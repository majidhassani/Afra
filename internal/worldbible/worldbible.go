// Package worldbible stores the private Truth Layer of every mission — the
// backend game master document.
//
// SECURITY INVARIANT: nothing in this package may ever be serialized into a
// client response. The World Bible is only read by application services to
// build internal agent context. The struct deliberately has NO json tags.
package worldbible

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

// Bible is the internal-only truth record for one mission.
type Bible struct {
	ID               uuid.UUID
	MissionID        uuid.UUID
	Truth            json.RawMessage
	HiddenState      json.RawMessage
	CharacterSecrets json.RawMessage
	ClueTruth        json.RawMessage
	MapTruth         json.RawMessage
	TimelineTruth    json.RawMessage
	FailureRules     json.RawMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Repository interface {
	Create(ctx context.Context, b *Bible) error
	GetByMissionID(ctx context.Context, missionID uuid.UUID) (*Bible, error)
	// UpdateHiddenState replaces the world simulation state (time engine).
	UpdateHiddenState(ctx context.Context, missionID uuid.UUID, hiddenState json.RawMessage) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func orEmpty(raw json.RawMessage, def string) []byte {
	if len(raw) == 0 {
		return []byte(def)
	}
	return raw
}

func (r *PGRepository) Create(ctx context.Context, b *Bible) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO world_bibles
		 (mission_id, truth, hidden_state, character_secrets, clue_truth, map_truth, timeline_truth, failure_rules)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at, updated_at`,
		b.MissionID,
		orEmpty(b.Truth, `{}`), orEmpty(b.HiddenState, `{}`), orEmpty(b.CharacterSecrets, `{}`),
		orEmpty(b.ClueTruth, `{}`), orEmpty(b.MapTruth, `{}`), orEmpty(b.TimelineTruth, `[]`),
		orEmpty(b.FailureRules, `[]`),
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create world bible")
	}
	return nil
}

func (r *PGRepository) GetByMissionID(ctx context.Context, missionID uuid.UUID) (*Bible, error) {
	b := &Bible{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, mission_id, truth, hidden_state, character_secrets, clue_truth, map_truth,
		        timeline_truth, failure_rules, created_at, updated_at
		 FROM world_bibles WHERE mission_id = $1`, missionID,
	).Scan(&b.ID, &b.MissionID, &b.Truth, &b.HiddenState, &b.CharacterSecrets, &b.ClueTruth,
		&b.MapTruth, &b.TimelineTruth, &b.FailureRules, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("world_bible_not_found", "world bible not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get world bible")
	}
	return b, nil
}

func (r *PGRepository) UpdateHiddenState(ctx context.Context, missionID uuid.UUID, hiddenState json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE world_bibles SET hidden_state = $2, updated_at = now() WHERE mission_id = $1`,
		missionID, orEmpty(hiddenState, `{}`))
	if err != nil {
		return apperrors.Internal(err, "update world bible hidden state")
	}
	return nil
}
