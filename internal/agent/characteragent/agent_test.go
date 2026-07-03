package characteragent

import (
	"testing"

	"casemind/internal/agent/runtime"
)

func TestFallbackBuildsValidCharacters(t *testing.T) {
	agent := New()
	outAny, ok := agent.Fallback(runtime.Task{
		Type: TaskType,
		Input: Input{
			MissionType: "wildlife_rescue",
			Summary:     "A rescue mission with unstable LLM output.",
			Locations: []LocationRef{
				{Key: "loc_station", Name: "Station"},
				{Key: "loc_river", Name: "River"},
				{Key: "loc_market", Name: "Market"},
			},
			Language: "en",
		},
	}, []byte(""), nil)
	if !ok {
		t.Fatal("expected fallback")
	}
	out := outAny.(*Output)
	if len(out.Characters) < 4 {
		t.Fatalf("expected at least 4 characters, got %d", len(out.Characters))
	}
	guides := 0
	for _, c := range out.Characters {
		if c.Category == "guide" {
			guides++
		}
		if c.Key == "" || c.Name == "" || c.AvatarPrompt == "" {
			t.Fatalf("invalid fallback character: %+v", c)
		}
	}
	if guides != 1 {
		t.Fatalf("expected exactly one guide, got %d", guides)
	}
}
