package detective

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Rank         string    `json:"rank"`
	XP           int       `json:"xp"`
	SolvedCases  int       `json:"solved_cases"`
	FailedCases  int       `json:"failed_cases"`
	TotalCases   int       `json:"total_cases"`
	AccuracyRate float64   `json:"accuracy_rate"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// HistoryEntry summarizes one case for the detective's history view.
type HistoryEntry struct {
	CaseID     uuid.UUID  `json:"case_id"`
	Title      string     `json:"title"`
	Type       string     `json:"type"`
	Difficulty string     `json:"difficulty"`
	Status     string     `json:"status"`
	Attempts   int        `json:"attempts"`
	CreatedAt  time.Time  `json:"created_at"`
	SolvedAt   *time.Time `json:"solved_at,omitempty"`
}

var rankThresholds = []struct {
	XP   int
	Rank string
}{
	{6000, "Master Detective"},
	{3000, "Chief Inspector"},
	{1500, "Inspector"},
	{700, "Senior Detective"},
	{300, "Detective"},
	{100, "Junior Detective"},
	{0, "Rookie"},
}

// RankForXP maps accumulated XP to a rank title.
func RankForXP(xp int) string {
	for _, t := range rankThresholds {
		if xp >= t.XP {
			return t.Rank
		}
	}
	return "Rookie"
}
