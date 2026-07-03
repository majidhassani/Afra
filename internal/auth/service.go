package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

// ProfileCreator lets registration create the detective profile without a
// package cycle onto the detective module.
type ProfileCreator interface {
	CreateProfile(ctx context.Context, userID uuid.UUID) error
}

type Service struct {
	users      UserRepository
	tokens     RefreshTokenRepository
	profiles   ProfileCreator
	jwt        *TokenManager
	refreshTTL time.Duration
}

func NewService(users UserRepository, tokens RefreshTokenRepository, profiles ProfileCreator, jwt *TokenManager, refreshTTL time.Duration) *Service {
	return &Service{users: users, tokens: tokens, profiles: profiles, jwt: jwt, refreshTTL: refreshTTL}
}

func (s *Service) Register(ctx context.Context, email, password, displayName string) (*User, *TokenPair, error) {
	if err := validator.New().
		Required("email", email).Email("email", email).
		Required("password", password).MinLen("password", password, 8).MaxLen("password", password, 72).
		Required("display_name", displayName).MaxLen("display_name", displayName, 60).
		Err(); err != nil {
		return nil, nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, apperrors.Internal(err, "hash password")
	}
	user := &User{Email: email, PasswordHash: string(hash), DisplayName: displayName}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, nil, err
	}
	if err := s.profiles.CreateProfile(ctx, user.ID); err != nil {
		return nil, nil, err
	}
	pair, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*User, *TokenPair, error) {
	if err := validator.New().
		Required("email", email).
		Required("password", password).
		Err(); err != nil {
		return nil, nil, err
	}
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Do not reveal whether the email exists.
		if apperrors.Is(err, apperrors.KindNotFound) {
			return nil, nil, apperrors.Unauthorized("invalid_credentials", "invalid email or password")
		}
		return nil, nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, nil, apperrors.Unauthorized("invalid_credentials", "invalid email or password")
	}
	pair, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

// Refresh validates and rotates the refresh token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	stored, err := s.validRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if err := s.tokens.Revoke(ctx, stored.ID); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, stored.UserID)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	stored, err := s.validRefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}
	return s.tokens.Revoke(ctx, stored.ID)
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *Service) validRefreshToken(ctx context.Context, raw string) (*RefreshToken, error) {
	if raw == "" {
		return nil, apperrors.Invalid("missing_refresh_token", "refresh_token is required")
	}
	stored, err := s.tokens.GetByHash(ctx, hashToken(raw))
	if err != nil {
		return nil, err
	}
	if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		return nil, apperrors.Unauthorized("invalid_refresh_token", "refresh token expired or revoked")
	}
	return stored, nil
}

func (s *Service) issueTokens(ctx context.Context, userID uuid.UUID) (*TokenPair, error) {
	access, accessExp, err := s.jwt.Generate(userID)
	if err != nil {
		return nil, err
	}
	raw, err := randomToken()
	if err != nil {
		return nil, err
	}
	refreshExp := time.Now().Add(s.refreshTTL)
	if err := s.tokens.Create(ctx, &RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: refreshExp,
	}); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		RefreshToken:     raw,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", apperrors.Internal(err, "generate token")
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
