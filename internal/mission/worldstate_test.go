package mission

import (
	"encoding/json"
	"testing"
)

func ps(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

// TestDeriveWorldStateWeatherAndTime: weather keywords and the clock hour drive
// the living-world snapshot deterministically.
func TestDeriveWorldStateWeatherAndTime(t *testing.T) {
	ws := DeriveWorldState(ps(t, map[string]any{"weather": "cold rain over the city"}),
		"Day 1 — 21:30", "city", 20, 10, false, false)
	if ws.Weather != "rain" {
		t.Errorf("weather = %q, want rain", ws.Weather)
	}
	if ws.TimeOfDay != "night" {
		t.Errorf("time_of_day = %q, want night", ws.TimeOfDay)
	}
	if ws.Visibility != "low" {
		t.Errorf("visibility = %q, want low (night)", ws.Visibility)
	}
	// Rain (not danger) at night → rain modifier wins over night.
	if ws.ThemeID != "city_rain" {
		t.Errorf("theme_id = %q, want city_rain", ws.ThemeID)
	}
}

// TestDeriveWorldStateDanger: high risk forces the danger theme and urgency.
func TestDeriveWorldStateDanger(t *testing.T) {
	ws := DeriveWorldState(ps(t, map[string]any{}), "Day 2 — 03:00", "forest", 85, 40, true, true)
	if !ws.Danger {
		t.Errorf("expected danger at risk 85")
	}
	if ws.ThemeID != "forest_danger" {
		t.Errorf("theme_id = %q, want forest_danger", ws.ThemeID)
	}
	if ws.Urgency != "critical" {
		t.Errorf("urgency = %q, want critical", ws.Urgency)
	}
	if ws.TimeOfDay != "night" {
		t.Errorf("03:00 should be night, got %q", ws.TimeOfDay)
	}
}

// TestDeriveWorldStatePhase: world phase tracks progress.
func TestDeriveWorldStatePhase(t *testing.T) {
	cases := map[int]string{5: "opening", 40: "investigation", 80: "closing"}
	for progress, want := range cases {
		ws := DeriveWorldState(ps(t, map[string]any{}), "Day 1 — 09:00", "desert", 10, progress, false, false)
		if ws.WorldPhase != want {
			t.Errorf("progress %d: phase = %q, want %q", progress, ws.WorldPhase, want)
		}
	}
	// Clear day with no danger → day theme.
	ws := DeriveWorldState(ps(t, map[string]any{}), "Day 1 — 10:00", "desert", 10, 5, false, false)
	if ws.ThemeID != "desert_day" {
		t.Errorf("theme_id = %q, want desert_day", ws.ThemeID)
	}
}
