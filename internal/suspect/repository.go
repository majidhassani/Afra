package suspect

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, s *Suspect) error
	ListByCase(ctx context.Context, caseID uuid.UUID) ([]Suspect, error)
	GetByID(ctx context.Context, caseID, suspectID uuid.UUID) (*Suspect, error)
	UpdateInterrogationState(ctx context.Context, suspectID uuid.UUID, stress, trust, count int) error
	AppendPrivateMemory(ctx context.Context, suspectID uuid.UUID, entry string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const suspectColumns = `id, case_id, name, age, job, relation_to_victim, public_profile, personality,
	known_facts, stress_level, trust_level, interrogation_count, is_culprit, secrets, lie_profile,
	private_memory, created_at, updated_at`

func scanSuspect(row pgx.Row, s *Suspect) error {
	return row.Scan(&s.ID, &s.CaseID, &s.Name, &s.Age, &s.Job, &s.RelationToVictim, &s.PublicProfile,
		&s.Personality, &s.KnownFacts, &s.StressLevel, &s.TrustLevel, &s.InterrogationCount,
		&s.IsCulprit, &s.Secrets, &s.LieProfile, &s.PrivateMemory, &s.CreatedAt, &s.UpdatedAt)
}

func orEmpty(raw []byte, def string) []byte {
	if len(raw) == 0 {
		return []byte(def)
	}
	return raw
}

func (r *PGRepository) Create(ctx context.Context, s *Suspect) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO suspects
		 (case_id, name, age, job, relation_to_victim, public_profile, personality, known_facts,
		  stress_level, trust_level, is_culprit, secrets, lie_profile, private_memory)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING id, created_at, updated_at`,
		s.CaseID, s.Name, s.Age, s.Job, s.RelationToVictim,
		orEmpty(s.PublicProfile, `{}`), orEmpty(s.Personality, `{}`), orEmpty(s.KnownFacts, `[]`),
		s.StressLevel, s.TrustLevel, s.IsCulprit,
		orEmpty(s.Secrets, `[]`), orEmpty(s.LieProfile, `{}`), orEmpty(s.PrivateMemory, `[]`),
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return apperrors.Internal(err, "create suspect")
	}
	return nil
}

func (r *PGRepository) ListByCase(ctx context.Context, caseID uuid.UUID) ([]Suspect, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+suspectColumns+` FROM suspects WHERE case_id = $1 ORDER BY created_at`, caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list suspects")
	}
	defer rows.Close()
	suspects := []Suspect{}
	for rows.Next() {
		var s Suspect
		if err := scanSuspect(rows, &s); err != nil {
			return nil, apperrors.Internal(err, "scan suspect")
		}
		suspects = append(suspects, s)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate suspects")
	}
	return suspects, nil
}

func (r *PGRepository) GetByID(ctx context.Context, caseID, suspectID uuid.UUID) (*Suspect, error) {
	s := &Suspect{}
	err := scanSuspect(r.pool.QueryRow(ctx,
		`SELECT `+suspectColumns+` FROM suspects WHERE id = $1 AND case_id = $2`, suspectID, caseID), s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NotFound("suspect_not_found", "suspect not found")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "get suspect")
	}
	return s, nil
}

func (r *PGRepository) UpdateInterrogationState(ctx context.Context, suspectID uuid.UUID, stress, trust, count int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE suspects SET stress_level = $2, trust_level = $3, interrogation_count = $4, updated_at = now()
		 WHERE id = $1`, suspectID, stress, trust, count)
	if err != nil {
		return apperrors.Internal(err, "update suspect interrogation state")
	}
	return nil
}

// AppendPrivateMemory records an internal note about what the suspect has
// already been asked/claimed, so future interrogations stay consistent.
func (r *PGRepository) AppendPrivateMemory(ctx context.Context, suspectID uuid.UUID, entry string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE suspects SET private_memory = private_memory || to_jsonb($2::text), updated_at = now()
		 WHERE id = $1`, suspectID, entry)
	if err != nil {
		return apperrors.Internal(err, "append suspect private memory")
	}
	return nil
}
