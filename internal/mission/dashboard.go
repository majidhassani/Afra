package mission

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"casemind/internal/character"
	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/missionevent"
)

// MissionDashboard is the structured, player-safe mission state that lets the
// Swift app answer, on any screen: what am I trying to do, where should I go
// next, why does it matter, am I getting closer, what happens if I wait, and
// what do I need before finishing.
type MissionDashboard struct {
	MissionID     uuid.UUID `json:"mission_id"`
	Title         string    `json:"title"`
	MissionStatus string    `json:"mission_status"`
	Mission       *Mission  `json:"mission"`

	PrimaryObjective    *Objective  `json:"primary_objective"`
	Objectives          []Objective `json:"objectives"`
	CompletedObjectives []Objective `json:"completed_objectives"`

	MissionProgress int             `json:"mission_progress"`
	RiskScore       int             `json:"risk_score"`
	ProgressWeights ProgressWeights `json:"progress_weights"`

	CurrentTime   string `json:"current_time"`
	TimeRemaining string `json:"time_remaining,omitempty"`
	HasDeadline   bool   `json:"has_deadline"`

	NextRecommendedActions []DashboardAction `json:"next_recommended_actions"`
	Guidance               DashboardGuidance `json:"guidance"`

	WinConditions     []string `json:"win_conditions"`
	FailureConditions []string `json:"failure_conditions"`

	CanComplete         bool     `json:"can_complete"`
	MissingRequirements []string `json:"missing_requirements"`

	Characters      []character.PublicCharacter `json:"characters"`
	Clues           []clue.PublicClue           `json:"clues"`
	Locations       []gamemap.Marker            `json:"locations"`
	TimelinePreview []missionevent.Event        `json:"timeline_preview"`
	WalletBalance   int                         `json:"wallet_balance"`
	Result          json.RawMessage             `json:"result,omitempty"`
}

// DashboardAction is the player-safe "what to do next" shape the game client
// consumes. It flattens the internal guidance action (whose target is either a
// location or a character) into a single target_type/target_id pair so the UI
// can deep-link without knowing the internal field layout.
type DashboardAction struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	TargetType  string `json:"target_type,omitempty"` // location | character
	TargetID    string `json:"target_id,omitempty"`
	Priority    string `json:"priority,omitempty"`
}

// DashboardGuidance is the summary form of the guidance the client shows in the
// mission HUD.
type DashboardGuidance struct {
	Summary            string            `json:"summary"`
	Warning            string            `json:"warning,omitempty"`
	RecommendedActions []DashboardAction `json:"recommended_actions"`
}

// toDashboardActions maps internal guidance actions to the client contract.
func toDashboardActions(actions []RecommendedAction) []DashboardAction {
	out := make([]DashboardAction, 0, len(actions))
	for _, a := range actions {
		da := DashboardAction{
			Type:        a.Action,
			Title:       a.Title,
			Description: a.Reason,
			Priority:    a.Priority,
		}
		switch {
		case a.LocationID != "":
			da.TargetType = "location"
			da.TargetID = a.LocationID
		case a.CharacterID != "":
			da.TargetType = "character"
			da.TargetID = a.CharacterID
		}
		out = append(out, da)
	}
	return out
}

// Dashboard assembles the structured mission dashboard for the owner.
func (s *Service) Dashboard(ctx context.Context, userID, missionID uuid.UUID) (*MissionDashboard, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}

	discovered, totalClues, err := s.clues.Counts(ctx, missionID)
	if err != nil {
		return nil, err
	}
	visited, totalLocations, err := s.locations.CountVisited(ctx, missionID)
	if err != nil {
		return nil, err
	}
	interactedChars, err := s.interactions.CountInteractedCharacters(ctx, missionID)
	if err != nil {
		return nil, err
	}
	characters, err := s.characters.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	clues, err := s.clues.ListDiscovered(ctx, missionID)
	if err != nil {
		return nil, err
	}
	locations, err := s.locations.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	timelinePreview, err := s.events.ListByMission(ctx, missionID, 5)
	if err != nil {
		return nil, err
	}
	walletBalance, err := s.wallet.Balance(ctx, userID)
	if err != nil {
		return nil, err
	}
	facts := s.factTexts(ctx, missionID)

	objectives := m.ParsedObjectives()
	public := parsePublicState(m.PublicState)

	clueTarget := MandatoryClueTarget(objectives)
	finalReady := discovered >= clueTarget && clueTarget > 0

	// Deterministic progress.
	weights := DefaultProgressWeights
	progress := MissionProgress(ProgressInputs{
		DiscoveredClues:  discovered,
		TotalClues:       totalClues,
		InteractedChars:  interactedChars,
		TotalChars:       nonGuideCount(characters),
		VisitedLocations: visited,
		TotalLocations:   totalLocations,
		CorrectDecisions: len(facts),
		DecisionTarget:   decisionTarget(totalClues),
		FinalReady:       finalReady,
	}, weights)

	// Deterministic risk.
	elapsed := MinutesSinceStart(m.MissionTime)
	deadline := public.DeadlineMinutes
	risk := RiskScore(RiskInputs{
		AmbientRisk:       public.RiskLevel,
		ElapsedMinutes:    elapsed,
		DeadlineMinutes:   deadline,
		HighRiskLocations: highRiskVisited(locations),
	})
	timeLabel, remaining, hasDeadline := TimeRemaining(elapsed, deadline)

	// Visible map + guidance inputs (hidden locations never leave the server).
	charactersAt := charactersByLocation(characters)
	markers, visibleLocs := s.buildMarkers(ctx, missionID, locations, charactersAt)
	visibleChars := visibleCharacters(characters, interactedChars)

	guidance := BuildGuidance(GuidanceInputs{
		Objectives:          objectives,
		Locations:           visibleLocs,
		Characters:          visibleChars,
		DiscoveredClues:     discovered,
		Progress:            progress,
		Risk:                risk,
		TimeRemaining:       timeLabel,
		HasDeadline:         hasDeadline,
		DeadlineApproaching: deadlineApproaching(remaining, deadline),
	})

	// Objective views with derived per-objective progress + split completed.
	active := []Objective{}
	completed := []Objective{}
	var primary *Objective
	for i := range objectives {
		v := objectiveView(objectives[i], discovered, progress, finalReady)
		if v.NormalizedType() == ObjectivePrimary {
			p := v
			primary = &p
		}
		if v.Status == ObjStatusCompleted {
			completed = append(completed, v)
		} else {
			active = append(active, v)
		}
	}

	canComplete, missing := EvaluateReadiness(ReadinessInputs{
		DiscoveredClues:    discovered,
		InteractedChars:    interactedChars,
		VisitedLocations:   visited,
		RequiredClueTarget: clueTarget,
	})

	return &MissionDashboard{
		MissionID:              m.ID,
		Title:                  m.Title,
		MissionStatus:          m.Status,
		Mission:                m,
		PrimaryObjective:       primary,
		Objectives:             active,
		CompletedObjectives:    completed,
		MissionProgress:        progress,
		RiskScore:              risk,
		ProgressWeights:        weights,
		CurrentTime:            m.CurrentTime,
		TimeRemaining:          timeLabel,
		HasDeadline:            hasDeadline,
		NextRecommendedActions: toDashboardActions(guidance.RecommendedActions),
		Guidance: DashboardGuidance{
			Summary:            guidance.Summary,
			Warning:            firstOrEmpty(guidance.Warnings),
			RecommendedActions: toDashboardActions(guidance.RecommendedActions),
		},
		WinConditions:          public.WinConditions,
		FailureConditions:      public.FailureConditions,
		CanComplete:            canComplete && m.Playable(),
		MissingRequirements:    missing,
		Characters:             character.PublicList(characters),
		Clues:                  clue.PublicList(clues),
		Locations:              markers,
		TimelinePreview:        timelinePreview,
		WalletBalance:          walletBalance,
		Result:                 m.Result,
	}, nil
}

// buildMarkers builds annotated map markers and the parallel guidance-visible
// location list in a single pass over the mission's locations.
func (s *Service) buildMarkers(ctx context.Context, missionID uuid.UUID, locations []gamemap.Location, charactersAt map[uuid.UUID]bool) ([]gamemap.Marker, []VisibleLocation) {
	markers := []gamemap.Marker{}
	visible := []VisibleLocation{}
	recommendedChosen := false
	for i := range locations {
		l := &locations[i]
		if !l.Visible() {
			continue
		}
		hasMore := false
		if undiscovered, err := s.clues.ListUndiscoveredAtLocation(ctx, missionID, l.ID); err == nil {
			hasMore = len(undiscovered) > 0
		}
		hasNewClue := false
		if discovered, err := s.clues.ListDiscoveredAtLocation(ctx, missionID, l.ID); err == nil {
			hasNewClue = len(discovered) > 0
		}
		m := gamemap.Marker{
			ID: l.ID, Name: l.Name, Type: l.Type, Lat: l.Latitude, Lng: l.Longitude,
			Status: l.Status, RiskLevel: l.RiskLevel,
			HasNewClue: hasNewClue, HasCharacter: charactersAt[l.ID],
			IsLocked: l.Status == gamemap.StatusLocked,
		}
		recommend := !recommendedChosen && l.Status == gamemap.StatusDiscovered
		m.Annotate(hasMore, recommend)
		if m.Recommended {
			recommendedChosen = true
		}
		markers = append(markers, m)
		visible = append(visible, VisibleLocation{
			ID: l.ID.String(), Name: l.Name, Status: l.Status,
			RiskLevel: l.RiskLevel, HasUndiscoveredClues: hasMore,
		})
	}
	return markers, visible
}

// publicStateView is the subset of a mission's public_state the dashboard reads.
type publicStateView struct {
	WinConditions     []string
	FailureConditions []string
	DeadlineMinutes   int
	RiskLevel         int
}

func parsePublicState(raw json.RawMessage) publicStateView {
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return publicStateView{
		WinConditions:     stringSlice(m[PublicKeyWinConditions]),
		FailureConditions: stringSlice(m[PublicKeyFailureConditions]),
		DeadlineMinutes:   intOf(m[PublicKeyDeadlineMinutes]),
		RiskLevel:         intOf(m[PublicKeyRiskLevel]),
	}
}

func stringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func intOf(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

// objectiveView derives a player-facing objective, computing per-objective
// progress from clue coverage without mutating stored objective state. A
// locked objective stays locked; an objective whose derived progress reaches
// 100 is shown as completed.
func objectiveView(o Objective, discoveredClues, missionProgress int, finalReady bool) Objective {
	v := o
	if v.Type == "" {
		v.Type = o.NormalizedType()
	}
	if v.Status == ObjStatusLocked || v.Status == ObjStatusFailed || v.Status == ObjStatusSkipped {
		return v
	}
	switch {
	case o.NormalizedType() == ObjectivePrimary:
		v.Progress = missionProgress
	case o.NormalizedType() == ObjectiveFinal:
		if finalReady {
			v.Progress = 100
		} else {
			v.Progress = int(ratio(discoveredClues, o.RequiredClues)*100 + 0.5)
		}
	case o.RequiredClues > 0:
		v.Progress = int(ratio(discoveredClues, o.RequiredClues)*100 + 0.5)
	default:
		v.Progress = missionProgress
	}
	v.Progress = clampScore(v.Progress)
	if v.Progress >= 100 && v.Status == ObjStatusActive {
		v.Status = ObjStatusCompleted
	}
	return v
}

func nonGuideCount(chars []character.Character) int {
	n := 0
	for i := range chars {
		if chars[i].Category != character.CategoryGuide {
			n++
		}
	}
	return n
}

func highRiskVisited(locations []gamemap.Location) int {
	n := 0
	for i := range locations {
		if locations[i].Status == gamemap.StatusVisited && locations[i].RiskLevel >= gamemap.HighRiskThreshold {
			n++
		}
	}
	return n
}

func charactersByLocation(chars []character.Character) map[uuid.UUID]bool {
	at := map[uuid.UUID]bool{}
	for i := range chars {
		if chars[i].CurrentLocationID != nil {
			at[*chars[i].CurrentLocationID] = true
		}
	}
	return at
}

// visibleCharacters projects characters into the guidance-visible shape. Since
// per-character interaction detail is not tracked, the first interactedCount
// non-guide characters are marked as already spoken with — enough for
// deterministic "who to talk to next" guidance without leaking anything.
func visibleCharacters(chars []character.Character, interactedCount int) []VisibleCharacter {
	out := make([]VisibleCharacter, 0, len(chars))
	marked := 0
	for i := range chars {
		if chars[i].Category == character.CategoryGuide {
			continue
		}
		interacted := marked < interactedCount
		if interacted {
			marked++
		}
		out = append(out, VisibleCharacter{
			ID: chars[i].ID.String(), Name: chars[i].Name,
			Role: chars[i].Role, Interacted: interacted,
		})
	}
	return out
}

// decisionTarget is how many confirmed facts earn full credit for the
// "correct decisions" progress slice — scaled to the mission's clue count.
func decisionTarget(totalClues int) int {
	if totalClues <= 0 {
		return 3
	}
	return totalClues
}

func firstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func deadlineApproaching(remaining, deadline int) bool {
	if deadline <= 0 {
		return false
	}
	return remaining <= deadline/4
}

func (s *Service) factTexts(ctx context.Context, missionID uuid.UUID) []string {
	events, err := s.events.ListByMission(ctx, missionID, 200)
	if err != nil {
		s.log.Error("list mission facts", "error", err)
		return nil
	}
	return missionevent.FactTexts(events)
}
