package solve

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	judgeagent "casemind/internal/agent/judge"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/casebible"
	"casemind/internal/history"
	llmmock "casemind/internal/llm/mock"
	"casemind/internal/memory"
	"casemind/internal/notification"
	"casemind/internal/suspect"
	apperrors "casemind/pkg/errors"
)

// --- fakes ---

type memAttempts struct{ items []Attempt }

func (r *memAttempts) Insert(_ context.Context, a *Attempt) error {
	a.ID = uuid.New()
	r.items = append(r.items, *a)
	return nil
}
func (r *memAttempts) ListByCase(_ context.Context, _ uuid.UUID) ([]Attempt, error) {
	return r.items, nil
}
func (r *memAttempts) CountByCase(_ context.Context, _ uuid.UUID) (int, error) {
	return len(r.items), nil
}

type fakeGateway struct {
	status string
	owner  uuid.UUID
}

func (g *fakeGateway) SolveInfo(_ context.Context, userID, _ uuid.UUID) (string, string, string, error) {
	if userID != g.owner {
		return "", "", "", apperrors.NotFound("case_not_found", "case not found")
	}
	return g.status, "medium", "A gallery owner was found dead.", nil
}
func (g *fakeGateway) MarkSolved(_ context.Context, _ uuid.UUID) error {
	g.status = "solved"
	return nil
}
func (g *fakeGateway) MarkFailed(_ context.Context, _ uuid.UUID) error {
	g.status = "failed"
	return nil
}
func (g *fakeGateway) EnsureOwned(_ context.Context, userID, _ uuid.UUID) error {
	if userID != g.owner {
		return apperrors.NotFound("case_not_found", "case not found")
	}
	return nil
}

type fakeDetectives struct {
	solved, failed int
	xp             int
}

func (d *fakeDetectives) ApplyCaseResult(_ context.Context, _ uuid.UUID, solved bool, xpDelta int) error {
	if solved {
		d.solved++
	} else {
		d.failed++
	}
	d.xp += xpDelta
	return nil
}

type fakeSuspects struct {
	byID map[uuid.UUID]*suspect.Suspect
}

func (r *fakeSuspects) Create(_ context.Context, _ *suspect.Suspect) error { return nil }
func (r *fakeSuspects) ListByCase(_ context.Context, _ uuid.UUID) ([]suspect.Suspect, error) {
	return nil, nil
}
func (r *fakeSuspects) GetByID(_ context.Context, _, id uuid.UUID) (*suspect.Suspect, error) {
	if s, ok := r.byID[id]; ok {
		return s, nil
	}
	return nil, apperrors.NotFound("suspect_not_found", "suspect not found")
}
func (r *fakeSuspects) UpdateInterrogationState(_ context.Context, _ uuid.UUID, _, _, _ int) error {
	return nil
}
func (r *fakeSuspects) AppendPrivateMemory(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

type fakeBibles struct{ bible *casebible.Bible }

func (r *fakeBibles) Create(_ context.Context, _ *casebible.Bible) error { return nil }
func (r *fakeBibles) GetByCaseID(_ context.Context, _ uuid.UUID) (*casebible.Bible, error) {
	return r.bible, nil
}

type memFacts struct{ facts []memory.Fact }

func (r *memFacts) Add(_ context.Context, caseID uuid.UUID, fact, source string) (*memory.Fact, error) {
	f := memory.Fact{ID: uuid.New(), CaseID: caseID, Fact: fact, Source: source}
	r.facts = append(r.facts, f)
	return &f, nil
}
func (r *memFacts) ListByCase(_ context.Context, _ uuid.UUID) ([]memory.Fact, error) {
	return r.facts, nil
}

type memRuns struct{}

func (memRuns) Insert(_ context.Context, _ *runtime.Run) error { return nil }

type memEvents struct{ events []string }

func (r *memEvents) Insert(_ context.Context, caseID uuid.UUID, t string, _ []byte) (*history.CaseEvent, error) {
	r.events = append(r.events, t)
	return &history.CaseEvent{ID: uuid.New(), CaseID: caseID, Type: t}, nil
}
func (r *memEvents) ListByCase(_ context.Context, _ uuid.UUID, _ int) ([]history.CaseEvent, error) {
	return nil, nil
}

type solveHarness struct {
	svc        *Service
	gateway    *fakeGateway
	detectives *fakeDetectives
	attempts   *memAttempts
	events     *memEvents
	owner      uuid.UUID
	caseID     uuid.UUID
	culpritID  uuid.UUID
	innocentID uuid.UUID
}

func newSolveHarness(t *testing.T) *solveHarness {
	t.Helper()
	owner, caseID := uuid.New(), uuid.New()
	culprit := &suspect.Suspect{ID: uuid.New(), CaseID: caseID, Name: "Elena Voss", IsCulprit: true}
	innocent := &suspect.Suspect{ID: uuid.New(), CaseID: caseID, Name: "Marcus Reed"}

	rt := runtime.New(llmmock.New(), memRuns{}, slog.Default())
	orch := orchestrator.New(rt)
	orch.Register(judgeagent.New())

	gateway := &fakeGateway{status: "open", owner: owner}
	detectives := &fakeDetectives{}
	attempts := &memAttempts{}
	events := &memEvents{}
	recorder := history.NewRecorder(events, notification.NewBus(), slog.Default())
	mem := memory.NewService(&memFacts{}, nil, slog.Default())

	svc := NewService(attempts, gateway, detectives,
		&fakeSuspects{byID: map[uuid.UUID]*suspect.Suspect{culprit.ID: culprit, innocent.ID: innocent}},
		&fakeBibles{bible: &casebible.Bible{
			CaseID: caseID, CulpritID: culprit.ID, Motive: "forgery exposure",
			Truth: json.RawMessage(`{}`),
		}},
		mem, orch, recorder, slog.Default())

	return &solveHarness{
		svc: svc, gateway: gateway, detectives: detectives, attempts: attempts, events: events,
		owner: owner, caseID: caseID, culpritID: culprit.ID, innocentID: innocent.ID,
	}
}

func TestSolveCorrectAccusation(t *testing.T) {
	h := newSolveHarness(t)
	verdict, err := h.svc.Solve(context.Background(), h.owner, h.caseID, Accusation{
		AccusedSuspectID: h.culpritID, Motive: "forgery exposure", Reasoning: "timeline contradictions",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !verdict.Correct {
		t.Fatal("accusing the culprit must be correct")
	}
	if verdict.CaseStatus != "solved" || h.gateway.status != "solved" {
		t.Fatalf("case must be solved, got %s", verdict.CaseStatus)
	}
	if h.detectives.solved != 1 || h.detectives.xp <= 0 {
		t.Fatalf("detective progression not applied: %+v", h.detectives)
	}
	if verdict.Score < 60 {
		t.Fatalf("correct accusation must score at least 60, got %d", verdict.Score)
	}
	// Feedback must never name-drop internals like the bible motive verbatim
	// beyond what the player already supplied — sanity: no forbidden keys.
	if strings.Contains(verdict.Feedback, "culprit_id") {
		t.Fatal("feedback leaks internals")
	}
}

func TestSolveWrongAccusationConsumesAttemptsThenFailsCase(t *testing.T) {
	h := newSolveHarness(t)
	ctx := context.Background()

	for i := 1; i <= MaxAttempts; i++ {
		verdict, err := h.svc.Solve(ctx, h.owner, h.caseID, Accusation{
			AccusedSuspectID: h.innocentID, Motive: "money", Reasoning: "he argued loudly",
		})
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if verdict.Correct {
			t.Fatal("accusing an innocent must be incorrect")
		}
		if verdict.Score > 50 {
			t.Fatalf("incorrect accusation must not score above 50, got %d", verdict.Score)
		}
		if i < MaxAttempts && verdict.CaseStatus != "open" {
			t.Fatalf("case must stay open after attempt %d, got %s", i, verdict.CaseStatus)
		}
	}
	if h.gateway.status != "failed" {
		t.Fatalf("case must fail after %d wrong attempts, got %s", MaxAttempts, h.gateway.status)
	}
	if h.detectives.failed != 1 {
		t.Fatal("failed case must be recorded on the detective profile")
	}

	// A fourth attempt must be rejected.
	if _, err := h.svc.Solve(ctx, h.owner, h.caseID, Accusation{
		AccusedSuspectID: h.culpritID, Motive: "m",
	}); !apperrors.Is(err, apperrors.KindConflict) {
		t.Fatalf("expected conflict after case failed, got %v", err)
	}
}

func TestSolveOwnershipEnforced(t *testing.T) {
	h := newSolveHarness(t)
	stranger := uuid.New()
	if _, err := h.svc.Solve(context.Background(), stranger, h.caseID, Accusation{
		AccusedSuspectID: h.culpritID, Motive: "m",
	}); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("stranger must get not-found, got %v", err)
	}
	if _, err := h.svc.Attempts(context.Background(), stranger, h.caseID); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("stranger must not list attempts, got %v", err)
	}
}

func TestSolveRejectsAccusedFromAnotherCase(t *testing.T) {
	h := newSolveHarness(t)
	if _, err := h.svc.Solve(context.Background(), h.owner, h.caseID, Accusation{
		AccusedSuspectID: uuid.New(), Motive: "m",
	}); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("unknown suspect must be not-found, got %v", err)
	}
}
