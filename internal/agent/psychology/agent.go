// Package psychologyagent implements the PsychologyAgent: it produces
// behavioral insights about a suspect after tense interrogations.
package psychologyagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "psychology"
	TaskType = "psychology_profile"
)

type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Input struct {
	SuspectName    string         `json:"suspect_name"`
	Personality    map[string]any `json:"personality"`
	StressLevel    int            `json:"stress_level"`
	TrustLevel     int            `json:"trust_level"`
	RecentMessages []Turn         `json:"recent_messages"`
}

type Output struct {
	Insight             string `json:"insight"`
	StressAssessment    string `json:"stress_assessment"`
	RecommendedApproach string `json:"recommended_approach"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Analyzes suspect behavior and suggests interrogation approaches.",
		Capabilities: []string{"psychology_profile"},
		AllowedTools: []string{},
		InputSchema:  "psychologyagent.Input",
		OutputSchema: "psychologyagent.Output",
		MaxRetries:   2,
		Timeout:      60 * time.Second,
		Temperature:  0.6,
		MaxTokens:    1500,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("psychologyagent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

You are the PsychologyAgent, a behavioral analyst observing an interrogation. Based ONLY on the observable conversation and stress/trust levels, produce a short insight for the detective. You do not know who the culprit is and must not speculate about guilt directly — describe behavior, not verdicts.

Respond with ONLY one JSON object:
{"insight": string, "stress_assessment": string, "recommended_approach": string}`,
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
	if out.Insight == "" {
		return nil, fmt.Errorf("insight is required")
	}
	return &out, nil
}
