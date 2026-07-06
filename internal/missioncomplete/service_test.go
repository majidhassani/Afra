package missioncomplete

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"casemind/internal/agent/missionjudge"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/interaction"
	llmmock "casemind/internal/llm/mock"
	"casemind/internal/mission"
	"casemind/internal/missionevent"
	"casemind/internal/notification"
	"casemind/internal/wallet"
	"casemind/internal/worldbible"
)

// --- fakes ---

type fakeGateway struct {
	snap       *mission.CompletionSnapshot
	finished   bool
	success    bool
	completed  map[string]bool
	resultJSON json.RawMessage
}

func (g *fakeGateway) CompletionSnapshot(_ context.Context, _, _ uuid.UUID) (*mission.CompletionSnapshot, error) {
	return g.snap, nil
}
func (g *fakeGateway) Finish(_ context.Context, _, _ uuid.UUID, result json.RawMessage, success bool, keys map[string]bool) error {
	g.finished = true
	g.success = success
	g.completed = keys
	g.resultJSON = result
	return nil
}

type fakeProfiles struct {
	applied   bool
	completed bool
	xp        int
}

func (p *fakeProfiles) ApplyMissionResult(_ context.Context, _ uuid.UUID, completed bool, xp int) error {
	p.applied = true
	p.completed = completed
	p.xp = xp
	return nil
}

// clue repo fake — only Counts and ListCritical matter here.
type fakeClues struct {
	discovered, total int
	critical          []clue.Clue
}

func (r *fakeClues) Create(context.Context, *clue.Clue) error { return nil }
func (r *fakeClues) GetByID(context.Context, uuid.UUID, uuid.UUID) (*clue.Clue, error) {
	return nil, nil
}
func (r *fakeClues) ListDiscovered(context.Context, uuid.UUID) ([]clue.Clue, error) { return nil, nil }
func (r *fakeClues) ListDiscoveredAtLocation(context.Context, uuid.UUID, uuid.UUID) ([]clue.Clue, error) {
	return nil, nil
}
func (r *fakeClues) ListUndiscoveredAtLocation(context.Context, uuid.UUID, uuid.UUID) ([]clue.Clue, error) {
	return nil, nil
}
func (r *fakeClues) FindUndiscoveredByTitle(context.Context, uuid.UUID, string) (*clue.Clue, error) {
	return nil, nil
}
func (r *fakeClues) MarkDiscovered(context.Context, uuid.UUID) error         { return nil }
func (r *fakeClues) AdjustReliability(context.Context, uuid.UUID, int) error { return nil }
func (r *fakeClues) Counts(context.Context, uuid.UUID) (int, int, error) {
	return r.discovered, r.total, nil
}
func (r *fakeClues) ListCritical(context.Context, uuid.UUID) ([]clue.Clue, error) {
	return r.critical, nil
}

// gamemap repo fake — only CountVisited matters.
type fakeLocations struct{ visited, total int }

func (r *fakeLocations) Create(context.Context, *gamemap.Location) error { return nil }
func (r *fakeLocations) GetByID(context.Context, uuid.UUID, uuid.UUID) (*gamemap.Location, error) {
	return nil, nil
}
func (r *fakeLocations) ListByMission(context.Context, uuid.UUID) ([]gamemap.Location, error) {
	return nil, nil
}
func (r *fakeLocations) UpdateStatus(context.Context, uuid.UUID, string) error { return nil }
func (r *fakeLocations) UpdateRisk(context.Context, uuid.UUID, int) error      { return nil }
func (r *fakeLocations) CountVisited(context.Context, uuid.UUID) (int, int, error) {
	return r.visited, r.total, nil
}

// interaction repo fake — only CountInteractedCharacters matters.
type interactionRepo struct{ chars int }

func (interactionRepo) FindOrCreate(context.Context, uuid.UUID, uuid.UUID, string, *uuid.UUID) (*interaction.Interaction, error) {
	return nil, nil
}
func (interactionRepo) AddMessage(context.Context, *interaction.Message) error { return nil }
func (interactionRepo) Recent(context.Context, uuid.UUID, int) ([]interaction.Message, error) {
	return nil, nil
}
func (interactionRepo) CountByMission(context.Context, uuid.UUID) (int, error) { return 0, nil }
func (r interactionRepo) CountInteractedCharacters(context.Context, uuid.UUID) (int, error) {
	return r.chars, nil
}

type fakeEvents struct{ events []missionevent.Event }

func (r *fakeEvents) Insert(_ context.Context, missionID uuid.UUID, t string, payload []byte) (*missionevent.Event, error) {
	ev := missionevent.Event{ID: uuid.New(), MissionID: missionID, Type: t, Payload: payload}
	r.events = append([]missionevent.Event{ev}, r.events...)
	return &ev, nil
}
func (r *fakeEvents) ListByMission(context.Context, uuid.UUID, int) ([]missionevent.Event, error) {
	return r.events, nil
}

type fakeBibles struct{ bible *worldbible.Bible }

func (r *fakeBibles) Create(context.Context, *worldbible.Bible) error { return nil }
func (r *fakeBibles) GetByMissionID(context.Context, uuid.UUID) (*worldbible.Bible, error) {
	return r.bible, nil
}
func (r *fakeBibles) UpdateHiddenState(context.Context, uuid.UUID, json.RawMessage) error {
	return nil
}

// in-memory wallet repo — enough for reserve/settle/credit/pricing.
type fakeWalletRepo struct{}

func (fakeWalletRepo) GetOrCreate(_ context.Context, userID uuid.UUID) (*wallet.Wallet, error) {
	return &wallet.Wallet{UserID: userID, Balance: 500}, nil
}
func (fakeWalletRepo) Reserve(_ context.Context, userID uuid.UUID, missionID *uuid.UUID, action string, amount int) (*wallet.Reservation, error) {
	return &wallet.Reservation{ID: uuid.New(), UserID: userID, MissionID: missionID, ActionType: action, Amount: amount, Status: wallet.ReservationActive}, nil
}
func (fakeWalletRepo) Settle(_ context.Context, res *wallet.Reservation, _ map[string]any) (*wallet.Transaction, error) {
	return &wallet.Transaction{ID: uuid.New(), UserID: res.UserID, Amount: -res.Amount}, nil
}
func (fakeWalletRepo) Release(context.Context, *wallet.Reservation) error { return nil }
func (fakeWalletRepo) Credit(_ context.Context, userID uuid.UUID, missionID *uuid.UUID, txType string, amount int, _ map[string]any) (*wallet.Transaction, error) {
	return &wallet.Transaction{ID: uuid.New(), UserID: userID, MissionID: missionID, Type: txType, Amount: amount}, nil
}
func (fakeWalletRepo) Transactions(context.Context, uuid.UUID, int) ([]wallet.Transaction, error) {
	return nil, nil
}
func (fakeWalletRepo) Pricing(context.Context) (map[string]int, error) {
	return map[string]int{wallet.ActionFinalJudgment: 10}, nil
}
func (fakeWalletRepo) InsertUsageLog(context.Context, *wallet.UsageLog) error { return nil }
func (fakeWalletRepo) CountAdClaimsSince(context.Context, uuid.UUID, time.Time) (int, error) {
	return 0, nil
}
func (fakeWalletRepo) InsertAdClaim(context.Context, uuid.UUID, int) error { return nil }
func (fakeWalletRepo) InsertReceipt(context.Context, uuid.UUID, string, string, string, int, string) error {
	return nil
}
func (fakeWalletRepo) SpentAndEarned(context.Context, uuid.UUID) (int, int, error) { return 0, 0, nil }

// --- harness ---

type harness struct {
	svc      *Service
	gateway  *fakeGateway
	profiles *fakeProfiles
	owner    uuid.UUID
	mission  uuid.UUID
}

func newHarness(t *testing.T, discovered, total, target int) *harness {
	t.Helper()
	owner, missionID := uuid.New(), uuid.New()

	rt := runtime.New(llmmock.New(), memRuns{}, slog.Default())
	orch := orchestrator.New(rt)
	orch.Register(missionjudge.New())

	gateway := &fakeGateway{snap: &mission.CompletionSnapshot{
		Status: mission.StatusActive, Type: "wildlife_rescue", Difficulty: "medium",
		Summary: "Cheetahs are vanishing from the reserve.",
		Objectives: []mission.Objective{
			{ID: "obj_source", Type: mission.ObjectiveRequired, Title: "Find the source", RequiredClues: total / 2},
			{ID: "obj_network", Type: mission.ObjectiveRequired, Title: "Identify the network", RequiredClues: target - total/2},
			{ID: "obj_trust", Type: mission.ObjectiveOptional, Title: "Win trust", RequiredClues: 1},
		},
		RequiredClueTarget: target,
		ElapsedMinutes:     600,
	}}
	profiles := &fakeProfiles{}
	events := &fakeEvents{}
	// Seed a discovered fact so the judge has some player memory.
	events.Insert(context.Background(), missionID, "fact_discovered", []byte(`{"fact":"collars were cut cleanly"}`))
	recorder := missionevent.NewRecorder(events, notification.NewBus(), slog.Default())

	walletSvc := wallet.NewService(fakeWalletRepo{}, true)
	guard := wallet.NewGuard(walletSvc, slog.Default())

	svc := NewService(gateway,
		&fakeClues{discovered: discovered, total: total, critical: []clue.Clue{
			{ID: uuid.New(), Title: "Torn Conservation Permit", Discovered: discovered >= total, Importance: "high"},
			{ID: uuid.New(), Title: "Altered Equipment Requisition", Discovered: true, Importance: "high"},
		}},
		&fakeLocations{visited: 3, total: 5},
		interactionRepo{chars: 2},
		events, &fakeBibles{bible: &worldbible.Bible{
			Truth:        json.RawMessage(`{"real_cause":"organized poaching network"}`),
			FailureRules: json.RawMessage(`["shipment leaves the reserve"]`),
		}},
		orch, guard, walletSvc, profiles, recorder, slog.Default())

	return &harness{svc: svc, gateway: gateway, profiles: profiles, owner: owner, mission: missionID}
}

type memRuns struct{}

func (memRuns) Insert(context.Context, *runtime.Run) error { return nil }

func TestCannotCompleteBeforeRequirements(t *testing.T) {
	h := newHarness(t, 1, 7, 4) // only 1 of 4 required clues found
	result, check, err := h.svc.Complete(context.Background(), h.owner, h.mission, Decision{
		Outcome: "The logistics officer ran a poaching network",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Fatal("mission must not be judged before requirements are met")
	}
	if check == nil || check.CanComplete {
		t.Fatalf("expected a not-ready completion check, got %+v", check)
	}
	if len(check.MissingRequirements) == 0 || check.Reason == "" {
		t.Fatalf("not-ready check must explain what is missing: %+v", check)
	}
	if h.gateway.finished {
		t.Fatal("mission must not be finalised when not ready")
	}
	if h.profiles.applied {
		t.Fatal("player must not be rewarded when the mission is not completable")
	}
}

func TestCanCompleteAfterRequirementsSucceeds(t *testing.T) {
	h := newHarness(t, 4, 7, 4) // requirements met, strong outcome
	result, check, err := h.svc.Complete(context.Background(), h.owner, h.mission, Decision{
		Outcome:   "The regional logistics officer ran a poaching network jamming the collars",
		Reasoning: "The permit, requisition, and fuel purchases line up.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if check != nil {
		t.Fatalf("mission should be ready, got not-ready check %+v", check)
	}
	if result == nil {
		t.Fatal("a ready mission must return a result")
	}
	if !result.Success {
		t.Fatalf("a strong, evidence-backed outcome should succeed: %+v", result)
	}
	if result.ResultTitle == "" || result.ResultSummary == "" {
		t.Fatal("result must clearly title and explain the outcome")
	}
	if result.Stars < 4 || result.Score < 60 {
		t.Fatalf("a success should score well: score=%d stars=%d", result.Score, result.Stars)
	}
	if len(result.CompletedObjectives) == 0 {
		t.Fatal("a successful mission should list completed objectives")
	}
	if result.XPReward <= 0 || result.CoinReward <= 0 {
		t.Fatalf("a success should reward XP and coins: %+v", result)
	}
	if !h.gateway.finished || !h.gateway.success {
		t.Fatal("gateway must finalise the mission as a success")
	}
	if !h.profiles.applied || !h.profiles.completed {
		t.Fatal("player profile must record the completed mission")
	}
}

func TestCompleteFailureExplainsResult(t *testing.T) {
	h := newHarness(t, 4, 7, 4) // requirements met, but a weak/wrong outcome
	result, check, err := h.svc.Complete(context.Background(), h.owner, h.mission, Decision{
		Outcome:   "I believe the head ranger is personally responsible.",
		Reasoning: "He seemed nervous.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if check != nil {
		t.Fatalf("mission was ready; expected a judged result, got %+v", check)
	}
	if result.Success {
		t.Fatal("an unsupported accusation must not succeed")
	}
	if result.ResultTitle == "" || result.ResultSummary == "" {
		t.Fatal("a failed mission must still clearly explain why")
	}
	if len(result.FailedObjectives) == 0 {
		t.Fatal("a failed mission should list which objectives failed")
	}
	if !h.gateway.finished || h.gateway.success {
		t.Fatal("gateway must finalise the mission as a failure")
	}
}
