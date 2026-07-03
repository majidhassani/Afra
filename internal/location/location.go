// Package location manages case locations; only discovered locations are
// ever returned to the client.
package location

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Location struct {
	ID                 uuid.UUID
	CaseID             uuid.UUID
	Name               string
	Type               string
	Latitude           float64
	Longitude          float64
	Description        string
	Discovered         bool
	RelatedEvidenceIDs json.RawMessage
	RelatedSuspectIDs  json.RawMessage
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// PublicLocation is the client-facing shape (only for discovered locations).
type PublicLocation struct {
	ID                 uuid.UUID       `json:"id"`
	CaseID             uuid.UUID       `json:"case_id"`
	Name               string          `json:"name"`
	Type               string          `json:"type"`
	Latitude           float64         `json:"latitude"`
	Longitude          float64         `json:"longitude"`
	Description        string          `json:"description"`
	RelatedEvidenceIDs json.RawMessage `json:"related_evidence_ids"`
	RelatedSuspectIDs  json.RawMessage `json:"related_suspect_ids"`
}

func (l *Location) Public() PublicLocation {
	return PublicLocation{
		ID: l.ID, CaseID: l.CaseID, Name: l.Name, Type: l.Type,
		Latitude: l.Latitude, Longitude: l.Longitude, Description: l.Description,
		RelatedEvidenceIDs: l.RelatedEvidenceIDs, RelatedSuspectIDs: l.RelatedSuspectIDs,
	}
}

type Repository interface {
	Create(ctx context.Context, l *Location) error
	ListDiscovered(ctx context.Context, caseID uuid.UUID) ([]Location, error)
	RevealByIDs(ctx context.Context, caseID uuid.UUID, ids []uuid.UUID) (int, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func orEmpty(raw []byte, def string) []byte {
	if len(raw) == 0 {
		return []byte(def)
	}
	return raw
}

func (r *PGRepository) Create(ctx context.Context, l *Location) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO locations
		 (case_id, name, type, latitude, longitude, description, discovered, related_evidence_ids, related_suspect_ids)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at, updated_at`,
		l.CaseID, l.Name, l.Type, l.Latitude, l.Longitude, l.Description, l.Discovered,
		orEmpty(l.RelatedEvidenceIDs, `[]`), orEmpty(l.RelatedSuspectIDs, `[]`),
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create location")
	}
	return nil
}

func (r *PGRepository) ListDiscovered(ctx context.Context, caseID uuid.UUID) ([]Location, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, case_id, name, type, latitude, longitude, description, discovered,
		        related_evidence_ids, related_suspect_ids, created_at, updated_at
		 FROM locations WHERE case_id = $1 AND discovered = true ORDER BY created_at`, caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list locations")
	}
	defer rows.Close()
	items := []Location{}
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.CaseID, &l.Name, &l.Type, &l.Latitude, &l.Longitude, &l.Description,
			&l.Discovered, &l.RelatedEvidenceIDs, &l.RelatedSuspectIDs, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan location")
		}
		items = append(items, l)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate locations")
	}
	return items, nil
}

func (r *PGRepository) RevealByIDs(ctx context.Context, caseID uuid.UUID, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE locations SET discovered = true, updated_at = now() WHERE case_id = $1 AND id = ANY($2)`,
		caseID, ids)
	if err != nil {
		return 0, apperrors.Internal(err, "reveal locations")
	}
	return int(tag.RowsAffected()), nil
}
