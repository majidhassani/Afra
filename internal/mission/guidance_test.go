package mission

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleGuidanceInputs() GuidanceInputs {
	return GuidanceInputs{
		Objectives: []Objective{
			{ID: "obj_primary", Type: ObjectivePrimary, Title: "Save the remaining cheetahs", Status: ObjStatusActive},
			{ID: "obj_final", Type: ObjectiveFinal, Title: "Close the migration corridor", Status: ObjStatusActive},
		},
		Locations: []VisibleLocation{
			{ID: "loc_ranger", Name: "Ranger Station", Status: "visited"},
			{ID: "loc_river", Name: "River Crossing", Status: "discovered"},
		},
		Characters: []VisibleCharacter{
			{ID: "char_ctrl", Name: "KAVIR Control", Role: "Mission Control AI", Interacted: true},
			{ID: "char_ranger", Name: "Thomas Reed", Role: "Head Ranger", Interacted: false},
		},
		DiscoveredClues: 2,
		Progress:        40,
		Risk:            30,
	}
}

func TestGuidanceReturnsNextRecommendedAction(t *testing.T) {
	g := BuildGuidance(sampleGuidanceInputs())
	if len(g.RecommendedActions) == 0 {
		t.Fatal("guidance must always return at least one recommended action")
	}
	first := g.RecommendedActions[0]
	if first.Action != ActionVisitLocation || first.LocationID != "loc_river" {
		t.Fatalf("expected the unvisited River Crossing to be recommended first, got %+v", first)
	}
	if first.Priority != PriorityHigh {
		t.Fatalf("the next best step should be high priority, got %q", first.Priority)
	}
	if first.Reason == "" {
		t.Fatal("a recommended action must explain why it matters")
	}
	if g.CurrentGoal != "Save the remaining cheetahs" {
		t.Fatalf("current goal should be the active primary objective, got %q", g.CurrentGoal)
	}
}

func TestGuidanceRecommendsTalkingToNewCharacter(t *testing.T) {
	in := sampleGuidanceInputs()
	in.Locations = []VisibleLocation{{ID: "loc_ranger", Name: "Ranger Station", Status: "visited"}}
	g := BuildGuidance(in)
	var talked bool
	for _, a := range g.RecommendedActions {
		if a.Action == ActionTalkToCharacter && a.CharacterID == "char_ranger" {
			talked = true
		}
		if a.CharacterID == "char_ctrl" {
			t.Fatal("guidance must not recommend talking to the mission-control guide")
		}
	}
	if !talked {
		t.Fatal("expected guidance to recommend talking to the un-interviewed ranger")
	}
}

// hiddenTruthTokens are strings from the World Bible / secret state that must
// never appear in player-facing guidance.
var hiddenTruthTokens = []string{
	"Farid Kia", "logistics officer", "jammer", "quarry", "poacher",
	"patrol schedules", "cloned", "antagonist", "hidden_antagonist",
}

func TestGuidanceDoesNotRevealHiddenTruth(t *testing.T) {
	// Even if visible entity names are benign, the guidance builder only ever
	// consumes visible state, so no bible secret can leak into its output.
	g := BuildGuidance(sampleGuidanceInputs())
	blob, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(blob))
	for _, tok := range hiddenTruthTokens {
		if strings.Contains(lower, strings.ToLower(tok)) {
			t.Fatalf("guidance leaked hidden-truth token %q: %s", tok, blob)
		}
	}
	if g.SpoilerLevel != "low" {
		t.Fatalf("deterministic guidance must be low spoiler, got %q", g.SpoilerLevel)
	}
}

func TestGuidanceWarnsOnDeadlineAndRisk(t *testing.T) {
	in := sampleGuidanceInputs()
	in.HasDeadline = true
	in.DeadlineApproaching = true
	in.TimeRemaining = "3 hours"
	in.Risk = 80
	g := BuildGuidance(in)
	if len(g.Warnings) < 2 {
		t.Fatalf("expected deadline + high-risk warnings, got %v", g.Warnings)
	}
	joined := strings.ToLower(strings.Join(g.Warnings, " "))
	if !strings.Contains(joined, "time") || !strings.Contains(joined, "risk") {
		t.Fatalf("warnings should cover time and risk, got %v", g.Warnings)
	}
}
