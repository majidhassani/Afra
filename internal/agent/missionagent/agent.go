// Package missionagent implements the MissionAgent: it creates the mission
// plan — title, briefing, objectives, region, and win/fail conditions.
package missionagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "mission"
	TaskType = "mission_plan"
)

type Input struct {
	MissionType string `json:"mission_type"`
	Difficulty  string `json:"difficulty"`
	RegionHint  string `json:"region_hint,omitempty"`
	Language    string `json:"language"`
}

type Objective struct {
	Key           string `json:"key"`
	Type          string `json:"type"` // primary | required | optional | hidden | final
	Title         string `json:"title"`
	Description   string `json:"description"`
	RequiredClues int    `json:"required_clues"`
	Optional      bool   `json:"optional"`
}

type Output struct {
	Title          string      `json:"title"`
	Summary        string      `json:"summary"`
	Briefing       string      `json:"briefing"`
	Region         string      `json:"region"`
	Objectives     []Objective `json:"objectives"`
	WinConditions  []string    `json:"win_conditions"`
	FailConditions []string    `json:"fail_conditions"`
	// DeadlineHours is the in-world deadline in mission-clock hours (0 = no
	// hard deadline). Drives the mission timer.
	DeadlineHours int `json:"deadline_hours"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Creates the mission plan: title, briefing, objectives, win/fail conditions.",
		Capabilities: []string{TaskType},
		InputSchema:  "missionagent.Input",
		OutputSchema: "missionagent.Output",
		MaxRetries:   2,
		Timeout:      90 * time.Second,
		Temperature:  0.8,
		MaxTokens:    3000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("missionagent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
MISSION_TYPE: %s
%s

You are the MissionAgent of AgentVerse, an AI-native mission game. Design a playable field mission of type %q at difficulty %q. Language: %s.

Rules:
- The briefing addresses the player as a field agent and must NOT reveal the hidden solution.
- 3-5 structured objectives. Each has a "type": exactly one "primary" (the headline goal), one or more "required", at most one "optional", and exactly one "final" (the closing intervention/decision). You may add one "hidden" objective that unlocks later.
- "required_clues" is how many clues that objective needs (0-4). The final objective usually needs the most.
- The region is a short real-world-plausible area name (e.g. "Kavir Reserve, Iran").
- Win/fail conditions are short player-safe statements the player can read without spoiling the solution.
- "deadline_hours" is the in-world time budget in hours (24-120) before the mission fails; use 0 only for open-ended missions.

Respond with ONLY one JSON object:
{"title": string, "summary": string, "briefing": string, "region": string,
 "objectives": [{"key": string, "type": "primary"|"required"|"optional"|"hidden"|"final", "title": string, "description": string, "required_clues": int, "optional": bool}],
 "win_conditions": [string], "fail_conditions": [string], "deadline_hours": int}`,
		TaskType, in.MissionType, runtime.MissionSecurityPreamble, in.MissionType, in.Difficulty, in.Language)

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
	if out.Title == "" || out.Summary == "" || out.Briefing == "" {
		return nil, fmt.Errorf("title, summary, and briefing are required")
	}
	if len(out.Objectives) < 1 || len(out.Objectives) > 6 {
		return nil, fmt.Errorf("expected 1-6 objectives, got %d", len(out.Objectives))
	}
	for i := range out.Objectives {
		o := &out.Objectives[i]
		if o.Key == "" || o.Title == "" {
			return nil, fmt.Errorf("objective %d missing key or title", i)
		}
		if o.RequiredClues < 0 {
			o.RequiredClues = 0
		}
		if o.RequiredClues > 6 {
			o.RequiredClues = 6
		}
		o.Type = normalizeType(o.Type, o.Optional)
	}
	ensurePrimaryAndFinal(out.Objectives)
	if out.DeadlineHours < 0 {
		out.DeadlineHours = 0
	}
	return &out, nil
}

var validTypes = map[string]bool{
	"primary": true, "required": true, "optional": true,
	"hidden": true, "dynamic": true, "final": true,
}

// normalizeType coerces an objective type into the supported set, falling back
// to the legacy optional flag when the model omitted or mis-typed it.
func normalizeType(t string, optional bool) string {
	if validTypes[t] {
		return t
	}
	if optional {
		return "optional"
	}
	return "required"
}

// ensurePrimaryAndFinal guarantees the mission always has exactly one primary
// objective and at least one final objective, so the win model is well-formed
// even when the model under-specifies. It never invents objectives — it only
// re-labels existing ones.
func ensurePrimaryAndFinal(objectives []Objective) {
	if len(objectives) == 0 {
		return
	}
	primary := -1
	final := -1
	for i := range objectives {
		switch objectives[i].Type {
		case "primary":
			if primary == -1 {
				primary = i
			} else {
				objectives[i].Type = "required" // demote extra primaries
			}
		case "final":
			final = i
		}
	}
	if primary == -1 {
		objectives[0].Type = "primary"
		primary = 0
	}
	if final == -1 {
		// Promote the last non-primary objective to the closing decision.
		for i := len(objectives) - 1; i >= 0; i-- {
			if i != primary {
				objectives[i].Type = "final"
				final = i
				break
			}
		}
	}
}
