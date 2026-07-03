// Package worldagent implements the WorldAgent: it creates the World Bible
// core — hidden truth, world simulation state, scheduled events, and failure
// rules — plus the small player-safe public state (weather, ambient risk).
package worldagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "world"
	TaskType = "world_generation"
)

type Input struct {
	MissionType string `json:"mission_type"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Briefing    string `json:"briefing"`
	Difficulty  string `json:"difficulty"`
	Region      string `json:"region"`
	Language    string `json:"language"`
}

type ScheduledEvent struct {
	AtMinutes   int    `json:"at_minutes"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Output struct {
	// Truth is the hidden causal chain: core problem, real cause, hidden
	// antagonist. World Bible only.
	Truth map[string]any `json:"truth"`
	// HiddenState is the private world simulation state.
	HiddenState map[string]any `json:"hidden_state"`
	// PublicState is player-safe ambient state (weather, risk_level 0-100).
	PublicState map[string]any `json:"public_state"`
	// EventSchedule is the private list of future world events.
	EventSchedule []ScheduledEvent `json:"event_schedule"`
	FailureRules  []string         `json:"failure_rules"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Creates the hidden world truth, simulation state, event schedule, and failure rules.",
		Capabilities: []string{TaskType},
		InputSchema:  "worldagent.Input",
		OutputSchema: "worldagent.Output",
		MaxRetries:   2,
		Timeout:      90 * time.Second,
		Temperature:  0.8,
		MaxTokens:    3000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("worldagent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

You are the WorldAgent of AgentVerse. Create the private World Bible core for this mission (type %q, difficulty %q). Language: %s.

Rules:
- "truth" must include: core_problem, real_cause, hidden_antagonist (or null when the antagonist is environmental).
- "hidden_state" is the private simulation state (risk variables, factions, resources, antagonist plans).
- "public_state" is player-safe only: {"weather": string, "risk_level": int 0-100}.
- "event_schedule": 3-6 future world events with "at_minutes" (mission-clock minutes from start, ascending, 30-720).
- "failure_rules": 2-4 short conditions under which the mission fails.

Respond with ONLY one JSON object:
{"truth": object, "hidden_state": object, "public_state": {"weather": string, "risk_level": int},
 "event_schedule": [{"at_minutes": int, "type": string, "title": string, "description": string}],
 "failure_rules": [string]}`,
		TaskType, runtime.MissionSecurityPreamble, in.MissionType, in.Difficulty, in.Language)

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
	if len(out.Truth) == 0 {
		return nil, fmt.Errorf("truth is required")
	}
	if out.PublicState == nil {
		out.PublicState = map[string]any{"weather": "clear", "risk_level": 20}
	}
	if out.HiddenState == nil {
		out.HiddenState = map[string]any{}
	}
	return &out, nil
}
