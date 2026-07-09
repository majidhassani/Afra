// Package mission owns the mission lifecycle: the root aggregate of
// AgentVerse. A mission is a map-based, AI-driven scenario; the old
// detective case is one mission type among several.
package mission

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"casemind/pkg/validator"
)

// Lifecycle statuses.
const (
	StatusDraft      = "draft"
	StatusGenerating = "generating"
	StatusReady      = "ready"
	StatusActive     = "active"
	StatusPaused     = "paused"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusArchived   = "archived"
)

var (
	ValidTypes = []string{
		"detective", "wildlife_rescue", "disaster_response", "exploration",
		"survival", "diplomacy", "medical_mystery",
	}
	ValidDifficulties = []string{"easy", "medium", "hard", "expert"}
	ValidLanguages    = []string{"en", "fa"}
)

// Epoch is mission-clock zero: every mission starts at "Day 1 - 08:00".
var Epoch = time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

// FormatClock renders a mission timestamp as "Day N - HH:MM".
func FormatClock(t time.Time) string {
	if t.Before(Epoch) {
		t = Epoch
	}
	elapsed := t.Sub(Epoch)
	day := int(elapsed.Hours()/24) + 1
	return fmt.Sprintf("Day %d - %02d:%02d", day, t.UTC().Hour(), t.UTC().Minute())
}

// MinutesSinceStart is the mission-clock minutes elapsed.
func MinutesSinceStart(t time.Time) int {
	if t.Before(Epoch) {
		return 0
	}
	return int(t.Sub(Epoch).Minutes())
}

// Objective is one structured mission objective (stored as JSONB).
//
// Type and Progress were added by the Mission Guidance upgrade; older stored
// objectives predate them, so read them through NormalizedType()/ClampProgress
// which fill sensible defaults from the legacy Optional flag.
type Objective struct {
	ID            string `json:"id"`
	Type          string `json:"type"` // primary | required | optional | hidden | dynamic | final
	Title         string `json:"title"`
	Description   string `json:"description"`
	Status        string `json:"status"`   // locked | active | completed | failed | skipped
	Progress      int    `json:"progress"` // 0-100
	RequiredClues int    `json:"required_clues"`
	Optional      bool   `json:"optional"`
}

// NormalizedType returns the objective type, deriving one from the legacy
// Optional flag when Type is empty (objectives generated before this upgrade).
func (o Objective) NormalizedType() string {
	if o.Type != "" {
		return o.Type
	}
	if o.Optional {
		return ObjectiveOptional
	}
	return ObjectiveRequired
}

// Mandatory reports whether the objective must be completed to win.
func (o Objective) Mandatory() bool {
	switch o.NormalizedType() {
	case ObjectivePrimary, ObjectiveRequired, ObjectiveFinal:
		return true
	default:
		return false
	}
}

type Mission struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	Status      string          `json:"status"`
	Difficulty  string          `json:"difficulty"`
	Region      string          `json:"region"`
	Summary     string          `json:"summary"`
	Briefing    string          `json:"briefing"`
	Objectives  json.RawMessage `json:"objectives"`
	Stages      json.RawMessage `json:"stages"`
	PublicState json.RawMessage `json:"public_state"`
	BoardArt    json.RawMessage `json:"board_art,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
	CenterLat   float64         `json:"center_lat"`
	CenterLng   float64         `json:"center_lng"`
	MapZoom     int             `json:"map_zoom"`
	MissionTime time.Time       `json:"-"`
	CurrentTime string          `json:"current_time"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

// Playable reports whether AI actions are allowed.
func (m *Mission) Playable() bool { return m.Status == StatusActive }

// ParsedObjectives decodes the JSONB objectives.
func (m *Mission) ParsedObjectives() []Objective {
	var objectives []Objective
	_ = json.Unmarshal(m.Objectives, &objectives)
	return objectives
}

func ValidateNewMission(missionType, difficulty, language string) error {
	return validator.New().
		Required("type", missionType).OneOf("type", missionType, ValidTypes...).
		Required("difficulty", difficulty).OneOf("difficulty", difficulty, ValidDifficulties...).
		OneOf("language", language, ValidLanguages...).
		Err()
}

// XPForDifficulty is the base XP for completing a mission at full score.
func XPForDifficulty(difficulty string) int {
	switch difficulty {
	case "easy":
		return 100
	case "medium":
		return 250
	case "hard":
		return 500
	case "expert":
		return 1000
	default:
		return 100
	}
}

// CoinRewardForScore converts a judge score into a coin reward.
func CoinRewardForScore(score int) int {
	if score < 0 {
		return 0
	}
	return score * 2
}
