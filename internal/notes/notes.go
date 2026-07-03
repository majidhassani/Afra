// Package notes manages the detective's player notes.
package notes

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Note struct {
	ID        uuid.UUID `json:"id"`
	CaseID    uuid.UUID `json:"case_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, n *Note) error
	ListByCase(ctx context.Context, caseID uuid.UUID) ([]Note, error)
	Update(ctx context.Context, n *Note) error
	Delete(ctx context.Context, caseID, noteID uuid.UUID) error
	GetByID(ctx context.Context, caseID, noteID uuid.UUID) (*Note, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Create(ctx context.Context, n *Note) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO player_notes (case_id, title, content) VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		n.CaseID, n.Title, n.Content,
	).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create note")
	}
	return nil
}

func (r *PGRepository) ListByCase(ctx context.Context, caseID uuid.UUID) ([]Note, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, case_id, title, content, created_at, updated_at
		 FROM player_notes WHERE case_id = $1 ORDER BY updated_at DESC`, caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list notes")
	}
	defer rows.Close()
	items := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.CaseID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan note")
		}
		items = append(items, n)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate notes")
	}
	return items, nil
}

func (r *PGRepository) GetByID(ctx context.Context, caseID, noteID uuid.UUID) (*Note, error) {
	n := &Note{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, case_id, title, content, created_at, updated_at
		 FROM player_notes WHERE id = $1 AND case_id = $2`, noteID, caseID,
	).Scan(&n.ID, &n.CaseID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("note_not_found", "note not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get note")
	}
	return n, nil
}

func (r *PGRepository) Update(ctx context.Context, n *Note) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE player_notes SET title = $3, content = $4, updated_at = now()
		 WHERE id = $1 AND case_id = $2`, n.ID, n.CaseID, n.Title, n.Content)
	if err != nil {
		return apperrors.Internal(err, "update note")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("note_not_found", "note not found")
	}
	return nil
}

func (r *PGRepository) Delete(ctx context.Context, caseID, noteID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM player_notes WHERE id = $1 AND case_id = $2`, noteID, caseID)
	if err != nil {
		return apperrors.Internal(err, "delete note")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("note_not_found", "note not found")
	}
	return nil
}
