// Package timeengine owns the mission clock: reading mission time and the
// paid advance-time flow (TimeAgent consequences + DirectorAgent pacing).
package timeengine

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"casemind/internal/agent/missiondirector"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/agent/timeagent"
	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/interaction"
	"casemind/internal/missionevent"
	"casemind/internal/wallet"
	"casemind/internal/worldbible"
	apperrors "casemind/pkg/errors"
)

// MissionGateway is the slice of the mission module this service needs; it
// is implemented by mission.Service.
type MissionGateway interface {
	EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error
	EnsureOwnedActive(ctx context.Context, userID, missionID uuid.UUID) error
	SummaryOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	ObjectiveTitles(ctx context.Context, userID, missionID uuid.UUID) ([]string, error)
	ClockOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	// PreviewClock formats the clock after advancing by minutes, without
	// mutating anything. Also returns the minutes elapsed since start.
	PreviewClock(ctx context.Context, userID, missionID uuid.UUID, minutes int) (newClock string, elapsed int, err error)
	// CommitClock advances the stored mission time.
	CommitClock(ctx context.Context, missionID uuid.UUID, minutes int) (newClock string, err error)
	PublicStateOf(ctx context.Context, userID, missionID uuid.UUID) (map[string]any, error)
	MergePublicState(ctx context.Context, missionID uuid.UUID, changes map[string]any) error
}

type Service struct {
	guard     MissionGateway
	wallet    *wallet.Guard
	orch      *orchestrator.Orchestrator
	bibles    worldbible.Repository
	clues     clue.Repository
	locations gamemap.Repository
	convs     interaction.Repository
	events    missionevent.Repository
	recorder  *missionevent.Recorder
	log       *slog.Logger
}

func NewService(
	guard MissionGateway,
	walletGuard *wallet.Guard,
	orch *orchestrator.Orchestrator,
	bibles worldbible.Repository,
	clues clue.Repository,
	locations gamemap.Repository,
	convs interaction.Repository,
	events missionevent.Repository,
	recorder *missionevent.Recorder,
	log *slog.Logger,
) *Service {
	return &Service{
		guard: guard, wallet: walletGuard, orch: orch, bibles: bibles, clues: clues,
		locations: locations, convs: convs, events: events, recorder: recorder, log: log,
	}
}

// TimeInfo is the current mission clock state.
type TimeInfo struct {
	CurrentTime string         `json:"current_time"`
	PublicState map[string]any `json:"public_state"`
}

func (s *Service) Get(ctx context.Context, userID, missionID uuid.UUID) (*TimeInfo, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	clock, err := s.guard.ClockOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	state, err := s.guard.PublicStateOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	return &TimeInfo{CurrentTime: clock, PublicState: state}, nil
}

// AdvanceResult is the player-safe outcome of advancing mission time.
type AdvanceResult struct {
	NewTime string           `json:"new_time"`
	Summary string           `json:"summary"`
	Events  []timeagent.Event `json:"events"`
	Cost    wallet.Cost      `json:"cost"`
}

// Advance runs the paid time-advance flow: WalletGuard -> TimeAgent ->
// domain application (clock, public state, hidden state, events) ->
// async DirectorAgent.
func (s *Service) Advance(ctx context.Context, userID, missionID uuid.UUID, amount int, unit string) (*AdvanceResult, error) {
	minutes, err := toMinutes(amount, unit)
	if err != nil {
		return nil, err
	}
	if err := s.guard.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}
	summary, err := s.guard.SummaryOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	currentClock, err := s.guard.ClockOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	newClock, elapsed, err := s.guard.PreviewClock(ctx, userID, missionID, minutes)
	if err != nil {
		return nil, err
	}
	publicState, err := s.guard.PublicStateOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}

	// Private context: due scheduled events + hidden state.
	bible, err := s.bibles.GetByMissionID(ctx, missionID)
	if err != nil {
		return nil, err
	}
	hiddenState := rawToMap(bible.HiddenState)
	due := dueEvents(bible.TimelineTruth, elapsed, elapsed+minutes)

	discoveredLocations := []string{}
	if locations, err := s.locations.ListByMission(ctx, missionID); err == nil {
		for i := range locations {
			if locations[i].Visible() {
				discoveredLocations = append(discoveredLocations, locations[i].Name)
			}
		}
	}

	res, err := s.wallet.Reserve(ctx, userID, &missionID, wallet.ActionAdvanceTime)
	if err != nil {
		return nil, err
	}
	outAny, meta, err := s.orch.RunWithMeta(ctx, timeagent.Name, runtime.Task{
		Type: timeagent.TaskType, MissionID: missionID, UserID: userID,
		Input: timeagent.Input{
			MissionSummary:      summary,
			CurrentClock:        currentClock,
			NewClock:            newClock,
			AdvanceMinutes:      minutes,
			PublicState:         publicState,
			DueEvents:           due,
			HiddenState:         hiddenState,
			DiscoveredLocations: discoveredLocations,
			Language:            "en",
		},
	})
	if err != nil {
		s.wallet.Release(ctx, res, timeagent.Name)
		return nil, err
	}
	charged := s.wallet.Settle(ctx, res, timeagent.Name, meta)
	out := outAny.(*timeagent.Output)

	// Apply the validated intention.
	committedClock, err := s.guard.CommitClock(ctx, missionID, minutes)
	if err != nil {
		return nil, err
	}
	if len(out.PublicStateChanges) > 0 {
		if err := s.guard.MergePublicState(ctx, missionID, sanitizePublicState(out.PublicStateChanges)); err != nil {
			s.log.Error("merge public state", "error", err)
		}
	}
	if len(out.HiddenStateChanges) > 0 {
		for k, v := range out.HiddenStateChanges {
			hiddenState[k] = v
		}
		if raw, err := json.Marshal(hiddenState); err == nil {
			if err := s.bibles.UpdateHiddenState(ctx, missionID, raw); err != nil {
				s.log.Error("update hidden state", "error", err)
			}
		}
	}
	s.recorder.Emit(ctx, missionID, "time_advanced", map[string]any{
		"new_time": committedClock, "minutes": minutes, "summary": out.Summary,
	})
	for _, ev := range out.Events {
		s.recorder.Emit(ctx, missionID, ev.Type, map[string]any{"title": ev.Title})
	}

	// DirectorAgent pacing pass (best effort, async).
	s.directorPass(missionID, userID, summary, committedClock)

	return &AdvanceResult{
		NewTime: committedClock,
		Summary: out.Summary,
		Events:  out.Events,
		Cost:    wallet.Cost{CoinsCharged: charged},
	}, nil
}

// directorPass runs the DirectorAgent asynchronously; its output arrives as
// mission events.
func (s *Service) directorPass(missionID, userID uuid.UUID, summary, clock string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		discovered, total, err := s.clues.Counts(ctx, missionID)
		if err != nil {
			return
		}
		visited, totalLocs, err := s.locations.CountVisited(ctx, missionID)
		if err != nil {
			return
		}
		interactions, err := s.convs.CountByMission(ctx, missionID)
		if err != nil {
			return
		}
		facts := []string{}
		if events, err := s.events.ListByMission(ctx, missionID, 100); err == nil {
			facts = missionevent.FactTexts(events)
		}

		outAny, err := s.orch.Run(ctx, missiondirector.Name, runtime.Task{
			Type: missiondirector.TaskType, MissionID: missionID, UserID: userID,
			Input: missiondirector.Input{
				MissionSummary:   summary,
				MissionTime:      clock,
				DiscoveredFacts:  facts,
				DiscoveredClues:  discovered,
				TotalClues:       total,
				VisitedLocations: visited,
				TotalLocations:   totalLocs,
				AIInteractions:   interactions,
				Language:         "en",
			},
		})
		if err != nil {
			return
		}
		out := outAny.(*missiondirector.Output)
		for _, ev := range out.Events {
			s.recorder.Emit(ctx, missionID, ev.Type, map[string]any{"title": ev.Title})
		}
		if out.Hint != "" {
			s.recorder.Emit(ctx, missionID, "director_hint", map[string]any{
				"hint": out.Hint, "tone": out.Tone,
			})
		}
	}()
}

func toMinutes(amount int, unit string) (int, error) {
	var minutes int
	switch unit {
	case "minutes", "minute", "":
		minutes = amount
	case "hours", "hour":
		minutes = amount * 60
	case "days", "day":
		minutes = amount * 24 * 60
	default:
		return 0, apperrors.Invalid("invalid_unit", "unit must be minutes, hours, or days")
	}
	if minutes < 5 || minutes > 24*60 {
		return 0, apperrors.Invalid("invalid_amount", "time advance must be between 5 minutes and 24 hours")
	}
	return minutes, nil
}

// dueEvents extracts scheduled events with at_minutes in (from, to].
func dueEvents(schedule json.RawMessage, from, to int) []timeagent.ScheduledEvent {
	var all []timeagent.ScheduledEvent
	if err := json.Unmarshal(schedule, &all); err != nil {
		return nil
	}
	due := []timeagent.ScheduledEvent{}
	for _, ev := range all {
		if ev.AtMinutes > from && ev.AtMinutes <= to {
			due = append(due, ev)
		}
	}
	return due
}

// sanitizePublicState only allows player-safe keys through.
func sanitizePublicState(changes map[string]any) map[string]any {
	allowed := map[string]bool{"weather": true, "risk_level": true}
	out := map[string]any{}
	for k, v := range changes {
		if allowed[k] {
			out[k] = v
		}
	}
	return out
}

func rawToMap(raw json.RawMessage) map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return m
}
