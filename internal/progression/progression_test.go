package progression

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/missionevent"
	"casemind/internal/notification"
)

// --- in-memory fakes -------------------------------------------------------

type fakeClueRepo struct {
	byID      map[uuid.UUID]*clue.Clue
	confirmed int
}

func (r *fakeClueRepo) Create(context.Context, *clue.Clue) error { return nil }
func (r *fakeClueRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*clue.Clue, error) {
	if c, ok := r.byID[id]; ok {
		return c, nil
	}
	return nil, clueNotFound()
}
func (r *fakeClueRepo) ListDiscovered(context.Context, uuid.UUID) ([]clue.Clue, error) {
	return nil, nil
}
func (r *fakeClueRepo) ListDiscoveredAtLocation(context.Context, uuid.UUID, uuid.UUID) ([]clue.Clue, error) {
	return nil, nil
}
func (r *fakeClueRepo) ListUndiscoveredAtLocation(context.Context, uuid.UUID, uuid.UUID) ([]clue.Clue, error) {
	return nil, nil
}
func (r *fakeClueRepo) FindUndiscoveredByTitle(context.Context, uuid.UUID, string) (*clue.Clue, error) {
	return nil, nil
}
func (r *fakeClueRepo) MarkDiscovered(context.Context, uuid.UUID) error         { return nil }
func (r *fakeClueRepo) AdjustReliability(context.Context, uuid.UUID, int) error { return nil }
func (r *fakeClueRepo) UpdateImage(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (r *fakeClueRepo) ListCritical(context.Context, uuid.UUID) ([]clue.Clue, error) {
	return nil, nil
}
func (r *fakeClueRepo) Counts(context.Context, uuid.UUID) (int, int, error) { return 0, 0, nil }
func (r *fakeClueRepo) SetStatus(_ context.Context, id uuid.UUID, status string) error {
	if c, ok := r.byID[id]; ok {
		if c.Status != clue.StatusConfirmed && status == clue.StatusConfirmed {
			r.confirmed++
		}
		c.Status = status
	}
	return nil
}
func (r *fakeClueRepo) CountConfirmed(context.Context, uuid.UUID) (int, error) {
	return r.confirmed, nil
}

func clueNotFound() error { return errNotFound{} }

type errNotFound struct{}

func (errNotFound) Error() string { return "not found" }

type fakeLocRepo struct {
	locked  []gamemap.Location
	updated map[uuid.UUID]string
}

func (r *fakeLocRepo) Create(context.Context, *gamemap.Location) error { return nil }
func (r *fakeLocRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*gamemap.Location, error) {
	return nil, nil
}
func (r *fakeLocRepo) ListByMission(context.Context, uuid.UUID) ([]gamemap.Location, error) {
	return nil, nil
}
func (r *fakeLocRepo) ListLocked(context.Context, uuid.UUID) ([]gamemap.Location, error) {
	out := []gamemap.Location{}
	for _, l := range r.locked {
		if r.updated[l.ID] == "" {
			out = append(out, l)
		}
	}
	return out, nil
}
func (r *fakeLocRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	if r.updated == nil {
		r.updated = map[uuid.UUID]string{}
	}
	r.updated[id] = status
	return nil
}
func (r *fakeLocRepo) UpdateRisk(context.Context, uuid.UUID, int) error { return nil }
func (r *fakeLocRepo) CountVisited(context.Context, uuid.UUID) (int, int, error) {
	return 0, 0, nil
}

type fakeOwner struct{}

func (fakeOwner) EnsureOwnedActive(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func newTestService(clues *fakeClueRepo, locs *fakeLocRepo) *Service {
	rec := missionevent.NewRecorder(&nullEventRepo{}, notification.NewBus(), slog.Default())
	return NewService(clues, locs, rec, fakeOwner{}, slog.Default())
}

type nullEventRepo struct{}

func (nullEventRepo) Insert(context.Context, uuid.UUID, string, []byte) (*missionevent.Event, error) {
	return &missionevent.Event{ID: uuid.New()}, nil
}
func (nullEventRepo) ListByMission(context.Context, uuid.UUID, int) ([]missionevent.Event, error) {
	return nil, nil
}

// --- tests -----------------------------------------------------------------

// TestConfirmEvidenceUnlocksNextLocation: confirming a discovered clue promotes
// it to confirmed and opens the earliest locked location with a reason.
func TestConfirmEvidenceUnlocksNextLocation(t *testing.T) {
	clueID := uuid.New()
	locID := uuid.New()
	clues := &fakeClueRepo{byID: map[uuid.UUID]*clue.Clue{
		clueID: {ID: clueID, Discovered: true, Status: clue.StatusDiscovered, Title: "Broken Collar"},
	}}
	locs := &fakeLocRepo{locked: []gamemap.Location{{ID: locID, Name: "Old Cabin", Status: gamemap.StatusLocked}}}
	svc := newTestService(clues, locs)

	env, err := svc.ConfirmEvidence(context.Background(), uuid.New(), uuid.New(), clueID)
	if err != nil {
		t.Fatalf("ConfirmEvidence: %v", err)
	}
	if clues.byID[clueID].Status != clue.StatusConfirmed {
		t.Errorf("clue status = %q, want confirmed", clues.byID[clueID].Status)
	}
	if len(env.UnlockedLocations) != 1 || env.UnlockedLocations[0].ID != locID {
		t.Fatalf("expected Old Cabin unlocked, got %+v", env.UnlockedLocations)
	}
	if env.UnlockedLocations[0].Reason == "" {
		t.Errorf("unlock must carry a reason")
	}
	if locs.updated[locID] != gamemap.StatusDiscovered {
		t.Errorf("location not moved to discovered")
	}
}

// TestConfirmEvidenceIdempotent: confirming an already-confirmed clue does not
// unlock a second location.
func TestConfirmEvidenceIdempotent(t *testing.T) {
	clueID := uuid.New()
	clues := &fakeClueRepo{byID: map[uuid.UUID]*clue.Clue{
		clueID: {ID: clueID, Discovered: true, Status: clue.StatusConfirmed, Title: "Already"},
	}}
	locs := &fakeLocRepo{locked: []gamemap.Location{{ID: uuid.New(), Name: "Cabin", Status: gamemap.StatusLocked}}}
	svc := newTestService(clues, locs)

	env, err := svc.ConfirmEvidence(context.Background(), uuid.New(), uuid.New(), clueID)
	if err != nil {
		t.Fatalf("ConfirmEvidence: %v", err)
	}
	if len(env.UnlockedLocations) != 0 {
		t.Errorf("already-confirmed clue must not unlock again: %+v", env.UnlockedLocations)
	}
}

// TestHypothesisTooEarly: fewer than the minimum confirmed clues → too_early.
func TestHypothesisTooEarly(t *testing.T) {
	clues := &fakeClueRepo{byID: map[uuid.UUID]*clue.Clue{}, confirmed: 1}
	svc := newTestService(clues, &fakeLocRepo{})
	res, err := svc.SubmitHypothesis(context.Background(), uuid.New(), uuid.New(), "It was the ranger", nil)
	if err != nil {
		t.Fatalf("SubmitHypothesis: %v", err)
	}
	if res.Verdict != VerdictTooEarly || !res.NeedsMoreEvidence {
		t.Errorf("verdict = %q needsMore=%v, want too_early/true", res.Verdict, res.NeedsMoreEvidence)
	}
}

// TestHypothesisUnsupported: enough confirmed evidence overall but none linked.
func TestHypothesisUnsupported(t *testing.T) {
	clues := &fakeClueRepo{byID: map[uuid.UUID]*clue.Clue{}, confirmed: 3}
	svc := newTestService(clues, &fakeLocRepo{})
	res, err := svc.SubmitHypothesis(context.Background(), uuid.New(), uuid.New(), "Theory", nil)
	if err != nil {
		t.Fatalf("SubmitHypothesis: %v", err)
	}
	if res.Verdict != VerdictUnsupported {
		t.Errorf("verdict = %q, want unsupported", res.Verdict)
	}
}

// TestHypothesisPartiallyCorrect: confirmed evidence linked → partially_correct
// with a prepare-final next action.
func TestHypothesisPartiallyCorrect(t *testing.T) {
	linked := uuid.New()
	clues := &fakeClueRepo{
		byID:      map[uuid.UUID]*clue.Clue{linked: {ID: linked, Discovered: true, Status: clue.StatusConfirmed}},
		confirmed: 3,
	}
	svc := newTestService(clues, &fakeLocRepo{})
	res, err := svc.SubmitHypothesis(context.Background(), uuid.New(), uuid.New(), "Theory", []uuid.UUID{linked})
	if err != nil {
		t.Fatalf("SubmitHypothesis: %v", err)
	}
	if res.Verdict != VerdictPartiallyCorrect {
		t.Fatalf("verdict = %q, want partially_correct", res.Verdict)
	}
	if len(res.Envelope.NextRecommendedActions) == 0 {
		t.Errorf("expected a prepare-final next action")
	}
}
