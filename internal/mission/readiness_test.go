package mission

import (
	"strings"
	"testing"
)

func TestReadinessBlocksUntilRequirementsMet(t *testing.T) {
	canComplete, missing := EvaluateReadiness(ReadinessInputs{
		DiscoveredClues:    1,
		InteractedChars:    0,
		VisitedLocations:   0,
		RequiredClueTarget: 4,
	})
	if canComplete {
		t.Fatal("mission must not be completable with unmet requirements")
	}
	joined := strings.ToLower(strings.Join(missing, " | "))
	for _, want := range []string{"clue", "location", "character"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing requirement for %q not reported: %v", want, missing)
		}
	}
}

func TestReadinessAllowsWhenRequirementsMet(t *testing.T) {
	canComplete, missing := EvaluateReadiness(ReadinessInputs{
		DiscoveredClues:    4,
		InteractedChars:    2,
		VisitedLocations:   3,
		RequiredClueTarget: 4,
	})
	if !canComplete {
		t.Fatalf("mission should be completable once requirements are met, missing=%v", missing)
	}
	if len(missing) != 0 {
		t.Fatalf("no requirements should be missing, got %v", missing)
	}
}

func TestMandatoryClueTargetIgnoresOptional(t *testing.T) {
	objectives := []Objective{
		{Type: ObjectivePrimary, RequiredClues: 3},
		{Type: ObjectiveRequired, RequiredClues: 2},
		{Type: ObjectiveOptional, RequiredClues: 5},
		{Type: ObjectiveFinal, RequiredClues: 4},
	}
	if got := MandatoryClueTarget(objectives); got != 9 {
		t.Fatalf("expected mandatory target 3+2+4=9, got %d", got)
	}
}
