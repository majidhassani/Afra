package mission

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMissionProgressRewardsClueDiscovery(t *testing.T) {
	base := ProgressInputs{
		TotalClues: 4, TotalChars: 3, TotalLocations: 4, DecisionTarget: 4,
	}
	before := MissionProgress(base, DefaultProgressWeights)

	after := base
	after.DiscoveredClues = 2
	got := MissionProgress(after, DefaultProgressWeights)
	if got <= before {
		t.Fatalf("discovering clues must raise progress: before=%d after=%d", before, got)
	}
	// 2/4 clues at 30% weight = 15 points.
	if got != 15 {
		t.Fatalf("expected 15%% from 2/4 clues, got %d", got)
	}
}

func TestMissionProgressRewardsCharacterInteraction(t *testing.T) {
	base := ProgressInputs{
		TotalClues: 4, TotalChars: 4, TotalLocations: 4, DecisionTarget: 4,
	}
	before := MissionProgress(base, DefaultProgressWeights)

	after := base
	after.InteractedChars = 2 // 2/4 at 25% weight = ~12.5 -> 13
	got := MissionProgress(after, DefaultProgressWeights)
	if got <= before {
		t.Fatalf("interacting with characters must raise progress: before=%d after=%d", before, got)
	}
}

func TestMissionProgressFullIsHundred(t *testing.T) {
	in := ProgressInputs{
		DiscoveredClues: 5, TotalClues: 5,
		InteractedChars: 3, TotalChars: 3,
		VisitedLocations: 4, TotalLocations: 4,
		CorrectDecisions: 6, DecisionTarget: 6,
		FinalReady: true,
	}
	if got := MissionProgress(in, DefaultProgressWeights); got != 100 {
		t.Fatalf("fully complete mission must be 100%%, got %d", got)
	}
}

func TestRiskIncreasesAfterTimeAdvance(t *testing.T) {
	base := RiskInputs{AmbientRisk: 30, DeadlineMinutes: 1440}
	before := RiskScore(base)

	advanced := base
	advanced.ElapsedMinutes = 600 // advanced the clock
	after := RiskScore(advanced)
	if after <= before {
		t.Fatalf("advancing time must raise risk: before=%d after=%d", before, after)
	}
}

func TestRiskIncreasesWithoutDeadline(t *testing.T) {
	base := RiskInputs{AmbientRisk: 20}
	before := RiskScore(base)
	base.ElapsedMinutes = 720
	if after := RiskScore(base); after <= before {
		t.Fatalf("time pressure must apply even without a deadline: before=%d after=%d", before, after)
	}
}

func TestTimeRemainingFormatting(t *testing.T) {
	label, remaining, has := TimeRemaining(6*60, 5*24*60) // 6h into a 5-day budget
	if !has || remaining <= 0 {
		t.Fatalf("expected a live deadline, got has=%v remaining=%d", has, remaining)
	}
	if !strings.Contains(label, "day") {
		t.Fatalf("expected a day-scale label, got %q", label)
	}

	if _, _, has := TimeRemaining(10, 0); has {
		t.Fatal("no deadline must report hasDeadline=false")
	}
	if label, _, _ := TimeRemaining(100, 50); label != "expired" {
		t.Fatalf("passed deadline must be expired, got %q", label)
	}
}

func TestObjectiveTypeAndMandatoryDefaults(t *testing.T) {
	// A legacy objective with no Type falls back to the optional flag.
	legacy := Objective{Optional: true}
	if legacy.NormalizedType() != ObjectiveOptional {
		t.Fatalf("optional legacy objective should normalize to optional, got %q", legacy.NormalizedType())
	}
	if legacy.Mandatory() {
		t.Fatal("optional objective must not be mandatory")
	}
	for _, ty := range []string{ObjectivePrimary, ObjectiveRequired, ObjectiveFinal} {
		if !(Objective{Type: ty}).Mandatory() {
			t.Fatalf("%s objective must be mandatory", ty)
		}
	}
}

func TestParsedObjectivesRoundTrip(t *testing.T) {
	raw := json.RawMessage(`[{"id":"o1","type":"primary","title":"Save them","status":"active","progress":40,"required_clues":3}]`)
	m := &Mission{Objectives: raw}
	objs := m.ParsedObjectives()
	if len(objs) != 1 || objs[0].Type != ObjectivePrimary || objs[0].Progress != 40 {
		t.Fatalf("objective did not round-trip: %+v", objs)
	}
}
