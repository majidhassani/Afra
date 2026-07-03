package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrors "casemind/pkg/errors"
)

type fakeUserRepo struct {
	byEmail map[string]*User
	byID    map[uuid.UUID]*User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]*User{}, byID: map[uuid.UUID]*User{}}
}

func (r *fakeUserRepo) Create(_ context.Context, u *User) error {
	if _, exists := r.byEmail[u.Email]; exists {
		return apperrors.Conflict("email_taken", "email taken")
	}
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	r.byEmail[u.Email] = u
	r.byID[u.ID] = u
	return nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*User, error) {
	if u, ok := r.byEmail[email]; ok {
		return u, nil
	}
	return nil, apperrors.NotFound("user_not_found", "not found")
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*User, error) {
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, apperrors.NotFound("user_not_found", "not found")
}

type fakeTokenRepo struct {
	byHash map[string]*RefreshToken
}

func newFakeTokenRepo() *fakeTokenRepo { return &fakeTokenRepo{byHash: map[string]*RefreshToken{}} }

func (r *fakeTokenRepo) Create(_ context.Context, t *RefreshToken) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	r.byHash[t.TokenHash] = t
	return nil
}

func (r *fakeTokenRepo) GetByHash(_ context.Context, hash string) (*RefreshToken, error) {
	if t, ok := r.byHash[hash]; ok {
		return t, nil
	}
	return nil, apperrors.Unauthorized("invalid_refresh_token", "invalid")
}

func (r *fakeTokenRepo) Revoke(_ context.Context, id uuid.UUID) error {
	for _, t := range r.byHash {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (r *fakeTokenRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	for _, t := range r.byHash {
		if t.UserID == userID {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

type fakeProfiles struct{ created []uuid.UUID }

func (p *fakeProfiles) CreateProfile(_ context.Context, userID uuid.UUID) error {
	p.created = append(p.created, userID)
	return nil
}

func newTestService() (*Service, *fakeProfiles) {
	profiles := &fakeProfiles{}
	svc := NewService(
		newFakeUserRepo(), newFakeTokenRepo(), profiles,
		NewTokenManager("test-secret", 15*time.Minute), 24*time.Hour,
	)
	return svc, profiles
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	ctx := context.Background()
	svc, profiles := newTestService()

	user, tokens, err := svc.Register(ctx, "sherlock@example.com", "strongpass123", "Sherlock")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected token pair")
	}
	if len(profiles.created) != 1 || profiles.created[0] != user.ID {
		t.Fatal("detective profile was not created on register")
	}

	// Login.
	_, loginTokens, err := svc.Login(ctx, "sherlock@example.com", "strongpass123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	// Wrong password must not authenticate and must not reveal user existence.
	if _, _, err := svc.Login(ctx, "sherlock@example.com", "wrongpass"); !apperrors.Is(err, apperrors.KindUnauthorized) {
		t.Fatalf("expected unauthorized for wrong password, got %v", err)
	}
	if _, _, err := svc.Login(ctx, "nobody@example.com", "whatever"); !apperrors.Is(err, apperrors.KindUnauthorized) {
		t.Fatalf("expected unauthorized for unknown email, got %v", err)
	}

	// Refresh rotates: old token becomes invalid.
	newTokens, err := svc.Refresh(ctx, loginTokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := svc.Refresh(ctx, loginTokens.RefreshToken); err == nil {
		t.Fatal("expected rotated refresh token to be rejected")
	}

	// Logout revokes.
	if err := svc.Logout(ctx, newTokens.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(ctx, newTokens.RefreshToken); err == nil {
		t.Fatal("expected revoked refresh token to be rejected")
	}
}

func TestRegisterValidation(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()

	cases := []struct{ email, password, name string }{
		{"", "strongpass123", "X"},
		{"not-an-email", "strongpass123", "X"},
		{"a@b.co", "short", "X"},
		{"a@b.co", "strongpass123", ""},
	}
	for _, c := range cases {
		if _, _, err := svc.Register(ctx, c.email, c.password, c.name); !apperrors.Is(err, apperrors.KindInvalid) {
			t.Errorf("expected invalid for %+v, got %v", c, err)
		}
	}

	if _, _, err := svc.Register(ctx, "dupe@example.com", "strongpass123", "A"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Register(ctx, "dupe@example.com", "strongpass123", "B"); !apperrors.Is(err, apperrors.KindConflict) {
		t.Errorf("expected conflict for duplicate email, got %v", err)
	}
}

func TestJWTRoundTrip(t *testing.T) {
	tm := NewTokenManager("secret", time.Minute)
	userID := uuid.New()
	token, _, err := tm.Generate(userID)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := tm.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != userID {
		t.Fatal("round-tripped user id mismatch")
	}
	if _, err := tm.Parse(token + "tampered"); err == nil {
		t.Fatal("expected tampered token to be rejected")
	}
	other := NewTokenManager("other-secret", time.Minute)
	if _, err := other.Parse(token); err == nil {
		t.Fatal("expected token signed with different secret to be rejected")
	}
}
