// Package casebible stores the private Truth Layer of every case.
//
// SECURITY INVARIANT: nothing in this package may ever be serialized into a
// client response. The Bible is only read by application services to build
// internal agent context.
package casebible

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

// Bible is the internal-only truth record. It deliberately has NO json tags
// for client serialization; raw JSONB fields are kept opaque.
type Bible struct {
	ID             uuid.UUID
	CaseID         uuid.UUID
	CulpritID      uuid.UUID
	Motive         string
	Truth          json.RawMessage
	RealTimeline   json.RawMessage
	HiddenFacts    json.RawMessage
	FalseLeads     json.RawMessage
	EvidenceTruth  json.RawMessage
	SuspectSecrets json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Repository interface {
	Create(ctx context.Context, b *Bible) error
	GetByCaseID(ctx context.Context, caseID uuid.UUID) (*Bible, error)
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
		`INSERT INTO case_bibles
		 (case_id, culprit_id, motive, truth, real_timeline, hidden_facts, false_leads, evidence_truth, suspect_secrets)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at, updated_at`,
		b.CaseID, b.CulpritID, b.Motive,
		orEmpty(b.Truth, `{}`), orEmpty(b.RealTimeline, `[]`), orEmpty(b.HiddenFacts, `[]`),
		orEmpty(b.FalseLeads, `[]`), orEmpty(b.EvidenceTruth, `{}`), orEmpty(b.SuspectSecrets, `{}`),
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create case bible")
	}
	return nil
}

func (r *PGRepository) GetByCaseID(ctx context.Context, caseID uuid.UUID) (*Bible, error) {
	b := &Bible{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, case_id, culprit_id, motive, truth, real_timeline, hidden_facts, false_leads, evidence_truth, suspect_secrets, created_at, updated_at
		 FROM case_bibles WHERE case_id = $1`, caseID,
	).Scan(&b.ID, &b.CaseID, &b.CulpritID, &b.Motive, &b.Truth, &b.RealTimeline, &b.HiddenFacts,
		&b.FalseLeads, &b.EvidenceTruth, &b.SuspectSecrets, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("case_bible_not_found", "case bible not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get case bible")
	}
	return b, nil
}
