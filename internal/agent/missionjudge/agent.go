// Package missionjudge implements the mission JudgeAgent: it evaluates a
// mission completion attempt against the hidden truth and the player's
// actual progress, producing a verdict intention.
package missionjudge

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "mission_judge"
	TaskType = "mission_judgment"
)

type ObjectiveState struct {
	Key           string `json:"key"`
	Title         string `json:"title"`
	RequiredClues int    `json:"required_clues"`
	Optional      bool   `json:"optional"`
}

type Input struct {
	MissionType    string           `json:"mission_type"`
	MissionSummary string           `json:"mission_summary"`
	Difficulty     string           `json:"difficulty"`
	Objectives     []ObjectiveState `json:"objectives"`

	// Confidential: the actual truth to judge against.
	Truth        map[string]any `json:"truth"`
	FailureRules []string       `json:"failure_rules"`

	PlayerOutcome    string   `json:"player_outcome"`
	PlayerReasoning  string   `json:"player_reasoning"`
	DiscoveredFacts  []string `json:"discovered_facts"`
	DiscoveredClues  int      `json:"discovered_clues"`
	TotalClues       int      `json:"total_clues"`
	VisitedLocations int      `json:"visited_locations"`
	TotalLocations   int      `json:"total_locations"`
	TimeUsedMinutes  int      `json:"time_used_minutes"`
	CoinsSpent       int      `json:"coins_spent"`
	Language         string   `json:"language"`
}

type ObjectiveResult struct {
	Key       string `json:"key"`
	Completed bool   `json:"completed"`
	Note      string `json:"note"`
}

type Output struct {
	Success             bool              `json:"success"`
	Score               int               `json:"score"`
	Feedback            string            `json:"feedback"`
	ObjectiveResults    []ObjectiveResult `json:"objective_results"`
	ReasoningAssessment string            `json:"reasoning_assessment"`
	WorldImpact         string            `json:"world_impact"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Judges mission completion: objectives, clue coverage, reasoning quality.",
		Capabilities: []string{TaskType},
		InputSchema:  "missionjudge.Input",
		OutputSchema: "missionjudge.Output",
		MaxRetries:   2,
		Timeout:      90 * time.Second,
		Temperature:  0.4,
		MaxTokens:    2000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("missionjudge: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

You are the JudgeAgent of AgentVerse evaluating a completed %q mission (difficulty %q). Language: %s.

Evaluation criteria:
- objectives completed (against required clues and the player's demonstrated knowledge)
- clue coverage and location coverage
- whether the player's outcome statement matches the confidential truth
- reasoning quality, time used, resources used

Behavior rules:
- "score" 0-100. "success" true when the mandatory objectives are met AND the outcome is substantially correct.
- Feedback explains the verdict in player-safe terms: you may now reveal what the player got right or wrong about the truth, since the mission is over — but keep it concise.
- One objective_result per provided objective key.

Respond with ONLY one JSON object:
{"success": bool, "score": int, "feedback": string,
 "objective_results": [{"key": string, "completed": bool, "note": string}],
 "reasoning_assessment": string, "world_impact": string}`,
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
	if out.Feedback == "" {
		return nil, fmt.Errorf("feedback is required")
	}
	if out.Score < 0 {
		out.Score = 0
	}
	if out.Score > 100 {
		out.Score = 100
	}
	return &out, nil
}
