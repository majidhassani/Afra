package mission

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"casemind/internal/character"
)

// Suspect statuses for the HUD.
const (
	SuspectHidden     = "hidden"     // reveal stage not reached
	SuspectIdentified = "identified" // reveal stage completed
)

// BoardImage is one generated scenario board (background art).
type BoardImage struct {
	URL     string `json:"url"`
	Status  string `json:"status"`
	Version int    `json:"version"`
}

// ParsedBoardArt decodes the board_art JSONB map (board_type → image).
func (m *Mission) ParsedBoardArt() map[string]BoardImage {
	out := map[string]BoardImage{}
	_ = json.Unmarshal(m.BoardArt, &out)
	return out
}

// GameplayStatus is the single payload the game HUD lives on: everything the
// player must always see (stage, objective, clue count, suspect status,
// progress, time, next reward, report CTA, next action) in one read.
type GameplayStatus struct {
	*MissionDashboard

	Stages       []Stage `json:"stages"`
	CurrentStage *Stage  `json:"current_stage,omitempty"`
	StageIndex   int     `json:"stage_index"`
	StageCount   int     `json:"stage_count"`

	// Mission-wide clue goal ("2 / 5 clues found") — the identify-stage
	// target, the strongest single number to put on the HUD.
	ClueGoal  int `json:"clue_goal"`
	CluesFound int `json:"clues_found"`

	SuspectStatus string                      `json:"suspect_status"`
	Suspect       *character.PublicCharacter  `json:"suspect,omitempty"`

	// NextReward is the active stage's reward — what completing the current
	// stage pays.
	NextReward *StageReward `json:"next_reward,omitempty"`

	// ReportPending glows the report CTA: the active stage requires a report
	// that has not been accepted yet.
	ReportPending     bool   `json:"report_pending"`
	PendingReportType string `json:"pending_report_type,omitempty"`

	BoardArt map[string]BoardImage `json:"board_art"`

	// StageUpdate carries transitions that happened during this (lazy)
	// evaluation so even a read can raise reward popups.
	StageUpdate *StageUpdate `json:"stage_update,omitempty"`
}

// GameplayStatus assembles the full HUD payload, running a lazy stage
// evaluation first so derived progress is always current.
func (s *Service) GameplayStatus(ctx context.Context, userID, missionID uuid.UUID) (*GameplayStatus, error) {
	update, err := s.EvaluateStages(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	dash, err := s.Dashboard(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}

	status := &GameplayStatus{
		MissionDashboard: dash,
		Stages:           update.Stages,
		CurrentStage:     update.CurrentStage,
		StageIndex:       update.StageIndex,
		StageCount:       update.StageCount,
		SuspectStatus:    SuspectHidden,
		Suspect:          update.Suspect,
		BoardArt:         dash.Mission.ParsedBoardArt(),
	}
	if update.Changed() {
		status.StageUpdate = update
	}
	if update.Suspect != nil {
		status.SuspectStatus = SuspectIdentified
	}

	// Mission-wide clue goal: the largest find_clues requirement across stages.
	discovered, _, err := s.clues.Counts(ctx, missionID)
	if err == nil {
		status.CluesFound = discovered
	}
	for i := range update.Stages {
		st := &update.Stages[i]
		for _, a := range st.RequiredActions {
			if a.Type == StageActionFindClues && a.Count > status.ClueGoal {
				status.ClueGoal = a.Count
			}
		}
	}

	if cur := update.CurrentStage; cur != nil {
		r := cur.Reward
		if r.XP > 0 || r.Coins > 0 || r.Badge != "" {
			status.NextReward = &r
		}
		for _, a := range cur.RequiredActions {
			if a.Type == StageActionSubmitReport && a.Done < a.Count {
				status.ReportPending = true
				status.PendingReportType = a.Report
				break
			}
			if a.Type == StageActionFinalDecision && a.Done < a.Count {
				status.ReportPending = true
				status.PendingReportType = "final_report"
				break
			}
		}
	}
	return status, nil
}

// --- Board art gateway (used by the visualasset module) ---

// BoardContext is the player-safe prompt context for board generation.
// It never includes WorldBible content.
func (s *Service) BoardContext(ctx context.Context, userID, missionID uuid.UUID) (missionType, region, weather, timeOfDay string, risk int, err error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return "", "", "", "", 0, err
	}
	ws := DeriveWorldState(m.PublicState, m.CurrentTime, BiomeForType(m.Type), 0, 0, false, false)
	public := parsePublicState(m.PublicState)
	return m.Type, m.Region, ws.Weather, ws.TimeOfDay, public.RiskLevel, nil
}

// SaveBoardArt persists one generated board image and bumps its version.
func (s *Service) SaveBoardArt(ctx context.Context, missionID uuid.UUID, boardType, dataURL, status string) (int, error) {
	m, err := s.repo.GetByID(ctx, missionID)
	if err != nil {
		return 0, err
	}
	art := m.ParsedBoardArt()
	img := art[boardType]
	img.URL = dataURL
	img.Status = status
	img.Version++
	art[boardType] = img
	raw, err := json.Marshal(art)
	if err != nil {
		return 0, err
	}
	if err := s.repo.UpdateBoardArt(ctx, missionID, raw); err != nil {
		return 0, err
	}
	return img.Version, nil
}

// BoardArtOf returns the stored board images for the owner.
func (s *Service) BoardArtOf(ctx context.Context, userID, missionID uuid.UUID) (map[string]BoardImage, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	return m.ParsedBoardArt(), nil
}
