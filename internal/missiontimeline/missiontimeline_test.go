package missiontimeline

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"casemind/internal/missionevent"
)

func event(t *testing.T, evType string, payload map[string]any, at time.Time) missionevent.Event {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return missionevent.Event{
		ID: uuid.New(), MissionID: uuid.New(),
		Type: evType, Payload: raw, CreatedAt: at,
	}
}

// TestCurateOrdersOldestFirstAndNormalizesTypes: raw events arrive newest
// first; the curated timeline reads as a story, oldest first, with
// normalized player-facing categories.
func TestCurateOrdersOldestFirstAndNormalizesTypes(t *testing.T) {
	base := time.Now()
	events := []missionevent.Event{ // newest first, as ListByMission returns
		event(t, "mission_completed", map[string]any{"score": 84, "title": "Mission Successful"}, base.Add(3*time.Minute)),
		event(t, "dialogue", map[string]any{"character_name": "Head Ranger"}, base.Add(2*time.Minute)),
		event(t, "clue_discovered", map[string]any{"title": "Broken GPS Collar"}, base.Add(time.Minute)),
		event(t, "mission_ready", map[string]any{"title": "Vanished Herd"}, base),
	}
	items := Curate(events)
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}
	wantTypes := []string{TypeMissionStarted, TypeClueDiscovered, TypeCharacterTalked, TypeMissionCompleted}
	for i, want := range wantTypes {
		if items[i].Type != want {
			t.Errorf("item %d: type = %q, want %q", i, items[i].Type, want)
		}
	}
	if items[1].Title != "Broken GPS Collar" {
		t.Errorf("clue title = %q", items[1].Title)
	}
	if items[1].Importance != ImportanceHigh {
		t.Errorf("clue importance = %q, want high", items[1].Importance)
	}
}

// TestCurateDropsInternalNoise: generation progress, raw facts, and director
// pacing hints never appear in the curated player timeline.
func TestCurateDropsInternalNoise(t *testing.T) {
	base := time.Now()
	events := []missionevent.Event{
		event(t, "fact_discovered", map[string]any{"fact": "secret-ish internal memory"}, base.Add(3*time.Second)),
		event(t, "generation_progress", map[string]any{"stage": "world"}, base.Add(2*time.Second)),
		event(t, "director_hint", map[string]any{"hint": "pace up"}, base.Add(time.Second)),
		event(t, "player_action", map[string]any{"action": "inspect_area"}, base),
	}
	if items := Curate(events); len(items) != 0 {
		t.Fatalf("expected internal events to be dropped, got %d items", len(items))
	}
}

// TestCurateUnknownTypesBecomeWorldEvents: dynamic time/director agent event
// types surface as world events only when they carry player-safe text.
func TestCurateUnknownTypesBecomeWorldEvents(t *testing.T) {
	base := time.Now()
	events := []missionevent.Event{
		event(t, "sandstorm_incoming", map[string]any{"title": "A sandstorm approaches"}, base.Add(time.Second)),
		event(t, "mystery_blob", map[string]any{}, base),
	}
	items := Curate(events)
	if len(items) != 1 {
		t.Fatalf("expected 1 world event, got %d", len(items))
	}
	if items[0].Type != TypeWorldEvent || items[0].Title != "A sandstorm approaches" {
		t.Errorf("unexpected item: %+v", items[0])
	}
}

// TestCurateCollapsesRepeatedReadyNotices: repeated ready-to-complete checks
// must not spam the story.
func TestCurateCollapsesRepeatedReadyNotices(t *testing.T) {
	base := time.Now()
	events := []missionevent.Event{
		event(t, "mission_ready_to_complete", map[string]any{"title": "Ready for final decision"}, base.Add(2*time.Second)),
		event(t, "mission_ready_to_complete", map[string]any{"title": "Ready for final decision"}, base.Add(time.Second)),
	}
	items := Curate(events)
	if len(items) != 1 {
		t.Fatalf("expected collapsed ready notice, got %d items", len(items))
	}
}
