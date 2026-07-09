package mission

import (
	"encoding/json"
	"fmt"
)

// Stage IDs of the default mission-level template, in play order.
const (
	StageArrival         = "arrival"
	StageFirstClues      = "first_clues"
	StageIdentifySuspect = "identify_suspect"
	StageConfirmEvidence = "confirm_evidence"
	StageFinalReport     = "final_report"
	StageDebrief         = "debrief"
)

// Stage statuses.
const (
	StageStatusLocked    = "locked"
	StageStatusActive    = "active"
	StageStatusCompleted = "completed"
)

// Stage requirement action types.
const (
	StageActionVisitLocations      = "visit_locations"
	StageActionFindClues           = "find_clues"
	StageActionConfirmEvidence     = "confirm_evidence"
	StageActionInterviewCharacters = "interview_characters"
	StageActionSubmitReport        = "submit_report" // Report field selects the type
	StageActionFinalDecision       = "final_decision"
)

// Stage is one visible level-segment of a mission. Stored as JSONB on the
// mission (like objectives). Counters (FoundClueCount, StageAction.Done,
// Progress) are derived at evaluation time; Status is the persisted truth.
type Stage struct {
	ID                string        `json:"id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Status            string        `json:"status"`
	Progress          int           `json:"progress"`
	RequiredClueCount int           `json:"required_clue_count"`
	FoundClueCount    int           `json:"found_clue_count"`
	RequiredActions   []StageAction `json:"required_actions"`
	Reward            StageReward   `json:"reward"`
	UnlockOnComplete  StageUnlock   `json:"unlock_on_complete"`
}

// StageAction is one requirement inside a stage. Count is the cumulative
// mission-wide target (monotonic — progress can never regress), Done the
// derived current value clamped to Count.
type StageAction struct {
	Type   string `json:"type"`
	Report string `json:"report,omitempty"` // for submit_report: the report type
	Count  int    `json:"count"`
	Done   int    `json:"done"`
}

// Label renders a player-readable requirement line.
func (a StageAction) Label() string {
	switch a.Type {
	case StageActionVisitLocations:
		return fmt.Sprintf("Visit %d location(s)", a.Count)
	case StageActionFindClues:
		return fmt.Sprintf("Find %d clue(s)", a.Count)
	case StageActionConfirmEvidence:
		return fmt.Sprintf("Confirm %d piece(s) of evidence", a.Count)
	case StageActionInterviewCharacters:
		return fmt.Sprintf("Interview %d character(s)", a.Count)
	case StageActionSubmitReport:
		return fmt.Sprintf("Submit a %s report", a.Report)
	case StageActionFinalDecision:
		return "Submit your final decision"
	default:
		return a.Type
	}
}

// StageReward is granted exactly once, when the stage transitions to
// completed.
type StageReward struct {
	XP    int    `json:"xp"`
	Coins int    `json:"coins"`
	Badge string `json:"badge,omitempty"`
}

// StageUnlock describes what completing the stage opens.
type StageUnlock struct {
	NextStageID    string `json:"next_stage_id,omitempty"`
	UnlockLocation bool   `json:"unlock_location,omitempty"`
	RevealSuspect  bool   `json:"reveal_suspect,omitempty"`
}

// ParsedStages decodes the stored stages JSONB. Missions generated before the
// stage system return an empty slice — callers backfill via DefaultStages.
func (m *Mission) ParsedStages() []Stage {
	var stages []Stage
	_ = json.Unmarshal(m.Stages, &stages)
	return stages
}

// suspectStageTitles maps mission types to the identify-stage language. The
// detective family talks about suspects; other types identify a cause/source.
func suspectStageTitle(missionType string) (title, description string) {
	switch missionType {
	case "detective":
		return "Identify the Suspect", "Gather enough clues and testimony to name a prime suspect, then submit a suspect report."
	case "medical_mystery":
		return "Identify the Source", "Trace the outbreak: gather enough evidence to name the likely source, then submit a suspect report."
	default:
		return "Identify the Cause", "Gather enough evidence to explain what is behind this, then submit a suspect report."
	}
}

// DefaultStageInputs sizes the default stage template to the actual mission.
type DefaultStageInputs struct {
	MissionType    string
	Difficulty     string
	TotalClues     int
	ClueTarget     int // mandatory clue target from objectives (0 = derive)
	TotalChars     int // non-guide characters
	TotalLocations int
}

// DefaultStages builds the deterministic six-stage template scaled to the
// mission's real content. All counts are cumulative mission-wide targets, so
// evaluation is monotonic. The first stage starts active.
func DefaultStages(in DefaultStageInputs) []Stage {
	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}
	totalClues := in.TotalClues
	if totalClues <= 0 {
		totalClues = 5
	}
	clueTarget := in.ClueTarget
	if clueTarget <= 0 || clueTarget > totalClues {
		clueTarget = clamp((totalClues*2+2)/3, 1, totalClues) // ~2/3 of clues
	}
	firstClues := clamp(2, 1, clueTarget)
	confirmTarget := clamp(totalClues/2, 2, clueTarget)
	interviews := clamp(2, 0, in.TotalChars)

	identifyTitle, identifyDesc := suspectStageTitle(in.MissionType)

	identifyActions := []StageAction{
		{Type: StageActionFindClues, Count: clueTarget},
	}
	if interviews > 0 {
		identifyActions = append(identifyActions,
			StageAction{Type: StageActionInterviewCharacters, Count: interviews})
	}
	identifyActions = append(identifyActions,
		StageAction{Type: StageActionSubmitReport, Report: "suspect_report", Count: 1})

	return []Stage{
		{
			ID: StageArrival, Title: "Arrival",
			Description: "Get on site: open the map and visit your first location.",
			Status:      StageStatusActive,
			RequiredActions: []StageAction{
				{Type: StageActionVisitLocations, Count: 1},
			},
			Reward:           StageReward{XP: 25, Coins: 10},
			UnlockOnComplete: StageUnlock{NextStageID: StageFirstClues},
		},
		{
			ID: StageFirstClues, Title: "First Clues",
			Description:       "Search the area and secure your first pieces of evidence.",
			Status:            StageStatusLocked,
			RequiredClueCount: firstClues,
			RequiredActions: []StageAction{
				{Type: StageActionFindClues, Count: firstClues},
			},
			Reward:           StageReward{XP: 50, Coins: 15},
			UnlockOnComplete: StageUnlock{NextStageID: StageIdentifySuspect, UnlockLocation: true},
		},
		{
			ID: StageIdentifySuspect, Title: identifyTitle,
			Description:       identifyDesc,
			Status:            StageStatusLocked,
			RequiredClueCount: clueTarget,
			RequiredActions:   identifyActions,
			Reward:            StageReward{XP: 100, Coins: 25, Badge: "sharp_eye"},
			UnlockOnComplete: StageUnlock{
				NextStageID: StageConfirmEvidence, UnlockLocation: true, RevealSuspect: true,
			},
		},
		{
			ID: StageConfirmEvidence, Title: "Confirm the Evidence",
			Description: "Inspect and confirm the evidence that carries your case.",
			Status:      StageStatusLocked,
			RequiredActions: []StageAction{
				{Type: StageActionConfirmEvidence, Count: confirmTarget},
			},
			Reward:           StageReward{XP: 100, Coins: 25},
			UnlockOnComplete: StageUnlock{NextStageID: StageFinalReport},
		},
		{
			ID: StageFinalReport, Title: "Final Report",
			Description: "You have what you need. Submit your final decision to command.",
			Status:      StageStatusLocked,
			RequiredActions: []StageAction{
				{Type: StageActionFinalDecision, Count: 1},
			},
			Reward:           StageReward{XP: 150, Coins: 50},
			UnlockOnComplete: StageUnlock{NextStageID: StageDebrief},
		},
		{
			ID: StageDebrief, Title: "Debrief",
			Description:      "Review the outcome, rewards, and what you missed.",
			Status:           StageStatusLocked,
			RequiredActions:  []StageAction{},
			Reward:           StageReward{},
			UnlockOnComplete: StageUnlock{},
		},
	}
}

// CurrentStage returns the first non-completed stage (the active one), or nil
// when every stage is completed.
func CurrentStage(stages []Stage) *Stage {
	for i := range stages {
		if stages[i].Status != StageStatusCompleted {
			return &stages[i]
		}
	}
	return nil
}

// StageIndex returns the 1-based position of the stage, for "Stage 2/6" UI.
func StageIndex(stages []Stage, id string) int {
	for i := range stages {
		if stages[i].ID == id {
			return i + 1
		}
	}
	return 0
}
