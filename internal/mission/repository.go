package mission

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, m *Mission) error
	GetByID(ctx context.Context, missionID uuid.UUID) (*Mission, error)
	GetForUser(ctx context.Context, userID, missionID uuid.UUID) (*Mission, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Mission, error)
	UpdateStatus(ctx context.Context, missionID uuid.UUID, status string) error
	SetGeneratedContent(ctx context.Context, m *Mission) error
	UpdateObjectives(ctx context.Context, missionID uuid.UUID, objectives json.RawMessage) error
	UpdatePublicState(ctx context.Context, missionID uuid.UUID, state json.RawMessage) error
	AdvanceMissionTime(ctx context.Context, missionID uuid.UUID, minutes int) (time.Time, error)
	SetResult(ctx context.Context, missionID uuid.UUID, result json.RawMessage, status string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const missionColumns = `id, user_id, type, title, status, difficulty, region, summary, briefing,
	objectives, public_state, result, center_lat, center_lng, map_zoom, mission_time,
	created_at, updated_at, completed_at`

func scanMission(row pgx.Row) (*Mission, error) {
	m := &Mission{}
	err := row.Scan(&m.ID, &m.UserID, &m.Type, &m.Title, &m.Status, &m.Difficulty, &m.Region,
		&m.Summary, &m.Briefing, &m.Objectives, &m.PublicState, &m.Result, &m.CenterLat,
		&m.CenterLng, &m.MapZoom, &m.MissionTime, &m.CreatedAt, &m.UpdatedAt, &m.CompletedAt)
	if err != nil {
		return nil, err
	}
	m.CurrentTime = FormatClock(m.MissionTime)
	return m, nil
}

func (r *PGRepository) Create(ctx context.Context, m *Mission) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO missions (user_id, type, title, status, difficulty, region, summary, briefing, mission_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, objectives, public_state, mission_time, created_at, updated_at`,
		m.UserID, m.Type, m.Title, m.Status, m.Difficulty, m.Region, m.Summary, m.Briefing, Epoch,
	).Scan(&m.ID, &m.Objectives, &m.PublicState, &m.MissionTime, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create mission")
	}
	m.CurrentTime = FormatClock(m.MissionTime)
	return nil
}

func (r *PGRepository) GetByID(ctx context.Context, missionID uuid.UUID) (*Mission, error) {
	m, err := scanMission(r.pool.QueryRow(ctx,
		`SELECT `+missionColumns+` FROM missions WHERE id = $1`,
		missionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("mission_not_found", "mission not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get mission")
	}
	return m, nil
}

func (r *PGRepository) GetForUser(ctx context.Context, userID, missionID uuid.UUID) (*Mission, error) {
	m, err := scanMission(r.pool.QueryRow(ctx,
		`SELECT `+missionColumns+` FROM missions WHERE id = $1 AND user_id = $2`,
		missionID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		// Not-found for both missing and foreign missions: never reveal
		// other users' mission IDs.
		return nil, apperrors.NotFound("mission_not_found", "mission not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get mission")
	}
	return m, nil
}

func (r *PGRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Mission, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+missionColumns+` FROM missions WHERE user_id = $1 ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, apperrors.Internal(err, "list missions")
	}
	defer rows.Close()
	missions := []Mission{}
	for rows.Next() {
		m, err := scanMission(rows)
		if err != nil {
			return nil, apperrors.Internal(err, "scan mission")
		}
		missions = append(missions, *m)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate missions")
	}
	return missions, nil
}

func (r *PGRepository) UpdateStatus(ctx context.Context, missionID uuid.UUID, status string) error {
	var err error
	if status == StatusCompleted || status == StatusFailed {
		_, err = r.pool.Exec(ctx,
			`UPDATE missions SET status = $2, completed_at = now(), updated_at = now() WHERE id = $1`,
			missionID, status)
	} else {
		_, err = r.pool.Exec(ctx,
			`UPDATE missions SET status = $2, updated_at = now() WHERE id = $1`,
			missionID, status)
	}
	if err != nil {
		return apperrors.Internal(err, "update mission status")
	}
	return nil
}

// SetGeneratedContent stores everything the generation pipeline produced.
func (r *PGRepository) SetGeneratedContent(ctx context.Context, m *Mission) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE missions SET title = $2, summary = $3, briefing = $4, region = $5,
		     objectives = $6, public_state = $7, center_lat = $8, center_lng = $9,
		     map_zoom = $10, updated_at = now()
		 WHERE id = $1`,
		m.ID, m.Title, m.Summary, m.Briefing, m.Region,
		orEmpty(m.Objectives, `[]`), orEmpty(m.PublicState, `{}`),
		m.CenterLat, m.CenterLng, m.MapZoom)
	if err != nil {
		return apperrors.Internal(err, "set generated mission content")
	}
	return nil
}

func (r *PGRepository) UpdateObjectives(ctx context.Context, missionID uuid.UUID, objectives json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE missions SET objectives = $2, updated_at = now() WHERE id = $1`,
		missionID, orEmpty(objectives, `[]`))
	if err != nil {
		return apperrors.Internal(err, "update mission objectives")
	}
	return nil
}

func (r *PGRepository) UpdatePublicState(ctx context.Context, missionID uuid.UUID, state json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE missions SET public_state = $2, updated_at = now() WHERE id = $1`,
		missionID, orEmpty(state, `{}`))
	if err != nil {
		return apperrors.Internal(err, "update mission public state")
	}
	return nil
}

func (r *PGRepository) AdvanceMissionTime(ctx context.Context, missionID uuid.UUID, minutes int) (time.Time, error) {
	var newTime time.Time
	err := r.pool.QueryRow(ctx,
		`UPDATE missions SET mission_time = mission_time + ($2 * interval '1 minute'), updated_at = now()
		 WHERE id = $1 RETURNING mission_time`,
		missionID, minutes).Scan(&newTime)
	if err != nil {
		return time.Time{}, apperrors.Internal(err, "advance mission time")
	}
	return newTime, nil
}

func (r *PGRepository) SetResult(ctx context.Context, missionID uuid.UUID, result json.RawMessage, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE missions SET result = $2, status = $3, completed_at = now(), updated_at = now()
		 WHERE id = $1`, missionID, orEmpty(result, `{}`), status)
	if err != nil {
		return apperrors.Internal(err, "set mission result")
	}
	return nil
}

func orEmpty(raw json.RawMessage, def string) []byte {
	if len(raw) == 0 {
		return []byte(def)
	}
	return raw
}
