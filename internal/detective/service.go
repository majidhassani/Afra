package detective

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// CreateProfile satisfies auth.ProfileCreator.
func (s *Service) CreateProfile(ctx context.Context, userID uuid.UUID) error {
	return s.repo.CreateProfile(ctx, userID)
}

func (s *Service) Profile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *Service) History(ctx context.Context, userID uuid.UUID) ([]HistoryEntry, error) {
	return s.repo.History(ctx, userID)
}

func (s *Service) IncrementTotalCases(ctx context.Context, userID uuid.UUID) error {
	return s.repo.IncrementTotalCases(ctx, userID)
}

func (s *Service) ApplyCaseResult(ctx context.Context, userID uuid.UUID, solved bool, xpDelta int) error {
	return s.repo.ApplyCaseResult(ctx, userID, solved, xpDelta)
}
