package gamemap

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, l *Location) error
	GetByID(ctx context.Context, missionID, locationID uuid.UUID) (*Location, error)
	ListByMission(ctx context.Context, missionID uuid.UUID) ([]Location, error)
	ListLocked(ctx context.Context, missionID uuid.UUID) ([]Location, error)
	UpdateStatus(ctx context.Context, locationID uuid.UUID, status string) error
	UpdateRisk(ctx context.Context, locationID uuid.UUID, riskLevel int) error
	CountVisited(ctx context.Context, missionID uuid.UUID) (visited int, total int, err error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const locationColumns = `id, mission_id, name, type, latitude, longitude, status, risk_level,
	description, visual_prompt, available_actions, created_at, updated_at`

func scanLocation(row pgx.Row) (*Location, error) {
	l := &Location{}
	err := row.Scan(&l.ID, &l.MissionID, &l.Name, &l.Type, &l.Latitude, &l.Longitude, &l.Status,
		&l.RiskLevel, &l.Description, &l.VisualPrompt, &l.AvailableActions, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (r *PGRepository) Create(ctx context.Context, l *Location) error {
	actions := l.AvailableActions
	if len(actions) == 0 {
		actions = []byte(`[]`)
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO map_locations (mission_id, name, type, latitude, longitude, status, risk_level,
		     description, visual_prompt, available_actions)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, created_at, updated_at`,
		l.MissionID, l.Name, l.Type, l.Latitude, l.Longitude, l.Status, l.RiskLevel,
		l.Description, l.VisualPrompt, actions,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create map location")
	}
	return nil
}

func (r *PGRepository) GetByID(ctx context.Context, missionID, locationID uuid.UUID) (*Location, error) {
	l, err := scanLocation(r.pool.QueryRow(ctx,
		`SELECT `+locationColumns+` FROM map_locations WHERE mission_id = $1 AND id = $2`,
		missionID, locationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("location_not_found", "location not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get map location")
	}
	return l, nil
}

func (r *PGRepository) ListByMission(ctx context.Context, missionID uuid.UUID) ([]Location, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+locationColumns+` FROM map_locations WHERE mission_id = $1 ORDER BY created_at`,
		missionID)
	if err != nil {
		return nil, apperrors.Internal(err, "list map locations")
	}
	defer rows.Close()
	locations := []Location{}
	for rows.Next() {
		l, err := scanLocation(rows)
		if err != nil {
			return nil, apperrors.Internal(err, "scan map location")
		}
		locations = append(locations, *l)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate map locations")
	}
	return locations, nil
}

// ListLocked returns the mission's locked locations, oldest first — the order
// the unlock engine opens them in.
func (r *PGRepository) ListLocked(ctx context.Context, missionID uuid.UUID) ([]Location, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+locationColumns+` FROM map_locations
		 WHERE mission_id = $1 AND status = 'locked' ORDER BY created_at`, missionID)
	if err != nil {
		return nil, apperrors.Internal(err, "list locked locations")
	}
	defer rows.Close()
	locations := []Location{}
	for rows.Next() {
		l, err := scanLocation(rows)
		if err != nil {
			return nil, apperrors.Internal(err, "scan locked location")
		}
		locations = append(locations, *l)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate locked locations")
	}
	return locations, nil
}

func (r *PGRepository) UpdateStatus(ctx context.Context, locationID uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE map_locations SET status = $2, updated_at = now() WHERE id = $1`, locationID, status)
	if err != nil {
		return apperrors.Internal(err, "update location status")
	}
	return nil
}

func (r *PGRepository) UpdateRisk(ctx context.Context, locationID uuid.UUID, riskLevel int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE map_locations SET risk_level = LEAST(100, GREATEST(0, $2)), updated_at = now()
		 WHERE id = $1`, locationID, riskLevel)
	if err != nil {
		return apperrors.Internal(err, "update location risk")
	}
	return nil
}

func (r *PGRepository) CountVisited(ctx context.Context, missionID uuid.UUID) (int, int, error) {
	var visited, total int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE status = 'visited'), count(*)
		 FROM map_locations WHERE mission_id = $1`, missionID).Scan(&visited, &total)
	if err != nil {
		return 0, 0, apperrors.Internal(err, "count visited locations")
	}
	return visited, total, nil
}
