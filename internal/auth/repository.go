package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, t *RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// --- PostgreSQL implementations ---

type PGUserRepository struct{ pool *pgxpool.Pool }

func NewPGUserRepository(pool *pgxpool.Pool) *PGUserRepository { return &PGUserRepository{pool: pool} }

func (r *PGUserRepository) Create(ctx context.Context, u *User) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		u.Email, u.PasswordHash, u.DisplayName,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperrors.Conflict("email_taken", "an account with this email already exists")
		}
		return apperrors.Internal(err, "create user")
	}
	return nil
}

func (r *PGUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("user_not_found", "user not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get user by email")
	}
	return u, nil
}

func (r *PGUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("user_not_found", "user not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get user by id")
	}
	return u, nil
}

type PGRefreshTokenRepository struct{ pool *pgxpool.Pool }

func NewPGRefreshTokenRepository(pool *pgxpool.Pool) *PGRefreshTokenRepository {
	return &PGRefreshTokenRepository{pool: pool}
}

func (r *PGRefreshTokenRepository) Create(ctx context.Context, t *RefreshToken) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3) RETURNING id, created_at`,
		t.UserID, t.TokenHash, t.ExpiresAt,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "create refresh token")
	}
	return nil
}

func (r *PGRefreshTokenRepository) GetByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	t := &RefreshToken{}
	var revokedAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, hash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &revokedAt, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.Unauthorized("invalid_refresh_token", "invalid refresh token")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get refresh token")
	}
	t.RevokedAt = revokedAt
	return t, nil
}

func (r *PGRefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return apperrors.Internal(err, "revoke refresh token")
	}
	return nil
}

func (r *PGRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return apperrors.Internal(err, "revoke refresh tokens for user")
	}
	return nil
}
