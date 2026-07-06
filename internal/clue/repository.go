package clue

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, c *Clue) error
	GetByID(ctx context.Context, missionID, clueID uuid.UUID) (*Clue, error)
	ListDiscovered(ctx context.Context, missionID uuid.UUID) ([]Clue, error)
	ListDiscoveredAtLocation(ctx context.Context, missionID, locationID uuid.UUID) ([]Clue, error)
	ListUndiscoveredAtLocation(ctx context.Context, missionID, locationID uuid.UUID) ([]Clue, error)
	FindUndiscoveredByTitle(ctx context.Context, missionID uuid.UUID, title string) (*Clue, error)
	MarkDiscovered(ctx context.Context, clueID uuid.UUID) error
	AdjustReliability(ctx context.Context, clueID uuid.UUID, delta int) error
	UpdateImage(ctx context.Context, clueID uuid.UUID, url, status string) error
	Counts(ctx context.Context, missionID uuid.UUID) (discovered int, total int, err error)
	// ListCritical returns the high-importance clues for a mission (discovered
	// or not), so end-of-mission evaluation can report which critical clues
	// were found and which were missed.
	ListCritical(ctx context.Context, missionID uuid.UUID) ([]Clue, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const clueColumns = `id, mission_id, location_id, title, type, short_description, detailed_description,
	visual_description, avatar_or_thumbnail_prompt, image_url, image_status, discovered, reliability, importance,
	related_character_ids, public_data, internal_truth, created_at, updated_at`

func scanClue(row pgx.Row) (*Clue, error) {
	c := &Clue{}
	err := row.Scan(&c.ID, &c.MissionID, &c.LocationID, &c.Title, &c.Type, &c.ShortDescription,
		&c.DetailedDescription, &c.VisualDescription, &c.AvatarOrThumbnailPrompt, &c.ImageURL, &c.ImageStatus,
		&c.Discovered, &c.Reliability, &c.Importance, &c.RelatedCharacterIDs, &c.PublicData, &c.InternalTruth,
		&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *PGRepository) Create(ctx context.Context, c *Clue) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO clues (mission_id, location_id, title, type, short_description, detailed_description,
		     visual_description, avatar_or_thumbnail_prompt, discovered, reliability, importance,
		     related_character_ids, public_data, internal_truth)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING id, created_at, updated_at`,
		c.MissionID, c.LocationID, c.Title, c.Type, c.ShortDescription, c.DetailedDescription,
		c.VisualDescription, c.AvatarOrThumbnailPrompt, c.Discovered, c.Reliability, c.Importance,
		orEmpty(c.RelatedCharacterIDs, `[]`), orEmpty(c.PublicData, `{}`), orEmpty(c.InternalTruth, `{}`),
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create clue")
	}
	return nil
}

func (r *PGRepository) GetByID(ctx context.Context, missionID, clueID uuid.UUID) (*Clue, error) {
	c, err := scanClue(r.pool.QueryRow(ctx,
		`SELECT `+clueColumns+` FROM clues WHERE mission_id = $1 AND id = $2`, missionID, clueID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("clue_not_found", "clue not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get clue")
	}
	return c, nil
}

func (r *PGRepository) list(ctx context.Context, query string, args ...any) ([]Clue, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Internal(err, "list clues")
	}
	defer rows.Close()
	clues := []Clue{}
	for rows.Next() {
		c, err := scanClue(rows)
		if err != nil {
			return nil, apperrors.Internal(err, "scan clue")
		}
		clues = append(clues, *c)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate clues")
	}
	return clues, nil
}

func (r *PGRepository) ListDiscovered(ctx context.Context, missionID uuid.UUID) ([]Clue, error) {
	return r.list(ctx,
		`SELECT `+clueColumns+` FROM clues WHERE mission_id = $1 AND discovered ORDER BY created_at`, missionID)
}

func (r *PGRepository) ListDiscoveredAtLocation(ctx context.Context, missionID, locationID uuid.UUID) ([]Clue, error) {
	return r.list(ctx,
		`SELECT `+clueColumns+` FROM clues
		 WHERE mission_id = $1 AND location_id = $2 AND discovered ORDER BY created_at`,
		missionID, locationID)
}

func (r *PGRepository) ListUndiscoveredAtLocation(ctx context.Context, missionID, locationID uuid.UUID) ([]Clue, error) {
	return r.list(ctx,
		`SELECT `+clueColumns+` FROM clues
		 WHERE mission_id = $1 AND location_id = $2 AND NOT discovered ORDER BY created_at`,
		missionID, locationID)
}

func (r *PGRepository) FindUndiscoveredByTitle(ctx context.Context, missionID uuid.UUID, title string) (*Clue, error) {
	c, err := scanClue(r.pool.QueryRow(ctx,
		`SELECT `+clueColumns+` FROM clues
		 WHERE mission_id = $1 AND NOT discovered AND lower(title) = lower($2) LIMIT 1`,
		missionID, title))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("clue_not_found", "no undiscovered clue with that title")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "find clue by title")
	}
	return c, nil
}

func (r *PGRepository) MarkDiscovered(ctx context.Context, clueID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE clues SET discovered = true, updated_at = now() WHERE id = $1`, clueID)
	if err != nil {
		return apperrors.Internal(err, "mark clue discovered")
	}
	return nil
}

func (r *PGRepository) AdjustReliability(ctx context.Context, clueID uuid.UUID, delta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE clues SET reliability = LEAST(100, GREATEST(0, reliability + $2)), updated_at = now()
		 WHERE id = $1`, clueID, delta)
	if err != nil {
		return apperrors.Internal(err, "adjust clue reliability")
	}
	return nil
}

func (r *PGRepository) UpdateImage(ctx context.Context, clueID uuid.UUID, url, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE clues SET image_url = $2, image_status = $3, updated_at = now() WHERE id = $1`,
		clueID, url, status)
	if err != nil {
		return apperrors.Internal(err, "update clue image")
	}
	return nil
}

func (r *PGRepository) ListCritical(ctx context.Context, missionID uuid.UUID) ([]Clue, error) {
	return r.list(ctx,
		`SELECT `+clueColumns+` FROM clues
		 WHERE mission_id = $1 AND importance = 'high' ORDER BY created_at`, missionID)
}

func (r *PGRepository) Counts(ctx context.Context, missionID uuid.UUID) (int, int, error) {
	var discovered, total int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE discovered), count(*) FROM clues WHERE mission_id = $1`,
		missionID).Scan(&discovered, &total)
	if err != nil {
		return 0, 0, apperrors.Internal(err, "count clues")
	}
	return discovered, total, nil
}

func orEmpty(raw []byte, def string) []byte {
	if len(raw) == 0 {
		return []byte(def)
	}
	return raw
}
