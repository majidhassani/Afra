package character

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, c *Character) error
	GetByID(ctx context.Context, missionID, characterID uuid.UUID) (*Character, error)
	ListByMission(ctx context.Context, missionID uuid.UUID) ([]Character, error)
	ListAtLocation(ctx context.Context, missionID, locationID uuid.UUID) ([]Character, error)
	UpdateDialogueState(ctx context.Context, characterID uuid.UUID, trust, stress int, mood string) error
	UpdateLocation(ctx context.Context, characterID uuid.UUID, locationID *uuid.UUID) error
	UpdateAvatar(ctx context.Context, characterID uuid.UUID, url, status string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const characterColumns = `id, mission_id, name, role, category, age, public_profile, personality,
	current_location_id, trust_level, stress_level, mood, dialogue_style, avatar_prompt,
	thumbnail_prompt, avatar_url, avatar_status, avatar_version, visual_style_tags, private_state, created_at, updated_at`

func scanCharacter(row pgx.Row) (*Character, error) {
	c := &Character{}
	err := row.Scan(&c.ID, &c.MissionID, &c.Name, &c.Role, &c.Category, &c.Age, &c.PublicProfile,
		&c.Personality, &c.CurrentLocationID, &c.TrustLevel, &c.StressLevel, &c.Mood,
		&c.DialogueStyle, &c.AvatarPrompt, &c.ThumbnailPrompt, &c.AvatarURL, &c.AvatarStatus,
		&c.AvatarVersion, &c.VisualStyleTags, &c.PrivateState, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *PGRepository) Create(ctx context.Context, c *Character) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO characters (mission_id, name, role, category, age, public_profile, personality,
		     current_location_id, trust_level, stress_level, mood, dialogue_style, avatar_prompt,
		     thumbnail_prompt, visual_style_tags, private_state)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		 RETURNING id, created_at, updated_at`,
		c.MissionID, c.Name, c.Role, c.Category, c.Age, c.PublicProfile,
		orEmpty(c.Personality, `{}`), c.CurrentLocationID, c.TrustLevel, c.StressLevel,
		c.Mood, c.DialogueStyle, c.AvatarPrompt, c.ThumbnailPrompt,
		orEmpty(c.VisualStyleTags, `[]`), orEmpty(c.PrivateState, `{}`),
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create character")
	}
	return nil
}

func (r *PGRepository) GetByID(ctx context.Context, missionID, characterID uuid.UUID) (*Character, error) {
	c, err := scanCharacter(r.pool.QueryRow(ctx,
		`SELECT `+characterColumns+` FROM characters WHERE mission_id = $1 AND id = $2`,
		missionID, characterID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("character_not_found", "character not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get character")
	}
	return c, nil
}

func (r *PGRepository) list(ctx context.Context, query string, args ...any) ([]Character, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Internal(err, "list characters")
	}
	defer rows.Close()
	characters := []Character{}
	for rows.Next() {
		c, err := scanCharacter(rows)
		if err != nil {
			return nil, apperrors.Internal(err, "scan character")
		}
		characters = append(characters, *c)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate characters")
	}
	return characters, nil
}

func (r *PGRepository) ListByMission(ctx context.Context, missionID uuid.UUID) ([]Character, error) {
	return r.list(ctx,
		`SELECT `+characterColumns+` FROM characters WHERE mission_id = $1 ORDER BY created_at`, missionID)
}

func (r *PGRepository) ListAtLocation(ctx context.Context, missionID, locationID uuid.UUID) ([]Character, error) {
	return r.list(ctx,
		`SELECT `+characterColumns+` FROM characters
		 WHERE mission_id = $1 AND current_location_id = $2 ORDER BY created_at`,
		missionID, locationID)
}

func (r *PGRepository) UpdateDialogueState(ctx context.Context, characterID uuid.UUID, trust, stress int, mood string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE characters SET trust_level = $2, stress_level = $3, mood = $4, updated_at = now()
		 WHERE id = $1`, characterID, trust, stress, mood)
	if err != nil {
		return apperrors.Internal(err, "update character dialogue state")
	}
	return nil
}

func (r *PGRepository) UpdateLocation(ctx context.Context, characterID uuid.UUID, locationID *uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE characters SET current_location_id = $2, updated_at = now() WHERE id = $1`,
		characterID, locationID)
	if err != nil {
		return apperrors.Internal(err, "update character location")
	}
	return nil
}

func (r *PGRepository) UpdateAvatar(ctx context.Context, characterID uuid.UUID, url, status string) error {
	// Bump the version only when a real image was produced, so the client can
	// cache-bust regenerated portraits (unavailable attempts don't churn it).
	_, err := r.pool.Exec(ctx,
		`UPDATE characters SET avatar_url = $2, avatar_status = $3,
		     avatar_version = avatar_version + CASE WHEN $2 <> '' THEN 1 ELSE 0 END,
		     updated_at = now() WHERE id = $1`,
		characterID, url, status)
	if err != nil {
		return apperrors.Internal(err, "update character avatar")
	}
	return nil
}

func orEmpty(raw []byte, def string) []byte {
	if len(raw) == 0 {
		return []byte(def)
	}
	return raw
}
