// Package cases owns the case lifecycle. (Directory is /internal/case per
// the architecture spec; the package is named "cases" because "case" is a
// Go keyword.)
package cases

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"casemind/pkg/validator"
)

const (
	StatusGenerating = "generating"
	StatusOpen       = "open"
	StatusSolved     = "solved"
	StatusFailed     = "failed"
	StatusArchived   = "archived"
)

var (
	ValidTypes        = []string{"murder", "kidnapping", "missing_person", "robbery", "fraud"}
	ValidDifficulties = []string{"easy", "medium", "hard", "expert"}
	ValidLanguages    = []string{"en", "fa"}
)

// MaxSolveAttempts is the number of accusations allowed before a case fails.
const MaxSolveAttempts = 3

type Case struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Title      string     `json:"title"`
	Type       string     `json:"type"`
	Difficulty string     `json:"difficulty"`
	Status     string     `json:"status"`
	Summary    string     `json:"summary"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	SolvedAt   *time.Time `json:"solved_at,omitempty"`
}

func ValidateNewCase(caseType, difficulty string) error {
	return validator.New().
		Required("type", caseType).OneOf("type", caseType, ValidTypes...).
		Required("difficulty", difficulty).OneOf("difficulty", difficulty, ValidDifficulties...).
		Err()
}

func ValidateNewCaseLanguage(language string) error {
	return validator.New().
		Required("language", language).OneOf("language", language, ValidLanguages...).
		Err()
}

func NormalizeLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "":
		return "en"
	case "fa", "fa-ir", "farsi", "persian", "فارسی":
		return "fa"
	case "en", "en-us", "en-gb", "english", "انگلیسی":
		return "en"
	default:
		return language
	}
}

// XPForDifficulty is the base XP awarded for solving a case.
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
