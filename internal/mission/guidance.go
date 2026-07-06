package mission

import "fmt"

// This file builds the deterministic "What should I do next?" guidance shown on
// the dashboard. It is computed from strictly player-visible state — objectives,
// visible locations, known characters, discovered clue counts — and never reads
// the World Bible, so it structurally cannot reveal hidden truth. Its
// spoiler_level is therefore always "low".

// Recommended-action verbs.
const (
	ActionVisitLocation   = "visit_location"
	ActionTalkToCharacter = "talk_to_character"
	ActionSearchLocation  = "search_location"
	ActionReviewClues     = "review_clues"
	ActionPrepareFinal    = "prepare_final"
)

// Priorities.
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// VisibleLocation is a player-visible map location the guidance builder may
// reference. Hidden locations must never be passed in.
type VisibleLocation struct {
	ID                   string
	Name                 string
	Status               string // discovered | visited | locked
	RiskLevel            int
	HasUndiscoveredClues bool
}

// VisibleCharacter is a known character the guidance builder may reference.
type VisibleCharacter struct {
	ID         string
	Name       string
	Role       string
	Interacted bool
}

// GuidanceInputs is the player-visible snapshot the guidance builder consumes.
type GuidanceInputs struct {
	Objectives      []Objective
	Locations       []VisibleLocation
	Characters      []VisibleCharacter
	DiscoveredClues int
	Progress        int
	Risk            int
	TimeRemaining   string
	HasDeadline     bool
	// DeadlineApproaching is true when less than a quarter of the mission's
	// time budget remains.
	DeadlineApproaching bool
}

// RecommendedAction is one concrete, player-safe next step.
type RecommendedAction struct {
	Action      string `json:"action"`
	Title       string `json:"title"`
	Reason      string `json:"reason"`
	LocationID  string `json:"location_id,omitempty"`
	CharacterID string `json:"character_id,omitempty"`
	Priority    string `json:"priority"`
}

// Guidance is the structured answer to "What should I do next?".
type Guidance struct {
	Summary            string              `json:"summary"`
	CurrentGoal        string              `json:"current_goal"`
	RecommendedActions []RecommendedAction `json:"recommended_actions"`
	Warnings           []string            `json:"warnings"`
	SpoilerLevel       string              `json:"spoiler_level"`
}

// maxRecommendedActions caps how many next steps guidance offers at once.
const maxRecommendedActions = 3

// BuildGuidance produces deterministic next-step guidance from visible state.
func BuildGuidance(in GuidanceInputs) Guidance {
	g := Guidance{
		Summary:      guidanceSummary(in),
		CurrentGoal:  currentGoal(in.Objectives),
		Warnings:     guidanceWarnings(in),
		SpoilerLevel: "low",
	}

	actions := []RecommendedAction{}

	// 1. Point at an unexplored, accessible location first.
	if loc, ok := firstUnvisited(in.Locations); ok {
		actions = append(actions, RecommendedAction{
			Action:     ActionVisitLocation,
			Title:      "Go to " + loc.Name,
			Reason:     "You have not investigated this location yet — it may hold clues you still need.",
			LocationID: loc.ID,
			Priority:   PriorityHigh,
		})
	}

	// 2. Nudge toward a character the player has never spoken with.
	if ch, ok := firstUninteracted(in.Characters); ok {
		actions = append(actions, RecommendedAction{
			Action:      ActionTalkToCharacter,
			Title:       "Talk to " + ch.Name,
			Reason:      characterReason(ch),
			CharacterID: ch.ID,
			Priority:    PriorityMedium,
		})
	}

	// 3. Suggest searching a visited spot that still has more to find.
	if loc, ok := visitedWithMore(in.Locations); ok {
		actions = append(actions, RecommendedAction{
			Action:     ActionSearchLocation,
			Title:      "Search " + loc.Name + " again",
			Reason:     "There may be more to uncover here beyond what you have already found.",
			LocationID: loc.ID,
			Priority:   PriorityMedium,
		})
	}

	// Fallbacks when the map and cast are exhausted.
	if len(actions) == 0 {
		if in.DiscoveredClues > 0 {
			actions = append(actions, RecommendedAction{
				Action:   ActionPrepareFinal,
				Title:    "Prepare your final decision",
				Reason:   "You have explored the field and gathered evidence — review it and commit to a conclusion.",
				Priority: PriorityHigh,
			})
		} else {
			actions = append(actions, RecommendedAction{
				Action:   ActionReviewClues,
				Title:    "Review the briefing",
				Reason:   "Start from the mission briefing and objectives to decide where to head first.",
				Priority: PriorityMedium,
			})
		}
	}

	if len(actions) > maxRecommendedActions {
		actions = actions[:maxRecommendedActions]
	}
	g.RecommendedActions = actions
	return g
}

func guidanceSummary(in GuidanceInputs) string {
	switch {
	case in.DiscoveredClues == 0:
		return "You are just getting started — no clues gathered yet. Explore the map to begin building the picture."
	case in.DiscoveredClues == 1:
		return fmt.Sprintf("You have found 1 clue so far and are %d%% of the way through the mission.", in.Progress)
	default:
		return fmt.Sprintf("You have gathered %d clues and are %d%% of the way through the mission.", in.DiscoveredClues, in.Progress)
	}
}

func currentGoal(objectives []Objective) string {
	// Prefer the active primary objective, then any active mandatory one, then
	// any active objective at all.
	for _, o := range objectives {
		if o.NormalizedType() == ObjectivePrimary && o.Status == ObjStatusActive {
			return o.Title
		}
	}
	for _, o := range objectives {
		if o.Mandatory() && o.Status == ObjStatusActive {
			return o.Title
		}
	}
	for _, o := range objectives {
		if o.Status == ObjStatusActive {
			return o.Title
		}
	}
	if len(objectives) > 0 {
		return objectives[0].Title
	}
	return "Complete the mission objectives."
}

func guidanceWarnings(in GuidanceInputs) []string {
	warnings := []string{}
	if in.HasDeadline && in.DeadlineApproaching {
		warnings = append(warnings, "Time is running short — "+in.TimeRemaining+" left before the deadline.")
	}
	if in.Risk >= 60 {
		warnings = append(warnings, "Risk is high. Waiting or advancing time may let the situation deteriorate.")
	} else {
		warnings = append(warnings, "Advancing time moves the world forward and may let events unfold against you.")
	}
	return warnings
}

func firstUnvisited(locs []VisibleLocation) (VisibleLocation, bool) {
	for _, l := range locs {
		if l.Status == "discovered" {
			return l, true
		}
	}
	return VisibleLocation{}, false
}

func visitedWithMore(locs []VisibleLocation) (VisibleLocation, bool) {
	for _, l := range locs {
		if l.Status == "visited" && l.HasUndiscoveredClues {
			return l, true
		}
	}
	return VisibleLocation{}, false
}

func firstUninteracted(chars []VisibleCharacter) (VisibleCharacter, bool) {
	for _, c := range chars {
		// The guide/mission-control character is not an investigation target.
		if c.Role == "Mission Control AI" || c.Interacted {
			continue
		}
		return c, true
	}
	return VisibleCharacter{}, false
}

func characterReason(c VisibleCharacter) string {
	if c.Role != "" {
		return "As " + c.Role + ", they may know something relevant you have not heard yet."
	}
	return "They may know something relevant you have not heard yet."
}
