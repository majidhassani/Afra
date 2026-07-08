// Package missiontimeline curates raw mission events into the player-facing
// mission timeline: a readable story of what happened, where, and why it
// matters. Raw events (/events) stay available as the detailed feed; this
// package owns the official GET /missions/{id}/timeline contract.
//
// Privacy rule: items are built only from player-safe mission_events payloads.
// Nothing from the WorldBible, hidden state, or internal truth may appear here.
package missiontimeline

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"casemind/internal/missionevent"
)

// Normalized player-facing timeline item types.
const (
	TypeMissionStarted         = "mission_started"
	TypeLocationVisited        = "location_visited"
	TypeClueDiscovered         = "clue_discovered"
	TypeClueInspected          = "clue_inspected"
	TypeEvidenceConfirmed      = "evidence_confirmed"
	TypeHypothesisSubmitted    = "hypothesis_submitted"
	TypeCharacterTalked        = "character_talked"
	TypeAIGuidanceReceived     = "ai_guidance_received"
	TypeTimeAdvanced           = "time_advanced"
	TypeRiskChanged            = "risk_changed"
	TypeObjectiveCompleted     = "objective_completed"
	TypeObjectiveFailed        = "objective_failed"
	TypeNewLocationUnlocked    = "new_location_unlocked"
	TypeMissionReadyToComplete = "mission_ready_to_complete"
	TypeMissionCompleted       = "mission_completed"
	TypeMissionFailed          = "mission_failed"
	TypeWorldEvent             = "world_event"
)

// Importance levels for timeline items.
const (
	ImportanceHigh   = "high"
	ImportanceMedium = "medium"
	ImportanceLow    = "low"
)

// Item is one curated, player-facing timeline entry. Title carries mission
// content (already in the mission's language); the client renders the
// category label/icon from Type.
type Item struct {
	ID                 uuid.UUID  `json:"id"`
	Type               string     `json:"type"`
	Title              string     `json:"title"`
	Description        string     `json:"description,omitempty"`
	MissionTime        string     `json:"mission_time,omitempty"`
	OccurredAt         time.Time  `json:"occurred_at"`
	LocationID         *uuid.UUID `json:"location_id,omitempty"`
	RelatedClueID      *uuid.UUID `json:"related_clue_id,omitempty"`
	RelatedCharacterID *uuid.UUID `json:"related_character_id,omitempty"`
	Importance         string     `json:"importance"`
}

// View is the timeline endpoint response.
type View struct {
	MissionID uuid.UUID `json:"mission_id"`
	Items     []Item    `json:"items"`
}

// Ownership is the slice of the mission module this service needs.
type Ownership interface {
	EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error
}

type Service struct {
	events   missionevent.Repository
	missions Ownership
}

func NewService(events missionevent.Repository, missions Ownership) *Service {
	return &Service{events: events, missions: missions}
}

// Timeline returns the curated mission timeline, oldest first.
func (s *Service) Timeline(ctx context.Context, userID, missionID uuid.UUID, limit int) (*View, error) {
	if err := s.missions.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	events, err := s.events.ListByMission(ctx, missionID, limit)
	if err != nil {
		return nil, err
	}
	items := Curate(events)
	return &View{MissionID: missionID, Items: items}, nil
}

// Curate converts raw mission events (newest first, as stored) into curated
// player-facing items, oldest first. Noise events (generation progress,
// internal facts, director pacing) are dropped.
func Curate(events []missionevent.Event) []Item {
	items := make([]Item, 0, len(events))
	var lastReadyAt *time.Time
	// Walk oldest → newest.
	for i := len(events) - 1; i >= 0; i-- {
		ev := &events[i]
		item, ok := curateOne(ev)
		if !ok {
			continue
		}
		// Collapse repeated ready-to-complete notices into the first one.
		if item.Type == TypeMissionReadyToComplete {
			if lastReadyAt != nil {
				continue
			}
			lastReadyAt = &ev.CreatedAt
		}
		items = append(items, item)
	}
	return items
}

type payload struct {
	Title         string     `json:"title"`
	Name          string     `json:"name"`
	Summary       string     `json:"summary"`
	Message       string     `json:"message"`
	Fact          string     `json:"fact"`
	NewTime       string     `json:"new_time"`
	MissionTime   string     `json:"mission_time"`
	LocationID    *uuid.UUID `json:"location_id"`
	ClueID        *uuid.UUID `json:"clue_id"`
	CharacterID   *uuid.UUID `json:"character_id"`
	CharacterName string     `json:"character_name"`
	LocationName  string     `json:"location_name"`
	Score         *int       `json:"score"`
	Stars         *int       `json:"stars"`
	XPReward      *int       `json:"xp_reward"`
	CoinReward    *int       `json:"coin_reward"`
	Reason        string     `json:"reason"`
}

func curateOne(ev *missionevent.Event) (Item, bool) {
	var p payload
	_ = json.Unmarshal(ev.Payload, &p)

	item := Item{
		ID:          ev.ID,
		OccurredAt:  ev.CreatedAt,
		MissionTime: firstNonEmpty(p.MissionTime, p.NewTime),
		LocationID:  p.LocationID,
	}

	switch ev.Type {
	case "mission_ready":
		item.Type = TypeMissionStarted
		item.Title = firstNonEmpty(p.Title, p.Name)
		item.Importance = ImportanceHigh
	case "location_visited":
		item.Type = TypeLocationVisited
		item.Title = firstNonEmpty(p.Name, p.LocationName)
		item.Importance = ImportanceMedium
	case "clue_discovered":
		item.Type = TypeClueDiscovered
		item.Title = p.Title
		item.RelatedClueID = p.ClueID
		item.Importance = ImportanceHigh
	case "clue_inspected":
		item.Type = TypeClueInspected
		item.Title = p.Title
		item.RelatedClueID = p.ClueID
		item.Importance = ImportanceLow
	case "evidence_confirmed":
		item.Type = TypeEvidenceConfirmed
		item.Title = p.Title
		item.RelatedClueID = p.ClueID
		item.Importance = ImportanceHigh
	case "hypothesis_submitted":
		item.Type = TypeHypothesisSubmitted
		item.Title = firstNonEmpty(p.Summary, p.Title)
		item.Importance = ImportanceMedium
	case "dialogue":
		item.Type = TypeCharacterTalked
		item.Title = firstNonEmpty(p.CharacterName, p.Name)
		item.RelatedCharacterID = p.CharacterID
		item.Importance = ImportanceMedium
	case "ai_guidance":
		item.Type = TypeAIGuidanceReceived
		item.Title = firstNonEmpty(p.Summary, p.Message)
		item.Importance = ImportanceLow
	case "time_advanced":
		item.Type = TypeTimeAdvanced
		item.Title = p.NewTime
		item.Description = p.Summary
		item.Importance = ImportanceLow
	case "risk_changed":
		item.Type = TypeRiskChanged
		item.Title = firstNonEmpty(p.Title, p.Summary)
		item.Importance = ImportanceMedium
	case "objective_completed":
		item.Type = TypeObjectiveCompleted
		item.Title = p.Title
		item.Importance = ImportanceHigh
	case "objective_failed":
		item.Type = TypeObjectiveFailed
		item.Title = p.Title
		item.Importance = ImportanceHigh
	case "new_location_unlocked", "location_unlocked", "locations_discovered":
		item.Type = TypeNewLocationUnlocked
		item.Title = firstNonEmpty(p.Name, p.Title, p.LocationName)
		item.Importance = ImportanceMedium
	case "mission_ready_to_complete":
		item.Type = TypeMissionReadyToComplete
		item.Title = p.Title
		item.Description = p.Reason
		item.Importance = ImportanceHigh
	case "mission_completed":
		item.Type = TypeMissionCompleted
		item.Title = p.Title
		item.Description = p.Summary
		item.Importance = ImportanceHigh
	case "mission_failed":
		item.Type = TypeMissionFailed
		item.Title = p.Title
		item.Description = p.Summary
		item.Importance = ImportanceHigh
	case "initial_clues_available":
		// Curated as a world event so the player sees the mission opening up.
		item.Type = TypeWorldEvent
		item.Title = firstNonEmpty(p.Title, p.Summary)
		item.Importance = ImportanceLow
	case "fact_discovered", "player_action", "generation_progress",
		"mission_generation_started", "mission_generation_failed", "director_hint":
		// Internal/noise events: never part of the curated story.
		return Item{}, false
	default:
		// Dynamic world events emitted by the time/director agents.
		if p.Title == "" && p.Summary == "" {
			return Item{}, false
		}
		item.Type = TypeWorldEvent
		item.Title = firstNonEmpty(p.Title, p.Summary)
		item.Importance = ImportanceMedium
	}
	if item.Title == "" && item.Description == "" {
		return Item{}, false
	}
	return item, true
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
