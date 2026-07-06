// Package missiondirector implements the mission DirectorAgent: it keeps the
// mission alive by reacting to player progress with pacing events and an
// optional nudge, without revealing hidden truth.
package missiondirector

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "mission_director"
	TaskType = "mission_director"
)

type Input struct {
	MissionSummary   string   `json:"mission_summary"`
	MissionTime      string   `json:"mission_time"`
	Objectives       []string `json:"objectives"`
	DiscoveredFacts  []string `json:"discovered_facts"`
	DiscoveredClues  int      `json:"discovered_clues"`
	TotalClues       int      `json:"total_clues"`
	VisitedLocations int      `json:"visited_locations"`
	TotalLocations   int      `json:"total_locations"`
	AIInteractions   int      `json:"ai_interactions"`
	Language         string   `json:"language"`
}

type Event struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

type Output struct {
	Events []Event `json:"events"`
	Hint   string  `json:"hint"`
	Tone   string  `json:"tone"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Keeps the mission alive: pacing events and gentle nudges based on progress.",
		Capabilities: []string{TaskType},
		InputSchema:  "missiondirector.Input",
		OutputSchema: "missiondirector.Output",
		MaxRetries:   1,
		Timeout:      45 * time.Second,
		Temperature:  0.7,
		MaxTokens:    800,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("missiondirector: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

%s

You are the DirectorAgent of AgentVerse: an invisible game master watching mission pacing.

Behavior rules:
- Based only on the player-safe progress numbers and discovered facts, produce 0-2 small pacing events (type like map_activity, radio_chatter, weather_hint) and one short player-safe hint.
- If progress is healthy, keep events empty and the hint encouraging.
- Never reveal hidden truth or undiscovered clue locations.

Respond with ONLY one JSON object:
{"events": [{"type": string, "title": string}], "hint": string, "tone": string}`,
		TaskType, runtime.MissionSecurityPreamble, runtime.LanguageDirective(in.Language))

	ctxJSON, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: "PLAYER-VISIBLE CONTEXT:\n" + string(ctxJSON)},
	}, nil
}

func (a *Agent) Parse(raw []byte) (any, error) {
	var out Output
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if out.Hint == "" && len(out.Events) == 0 {
		return nil, fmt.Errorf("director output must contain a hint or events")
	}
	if len(out.Events) > 3 {
		out.Events = out.Events[:3]
	}
	return &out, nil
}
