// Package solve handles final accusations: the JudgeAgent scores and
// narrates, but correctness is decided deterministically against the Case
// Bible, and only player-safe feedback ever leaves the backend.
package solve

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Attempt struct {
	ID               uuid.UUID `json:"id"`
	CaseID           uuid.UUID `json:"case_id"`
	AccusedSuspectID uuid.UUID `json:"accused_suspect_id"`
	Motive           string    `json:"motive"`
	Reasoning        string    `json:"reasoning"`
	Correct          bool      `json:"correct"`
	Score            int       `json:"score"`
	Feedback         string    `json:"feedback"`
	CreatedAt        time.Time `json:"created_at"`
}

type Repository interface {
	Insert(ctx context.Context, a *Attempt) error
	ListByCase(ctx context.Context, caseID uuid.UUID) ([]Attempt, error)
	CountByCase(ctx context.Context, caseID uuid.UUID) (int, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Insert(ctx context.Context, a *Attempt) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO solve_attempts (case_id, accused_suspect_id, motive, reasoning, correct, score, feedback)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at`,
		a.CaseID, a.AccusedSuspectID, a.Motive, a.Reasoning, a.Correct, a.Score, a.Feedback,
	).Scan(&a.ID, &a.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "insert solve attempt")
	}
	return nil
}

func (r *PGRepository) ListByCase(ctx context.Context, caseID uuid.UUID) ([]Attempt, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, case_id, accused_suspect_id, motive, reasoning, correct, score, feedback, created_at
		 FROM solve_attempts WHERE case_id = $1 ORDER BY created_at`, caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list solve attempts")
	}
	defer rows.Close()
	items := []Attempt{}
	for rows.Next() {
		var a Attempt
		if err := rows.Scan(&a.ID, &a.CaseID, &a.AccusedSuspectID, &a.Motive, &a.Reasoning,
			&a.Correct, &a.Score, &a.Feedback, &a.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan solve attempt")
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return nil, apperrors.Internal(err, "iterate solve attempts")
	}
	return items, nil
}

func (r *PGRepository) CountByCase(ctx context.Context, caseID uuid.UUID) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM solve_attempts WHERE case_id = $1`, caseID).Scan(&n); err != nil {
		return 0, apperrors.Internal(err, "count solve attempts")
	}
	return n, nil
}
