// Package missioncomplete owns the final mission decision: it gates completion
// on readiness, runs the mission JudgeAgent against the private truth, and
// produces a clear win/loss result with objective-level and clue-level
// explanation, XP, and coin rewards.
package missioncomplete

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"casemind/internal/agent/missionjudge"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/interaction"
	"casemind/internal/mission"
	"casemind/internal/missionevent"
	"casemind/internal/wallet"
	"casemind/internal/worldbible"
	"casemind/pkg/validator"
)

// MissionGateway is the slice of the mission module this service needs; it is
// implemented by mission.Service.
type MissionGateway interface {
	CompletionSnapshot(ctx context.Context, userID, missionID uuid.UUID) (*mission.CompletionSnapshot, error)
	Finish(ctx context.Context, userID, missionID uuid.UUID, result json.RawMessage, success bool, completedKeys map[string]bool) error
	StoredResult(ctx context.Context, userID, missionID uuid.UUID) (json.RawMessage, string, error)
}

// ProfileProgression is the slice of the player-profile module this service
// uses to award mission results.
type ProfileProgression interface {
	ApplyMissionResult(ctx context.Context, userID uuid.UUID, completed bool, xpDelta int) error
}

type Service struct {
	gateway      MissionGateway
	clues        clue.Repository
	locations    gamemap.Repository
	interactions interaction.Repository
	events       missionevent.Repository
	bibles       worldbible.Repository
	orch         *orchestrator.Orchestrator
	wallet       *wallet.Guard
	walletSvc    *wallet.Service
	profiles     ProfileProgression
	recorder     *missionevent.Recorder
	log          *slog.Logger
}

func NewService(
	gateway MissionGateway,
	clues clue.Repository,
	locations gamemap.Repository,
	interactions interaction.Repository,
	events missionevent.Repository,
	bibles worldbible.Repository,
	orch *orchestrator.Orchestrator,
	walletGuard *wallet.Guard,
	walletSvc *wallet.Service,
	profiles ProfileProgression,
	recorder *missionevent.Recorder,
	log *slog.Logger,
) *Service {
	return &Service{
		gateway: gateway, clues: clues, locations: locations, interactions: interactions,
		events: events, bibles: bibles, orch: orch, wallet: walletGuard, walletSvc: walletSvc,
		profiles: profiles, recorder: recorder, log: log,
	}
}

// Decision is the player's final call submitted with the completion request.
type Decision struct {
	Outcome   string `json:"outcome"`
	Reasoning string `json:"reasoning"`
	// Language is the language the result feedback must be written in.
	Language string `json:"-"`
}

// CompletionCheck is returned when the mission is not yet ready to finish. Its
// CanComplete is always false; a ready mission returns a Result instead.
type CompletionCheck struct {
	CanComplete         bool     `json:"can_complete"`
	Reason              string   `json:"reason"`
	MissingRequirements []string `json:"missing_requirements"`
}

// Result is the player-safe end-of-mission evaluation.
type Result struct {
	CanComplete              bool        `json:"can_complete"`
	Success                  bool        `json:"success"`
	Score                    int         `json:"score"`
	Stars                    int         `json:"stars"`
	ResultTitle              string      `json:"result_title"`
	ResultSummary            string      `json:"result_summary"`
	CompletedObjectives      []string    `json:"completed_objectives"`
	FailedObjectives         []string    `json:"failed_objectives"`
	MissedOptionalObjectives []string    `json:"missed_optional_objectives"`
	CriticalCluesFound       []string    `json:"critical_clues_found"`
	CriticalCluesMissed      []string    `json:"critical_clues_missed"`
	GoodDecisions            []string    `json:"good_decisions"`
	BadDecisions             []string    `json:"bad_decisions"`
	XPReward                 int         `json:"xp_reward"`
	CoinReward               int         `json:"coin_reward"`
	Cost                     wallet.Cost `json:"cost"`
}

// Check reports whether the mission may be completed, without judging it or
// charging the player.
func (s *Service) Check(ctx context.Context, userID, missionID uuid.UUID) (*CompletionCheck, error) {
	snap, err := s.gateway.CompletionSnapshot(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	discovered, _, err := s.clues.Counts(ctx, missionID)
	if err != nil {
		return nil, err
	}
	visited, _, err := s.locations.CountVisited(ctx, missionID)
	if err != nil {
		return nil, err
	}
	interacted, err := s.interactions.CountInteractedCharacters(ctx, missionID)
	if err != nil {
		return nil, err
	}
	canComplete, missing := mission.EvaluateReadiness(mission.ReadinessInputs{
		DiscoveredClues:    discovered,
		InteractedChars:    interacted,
		VisitedLocations:   visited,
		RequiredClueTarget: snap.RequiredClueTarget,
	})
	if canComplete {
		s.emitReadyOnce(ctx, missionID)
	}
	return &CompletionCheck{
		CanComplete:         canComplete,
		Reason:              notReadyReason(canComplete, missing),
		MissingRequirements: missing,
	}, nil
}

// emitReadyOnce records the mission_ready_to_complete timeline milestone the
// first time readiness is observed. Repeated checks must not spam the story.
func (s *Service) emitReadyOnce(ctx context.Context, missionID uuid.UUID) {
	events, err := s.events.ListByMission(ctx, missionID, 200)
	if err != nil {
		return
	}
	for i := range events {
		if events[i].Type == "mission_ready_to_complete" {
			return
		}
	}
	s.recorder.Emit(ctx, missionID, "mission_ready_to_complete", map[string]any{
		"title":  "Ready for the final decision",
		"reason": "All completion requirements are met — submit your final judgment when you are confident.",
	})
}

// Complete gates on readiness, then runs the JudgeAgent and finalises the
// mission. When the mission is not ready it returns a CompletionCheck (nil
// Result) and charges nothing.
func (s *Service) Complete(ctx context.Context, userID, missionID uuid.UUID, dec Decision) (*Result, *CompletionCheck, error) {
	if err := validator.New().
		MaxLen("outcome", dec.Outcome, 2000).
		MaxLen("reasoning", dec.Reasoning, 5000).
		Err(); err != nil {
		return nil, nil, err
	}
	snap, err := s.gateway.CompletionSnapshot(ctx, userID, missionID)
	if err != nil {
		return nil, nil, err
	}

	discovered, totalClues, err := s.clues.Counts(ctx, missionID)
	if err != nil {
		return nil, nil, err
	}
	visited, totalLocations, err := s.locations.CountVisited(ctx, missionID)
	if err != nil {
		return nil, nil, err
	}
	interacted, err := s.interactions.CountInteractedCharacters(ctx, missionID)
	if err != nil {
		return nil, nil, err
	}

	// Readiness gate — never charged, never judged when unmet.
	canComplete, missing := mission.EvaluateReadiness(mission.ReadinessInputs{
		DiscoveredClues:    discovered,
		InteractedChars:    interacted,
		VisitedLocations:   visited,
		RequiredClueTarget: snap.RequiredClueTarget,
	})
	if !canComplete {
		return nil, &CompletionCheck{
			CanComplete:         false,
			Reason:              notReadyReason(false, missing),
			MissingRequirements: missing,
		}, nil
	}

	// Private truth is loaded here and never leaves this function except as
	// the JudgeAgent's already-sanitised feedback.
	bible, err := s.bibles.GetByMissionID(ctx, missionID)
	if err != nil {
		return nil, nil, err
	}
	facts := s.factTexts(ctx, missionID)

	res, err := s.wallet.Reserve(ctx, userID, &missionID, wallet.ActionFinalJudgment)
	if err != nil {
		return nil, nil, err
	}
	outAny, meta, err := s.orch.RunWithMeta(ctx, missionjudge.Name, runtime.Task{
		Type: missionjudge.TaskType, MissionID: missionID, UserID: userID,
		Input: missionjudge.Input{
			MissionType:      snap.Type,
			MissionSummary:   snap.Summary,
			Difficulty:       snap.Difficulty,
			Objectives:       judgeObjectives(snap.Objectives),
			Truth:            rawToMap(bible.Truth),
			FailureRules:     rawToStrings(bible.FailureRules),
			PlayerOutcome:    dec.Outcome,
			PlayerReasoning:  dec.Reasoning,
			DiscoveredFacts:  facts,
			DiscoveredClues:  discovered,
			TotalClues:       totalClues,
			VisitedLocations: visited,
			TotalLocations:   totalLocations,
			TimeUsedMinutes:  snap.ElapsedMinutes,
			Language:         runtime.Language(dec.Language),
		},
	})
	if err != nil {
		s.wallet.Release(ctx, res, missionjudge.Name)
		return nil, nil, err
	}
	charged := s.wallet.Settle(ctx, res, missionjudge.Name, meta)
	judged := outAny.(*missionjudge.Output)

	result := s.buildResult(ctx, missionID, snap, judged, discovered)
	result.Cost = wallet.Cost{CoinsCharged: charged}

	// Persist the terminal state and reward the player.
	resultJSON, _ := json.Marshal(result)
	if err := s.gateway.Finish(ctx, userID, missionID, resultJSON, judged.Success, completedKeys(judged)); err != nil {
		return nil, nil, err
	}
	if err := s.profiles.ApplyMissionResult(ctx, userID, judged.Success, result.XPReward); err != nil {
		s.log.Error("apply mission result", "error", err)
	}
	if result.CoinReward > 0 {
		if _, err := s.walletSvc.Credit(ctx, userID, &missionID, wallet.TxMissionReward, result.CoinReward,
			map[string]any{"score": result.Score, "success": result.Success}); err != nil {
			s.log.Error("credit mission reward", "error", err)
		}
	}
	s.recorder.Emit(ctx, missionID, missionCompletedEvent(judged.Success), map[string]any{
		"score": result.Score, "stars": result.Stars,
		"xp_reward": result.XPReward, "coin_reward": result.CoinReward,
	})

	return result, nil, nil
}

// StoredResult returns the persisted result of a finished mission, so the
// standalone result page and history can review a mission after the modal is
// gone. The response includes the mission status for win/loss framing.
func (s *Service) StoredResult(ctx context.Context, userID, missionID uuid.UUID) (json.RawMessage, string, error) {
	return s.gateway.StoredResult(ctx, userID, missionID)
}

func (s *Service) factTexts(ctx context.Context, missionID uuid.UUID) []string {
	events, err := s.events.ListByMission(ctx, missionID, 200)
	if err != nil {
		s.log.Error("list mission facts", "error", err)
		return nil
	}
	return missionevent.FactTexts(events)
}
