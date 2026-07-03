package playerprofile

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	// Create inserts the profile, sourcing the display name from users.
	Create(ctx context.Context, userID uuid.UUID) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error)
	UpdateDisplayName(ctx context.Context, userID uuid.UUID, displayName string) error
	IncrementTotalMissions(ctx context.Context, userID uuid.UUID) error
	// ApplyMissionResult updates completion counters, XP, level, rank, and
	// favorite mission type.
	ApplyMissionResult(ctx context.Context, userID uuid.UUID, completed bool, xpDelta int) error
	AddCounters(ctx context.Context, userID uuid.UUID, clues, aiInteractions, locations int) error
	History(ctx context.Context, userID uuid.UUID) ([]HistoryEntry, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Create(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO player_profiles (user_id, display_name)
		 SELECT id, display_name FROM users WHERE id = $1
		 ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		return apperrors.Internal(err, "create player profile")
	}
	return nil
}

func (r *PGRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	p := &Profile{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, display_name, rank, level, xp, total_missions, completed_missions,
		        failed_missions, success_rate, favorite_mission_type, total_clues_found,
		        total_ai_interactions, total_locations_visited, badges, created_at, updated_at
		 FROM player_profiles WHERE user_id = $1`, userID,
	).Scan(&p.ID, &p.UserID, &p.DisplayName, &p.Rank, &p.Level, &p.XP, &p.TotalMissions,
		&p.CompletedMissions, &p.FailedMissions, &p.SuccessRate, &p.FavoriteMissionType,
		&p.TotalCluesFound, &p.TotalAIInteractions, &p.TotalLocationsVisited, &p.Badges,
		&p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("profile_not_found", "player profile not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get player profile")
	}
	return p, nil
}

func (r *PGRepository) UpdateDisplayName(ctx context.Context, userID uuid.UUID, displayName string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE player_profiles SET display_name = $2, updated_at = now() WHERE user_id = $1`,
		userID, displayName)
	if err != nil {
		return apperrors.Internal(err, "update display name")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("profile_not_found", "player profile not found")
	}
	return nil
}

func (r *PGRepository) IncrementTotalMissions(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE player_profiles SET total_missions = total_missions + 1, updated_at = now()
		 WHERE user_id = $1`, userID)
	if err != nil {
		return apperrors.Internal(err, "increment total missions")
	}
	return nil
}

func (r *PGRepository) ApplyMissionResult(ctx context.Context, userID uuid.UUID, completed bool, xpDelta int) error {
	p, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if completed {
		p.CompletedMissions++
	} else {
		p.FailedMissions++
	}
	p.XP += xpDelta
	if p.XP < 0 {
		p.XP = 0
	}
	finished := p.CompletedMissions + p.FailedMissions
	if finished > 0 {
		p.SuccessRate = float64(p.CompletedMissions) / float64(finished)
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE player_profiles
		 SET completed_missions = $2, failed_missions = $3, xp = $4, level = $5, rank = $6,
		     success_rate = $7,
		     favorite_mission_type = COALESCE((
		         SELECT m.type FROM missions m
		         WHERE m.user_id = $1 GROUP BY m.type ORDER BY count(*) DESC LIMIT 1
		     ), favorite_mission_type),
		     updated_at = now()
		 WHERE user_id = $1`,
		userID, p.CompletedMissions, p.FailedMissions, p.XP, LevelForXP(p.XP), RankForXP(p.XP), p.SuccessRate)
	if err != nil {
		return apperrors.Internal(err, "apply mission result")
	}
	return nil
}

func (r *PGRepository) AddCounters(ctx context.Context, userID uuid.UUID, clues, aiInteractions, locations int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE player_profiles
		 SET total_clues_found = total_clues_found + $2,
		     total_ai_interactions = total_ai_interactions + $3,
		     total_locations_visited = total_locations_visited + $4,
		     updated_at = now()
		 WHERE user_id = $1`, userID, clues, aiInteractions, locations)
	if err != nil {
		return apperrors.Internal(err, "add profile counters")
	}
	return nil
}

func (r *PGRepository) History(ctx context.Context, userID uuid.UUID) ([]HistoryEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, type, difficulty, status, result, created_at, completed_at
		 FROM missions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, apperrors.Internal(err, "list mission history")
	}
	defer rows.Close()
	items := []HistoryEntry{}
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.MissionID, &e.Title, &e.Type, &e.Difficulty, &e.Status,
			&e.Result, &e.CreatedAt, &e.CompletedAt); err != nil {
			return nil, apperrors.Internal(err, "scan history entry")
		}
		items = append(items, e)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate history")
	}
	return items, nil
}
