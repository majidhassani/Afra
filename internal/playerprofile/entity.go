// Package playerprofile generalizes the detective profile into the
// AgentVerse player profile: level, XP, rank, mission statistics, and
// activity counters.
package playerprofile

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID                    uuid.UUID       `json:"id"`
	UserID                uuid.UUID       `json:"user_id"`
	DisplayName           string          `json:"display_name"`
	Rank                  string          `json:"rank"`
	Level                 int             `json:"level"`
	XP                    int             `json:"xp"`
	TotalMissions         int             `json:"total_missions"`
	CompletedMissions     int             `json:"completed_missions"`
	FailedMissions        int             `json:"failed_missions"`
	SuccessRate           float64         `json:"success_rate"`
	FavoriteMissionType   string          `json:"favorite_mission_type"`
	TotalCluesFound       int             `json:"total_clues_found"`
	TotalAIInteractions   int             `json:"total_ai_interactions"`
	TotalLocationsVisited int             `json:"total_locations_visited"`
	Badges                json.RawMessage `json:"badges"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// Stats is the profile plus live wallet aggregates.
type Stats struct {
	Profile          *Profile `json:"profile"`
	TotalCoinsSpent  int      `json:"total_coins_spent"`
	TotalCoinsEarned int      `json:"total_coins_earned"`
}

// HistoryEntry summarizes one mission for the player's history view.
type HistoryEntry struct {
	MissionID   uuid.UUID       `json:"mission_id"`
	Title       string          `json:"title"`
	Type        string          `json:"type"`
	Difficulty  string          `json:"difficulty"`
	Status      string          `json:"status"`
	Result      json.RawMessage `json:"result,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

// XPPerLevel keeps leveling linear and predictable.
const XPPerLevel = 300

func LevelForXP(xp int) int {
	if xp < 0 {
		xp = 0
	}
	return 1 + xp/XPPerLevel
}

var rankThresholds = []struct {
	XP   int
	Rank string
}{
	{6000, "Legend"},
	{3000, "Commander"},
	{1500, "Special Agent"},
	{700, "Senior Agent"},
	{300, "Agent"},
	{100, "Field Agent"},
	{0, "Recruit"},
}

// RankForXP maps accumulated XP to a rank title.
func RankForXP(xp int) string {
	for _, t := range rankThresholds {
		if xp >= t.XP {
			return t.Rank
		}
	}
	return "Recruit"
}
