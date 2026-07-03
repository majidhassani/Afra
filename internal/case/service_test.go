package cases

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"casemind/internal/evidence"
	"casemind/internal/history"
	"casemind/internal/suspect"
	apperrors "casemind/pkg/errors"
)

// fakeCaseRepo mimics the SQL contract: lookups are always scoped by
// (caseID, userID), so another user's case is indistinguishable from a
// missing one.
type fakeCaseRepo struct {
	items map[uuid.UUID]*Case
}

func (r *fakeCaseRepo) Create(_ context.Context, c *Case) error {
	c.ID = uuid.New()
	r.items[c.ID] = c
	return nil
}

func (r *fakeCaseRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]Case, error) {
	out := []Case{}
	for _, c := range r.items {
		if c.UserID == userID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (r *fakeCaseRepo) GetForUser(_ context.Context, userID, caseID uuid.UUID) (*Case, error) {
	c, ok := r.items[caseID]
	if !ok || c.UserID != userID {
		return nil, apperrors.NotFound("case_not_found", "case not found")
	}
	return c, nil
}

func (r *fakeCaseRepo) UpdateStatus(_ context.Context, caseID uuid.UUID, status string) error {
	if c, ok := r.items[caseID]; ok {
		c.Status = status
	}
	return nil
}

func (r *fakeCaseRepo) SetGeneratedContent(_ context.Context, caseID uuid.UUID, title, summary string) error {
	return nil
}

func (r *fakeCaseRepo) MarkSolved(_ context.Context, caseID uuid.UUID) error {
	return r.UpdateStatus(nil, caseID, StatusSolved)
}

type fakeSuspectRepo struct{ suspect.Repository }

func (fakeSuspectRepo) ListByCase(_ context.Context, _ uuid.UUID) ([]suspect.Suspect, error) {
	return []suspect.Suspect{}, nil
}

type fakeEvidenceRepo struct{ evidence.Repository }

func (fakeEvidenceRepo) ListDiscovered(_ context.Context, _ uuid.UUID) ([]evidence.Evidence, error) {
	return []evidence.Evidence{}, nil
}

type fakeEventRepo struct{}

func (fakeEventRepo) Insert(_ context.Context, caseID uuid.UUID, t string, p []byte) (*history.CaseEvent, error) {
	return &history.CaseEvent{CaseID: caseID, Type: t}, nil
}

func (fakeEventRepo) ListByCase(_ context.Context, _ uuid.UUID, _ int) ([]history.CaseEvent, error) {
	return []history.CaseEvent{}, nil
}

type fakeStats struct{}

func (fakeStats) IncrementTotalCases(_ context.Context, _ uuid.UUID) error { return nil }

func newOwnershipTestService(t *testing.T) (*Service, uuid.UUID, uuid.UUID) {
	t.Helper()
	repo := &fakeCaseRepo{items: map[uuid.UUID]*Case{}}
	owner := uuid.New()
	c := &Case{UserID: owner, Type: "murder", Difficulty: "easy", Status: StatusOpen}
	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	svc := NewService(repo, fakeSuspectRepo{}, fakeEvidenceRepo{}, fakeEventRepo{}, fakeStats{}, nil, slog.Default())
	return svc, owner, c.ID
}

// TestOwnershipEnforcement: every case accessor must 404 for another user.
func TestOwnershipEnforcement(t *testing.T) {
	ctx := context.Background()
	svc, owner, caseID := newOwnershipTestService(t)
	stranger := uuid.New()

	if _, err := svc.Get(ctx, owner, caseID); err != nil {
		t.Fatalf("owner should access own case: %v", err)
	}
	if err := svc.EnsureOwned(ctx, owner, caseID); err != nil {
		t.Fatalf("owner should pass guard: %v", err)
	}

	if _, err := svc.Get(ctx, stranger, caseID); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("stranger must get not-found, got %v", err)
	}
	if err := svc.EnsureOwned(ctx, stranger, caseID); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("stranger must fail guard with not-found, got %v", err)
	}
	if err := svc.EnsureOwnedOpen(ctx, stranger, caseID); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("stranger must fail open-guard with not-found, got %v", err)
	}
	if _, _, _, err := svc.SolveInfo(ctx, stranger, caseID); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("stranger must fail solve-info with not-found, got %v", err)
	}
	if _, err := svc.Archive(ctx, stranger, caseID); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("stranger must not archive, got %v", err)
	}
	if _, err := svc.Events(ctx, stranger, caseID, 10); !apperrors.Is(err, apperrors.KindNotFound) {
		t.Fatalf("stranger must not read events, got %v", err)
	}

	// Listing only returns own cases.
	own, err := svc.List(ctx, stranger)
	if err != nil {
		t.Fatal(err)
	}
	if len(own) != 0 {
		t.Fatal("stranger sees someone else's cases")
	}
}

func TestEnsureOwnedOpenRejectsNonOpenStatus(t *testing.T) {
	ctx := context.Background()
	svc, owner, caseID := newOwnershipTestService(t)
	if err := svc.EnsureOwnedOpen(ctx, owner, caseID); err != nil {
		t.Fatalf("open case should pass: %v", err)
	}
	if _, err := svc.Archive(ctx, owner, caseID); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureOwnedOpen(ctx, owner, caseID); !apperrors.Is(err, apperrors.KindConflict) {
		t.Fatalf("archived case must fail open-guard with conflict, got %v", err)
	}
}

func TestCreateValidatesTypeAndDifficulty(t *testing.T) {
	ctx := context.Background()
	svc, owner, _ := newOwnershipTestService(t)
	if _, err := svc.Create(ctx, owner, "heist", "easy"); !apperrors.Is(err, apperrors.KindInvalid) {
		t.Fatalf("expected invalid case type error, got %v", err)
	}
	if _, err := svc.Create(ctx, owner, "murder", "nightmare"); !apperrors.Is(err, apperrors.KindInvalid) {
		t.Fatalf("expected invalid difficulty error, got %v", err)
	}
	if _, err := svc.Create(ctx, owner, "murder", "easy", "spanish"); !apperrors.Is(err, apperrors.KindInvalid) {
		t.Fatalf("expected invalid language error, got %v", err)
	}
}

func TestNormalizeLanguageAcceptsAliases(t *testing.T) {
	cases := map[string]string{
		"":        "en",
		"en":      "en",
		"English": "en",
		"انگلیسی": "en",
		"fa":      "fa",
		"Farsi":   "fa",
		"Persian": "fa",
		"فارسی":   "fa",
	}
	for in, want := range cases {
		if got := NormalizeLanguage(in); got != want {
			t.Fatalf("NormalizeLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}
