// Package memory manages the player's discovered facts — the growing
// player-layer knowledge base fed into agent context — and optionally
// mirrors facts into Qdrant for future semantic recall.
package memory

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"casemind/internal/platform/vector"
	apperrors "casemind/pkg/errors"
)

type Fact struct {
	ID        uuid.UUID `json:"id"`
	CaseID    uuid.UUID `json:"case_id"`
	Fact      string    `json:"fact"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type FactRepository interface {
	Add(ctx context.Context, caseID uuid.UUID, fact, source string) (*Fact, error)
	ListByCase(ctx context.Context, caseID uuid.UUID) ([]Fact, error)
}

type PGFactRepository struct{ pool *pgxpool.Pool }

func NewPGFactRepository(pool *pgxpool.Pool) *PGFactRepository {
	return &PGFactRepository{pool: pool}
}

func (r *PGFactRepository) Add(ctx context.Context, caseID uuid.UUID, fact, source string) (*Fact, error) {
	f := &Fact{CaseID: caseID, Fact: fact, Source: source}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO discovered_facts (case_id, fact, source) VALUES ($1, $2, $3)
		 RETURNING id, created_at`, caseID, fact, source,
	).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return nil, apperrors.Internal(err, "add discovered fact")
	}
	return f, nil
}

func (r *PGFactRepository) ListByCase(ctx context.Context, caseID uuid.UUID) ([]Fact, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, case_id, fact, source, created_at FROM discovered_facts
		 WHERE case_id = $1 ORDER BY created_at`, caseID)
	if err != nil {
		return nil, apperrors.Internal(err, "list discovered facts")
	}
	defer rows.Close()
	facts := []Fact{}
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.CaseID, &f.Fact, &f.Source, &f.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan discovered fact")
		}
		facts = append(facts, f)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate discovered facts")
	}
	return facts, nil
}

// Service is the case memory used to assemble agent context.
type Service struct {
	facts  FactRepository
	vector *vector.Client // optional; nil when Qdrant is not configured
	log    *slog.Logger
}

func NewService(facts FactRepository, vc *vector.Client, log *slog.Logger) *Service {
	return &Service{facts: facts, vector: vc, log: log}
}

// AddFacts stores new discovered facts, skipping empties and duplicates of
// the exact same text.
func (s *Service) AddFacts(ctx context.Context, caseID uuid.UUID, source string, facts []string) []Fact {
	existing, err := s.facts.ListByCase(ctx, caseID)
	if err != nil {
		s.log.Error("list facts for dedupe", "error", err)
		existing = nil
	}
	known := map[string]bool{}
	for _, f := range existing {
		known[f.Fact] = true
	}
	added := []Fact{}
	for _, text := range facts {
		if text == "" || known[text] {
			continue
		}
		f, err := s.facts.Add(ctx, caseID, text, source)
		if err != nil {
			s.log.Error("add fact", "error", err)
			continue
		}
		known[text] = true
		added = append(added, *f)
	}
	return added
}

func (s *Service) Facts(ctx context.Context, caseID uuid.UUID) ([]Fact, error) {
	return s.facts.ListByCase(ctx, caseID)
}

// FactTexts returns just the fact strings for prompt building.
func (s *Service) FactTexts(ctx context.Context, caseID uuid.UUID) []string {
	facts, err := s.facts.ListByCase(ctx, caseID)
	if err != nil {
		s.log.Error("list facts", "error", err)
		return nil
	}
	out := make([]string, 0, len(facts))
	for _, f := range facts {
		out = append(out, f.Fact)
	}
	return out
}
