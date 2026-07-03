package detective

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	CreateProfile(ctx context.Context, userID uuid.UUID) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error)
	IncrementTotalCases(ctx context.Context, userID uuid.UUID) error
	// ApplyCaseResult records a solved/failed case and adds XP, recomputing
	// rank and accuracy.
	ApplyCaseResult(ctx context.Context, userID uuid.UUID, solved bool, xpDelta int) error
	History(ctx context.Context, userID uuid.UUID) ([]HistoryEntry, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) CreateProfile(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO detective_profiles (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		return apperrors.Internal(err, "create detective profile")
	}
	return nil
}

func (r *PGRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	p := &Profile{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, rank, xp, solved_cases, failed_cases, total_cases, accuracy_rate, created_at, updated_at
		 FROM detective_profiles WHERE user_id = $1`, userID,
	).Scan(&p.ID, &p.UserID, &p.Rank, &p.XP, &p.SolvedCases, &p.FailedCases, &p.TotalCases, &p.AccuracyRate, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("profile_not_found", "detective profile not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get detective profile")
	}
	return p, nil
}

func (r *PGRepository) IncrementTotalCases(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE detective_profiles SET total_cases = total_cases + 1, updated_at = now() WHERE user_id = $1`, userID)
	if err != nil {
		return apperrors.Internal(err, "increment total cases")
	}
	return nil
}

func (r *PGRepository) ApplyCaseResult(ctx context.Context, userID uuid.UUID, solved bool, xpDelta int) error {
	p, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if solved {
		p.SolvedCases++
	} else {
		p.FailedCases++
	}
	p.XP += xpDelta
	if p.XP < 0 {
		p.XP = 0
	}
	finished := p.SolvedCases + p.FailedCases
	accuracy := 0.0
	if finished > 0 {
		accuracy = float64(p.SolvedCases) / float64(finished)
	}
	rank := RankForXP(p.XP)
	_, err = r.pool.Exec(ctx,
		`UPDATE detective_profiles
		 SET solved_cases = $2, failed_cases = $3, xp = $4, accuracy_rate = $5, rank = $6, updated_at = now()
		 WHERE user_id = $1`,
		userID, p.SolvedCases, p.FailedCases, p.XP, accuracy, rank)
	if err != nil {
		return apperrors.Internal(err, "apply case result")
	}
	return nil
}

func (r *PGRepository) History(ctx context.Context, userID uuid.UUID) ([]HistoryEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.title, c.type, c.difficulty, c.status,
		        (SELECT count(*) FROM solve_attempts sa WHERE sa.case_id = c.id) AS attempts,
		        c.created_at, c.solved_at
		 FROM cases c
		 WHERE c.user_id = $1
		 ORDER BY c.created_at DESC`, userID)
	if err != nil {
		return nil, apperrors.Internal(err, "query history")
	}
	defer rows.Close()
	entries := []HistoryEntry{}
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.CaseID, &e.Title, &e.Type, &e.Difficulty, &e.Status, &e.Attempts, &e.CreatedAt, &e.SolvedAt); err != nil {
			return nil, apperrors.Internal(err, "scan history entry")
		}
		entries = append(entries, e)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate history")
	}
	return entries, nil
}
