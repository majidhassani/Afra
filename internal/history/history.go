// Package history persists case events (the investigation audit trail) and
// broadcasts them on the notification bus.
package history

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"casemind/internal/notification"
	apperrors "casemind/pkg/errors"
)

type CaseEvent struct {
	ID        uuid.UUID       `json:"id"`
	CaseID    uuid.UUID       `json:"case_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type Repository interface {
	Insert(ctx context.Context, caseID uuid.UUID, eventType string, payload []byte) (*CaseEvent, error)
	ListByCase(ctx context.Context, caseID uuid.UUID, limit int) ([]CaseEvent, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Insert(ctx context.Context, caseID uuid.UUID, eventType string, payload []byte) (*CaseEvent, error) {
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	ev := &CaseEvent{CaseID: caseID, Type: eventType, Payload: payload}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO case_events (case_id, type, payload) VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		caseID, eventType, payload,
	).Scan(&ev.ID, &ev.CreatedAt)
	if err != nil {
		return nil, apperrors.Internal(err, "insert case event")
	}
	return ev, nil
}

func (r *PGRepository) ListByCase(ctx context.Context, caseID uuid.UUID, limit int) ([]CaseEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, case_id, type, payload, created_at FROM case_events
		 WHERE case_id = $1 ORDER BY created_at DESC LIMIT $2`, caseID, limit)
	if err != nil {
		return nil, apperrors.Internal(err, "list case events")
	}
	defer rows.Close()
	events := []CaseEvent{}
	for rows.Next() {
		var ev CaseEvent
		if err := rows.Scan(&ev.ID, &ev.CaseID, &ev.Type, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan case event")
		}
		events = append(events, ev)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate case events")
	}
	return events, nil
}

// Recorder persists events and publishes them to live subscribers.
type Recorder struct {
	repo Repository
	bus  *notification.Bus
	log  *slog.Logger
}

func NewRecorder(repo Repository, bus *notification.Bus, log *slog.Logger) *Recorder {
	return &Recorder{repo: repo, bus: bus, log: log}
}

// Emit stores and broadcasts a case event. Payload must only contain
// player-safe data — it is delivered to the client.
func (rec *Recorder) Emit(ctx context.Context, caseID uuid.UUID, eventType string, payload map[string]any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		rec.log.Error("marshal event payload", "error", err, "type", eventType)
		raw = []byte(`{}`)
	}
	ev, err := rec.repo.Insert(ctx, caseID, eventType, raw)
	if err != nil {
		rec.log.Error("persist case event", "error", err, "type", eventType)
		return
	}
	rec.bus.Publish(notification.Event{
		ID:        ev.ID,
		CaseID:    caseID,
		Type:      eventType,
		Payload:   payload,
		CreatedAt: ev.CreatedAt,
	})
}
