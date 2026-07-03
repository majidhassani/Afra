package evidence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, e *Evidence) error
	ListDiscovered(ctx context.Context, caseID uuid.UUID) ([]Evidence, error)
	GetByID(ctx context.Context, caseID, evidenceID uuid.UUID) (*Evidence, error)
	MarkDiscovered(ctx context.Context, evidenceID uuid.UUID) error
	FindUndiscoveredByTitle(ctx context.Context, caseID uuid.UUID, title string) (*Evidence, error)
	ListUndiscoveredTitles(ctx context.Context, caseID uuid.UUID) ([]string, error)
	AdjustReliability(ctx context.Context, evidenceID uuid.UUID, delta int) error
	AddInspection(ctx context.Context, ins *Inspection) error
	ListInspections(ctx context.Context, caseID, evidenceID uuid.UUID) ([]Inspection, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const evidenceColumns = `id, case_id, title, type, description, public_data, internal_truth, discovered,
	reliability, related_suspect_ids, related_location_ids, related_timeline_event_ids, attachment_url,
	created_at, updated_at`

func scanEvidence(row pgx.Row, e *Evidence) error {
	return row.Scan(&e.ID, &e.CaseID, &e.Title, &e.Type, &e.Description, &e.PublicData, &e.InternalTruth,
		&e.Discovered, &e.Reliability, &e.RelatedSuspectIDs, &e.RelatedLocationIDs,
		&e.RelatedTimelineEventIDs, &e.AttachmentURL, &e.CreatedAt, &e.UpdatedAt)
}

func orEmpty(raw []byte, def string) []byte {
	if len(raw) == 0 {
		return []byte(def)
	}
	return raw
}

func (r *PGRepository) Create(ctx context.Context, e *Evidence) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO evidence
		 (case_id, title, type, description, public_data, internal_truth, discovered, reliability,
		  related_suspect_ids, related_location_ids, related_timeline_event_ids, attachment_url)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at, updated_at`,
		e.CaseID, e.Title, e.Type, e.Description,
		orEmpty(e.PublicData, `{}`), orEmpty(e.InternalTruth, `{}`), e.Discovered, e.Reliability,
		orEmpty(e.RelatedSuspectIDs, `[]`), orEmpty(e.RelatedLocationIDs, `[]`),
		orEmpty(e.RelatedTimelineEventIDs, `[]`), e.AttachmentURL,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create evidence")
	}
	return nil
}

func (r *PGRepository) ListDiscovered(ctx context.Context, caseID uuid.UUID) ([]Evidence, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+evidenceColumns+` FROM evidence WHERE case_id = $1 AND discovered = true ORDER BY created_at`,
		caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list evidence")
	}
	defer rows.Close()
	items := []Evidence{}
	for rows.Next() {
		var e Evidence
		if err := scanEvidence(rows, &e); err != nil {
			return nil, apperrors.Internal(err, "scan evidence")
		}
		items = append(items, e)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate evidence")
	}
	return items, nil
}

func (r *PGRepository) GetByID(ctx context.Context, caseID, evidenceID uuid.UUID) (*Evidence, error) {
	e := &Evidence{}
	err := scanEvidence(r.pool.QueryRow(ctx,
		`SELECT `+evidenceColumns+` FROM evidence WHERE id = $1 AND case_id = $2`, evidenceID, caseID), e)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("evidence_not_found", "evidence not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get evidence")
	}
	return e, nil
}

func (r *PGRepository) MarkDiscovered(ctx context.Context, evidenceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE evidence SET discovered = true, updated_at = now() WHERE id = $1`, evidenceID)
	if err != nil {
		return apperrors.Internal(err, "mark evidence discovered")
	}
	return nil
}

func (r *PGRepository) FindUndiscoveredByTitle(ctx context.Context, caseID uuid.UUID, title string) (*Evidence, error) {
	e := &Evidence{}
	err := scanEvidence(r.pool.QueryRow(ctx,
		`SELECT `+evidenceColumns+` FROM evidence
		 WHERE case_id = $1 AND discovered = false AND lower(title) = lower($2) LIMIT 1`, caseID, title), e)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("evidence_not_found", "evidence not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "find evidence by title")
	}
	return e, nil
}

func (r *PGRepository) ListUndiscoveredTitles(ctx context.Context, caseID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT title FROM evidence WHERE case_id = $1 AND discovered = false ORDER BY created_at`, caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list undiscovered evidence titles")
	}
	defer rows.Close()
	titles := []string{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, apperrors.Internal(err, "scan evidence title")
		}
		titles = append(titles, t)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate evidence titles")
	}
	return titles, nil
}

func (r *PGRepository) AdjustReliability(ctx context.Context, evidenceID uuid.UUID, delta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE evidence SET reliability = greatest(0, least(100, reliability + $2)), updated_at = now()
		 WHERE id = $1`, evidenceID, delta)
	if err != nil {
		return apperrors.Internal(err, "adjust evidence reliability")
	}
	return nil
}

func (r *PGRepository) AddInspection(ctx context.Context, ins *Inspection) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO evidence_inspections (evidence_id, case_id, question, analysis)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		ins.EvidenceID, ins.CaseID, ins.Question, ins.Analysis,
	).Scan(&ins.ID, &ins.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "add evidence inspection")
	}
	return nil
}

func (r *PGRepository) ListInspections(ctx context.Context, caseID, evidenceID uuid.UUID) ([]Inspection, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, evidence_id, case_id, question, analysis, created_at FROM evidence_inspections
		 WHERE case_id = $1 AND evidence_id = $2 ORDER BY created_at`, caseID, evidenceID)
	if err != nil {
		return nil, apperrors.Internal(err, "list inspections")
	}
	defer rows.Close()
	items := []Inspection{}
	for rows.Next() {
		var ins Inspection
		if err := rows.Scan(&ins.ID, &ins.EvidenceID, &ins.CaseID, &ins.Question, &ins.Analysis, &ins.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan inspection")
		}
		items = append(items, ins)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate inspections")
	}
	return items, nil
}
