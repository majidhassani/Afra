package playerprofile

import (
	"context"

	"github.com/google/uuid"

	"casemind/pkg/validator"
)

// CoinStats is the slice of the wallet module this service reads.
type CoinStats interface {
	SpentAndEarned(ctx context.Context, userID uuid.UUID) (spent int, earned int, err error)
}

type Service struct {
	repo  Repository
	coins CoinStats
}

func NewService(repo Repository, coins CoinStats) *Service {
	return &Service{repo: repo, coins: coins}
}

// CreateProfile satisfies auth.ProfileCreator.
func (s *Service) CreateProfile(ctx context.Context, userID uuid.UUID) error {
	return s.repo.Create(ctx, userID)
}

func (s *Service) Profile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *Service) UpdateDisplayName(ctx context.Context, userID uuid.UUID, displayName string) (*Profile, error) {
	if err := validator.New().
		Required("display_name", displayName).MaxLen("display_name", displayName, 60).
		Err(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDisplayName(ctx, userID, displayName); err != nil {
		return nil, err
	}
	return s.repo.GetByUserID(ctx, userID)
}

func (s *Service) Stats(ctx context.Context, userID uuid.UUID) (*Stats, error) {
	p, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	spent, earned, err := s.coins.SpentAndEarned(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &Stats{Profile: p, TotalCoinsSpent: spent, TotalCoinsEarned: earned}, nil
}

func (s *Service) History(ctx context.Context, userID uuid.UUID) ([]HistoryEntry, error) {
	return s.repo.History(ctx, userID)
}

// --- hooks used by the mission engine ---

func (s *Service) IncrementTotalMissions(ctx context.Context, userID uuid.UUID) error {
	return s.repo.IncrementTotalMissions(ctx, userID)
}

func (s *Service) ApplyMissionResult(ctx context.Context, userID uuid.UUID, completed bool, xpDelta int) error {
	return s.repo.ApplyMissionResult(ctx, userID, completed, xpDelta)
}

func (s *Service) AddCounters(ctx context.Context, userID uuid.UUID, clues, aiInteractions, locations int) error {
	return s.repo.AddCounters(ctx, userID, clues, aiInteractions, locations)
}
