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
- 2-4 structured objectives; at most one optional. "required_clues" between 1 and 4.
- The region is a short real-world-plausible area name (e.g. "Kavir Reserve, Iran").
- Win/fail conditions are short player-safe statements.

Respond with ONLY one JSON object:
{"title": string, "summary": string, "briefing": string, "region": string,
 "objectives": [{"key": string, "title": string, "description": string, "required_clues": int, "optional": bool}],
 "win_conditions": [string], "fail_conditions": [string]}`,
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
	}
	return &out, nil
}
