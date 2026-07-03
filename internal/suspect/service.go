package suspect

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	directoragent "casemind/internal/agent/director"
	"casemind/internal/agent/orchestrator"
	psychologyagent "casemind/internal/agent/psychology"
	"casemind/internal/agent/runtime"
	suspectagent "casemind/internal/agent/suspect"
	"casemind/internal/conversation"
	"casemind/internal/evidence"
	"casemind/internal/history"
	"casemind/internal/memory"
	"casemind/pkg/validator"
)

// CaseGateway is the slice of the case module this service needs; it is
// implemented by cases.Service.
type CaseGateway interface {
	EnsureOwned(ctx context.Context, userID, caseID uuid.UUID) error
	EnsureOwnedOpen(ctx context.Context, userID, caseID uuid.UUID) error
	SummaryOf(ctx context.Context, userID, caseID uuid.UUID) (string, error)
}

type Service struct {
	repo      Repository
	guard     CaseGateway
	convs     conversation.Repository
	evidences evidence.Repository
	memory    *memory.Service
	orch      *orchestrator.Orchestrator
	recorder  *history.Recorder
	log       *slog.Logger
}

func NewService(
	repo Repository,
	guard CaseGateway,
	convs conversation.Repository,
	evidences evidence.Repository,
	mem *memory.Service,
	orch *orchestrator.Orchestrator,
	recorder *history.Recorder,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo, guard: guard, convs: convs, evidences: evidences,
		memory: mem, orch: orch, recorder: recorder, log: log,
	}
}

func (s *Service) List(ctx context.Context, userID, caseID uuid.UUID) ([]PublicSuspect, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	suspects, err := s.repo.ListByCase(ctx, caseID)
	if err != nil {
		return nil, err
	}
	return PublicList(suspects), nil
}

// SuspectDetail is the suspect plus their interrogation transcript.
type SuspectDetail struct {
	Suspect  PublicSuspect          `json:"suspect"`
	Messages []conversation.Message `json:"messages"`
}

func (s *Service) Get(ctx context.Context, userID, caseID, suspectID uuid.UUID) (*SuspectDetail, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	sus, err := s.repo.GetByID(ctx, caseID, suspectID)
	if err != nil {
		return nil, err
	}
	conv, err := s.convs.FindOrCreate(ctx, caseID, suspectID)
	if err != nil {
		return nil, err
	}
	messages, err := s.convs.Recent(ctx, conv.ID, 50)
	if err != nil {
		return nil, err
	}
	return &SuspectDetail{Suspect: sus.Public(), Messages: messages}, nil
}

// InterrogationResult is the player-safe outcome of one exchange.
type InterrogationResult struct {
	Reply                 string                    `json:"reply"`
	Emotion               string                    `json:"emotion"`
	StressLevel           int                       `json:"stress_level"`
	TrustLevel            int                       `json:"trust_level"`
	UnlockedClues         []string                  `json:"unlocked_clues"`
	NewEvidence           []evidence.PublicEvidence `json:"new_evidence"`
	ContradictionDetected bool                      `json:"contradiction_detected"`
}

// Interrogate implements the interrogation flow from 05_CASE_ENGINE.md.
func (s *Service) Interrogate(ctx context.Context, userID, caseID, suspectID uuid.UUID, message string) (*InterrogationResult, error) {
	if err := validator.New().
		Required("message", message).MaxLen("message", message, 2000).
		Err(); err != nil {
		return nil, err
	}
	// 1-2. Authenticated detective + case ownership (+ playable status).
	if err := s.guard.EnsureOwnedOpen(ctx, userID, caseID); err != nil {
		return nil, err
	}
	// 3. Load suspect.
	sus, err := s.repo.GetByID(ctx, caseID, suspectID)
	if err != nil {
		return nil, err
	}
	// 4. Conversation memory.
	conv, err := s.convs.FindOrCreate(ctx, caseID, suspectID)
	if err != nil {
		return nil, err
	}
	recent, err := s.convs.Recent(ctx, conv.ID, 20)
	if err != nil {
		return nil, err
	}
	// 5. Discovered facts.
	facts := s.memory.FactTexts(ctx, caseID)
	// 6. Safe internal context.
	summary, err := s.guard.SummaryOf(ctx, userID, caseID)
	if err != nil {
		return nil, err
	}
	undiscovered, err := s.undiscoveredTitles(ctx, caseID)
	if err != nil {
		s.log.Error("list undiscovered evidence", "error", err)
	}

	injection := runtime.DetectInjection(message)
	input := suspectagent.Input{
		CaseSummary:      summary,
		SuspectName:      sus.Name,
		Age:              sus.Age,
		Job:              sus.Job,
		RelationToVictim: sus.RelationToVictim,
		Personality:      rawToMap(sus.Personality),
		KnownFacts:       rawToStrings(sus.KnownFacts),
		StressLevel:      sus.StressLevel,
		TrustLevel:       sus.TrustLevel,

		IsCulprit:     sus.IsCulprit,
		Secrets:       rawToStrings(sus.Secrets),
		LieProfile:    rawToMap(sus.LieProfile),
		PrivateMemory: rawToStrings(sus.PrivateMemory),

		DiscoveredFacts:            facts,
		UndiscoveredEvidenceTitles: undiscovered,
		RecentMessages:             toTurns(recent),
		PlayerMessage:              message,
		InjectionDetected:          len(injection) > 0,
	}

	// 7. Run SuspectAgent; 8. output already validated by the agent parser.
	outAny, err := s.orch.Run(ctx, suspectagent.Name, runtime.Task{
		Type: suspectagent.TaskType, CaseID: caseID, UserID: userID, Input: input,
	})
	if err != nil {
		// Fallback to a safe response: log the exchange, mutate nothing else.
		s.log.Error("suspect agent failed, using fallback", "error", err)
		return s.fallback(ctx, conv.ID, sus, message)
	}
	out := outAny.(*suspectagent.Output)

	// Domain output guard: the reply must never contain a secret verbatim.
	out.Reply = redactSecrets(out.Reply, rawToStrings(sus.Secrets))

	// 9. Store messages.
	if err := s.convs.AddMessage(ctx, &conversation.Message{
		ConversationID: conv.ID, Role: conversation.RoleDetective, Content: message,
	}); err != nil {
		return nil, err
	}
	if err := s.convs.AddMessage(ctx, &conversation.Message{
		ConversationID: conv.ID, Role: conversation.RoleSuspect, Content: out.Reply, Emotion: out.Emotion,
	}); err != nil {
		return nil, err
	}

	// 10. Update stress/trust within domain bounds.
	stress := clamp(sus.StressLevel+out.StressDelta, 0, 100)
	trust := clamp(sus.TrustLevel+out.TrustDelta, 0, 100)
	count := sus.InterrogationCount + 1
	if err := s.repo.UpdateInterrogationState(ctx, sus.ID, stress, trust, count); err != nil {
		return nil, err
	}
	if err := s.repo.AppendPrivateMemory(ctx, sus.ID,
		"Q: "+truncate(message, 200)+" | A: "+truncate(out.Reply, 200)); err != nil {
		s.log.Error("append private memory", "error", err)
	}

	// 11. Unlock clues and evidence if allowed.
	added := s.memory.AddFacts(ctx, caseID, "interrogation:"+sus.Name, out.UnlockedClues)
	clues := make([]string, 0, len(added))
	for _, f := range added {
		clues = append(clues, f.Fact)
	}
	newEvidence := s.revealEvidence(ctx, caseID, out.RevealEvidenceTitles)

	s.recorder.Emit(ctx, caseID, "interrogation_exchange", map[string]any{
		"suspect_id": sus.ID, "suspect_name": sus.Name,
		"stress_level": stress, "trust_level": trust,
		"unlocked_clues": clues,
	})

	// Post-steps: psychology insight and director pacing (best effort).
	s.postSteps(caseID, userID, sus, stress, trust, count, summary, recent, message, out.Reply, facts)

	// 12. Safe response.
	return &InterrogationResult{
		Reply:                 out.Reply,
		Emotion:               out.Emotion,
		StressLevel:           stress,
		TrustLevel:            trust,
		UnlockedClues:         clues,
		NewEvidence:           newEvidence,
		ContradictionDetected: out.ContradictionDetected,
	}, nil
}

func (s *Service) fallback(ctx context.Context, convID uuid.UUID, sus *Suspect, message string) (*InterrogationResult, error) {
	const safeReply = "The suspect looks at you for a long moment and says nothing. Perhaps another approach would work better."
	if err := s.convs.AddMessage(ctx, &conversation.Message{
		ConversationID: convID, Role: conversation.RoleDetective, Content: message,
	}); err != nil {
		return nil, err
	}
	if err := s.convs.AddMessage(ctx, &conversation.Message{
		ConversationID: convID, Role: conversation.RoleSuspect, Content: safeReply, Emotion: "silent",
	}); err != nil {
		return nil, err
	}
	return &InterrogationResult{
		Reply:         safeReply,
		Emotion:       "silent",
		StressLevel:   sus.StressLevel,
		TrustLevel:    sus.TrustLevel,
		UnlockedClues: []string{},
		NewEvidence:   []evidence.PublicEvidence{},
	}, nil
}

func (s *Service) undiscoveredTitles(ctx context.Context, caseID uuid.UUID) ([]string, error) {
	return s.evidences.ListUndiscoveredTitles(ctx, caseID)
}

func (s *Service) revealEvidence(ctx context.Context, caseID uuid.UUID, titles []string) []evidence.PublicEvidence {
	revealed := []evidence.PublicEvidence{}
	for _, title := range titles {
		e, err := s.evidences.FindUndiscoveredByTitle(ctx, caseID, title)
		if err != nil {
			continue // agent suggested something that doesn't exist — ignore
		}
		if err := s.evidences.MarkDiscovered(ctx, e.ID); err != nil {
			s.log.Error("mark evidence discovered", "error", err)
			continue
		}
		e.Discovered = true
		revealed = append(revealed, e.Public())
		s.recorder.Emit(ctx, caseID, "evidence_discovered", map[string]any{
			"evidence_id": e.ID, "title": e.Title,
		})
	}
	return revealed
}

// postSteps runs PsychologyAgent (high stress) and DirectorAgent (pacing)
// asynchronously; their output arrives as case events.
func (s *Service) postSteps(caseID, userID uuid.UUID, sus *Suspect, stress, trust, count int,
	summary string, recent []conversation.Message, question, reply string, facts []string) {

	turns := toTurns(recent)
	turns = append(turns,
		suspectagent.Turn{Role: conversation.RoleDetective, Content: question},
		suspectagent.Turn{Role: conversation.RoleSuspect, Content: reply},
	)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if stress >= 70 {
			psyTurns := make([]psychologyagent.Turn, 0, len(turns))
			for _, t := range turns {
				psyTurns = append(psyTurns, psychologyagent.Turn(t))
			}
			outAny, err := s.orch.Run(ctx, psychologyagent.Name, runtime.Task{
				Type: psychologyagent.TaskType, CaseID: caseID, UserID: userID,
				Input: psychologyagent.Input{
					SuspectName: sus.Name, Personality: rawToMap(sus.Personality),
					StressLevel: stress, TrustLevel: trust, RecentMessages: psyTurns,
				},
			})
			if err == nil {
				out := outAny.(*psychologyagent.Output)
				s.recorder.Emit(ctx, caseID, "psychology_insight", map[string]any{
					"suspect_id": sus.ID, "insight": out.Insight,
					"stress_assessment": out.StressAssessment, "recommended_approach": out.RecommendedApproach,
				})
			}
		}

		if count > 0 && count%3 == 0 {
			outAny, err := s.orch.Run(ctx, directoragent.Name, runtime.Task{
				Type: directoragent.TaskType, CaseID: caseID, UserID: userID,
				Input: directoragent.Input{
					CaseSummary: summary, DiscoveredFacts: facts, InterrogationCount: count,
				},
			})
			if err == nil {
				out := outAny.(*directoragent.Output)
				s.recorder.Emit(ctx, caseID, "director_hint", map[string]any{
					"hint": out.Hint, "tone": out.Tone,
				})
			}
		}
	}()
}

// --- helpers ---

func toTurns(messages []conversation.Message) []suspectagent.Turn {
	turns := make([]suspectagent.Turn, 0, len(messages))
	for _, m := range messages {
		turns = append(turns, suspectagent.Turn{Role: m.Role, Content: m.Content})
	}
	return turns
}

func rawToMap(raw json.RawMessage) map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func rawToStrings(raw json.RawMessage) []string {
	var s []string
	_ = json.Unmarshal(raw, &s)
	return s
}

// redactSecrets replaces any verbatim secret leak with an in-character
// deflection. This is a hard domain rule on top of prompt discipline.
func redactSecrets(reply string, secrets []string) string {
	lower := strings.ToLower(reply)
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(secret)) {
			return "I'm not going to talk about that."
		}
	}
	return reply
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
