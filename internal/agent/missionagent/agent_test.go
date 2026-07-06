package missionagent

import "testing"

func parse(t *testing.T, raw string) *Output {
	t.Helper()
	out, err := New().Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return out.(*Output)
}

func TestParseGuaranteesPrimaryAndFinal(t *testing.T) {
	// The model omitted types entirely; Parse must still produce a well-formed
	// win model: exactly one primary and at least one final objective.
	out := parse(t, `{
      "title": "Save the Reserve", "summary": "s", "briefing": "b", "region": "r",
      "objectives": [
        {"key": "a", "title": "Investigate", "required_clues": 2},
        {"key": "b", "title": "Interview staff", "required_clues": 1},
        {"key": "c", "title": "Act on findings", "required_clues": 3}
      ],
      "win_conditions": ["find the cause"], "fail_conditions": ["time runs out"],
      "deadline_hours": 96
    }`)

	primaries, finals, required := 0, 0, 0
	for _, o := range out.Objectives {
		switch o.Type {
		case "primary":
			primaries++
		case "final":
			finals++
		case "required":
			required++
		}
	}
	if primaries != 1 {
		t.Fatalf("expected exactly one primary objective, got %d", primaries)
	}
	if finals < 1 {
		t.Fatal("expected at least one final objective")
	}
	if required < 1 {
		t.Fatal("expected at least one required objective")
	}
	if out.DeadlineHours != 96 {
		t.Fatalf("deadline should round-trip, got %d", out.DeadlineHours)
	}
}

func TestParseDemotesExtraPrimaries(t *testing.T) {
	out := parse(t, `{
      "title": "t", "summary": "s", "briefing": "b", "region": "r",
      "objectives": [
        {"key": "a", "type": "primary", "title": "Goal A", "required_clues": 2},
        {"key": "b", "type": "primary", "title": "Goal B", "required_clues": 2},
        {"key": "c", "type": "final", "title": "Finish", "required_clues": 3}
      ],
      "win_conditions": [], "fail_conditions": []
    }`)

	primaries := 0
	for _, o := range out.Objectives {
		if o.Type == "primary" {
			primaries++
		}
	}
	if primaries != 1 {
		t.Fatalf("extra primaries must be demoted to a single primary, got %d", primaries)
	}
}

func TestParseNormalizesUnknownTypeFromOptionalFlag(t *testing.T) {
	out := parse(t, `{
      "title": "t", "summary": "s", "briefing": "b", "region": "r",
      "objectives": [
        {"key": "a", "type": "primary", "title": "Goal", "required_clues": 2},
        {"key": "b", "type": "nonsense", "title": "Bonus", "optional": true, "required_clues": 1},
        {"key": "c", "type": "final", "title": "Finish", "required_clues": 3}
      ],
      "win_conditions": [], "fail_conditions": []
    }`)

	for _, o := range out.Objectives {
		if !validTypes[o.Type] {
			t.Fatalf("objective %q has invalid type %q", o.Key, o.Type)
		}
		if o.Key == "b" && o.Type != "optional" {
			t.Fatalf("unknown type on an optional objective should become optional, got %q", o.Type)
		}
	}
}
