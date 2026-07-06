package mission

import "fmt"

// This file holds the deterministic "is the mission ready to finish?" check,
// shared by the dashboard (to show can_complete) and the completion flow (to
// gate the final JudgeAgent). Keeping it in one place means the answer the
// player sees and the gate the server enforces can never disagree.

// ReadinessInputs are the collected signals the readiness check consumes.
type ReadinessInputs struct {
	DiscoveredClues    int
	InteractedChars    int
	VisitedLocations   int
	RequiredClueTarget int
}

// EvaluateReadiness reports whether the mission may be submitted for a final
// verdict, and — when it may not — the concrete requirements still missing, in
// plain player-facing language.
func EvaluateReadiness(in ReadinessInputs) (canComplete bool, missing []string) {
	missing = []string{}
	if in.RequiredClueTarget > 0 && in.DiscoveredClues < in.RequiredClueTarget {
		missing = append(missing, fmt.Sprintf(
			"Collect at least %d critical clues (you have %d)",
			in.RequiredClueTarget, in.DiscoveredClues))
	}
	if in.VisitedLocations < 1 {
		missing = append(missing, "Explore at least one location")
	}
	if in.InteractedChars < 1 {
		missing = append(missing, "Interview at least one character")
	}
	return len(missing) == 0, missing
}

// MandatoryClueTarget is the total clue coverage the mandatory objectives
// (primary, required, final) demand before the mission can be completed.
func MandatoryClueTarget(objectives []Objective) int {
	total := 0
	for _, o := range objectives {
		if o.Mandatory() {
			total += o.RequiredClues
		}
	}
	return total
}
