package cases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, c *Case) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Case, error)
	// GetForUser enforces ownership at the query level: a case belonging to
	// another user is indistinguishable from a missing one (404).
	GetForUser(ctx context.Context, userID, caseID uuid.UUID) (*Case, error)
	UpdateStatus(ctx context.Context, caseID uuid.UUID, status string) error
	SetGeneratedContent(ctx context.Context, caseID uuid.UUID, title, summary string) error
	MarkSolved(ctx context.Context, caseID uuid.UUID) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const caseColumns = `id, user_id, title, type, difficulty, status, summary, created_at, updated_at, solved_at`

func scanCase(row pgx.Row, c *Case) error {
	return row.Scan(&c.ID, &c.UserID, &c.Title, &c.Type, &c.Difficulty, &c.Status, &c.Summary,
		&c.CreatedAt, &c.UpdatedAt, &c.SolvedAt)
}

func (r *PGRepository) Create(ctx context.Context, c *Case) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO cases (user_id, title, type, difficulty, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		c.UserID, c.Title, c.Type, c.Difficulty, c.Status,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create case")
	}
	return nil
}

func (r *PGRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Case, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+caseColumns+` FROM cases WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, apperrors.Internal(err, "list cases")
	}
	defer rows.Close()
	items := []Case{}
	for rows.Next() {
		var c Case
		if err := scanCase(rows, &c); err != nil {
			return nil, apperrors.Internal(err, "scan case")
		}
		items = append(items, c)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate cases")
	}
	return items, nil
}

func (r *PGRepository) GetForUser(ctx context.Context, userID, caseID uuid.UUID) (*Case, error) {
	c := &Case{}
	err := scanCase(r.pool.QueryRow(ctx,
		`SELECT `+caseColumns+` FROM cases WHERE id = $1 AND user_id = $2`, caseID, userID), c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("case_not_found", "case not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get case")
	}
	return c, nil
}

func (r *PGRepository) UpdateStatus(ctx context.Context, caseID uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE cases SET status = $2, updated_at = now() WHERE id = $1`, caseID, status)
	if err != nil {
		return apperrors.Internal(err, "update case status")
	}
	return nil
}

func (r *PGRepository) SetGeneratedContent(ctx context.Context, caseID uuid.UUID, title, summary string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE cases SET title = $2, summary = $3, updated_at = now() WHERE id = $1`, caseID, title, summary)
	if err != nil {
		return apperrors.Internal(err, "set case content")
	}
	return nil
}

func (r *PGRepository) MarkSolved(ctx context.Context, caseID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE cases SET status = 'solved', solved_at = now(), updated_at = now() WHERE id = $1`, caseID)
	if err != nil {
		return apperrors.Internal(err, "mark case solved")
	}
	return nil
}
