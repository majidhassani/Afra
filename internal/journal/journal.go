// Package journal owns mission-scoped player notes (the journal panel).
package journal

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

type Note struct {
	ID        uuid.UUID `json:"id"`
	MissionID uuid.UUID `json:"mission_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, n *Note) error
	ListByMission(ctx context.Context, missionID uuid.UUID) ([]Note, error)
	Update(ctx context.Context, missionID, noteID uuid.UUID, title, content string) (*Note, error)
	Delete(ctx context.Context, missionID, noteID uuid.UUID) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Create(ctx context.Context, n *Note) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO journal_notes (mission_id, title, content) VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		n.MissionID, n.Title, n.Content,
	).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create journal note")
	}
	return nil
}

func (r *PGRepository) ListByMission(ctx context.Context, missionID uuid.UUID) ([]Note, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, mission_id, title, content, created_at, updated_at
		 FROM journal_notes WHERE mission_id = $1 ORDER BY updated_at DESC`, missionID)
	if err != nil {
		return nil, apperrors.Internal(err, "list journal notes")
	}
	defer rows.Close()
	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.MissionID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan journal note")
		}
		notes = append(notes, n)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate journal notes")
	}
	return notes, nil
}

func (r *PGRepository) Update(ctx context.Context, missionID, noteID uuid.UUID, title, content string) (*Note, error) {
	n := &Note{ID: noteID, MissionID: missionID, Title: title, Content: content}
	err := r.pool.QueryRow(ctx,
		`UPDATE journal_notes SET title = $3, content = $4, updated_at = now()
		 WHERE mission_id = $1 AND id = $2 RETURNING created_at, updated_at`,
		missionID, noteID, title, content,
	).Scan(&n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("note_not_found", "journal note not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "update journal note")
	}
	return n, nil
}

func (r *PGRepository) Delete(ctx context.Context, missionID, noteID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM journal_notes WHERE mission_id = $1 AND id = $2`, missionID, noteID)
	if err != nil {
		return apperrors.Internal(err, "delete journal note")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("note_not_found", "journal note not found")
	}
	return nil
}

// MissionGateway is the slice of the mission module this service needs.
type MissionGateway interface {
	EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error
}

type Service struct {
	repo  Repository
	guard MissionGateway
}

func NewService(repo Repository, guard MissionGateway) *Service {
	return &Service{repo: repo, guard: guard}
}

func validateNote(title, content string) error {
	return validator.New().
		MaxLen("title", title, 200).
		Required("content", content).MaxLen("content", content, 10000).
		Err()
}

func (s *Service) List(ctx context.Context, userID, missionID uuid.UUID) ([]Note, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	return s.repo.ListByMission(ctx, missionID)
}

func (s *Service) Create(ctx context.Context, userID, missionID uuid.UUID, title, content string) (*Note, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	if err := validateNote(title, content); err != nil {
		return nil, err
	}
	n := &Note{MissionID: missionID, Title: title, Content: content}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *Service) Update(ctx context.Context, userID, missionID, noteID uuid.UUID, title, content string) (*Note, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	if err := validateNote(title, content); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, missionID, noteID, title, content)
}

func (s *Service) Delete(ctx context.Context, userID, missionID, noteID uuid.UUID) error {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, missionID, noteID)
}
