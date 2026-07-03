// Package directoragent implements the DirectorAgent: it paces the
// investigation, nudging a stuck detective with player-safe hints.
package directoragent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "director"
	TaskType = "director_hint"
)

type Input struct {
	CaseSummary        string   `json:"case_summary"`
	DiscoveredFacts    []string `json:"discovered_facts"`
	DiscoveredEvidence []string `json:"discovered_evidence"`
	InterrogationCount int      `json:"interrogation_count"`
	FailedAttempts     int      `json:"failed_attempts"`
}

type Output struct {
	Hint string `json:"hint"`
	Tone string `json:"tone"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Paces the investigation and produces player-safe hints.",
		Capabilities: []string{"director_hint"},
		AllowedTools: []string{},
		InputSchema:  "directoragent.Input",
		OutputSchema: "directoragent.Output",
		MaxRetries:   2,
		Timeout:      60 * time.Second,
		Temperature:  0.7,
		MaxTokens:    1000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("directoragent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

You are the DirectorAgent, the invisible game master pacing an investigation. Based on what the detective has discovered so far, produce ONE short hint that nudges them toward their next productive step. The hint must only build on already-discovered material — never leak hidden facts, hidden evidence, or the culprit.

Respond with ONLY one JSON object:
{"hint": string, "tone": string}`,
		TaskType, runtime.SecurityPreamble)

	ctxJSON, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: string(ctxJSON)},
	}, nil
}

func (a *Agent) Parse(raw []byte) (any, error) {
	var out Output
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if out.Hint == "" {
		return nil, fmt.Errorf("hint is required")
	}
	return &out, nil
}
