// Package timeline manages the private real timeline (Truth Layer, revealed
// piece by piece) and the player-editable timeline.
package timeline

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

// RealEvent is internal; only revealed events are shown as ConfirmedEvent.
type RealEvent struct {
	ID           uuid.UUID
	CaseID       uuid.UUID
	OccurredAt   time.Time
	Title        string
	Description  string
	Participants json.RawMessage
	Revealed     bool
	CreatedAt    time.Time
}

// ConfirmedEvent is the client-facing shape of a revealed real event.
type ConfirmedEvent struct {
	ID          uuid.UUID `json:"id"`
	OccurredAt  time.Time `json:"occurred_at"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

func (e *RealEvent) Confirmed() ConfirmedEvent {
	return ConfirmedEvent{ID: e.ID, OccurredAt: e.OccurredAt, Title: e.Title, Description: e.Description}
}

type PlayerEvent struct {
	ID                uuid.UUID       `json:"id"`
	CaseID            uuid.UUID       `json:"case_id"`
	OccurredAt        time.Time       `json:"occurred_at"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	LinkedEvidenceIDs json.RawMessage `json:"linked_evidence_ids"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type Repository interface {
	CreateReal(ctx context.Context, e *RealEvent) error
	ListRevealed(ctx context.Context, caseID uuid.UUID) ([]RealEvent, error)
	RevealByIDs(ctx context.Context, caseID uuid.UUID, ids []uuid.UUID) (int, error)
	CreatePlayer(ctx context.Context, e *PlayerEvent) error
	UpdatePlayer(ctx context.Context, e *PlayerEvent) error
	DeletePlayer(ctx context.Context, caseID, eventID uuid.UUID) error
	ListPlayer(ctx context.Context, caseID uuid.UUID) ([]PlayerEvent, error)
	GetPlayer(ctx context.Context, caseID, eventID uuid.UUID) (*PlayerEvent, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) CreateReal(ctx context.Context, e *RealEvent) error {
	participants := e.Participants
	if len(participants) == 0 {
		participants = []byte(`[]`)
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO real_timeline_events (case_id, occurred_at, title, description, participants, revealed)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		e.CaseID, e.OccurredAt, e.Title, e.Description, participants, e.Revealed,
	).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "create real timeline event")
	}
	return nil
}

func (r *PGRepository) ListRevealed(ctx context.Context, caseID uuid.UUID) ([]RealEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, case_id, occurred_at, title, description, participants, revealed, created_at
		 FROM real_timeline_events WHERE case_id = $1 AND revealed = true ORDER BY occurred_at`, caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list revealed timeline events")
	}
	defer rows.Close()
	events := []RealEvent{}
	for rows.Next() {
		var e RealEvent
		if err := rows.Scan(&e.ID, &e.CaseID, &e.OccurredAt, &e.Title, &e.Description, &e.Participants, &e.Revealed, &e.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan timeline event")
		}
		events = append(events, e)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate timeline events")
	}
	return events, nil
}

func (r *PGRepository) RevealByIDs(ctx context.Context, caseID uuid.UUID, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE real_timeline_events SET revealed = true WHERE case_id = $1 AND id = ANY($2)`, caseID, ids)
	if err != nil {
		return 0, apperrors.Internal(err, "reveal timeline events")
	}
	return int(tag.RowsAffected()), nil
}

func (r *PGRepository) CreatePlayer(ctx context.Context, e *PlayerEvent) error {
	linked := e.LinkedEvidenceIDs
	if len(linked) == 0 {
		linked = []byte(`[]`)
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO player_timeline_events (case_id, occurred_at, title, description, linked_evidence_ids)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		e.CaseID, e.OccurredAt, e.Title, e.Description, linked,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create player timeline event")
	}
	return nil
}

func (r *PGRepository) GetPlayer(ctx context.Context, caseID, eventID uuid.UUID) (*PlayerEvent, error) {
	e := &PlayerEvent{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, case_id, occurred_at, title, description, linked_evidence_ids, created_at, updated_at
		 FROM player_timeline_events WHERE id = $1 AND case_id = $2`, eventID, caseID,
	).Scan(&e.ID, &e.CaseID, &e.OccurredAt, &e.Title, &e.Description, &e.LinkedEvidenceIDs, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("timeline_event_not_found", "timeline event not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get player timeline event")
	}
	return e, nil
}

func (r *PGRepository) UpdatePlayer(ctx context.Context, e *PlayerEvent) error {
	linked := e.LinkedEvidenceIDs
	if len(linked) == 0 {
		linked = []byte(`[]`)
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE player_timeline_events
		 SET occurred_at = $3, title = $4, description = $5, linked_evidence_ids = $6, updated_at = now()
		 WHERE id = $1 AND case_id = $2`,
		e.ID, e.CaseID, e.OccurredAt, e.Title, e.Description, linked)
	if err != nil {
		return apperrors.Internal(err, "update player timeline event")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("timeline_event_not_found", "timeline event not found")
	}
	return nil
}

func (r *PGRepository) DeletePlayer(ctx context.Context, caseID, eventID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM player_timeline_events WHERE id = $1 AND case_id = $2`, eventID, caseID)
	if err != nil {
		return apperrors.Internal(err, "delete player timeline event")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("timeline_event_not_found", "timeline event not found")
	}
	return nil
}

func (r *PGRepository) ListPlayer(ctx context.Context, caseID uuid.UUID) ([]PlayerEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, case_id, occurred_at, title, description, linked_evidence_ids, created_at, updated_at
		 FROM player_timeline_events WHERE case_id = $1 ORDER BY occurred_at`, caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list player timeline events")
	}
	defer rows.Close()
	events := []PlayerEvent{}
	for rows.Next() {
		var e PlayerEvent
		if err := rows.Scan(&e.ID, &e.CaseID, &e.OccurredAt, &e.Title, &e.Description, &e.LinkedEvidenceIDs, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan player timeline event")
		}
		events = append(events, e)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate player timeline events")
	}
	return events, nil
}
