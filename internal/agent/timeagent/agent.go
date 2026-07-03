// Package timeagent implements the TimeAgent: when the player advances the
// mission clock, it narrates what changed and returns structured world-state
// deltas and events, guided by the World Bible's private event schedule.
package timeagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "time"
	TaskType = "time_advance"
)

type ScheduledEvent struct {
	AtMinutes   int    `json:"at_minutes"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Input struct {
	MissionSummary string `json:"mission_summary"`
	CurrentClock   string `json:"current_clock"`
	NewClock       string `json:"new_clock"`
	AdvanceMinutes int    `json:"advance_minutes"`

	PublicState map[string]any `json:"public_state"`

	// Confidential: schedule entries due within the advanced window and the
	// private world state.
	DueEvents   []ScheduledEvent `json:"due_events"`
	HiddenState map[string]any   `json:"hidden_state"`

	DiscoveredLocations []string `json:"discovered_locations"`
	Language            string   `json:"language"`
}

type Event struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

type Output struct {
	Summary string  `json:"summary"`
	Events  []Event `json:"events"`
	// PublicStateChanges merges into the mission's player-safe state
	// (weather, risk_level).
	PublicStateChanges map[string]any `json:"public_state_changes"`
	// HiddenStateChanges merges into the World Bible hidden state.
	HiddenStateChanges map[string]any `json:"hidden_state_changes"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Advances mission time, producing world change events and state deltas.",
		Capabilities: []string{TaskType},
		InputSchema:  "timeagent.Input",
		OutputSchema: "timeagent.Output",
		MaxRetries:   2,
		Timeout:      60 * time.Second,
		Temperature:  0.7,
		MaxTokens:    1500,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("timeagent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

You are the TimeAgent of AgentVerse. Mission time advances from %s to %s (%d minutes). Language: %s.

Behavior rules:
- Weave the confidential due events into player-safe observations: the player notices effects, not hidden causes.
- "summary": 1-3 sentences describing what visibly changed.
- "events": 1-4 player-safe events, "type" like weather_changed, map_activity, npc_movement, world_state_change.
- "public_state_changes": only player-safe keys (weather, risk_level 0-100).
- "hidden_state_changes": private simulation updates consistent with the due events.
- Never reveal hidden truth, future schedule, or undiscovered clues.

Respond with ONLY one JSON object:
{"summary": string, "events": [{"type": string, "title": string}],
 "public_state_changes": object, "hidden_state_changes": object}`,
		TaskType, runtime.MissionSecurityPreamble, in.CurrentClock, in.NewClock, in.AdvanceMinutes, in.Language)

	ctxJSON, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: "CONFIDENTIAL CONTEXT:\n" + string(ctxJSON)},
	}, nil
}

func (a *Agent) Parse(raw []byte) (any, error) {
	var out Output
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if out.Summary == "" {
		return nil, fmt.Errorf("summary is required")
	}
	if len(out.Events) > 6 {
		out.Events = out.Events[:6]
	}
	if out.PublicStateChanges == nil {
		out.PublicStateChanges = map[string]any{}
	}
	if out.HiddenStateChanges == nil {
		out.HiddenStateChanges = map[string]any{}
	}
	return &out, nil
}
