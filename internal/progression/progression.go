// Package progression owns the deterministic gameplay state loop that makes a
// mission visibly advance: confirming evidence, unlocking the next map
// location, and submitting a mid-mission hypothesis. Every state-changing
// action returns the same envelope so the client can react uniformly
// (refresh map, animate unlock, append timeline, show next action).
//
// Rules: the AI never mutates state here — these are deterministic player
// actions validated and applied by the backend. Hidden truth is never exposed;
// the hypothesis verdict is computed from player-visible evidence coverage only.
package progression

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/missionevent"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

// Ownership guards mission access. Implemented by mission.Service.
type Ownership interface {
	EnsureOwnedActive(ctx context.Context, userID, missionID uuid.UUID) error
}

// Envelope is the uniform result of every state-changing progression action.
type Envelope struct {
	Message                string           `json:"message"`
	StateChanges           []StateChange    `json:"state_changes"`
	TimelineEvents         []string         `json:"timeline_events"`
	UnlockedLocations      []UnlockedLoc    `json:"unlocked_locations"`
	NextRecommendedActions []RecommendedAct `json:"next_recommended_actions"`
}

type StateChange struct {
	Entity string `json:"entity"` // clue | location | mission
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
}

type UnlockedLoc struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Reason string    `json:"reason"`
}

type RecommendedAct struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

// HypothesisVerdict is the deterministic checkpoint outcome.
type HypothesisVerdict string

const (
	VerdictTooEarly          HypothesisVerdict = "too_early"
	VerdictUnsupported       HypothesisVerdict = "unsupported"
	VerdictPartiallyCorrect  HypothesisVerdict = "partially_correct"
	minEvidenceForHypothesis                   = 2
)

// HypothesisResult is returned by SubmitHypothesis.
type HypothesisResult struct {
	Verdict           HypothesisVerdict `json:"verdict"`
	Feedback          string            `json:"feedback"`
	NeedsMoreEvidence bool              `json:"needs_more_evidence"`
	ConfirmedCount    int               `json:"confirmed_count"`
	Envelope          Envelope          `json:"progression"`
}

type Service struct {
	clues     clue.Repository
	locations gamemap.Repository
	recorder  *missionevent.Recorder
	owner     Ownership
	log       *slog.Logger
}

func NewService(clues clue.Repository, locations gamemap.Repository, recorder *missionevent.Recorder, owner Ownership, log *slog.Logger) *Service {
	return &Service{clues: clues, locations: locations, recorder: recorder, owner: owner, log: log}
}

// ConfirmEvidence promotes a discovered/inspected clue to confirmed, records the
// timeline event, and runs the unlock engine (a confirmation may open the next
// locked location). Idempotent: confirming an already-confirmed clue does not
// unlock again.
func (s *Service) ConfirmEvidence(ctx context.Context, userID, missionID, clueID uuid.UUID) (*Envelope, error) {
	if err := s.owner.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}
	c, err := s.clues.GetByID(ctx, missionID, clueID)
	if err != nil {
		return nil, err
	}
	if !c.Discovered {
		return nil, apperrors.NotFound("clue_not_found", "clue not found")
	}

	env := &Envelope{
		StateChanges:           []StateChange{},
		TimelineEvents:         []string{},
		UnlockedLocations:      []UnlockedLoc{},
		NextRecommendedActions: []RecommendedAct{},
	}

	already := c.LifecycleStatus() == clue.StatusConfirmed
	if already {
		env.Message = "This evidence is already confirmed."
		s.appendNextActions(ctx, missionID, env)
		return env, nil
	}

	from := c.LifecycleStatus()
	if err := s.clues.SetStatus(ctx, clueID, clue.StatusConfirmed); err != nil {
		return nil, err
	}
	env.StateChanges = append(env.StateChanges, StateChange{
		Entity: "clue", ID: clueID.String(), From: from, To: clue.StatusConfirmed,
	})
	s.recorder.Emit(ctx, missionID, "evidence_confirmed", map[string]any{
		"clue_id": clueID, "title": c.Title,
	})
	env.TimelineEvents = append(env.TimelineEvents, "evidence_confirmed")

	// Unlock engine: confirming a piece of evidence opens the next locked spot.
	s.unlockNext(ctx, missionID, "Unlocked after confirming evidence: "+c.Title, env)

	s.appendNextActions(ctx, missionID, env)
	env.Message = "Evidence confirmed."
	return env, nil
}

// unlockNext opens the earliest still-locked location (if any) and records it.
func (s *Service) unlockNext(ctx context.Context, missionID uuid.UUID, reason string, env *Envelope) {
	locked, err := s.locations.ListLocked(ctx, missionID)
	if err != nil {
		s.log.Error("list locked locations", "error", err)
		return
	}
	if len(locked) == 0 {
		return
	}
	next := locked[0]
	if err := s.locations.UpdateStatus(ctx, next.ID, gamemap.StatusDiscovered); err != nil {
		s.log.Error("unlock location", "error", err)
		return
	}
	env.StateChanges = append(env.StateChanges, StateChange{
		Entity: "location", ID: next.ID.String(), From: gamemap.StatusLocked, To: gamemap.StatusDiscovered,
	})
	env.UnlockedLocations = append(env.UnlockedLocations, UnlockedLoc{
		ID: next.ID, Name: next.Name, Reason: reason,
	})
	s.recorder.Emit(ctx, missionID, "location_unlocked", map[string]any{
		"location_id": next.ID, "name": next.Name, "reason": reason,
	})
	env.TimelineEvents = append(env.TimelineEvents, "location_unlocked")
}

// SubmitHypothesis records a mid-mission working theory and returns a
// deterministic checkpoint verdict from confirmed-evidence coverage. It never
// claims correct/wrong (that is the final JudgeAgent) and never reveals truth.
func (s *Service) SubmitHypothesis(ctx context.Context, userID, missionID uuid.UUID, answer string, clueIDs []uuid.UUID) (*HypothesisResult, error) {
	if err := validator.New().
		Required("answer", answer).MaxLen("answer", answer, 2000).
		Err(); err != nil {
		return nil, err
	}
	if err := s.owner.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}
	confirmed, err := s.clues.CountConfirmed(ctx, missionID)
	if err != nil {
		return nil, err
	}
	// Count how many of the linked clues are actually confirmed evidence.
	linkedConfirmed := 0
	for _, id := range clueIDs {
		c, err := s.clues.GetByID(ctx, missionID, id)
		if err != nil {
			continue
		}
		if c.LifecycleStatus() == clue.StatusConfirmed {
			linkedConfirmed++
		}
	}

	res := &HypothesisResult{
		ConfirmedCount: confirmed,
		Envelope: Envelope{
			StateChanges:           []StateChange{},
			TimelineEvents:         []string{"hypothesis_submitted"},
			UnlockedLocations:      []UnlockedLoc{},
			NextRecommendedActions: []RecommendedAct{},
		},
	}

	switch {
	case confirmed < minEvidenceForHypothesis:
		res.Verdict = VerdictTooEarly
		res.NeedsMoreEvidence = true
		res.Feedback = "It is too early to commit to a theory — confirm more evidence first, then link it to your hypothesis."
	case linkedConfirmed == 0:
		res.Verdict = VerdictUnsupported
		res.NeedsMoreEvidence = true
		res.Feedback = "Your theory is not yet backed by confirmed evidence. Link confirmed clues that support it before submitting a final decision."
	default:
		res.Verdict = VerdictPartiallyCorrect
		res.NeedsMoreEvidence = false
		res.Feedback = "Your theory is consistent with the evidence you have confirmed. When you are confident, submit your final decision."
		res.Envelope.NextRecommendedActions = append(res.Envelope.NextRecommendedActions, RecommendedAct{
			Type: "prepare_final", Title: "Prepare your final decision",
		})
	}

	s.recorder.Emit(ctx, missionID, "hypothesis_submitted", map[string]any{
		"verdict": string(res.Verdict), "confirmed_count": confirmed,
		"summary": summarize(answer),
	})
	res.Envelope.Message = res.Feedback
	return res, nil
}

// appendNextActions suggests the obvious next step after a confirm.
func (s *Service) appendNextActions(ctx context.Context, missionID uuid.UUID, env *Envelope) {
	confirmed, err := s.clues.CountConfirmed(ctx, missionID)
	if err != nil {
		return
	}
	if confirmed >= minEvidenceForHypothesis {
		env.NextRecommendedActions = append(env.NextRecommendedActions, RecommendedAct{
			Type: "submit_hypothesis", Title: "Test a hypothesis with your confirmed evidence",
		})
	}
}

// summarize keeps a short, player-safe snippet of the hypothesis for the log.
func summarize(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}
