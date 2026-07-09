package mission

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"casemind/internal/missionevent"
)

// Action types with a fixed mission-time cost. Time is a real resource: every
// meaningful action moves the clock, and the passage of time can trigger
// scheduled world events. Costs are deterministic so they can be previewed.
const (
	ActionTravel       = "travel"
	ActionLocationScan = "location_action"
	ActionCharacterTalk = "character_chat"
	ActionClueInspect  = "clue_inspect"
	ActionReportSubmit = "report_submit"
)

// ActionTimeCosts maps action type → mission minutes consumed.
var ActionTimeCosts = map[string]int{
	ActionTravel:        15,
	ActionLocationScan:  20,
	ActionCharacterTalk: 10,
	ActionClueInspect:   10,
	ActionReportSubmit:  15,
}

// TimeCostOf returns the fixed cost for an action type (0 = free/unknown).
func TimeCostOf(action string) int { return ActionTimeCosts[action] }

// scheduledEvent mirrors the WorldBible timeline_truth entry shape.
type scheduledEvent struct {
	AtMinutes   int    `json:"at_minutes"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ApplyActionTime commits the fixed time cost of an action to the mission
// clock and runs the deterministic due-event sweep: scheduled world events
// whose time falls inside the advanced window fire as timeline events and are
// returned so the client can show them. Free actions return a zero update.
//
// This is the deterministic sibling of the paid TimeAgent advance: no AI, no
// wallet charge — just the world clock moving because the player acted.
func (s *Service) ApplyActionTime(ctx context.Context, missionID uuid.UUID, action string) (*missionevent.TimeUpdate, error) {
	minutes := TimeCostOf(action)
	m, err := s.repo.GetByID(ctx, missionID)
	if err != nil {
		return nil, err
	}
	if minutes <= 0 {
		return &missionevent.TimeUpdate{
			NewTime:         m.CurrentTime,
			TriggeredEvents: []missionevent.WorldEvent{},
		}, nil
	}
	elapsedBefore := MinutesSinceStart(m.MissionTime)

	newTime, err := s.repo.AdvanceMissionTime(ctx, missionID, minutes)
	if err != nil {
		return nil, err
	}
	newClock := FormatClock(newTime)
	elapsedAfter := MinutesSinceStart(newTime)

	update := &missionevent.TimeUpdate{
		NewTime:         newClock,
		MinutesAdvanced: minutes,
		TriggeredEvents: s.sweepDueEvents(ctx, missionID, elapsedBefore, elapsedAfter),
	}
	s.recorder.Emit(ctx, missionID, "time_advanced", map[string]any{
		"new_time": newClock, "minutes": minutes, "action": action,
	})
	return update, nil
}

// sweepDueEvents fires scheduled world events in (from, to]. Each event fires
// at most once because the clock is monotonic and windows never overlap.
// Only the player-safe fields of a scheduled event are surfaced.
func (s *Service) sweepDueEvents(ctx context.Context, missionID uuid.UUID, from, to int) []missionevent.WorldEvent {
	triggered := []missionevent.WorldEvent{}
	bible, err := s.bibles.GetByMissionID(ctx, missionID)
	if err != nil {
		return triggered
	}
	var all []scheduledEvent
	if err := json.Unmarshal(bible.TimelineTruth, &all); err != nil {
		return triggered
	}
	for _, ev := range all {
		if ev.AtMinutes <= from || ev.AtMinutes > to {
			continue
		}
		we := missionevent.WorldEvent{
			Type:        "world_event",
			Title:       ev.Title,
			Description: ev.Description,
		}
		triggered = append(triggered, we)
		s.recorder.Emit(ctx, missionID, "world_event", map[string]any{
			"title": ev.Title, "description": ev.Description, "at_minutes": ev.AtMinutes,
		})
	}
	return triggered
}

// ActionPreview is the player-facing "what will this cost" answer, shown
// before the action runs.
type ActionPreview struct {
	Action          string `json:"action"`
	TimeCostMinutes int    `json:"time_cost_minutes"`
	CoinCost        int    `json:"coin_cost"`
	RiskNote        string `json:"risk_note"`
	NewTimeIfDone   string `json:"new_time_if_done"`
}

// PreviewAction computes the deterministic cost of an action without
// changing anything. Risk notes come from the target location's risk level
// and the mission deadline.
func (s *Service) PreviewAction(ctx context.Context, userID, missionID uuid.UUID, action string, targetID *uuid.UUID) (*ActionPreview, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	minutes := TimeCostOf(action)
	newClock, _, err := s.PreviewClock(ctx, userID, missionID, minutes)
	if err != nil {
		return nil, err
	}

	risk := ""
	if targetID != nil {
		if loc, err := s.locations.GetByID(ctx, missionID, *targetID); err == nil && loc.RiskLevel >= 60 {
			risk = "High-risk area — expect complications."
		}
	}
	if risk == "" {
		public := parsePublicState(m.PublicState)
		if public.DeadlineMinutes > 0 {
			elapsed := MinutesSinceStart(m.MissionTime)
			remaining := public.DeadlineMinutes - elapsed
			if remaining > 0 && remaining-minutes <= public.DeadlineMinutes/4 {
				risk = "Time is running short — this brings you close to the deadline."
			}
		}
	}
	return &ActionPreview{
		Action:          action,
		TimeCostMinutes: minutes,
		CoinCost:        s.coinCostOf(ctx, action),
		RiskNote:        risk,
		NewTimeIfDone:   newClock,
	}, nil
}

// coinCostOf maps a previewable action to its wallet price (0 = free).
func (s *Service) coinCostOf(ctx context.Context, action string) int {
	walletAction := map[string]string{
		ActionLocationScan:  "location_search",
		ActionCharacterTalk: "character_chat",
		ActionClueInspect:   "clue_inspect",
	}[action]
	if walletAction == "" {
		return 0
	}
	price, err := s.walletSvc.PriceOf(ctx, walletAction)
	if err != nil {
		return 0
	}
	return price
}
