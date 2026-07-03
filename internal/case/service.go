package cases

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"casemind/internal/evidence"
	"casemind/internal/history"
	"casemind/internal/suspect"
	apperrors "casemind/pkg/errors"
)

// DetectiveStats is the slice of the detective module the case service needs.
type DetectiveStats interface {
	IncrementTotalCases(ctx context.Context, userID uuid.UUID) error
}

type Service struct {
	repo       Repository
	suspects   suspect.Repository
	evidences  evidence.Repository
	events     history.Repository
	detectives DetectiveStats
	generator  *Generator
	log        *slog.Logger
}

func NewService(
	repo Repository,
	suspects suspect.Repository,
	evidences evidence.Repository,
	events history.Repository,
	detectives DetectiveStats,
	generator *Generator,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo, suspects: suspects, evidences: evidences, events: events,
		detectives: detectives, generator: generator, log: log,
	}
}

// Create creates the case in `generating` status and runs the generation
// pipeline in the background.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, caseType, difficulty string, languages ...string) (*Case, error) {
	if err := ValidateNewCase(caseType, difficulty); err != nil {
		return nil, err
	}
	language := "en"
	if len(languages) > 0 {
		language = NormalizeLanguage(languages[0])
	}
	if err := ValidateNewCaseLanguage(language); err != nil {
		return nil, err
	}
	c := &Case{
		UserID:     userID,
		Title:      "Generating case...",
		Type:       caseType,
		Difficulty: difficulty,
		Status:     StatusGenerating,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	if err := s.detectives.IncrementTotalCases(ctx, userID); err != nil {
		s.log.Error("increment total cases", "error", err)
	}

	// Generation runs detached from the request context.
	genCase := *c
	go func() {
		genCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.generator.Run(genCtx, &genCase, language)
	}()
	return c, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Case, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Dashboard is the case detail view: the case plus the player layer
// (public suspects and discovered evidence).
type Dashboard struct {
	Case     *Case                     `json:"case"`
	Suspects []suspect.PublicSuspect   `json:"suspects"`
	Evidence []evidence.PublicEvidence `json:"evidence"`
}

func (s *Service) Get(ctx context.Context, userID, caseID uuid.UUID) (*Dashboard, error) {
	c, err := s.repo.GetForUser(ctx, userID, caseID)
	if err != nil {
		return nil, err
	}
	suspects, err := s.suspects.ListByCase(ctx, caseID)
	if err != nil {
		return nil, err
	}
	discovered, err := s.evidences.ListDiscovered(ctx, caseID)
	if err != nil {
		return nil, err
	}
	return &Dashboard{
		Case:     c,
		Suspects: suspect.PublicList(suspects),
		Evidence: evidence.PublicList(discovered),
	}, nil
}

func (s *Service) Archive(ctx context.Context, userID, caseID uuid.UUID) (*Case, error) {
	c, err := s.repo.GetForUser(ctx, userID, caseID)
	if err != nil {
		return nil, err
	}
	if c.Status == StatusGenerating {
		return nil, apperrors.Conflict("case_generating", "cannot archive a case while it is generating")
	}
	if err := s.repo.UpdateStatus(ctx, caseID, StatusArchived); err != nil {
		return nil, err
	}
	c.Status = StatusArchived
	return c, nil
}

func (s *Service) Events(ctx context.Context, userID, caseID uuid.UUID, limit int) ([]history.CaseEvent, error) {
	if err := s.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	return s.events.ListByCase(ctx, caseID, limit)
}

// --- Guard methods implemented for other modules' gateway interfaces ---

// EnsureOwned fails with not-found unless the case exists and belongs to
// the user.
func (s *Service) EnsureOwned(ctx context.Context, userID, caseID uuid.UUID) error {
	_, err := s.repo.GetForUser(ctx, userID, caseID)
	return err
}

// EnsureOwnedOpen additionally requires the case to be open for play.
func (s *Service) EnsureOwnedOpen(ctx context.Context, userID, caseID uuid.UUID) error {
	c, err := s.repo.GetForUser(ctx, userID, caseID)
	if err != nil {
		return err
	}
	if c.Status != StatusOpen {
		return apperrors.Conflict("case_not_open", "this case is not open for investigation (status: "+c.Status+")")
	}
	return nil
}

// SolveInfo returns what the solve module needs, with ownership enforced.
func (s *Service) SolveInfo(ctx context.Context, userID, caseID uuid.UUID) (status, difficulty, summary string, err error) {
	c, err := s.repo.GetForUser(ctx, userID, caseID)
	if err != nil {
		return "", "", "", err
	}
	return c.Status, c.Difficulty, c.Summary, nil
}

// SummaryOf returns the player-facing case summary (post-guard internal use).
func (s *Service) SummaryOf(ctx context.Context, userID, caseID uuid.UUID) (string, error) {
	c, err := s.repo.GetForUser(ctx, userID, caseID)
	if err != nil {
		return "", err
	}
	return c.Summary, nil
}

func (s *Service) MarkSolved(ctx context.Context, caseID uuid.UUID) error {
	return s.repo.MarkSolved(ctx, caseID)
}

func (s *Service) MarkFailed(ctx context.Context, caseID uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, caseID, StatusFailed)
}
