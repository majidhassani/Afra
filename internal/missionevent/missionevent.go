// Package missionevent persists mission events (the mission audit trail /
// activity feed) and broadcasts them on the notification bus for SSE.
package missionevent

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

type Event struct {
	ID        uuid.UUID       `json:"id"`
	MissionID uuid.UUID       `json:"mission_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type Repository interface {
	Insert(ctx context.Context, missionID uuid.UUID, eventType string, payload []byte) (*Event, error)
	ListByMission(ctx context.Context, missionID uuid.UUID, limit int) ([]Event, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Insert(ctx context.Context, missionID uuid.UUID, eventType string, payload []byte) (*Event, error) {
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	ev := &Event{MissionID: missionID, Type: eventType, Payload: payload}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO mission_events (mission_id, type, payload) VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		missionID, eventType, payload,
	).Scan(&ev.ID, &ev.CreatedAt)
	if err != nil {
		return nil, apperrors.Internal(err, "insert mission event")
	}
	return ev, nil
}

func (r *PGRepository) ListByMission(ctx context.Context, missionID uuid.UUID, limit int) ([]Event, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, mission_id, type, payload, created_at FROM mission_events
		 WHERE mission_id = $1 ORDER BY created_at DESC LIMIT $2`, missionID, limit)
	if err != nil {
		return nil, apperrors.Internal(err, "list mission events")
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var ev Event
		if err := rows.Scan(&ev.ID, &ev.MissionID, &ev.Type, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan mission event")
		}
		events = append(events, ev)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate mission events")
	}
	return events, nil
}

// FactTexts extracts discovered facts (events of type "fact_discovered")
// from a mission event list, oldest first. Facts are the player-safe memory
// used as agent context.
func FactTexts(events []Event) []string {
	facts := []string{}
	for i := len(events) - 1; i >= 0; i-- { // events arrive newest-first
		if events[i].Type != "fact_discovered" {
			continue
		}
		var payload struct {
			Fact string `json:"fact"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err == nil && payload.Fact != "" {
			facts = append(facts, payload.Fact)
		}
	}
	return facts
}

// Recorder persists events and publishes them to live SSE subscribers.
type Recorder struct {
	repo Repository
	bus  *notification.Bus
	log  *slog.Logger
}

func NewRecorder(repo Repository, bus *notification.Bus, log *slog.Logger) *Recorder {
	return &Recorder{repo: repo, bus: bus, log: log}
}

// Emit stores and broadcasts a mission event. Payload must only contain
// player-safe data — it is delivered to the client.
func (rec *Recorder) Emit(ctx context.Context, missionID uuid.UUID, eventType string, payload map[string]any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		rec.log.Error("marshal mission event payload", "error", err, "type", eventType)
		raw = []byte(`{}`)
	}
	ev, err := rec.repo.Insert(ctx, missionID, eventType, raw)
	if err != nil {
		rec.log.Error("persist mission event", "error", err, "type", eventType)
		return
	}
	rec.bus.Publish(notification.Event{
		ID:        ev.ID,
		CaseID:    missionID, // bus topic key; reused for missions
		Type:      eventType,
		Payload:   payload,
		CreatedAt: ev.CreatedAt,
	})
}
