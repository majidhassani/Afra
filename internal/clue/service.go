package clue

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"casemind/internal/agent/clueagent"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/missionevent"
	"casemind/internal/wallet"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

// MissionGateway is the slice of the mission module this service needs; it
// is implemented by mission.Service.
type MissionGateway interface {
	EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error
	EnsureOwnedActive(ctx context.Context, userID, missionID uuid.UUID) error
	SummaryOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	ObjectiveTitles(ctx context.Context, userID, missionID uuid.UUID) ([]string, error)
	ApplyActionTime(ctx context.Context, missionID uuid.UUID, action string) (*missionevent.TimeUpdate, error)
}

// ProfileCounter is the slice of the player profile module this service uses.
type ProfileCounter interface {
	AddCounters(ctx context.Context, userID uuid.UUID, clues, aiInteractions, locations int) error
}

type Service struct {
	repo     Repository
	guard    MissionGateway
	wallet   *wallet.Guard
	orch     *orchestrator.Orchestrator
	events   missionevent.Repository
	recorder *missionevent.Recorder
	profiles ProfileCounter
	log      *slog.Logger
}

func NewService(
	repo Repository,
	guard MissionGateway,
	walletGuard *wallet.Guard,
	orch *orchestrator.Orchestrator,
	events missionevent.Repository,
	recorder *missionevent.Recorder,
	profiles ProfileCounter,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo, guard: guard, wallet: walletGuard, orch: orch,
		events: events, recorder: recorder, profiles: profiles, log: log,
	}
}

// List returns the discovered clues only. Undiscovered clues do not exist as
// far as the client is concerned.
func (s *Service) List(ctx context.Context, userID, missionID uuid.UUID) ([]PublicClue, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	clues, err := s.repo.ListDiscovered(ctx, missionID)
	if err != nil {
		return nil, err
	}
	return PublicList(clues), nil
}

func (s *Service) Get(ctx context.Context, userID, missionID, clueID uuid.UUID) (*PublicClue, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	c, err := s.repo.GetByID(ctx, missionID, clueID)
	if err != nil {
		return nil, err
	}
	if !c.Discovered {
		// Hide the existence of undiscovered clues.
		return nil, apperrors.NotFound("clue_not_found", "clue not found")
	}
	pub := c.Public()
	return &pub, nil
}

// InspectResult is the player-safe outcome of a paid clue inspection.
type InspectResult struct {
	Analysis   string                   `json:"analysis"`
	NewFacts   []string                 `json:"new_facts"`
	Clue       PublicClue               `json:"clue"`
	Cost       wallet.Cost              `json:"cost"`
	TimeUpdate *missionevent.TimeUpdate `json:"time_update,omitempty"`
}

func (s *Service) Inspect(ctx context.Context, userID, missionID, clueID uuid.UUID, question, language string) (*InspectResult, error) {
	if err := validator.New().MaxLen("question", question, 1000).Err(); err != nil {
		return nil, err
	}
	if err := s.guard.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}
	c, err := s.repo.GetByID(ctx, missionID, clueID)
	if err != nil {
		return nil, err
	}
	if !c.Discovered {
		return nil, apperrors.NotFound("clue_not_found", "clue not found")
	}
	summary, err := s.guard.SummaryOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}

	res, err := s.wallet.Reserve(ctx, userID, &missionID, wallet.ActionClueInspect)
	if err != nil {
		return nil, err
	}
	outAny, meta, err := s.orch.RunWithMeta(ctx, clueagent.Name, runtime.Task{
		Type: clueagent.TaskInspect, MissionID: missionID, UserID: userID,
		Input: clueagent.InspectInput{
			MissionSummary:  summary,
			Title:           c.Title,
			Type:            c.Type,
			ShortDesc:       c.ShortDescription,
			DetailedDesc:    c.DetailedDescription,
			PublicData:      rawToMap(c.PublicData),
			InternalTruth:   rawToMap(c.InternalTruth),
			Question:        question,
			DiscoveredFacts: s.factTexts(ctx, missionID),
			Language:        runtime.Language(language),
		},
	})
	if err != nil {
		s.wallet.Release(ctx, res, clueagent.Name)
		return nil, err
	}
	charged := s.wallet.Settle(ctx, res, clueagent.Name, meta)
	out := outAny.(*clueagent.InspectOutput)

	if out.ReliabilityDelta != 0 {
		if err := s.repo.AdjustReliability(ctx, c.ID, out.ReliabilityDelta); err != nil {
			s.log.Error("adjust clue reliability", "error", err)
		} else {
			c.Reliability = clamp(c.Reliability+out.ReliabilityDelta, 0, 100)
		}
	}
	for _, fact := range out.NewFacts {
		s.recorder.Emit(ctx, missionID, "fact_discovered", map[string]any{
			"fact": fact, "source": "clue_inspection:" + c.Title,
		})
	}
	s.recorder.Emit(ctx, missionID, "clue_inspected", map[string]any{
		"clue_id": c.ID, "title": c.Title,
	})
	// Advance the evidence lifecycle: discovered → inspected (never downgrade a
	// clue that is already confirmed).
	if c.LifecycleStatus() == StatusDiscovered {
		if err := s.repo.SetStatus(ctx, c.ID, StatusInspected); err != nil {
			s.log.Error("advance clue status", "error", err)
		} else {
			c.Status = StatusInspected
		}
	}
	if err := s.profiles.AddCounters(ctx, userID, 0, 1, 0); err != nil {
		s.log.Error("profile counter", "error", err)
	}

	// Inspecting evidence costs mission time.
	var timeUpdate *missionevent.TimeUpdate
	if tu, err := s.guard.ApplyActionTime(ctx, missionID, "clue_inspect"); err == nil {
		timeUpdate = tu
	} else {
		s.log.Error("inspect time cost", "error", err)
	}

	pub := c.Public()
	return &InspectResult{
		Analysis:   out.Analysis,
		NewFacts:   out.NewFacts,
		Clue:       pub,
		Cost:       wallet.Cost{CoinsCharged: charged},
		TimeUpdate: timeUpdate,
	}, nil
}

// ExplainResult is the player-safe outcome of a paid clue explanation.
type ExplainResult struct {
	Explanation string      `json:"explanation"`
	NextSteps   []string    `json:"next_steps"`
	CompareWith []string    `json:"compare_with"`
	Cost        wallet.Cost `json:"cost"`
}

func (s *Service) Explain(ctx context.Context, userID, missionID, clueID uuid.UUID, language string) (*ExplainResult, error) {
	if err := s.guard.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}
	c, err := s.repo.GetByID(ctx, missionID, clueID)
	if err != nil {
		return nil, err
	}
	if !c.Discovered {
		return nil, apperrors.NotFound("clue_not_found", "clue not found")
	}
	summary, err := s.guard.SummaryOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	objectives, err := s.guard.ObjectiveTitles(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	discovered, err := s.repo.ListDiscovered(ctx, missionID)
	if err != nil {
		return nil, err
	}
	titles := make([]string, 0, len(discovered))
	for i := range discovered {
		titles = append(titles, discovered[i].Title)
	}

	res, err := s.wallet.Reserve(ctx, userID, &missionID, wallet.ActionClueExplain)
	if err != nil {
		return nil, err
	}
	// The explanation agent gets PUBLIC clue data only.
	outAny, meta, err := s.orch.RunWithMeta(ctx, clueagent.Name, runtime.Task{
		Type: clueagent.TaskExplain, MissionID: missionID, UserID: userID,
		Input: clueagent.ExplainInput{
			MissionSummary:  summary,
			Title:           c.Title,
			Type:            c.Type,
			ShortDesc:       c.ShortDescription,
			DetailedDesc:    c.DetailedDescription,
			PublicData:      rawToMap(c.PublicData),
			DiscoveredClues: titles,
			Objectives:      objectives,
			Language:        runtime.Language(language),
		},
	})
	if err != nil {
		s.wallet.Release(ctx, res, clueagent.Name)
		return nil, err
	}
	charged := s.wallet.Settle(ctx, res, clueagent.Name, meta)
	out := outAny.(*clueagent.ExplainOutput)

	if err := s.profiles.AddCounters(ctx, userID, 0, 1, 0); err != nil {
		s.log.Error("profile counter", "error", err)
	}
	return &ExplainResult{
		Explanation: out.Explanation,
		NextSteps:   out.NextSteps,
		CompareWith: out.CompareWith,
		Cost:        wallet.Cost{CoinsCharged: charged},
	}, nil
}

func (s *Service) factTexts(ctx context.Context, missionID uuid.UUID) []string {
	events, err := s.events.ListByMission(ctx, missionID, 100)
	if err != nil {
		s.log.Error("list mission facts", "error", err)
		return nil
	}
	return missionevent.FactTexts(events)
}
