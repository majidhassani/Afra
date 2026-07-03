package solve

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	directoragent "casemind/internal/agent/director"
	judgeagent "casemind/internal/agent/judge"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/casebible"
	"casemind/internal/history"
	"casemind/internal/memory"
	"casemind/internal/suspect"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

// MaxAttempts before the case fails.
const MaxAttempts = 3

// CaseGateway is implemented by cases.Service.
type CaseGateway interface {
	SolveInfo(ctx context.Context, userID, caseID uuid.UUID) (status, difficulty, summary string, err error)
	MarkSolved(ctx context.Context, caseID uuid.UUID) error
	MarkFailed(ctx context.Context, caseID uuid.UUID) error
	EnsureOwned(ctx context.Context, userID, caseID uuid.UUID) error
}

// DetectiveGateway is implemented by detective.Service.
type DetectiveGateway interface {
	ApplyCaseResult(ctx context.Context, userID uuid.UUID, solved bool, xpDelta int) error
}

// XPForDifficulty mirrors the case module's base XP (kept local to avoid a
// dependency on the case package).
func XPForDifficulty(difficulty string) int {
	switch difficulty {
	case "easy":
		return 100
	case "medium":
		return 250
	case "hard":
		return 500
	case "expert":
		return 1000
	default:
		return 100
	}
}

type Service struct {
	repo       Repository
	gateway    CaseGateway
	detectives DetectiveGateway
	suspects   suspect.Repository
	bibles     casebible.Repository
	memory     *memory.Service
	orch       *orchestrator.Orchestrator
	recorder   *history.Recorder
	log        *slog.Logger
}

func NewService(
	repo Repository,
	gateway CaseGateway,
	detectives DetectiveGateway,
	suspects suspect.Repository,
	bibles casebible.Repository,
	mem *memory.Service,
	orch *orchestrator.Orchestrator,
	recorder *history.Recorder,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo, gateway: gateway, detectives: detectives, suspects: suspects,
		bibles: bibles, memory: mem, orch: orch, recorder: recorder, log: log,
	}
}

type Accusation struct {
	AccusedSuspectID uuid.UUID `json:"accused_suspect_id"`
	Motive           string    `json:"motive"`
	Reasoning        string    `json:"reasoning"`
}

// Verdict is the player-safe result of a solve attempt.
type Verdict struct {
	Correct           bool   `json:"correct"`
	Score             int    `json:"score"`
	Feedback          string `json:"feedback"`
	AttemptsUsed      int    `json:"attempts_used"`
	AttemptsRemaining int    `json:"attempts_remaining"`
	CaseStatus        string `json:"case_status"`
	XPAwarded         int    `json:"xp_awarded"`
}

// Solve implements the solve flow from 05_CASE_ENGINE.md.
func (s *Service) Solve(ctx context.Context, userID, caseID uuid.UUID, acc Accusation) (*Verdict, error) {
	if err := validator.New().
		Required("motive", acc.Motive).MaxLen("motive", acc.Motive, 1000).
		MaxLen("reasoning", acc.Reasoning, 5000).
		Err(); err != nil {
		return nil, err
	}
	if acc.AccusedSuspectID == uuid.Nil {
		return nil, apperrors.Invalid("validation_failed", "accused_suspect_id is required")
	}

	// 1. Ownership + playable status.
	status, difficulty, summary, err := s.gateway.SolveInfo(ctx, userID, caseID)
	if err != nil {
		return nil, err
	}
	if status != "open" {
		return nil, apperrors.Conflict("case_not_open", "this case is not open (status: "+status+")")
	}
	used, err := s.repo.CountByCase(ctx, caseID)
	if err != nil {
		return nil, err
	}
	if used >= MaxAttempts {
		return nil, apperrors.Conflict("no_attempts_left", "no solve attempts remaining")
	}

	// Accused suspect must belong to this case.
	accused, err := s.suspects.GetByID(ctx, caseID, acc.AccusedSuspectID)
	if err != nil {
		return nil, err
	}

	// 2. Load the Case Bible internally; domain decides correctness.
	bible, err := s.bibles.GetByCaseID(ctx, caseID)
	if err != nil {
		return nil, err
	}
	correct := accused.ID == bible.CulpritID

	// 3. JudgeAgent narrates and scores. If it fails, no attempt is
	// consumed and no state changes.
	attemptNumber := used + 1
	outAny, err := s.orch.Run(ctx, judgeagent.Name, runtime.Task{
		Type: judgeagent.TaskType, CaseID: caseID, UserID: userID,
		Input: judgeagent.Input{
			CaseSummary:       summary,
			AccusedName:       accused.Name,
			AccusedMotive:     acc.Motive,
			Reasoning:         acc.Reasoning,
			AccusationCorrect: correct,
			RealMotive:        bible.Motive,
			DiscoveredFacts:   s.memory.FactTexts(ctx, caseID),
			AttemptNumber:     attemptNumber,
			AttemptsRemaining: MaxAttempts - attemptNumber,
		},
	})
	if err != nil {
		return nil, err
	}
	judged := outAny.(*judgeagent.Output)

	// 4-5. Domain rules cap the score, then persist the attempt.
	score := judged.Score
	if correct && score < 60 {
		score = 60
	}
	if !correct && score > 50 {
		score = 50
	}
	attempt := &Attempt{
		CaseID:           caseID,
		AccusedSuspectID: accused.ID,
		Motive:           acc.Motive,
		Reasoning:        acc.Reasoning,
		Correct:          correct,
		Score:            score,
		Feedback:         judged.Feedback,
	}
	if err := s.repo.Insert(ctx, attempt); err != nil {
		return nil, err
	}

	// 6-7. Update case status and detective progression.
	caseStatus := "open"
	xp := 0
	switch {
	case correct:
		if err := s.gateway.MarkSolved(ctx, caseID); err != nil {
			return nil, err
		}
		caseStatus = "solved"
		xp = XPForDifficulty(difficulty) * score / 100
		if err := s.detectives.ApplyCaseResult(ctx, userID, true, xp); err != nil {
			s.log.Error("apply case result", "error", err)
		}
		s.recorder.Emit(ctx, caseID, "case_solved", map[string]any{
			"score": score, "xp_awarded": xp,
		})
	case attemptNumber >= MaxAttempts:
		if err := s.gateway.MarkFailed(ctx, caseID); err != nil {
			return nil, err
		}
		caseStatus = "failed"
		if err := s.detectives.ApplyCaseResult(ctx, userID, false, 0); err != nil {
			s.log.Error("apply case result", "error", err)
		}
		s.recorder.Emit(ctx, caseID, "case_failed", map[string]any{
			"attempts_used": attemptNumber,
		})
	default:
		s.recorder.Emit(ctx, caseID, "solve_attempt_failed", map[string]any{
			"attempts_used": attemptNumber, "attempts_remaining": MaxAttempts - attemptNumber,
		})
		s.directorNudge(caseID, userID, summary, attemptNumber)
	}

	// 8. Safe feedback only.
	return &Verdict{
		Correct:           correct,
		Score:             score,
		Feedback:          judged.Feedback,
		AttemptsUsed:      attemptNumber,
		AttemptsRemaining: MaxAttempts - attemptNumber,
		CaseStatus:        caseStatus,
		XPAwarded:         xp,
	}, nil
}

func (s *Service) Attempts(ctx context.Context, userID, caseID uuid.UUID) ([]Attempt, error) {
	if err := s.gateway.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	return s.repo.ListByCase(ctx, caseID)
}

// directorNudge emits a pacing hint after a failed attempt (best effort).
func (s *Service) directorNudge(caseID, userID uuid.UUID, summary string, failedAttempts int) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		outAny, err := s.orch.Run(ctx, directoragent.Name, runtime.Task{
			Type: directoragent.TaskType, CaseID: caseID, UserID: userID,
			Input: directoragent.Input{
				CaseSummary:     summary,
				DiscoveredFacts: s.memory.FactTexts(ctx, caseID),
				FailedAttempts:  failedAttempts,
			},
		})
		if err != nil {
			return
		}
		out := outAny.(*directoragent.Output)
		s.recorder.Emit(ctx, caseID, "director_hint", map[string]any{
			"hint": out.Hint, "tone": out.Tone,
		})
	}()
}
