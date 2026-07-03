// Package evidenceagent implements the EvidenceAgent: forensic analysis of
// a discovered piece of evidence. It sees the internal truth but must only
// surface player-safe insights.
package evidenceagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "evidence"
	TaskType = "evidence_inspection"
)

type Input struct {
	CaseSummary     string         `json:"case_summary"`
	EvidenceTitle   string         `json:"evidence_title"`
	EvidenceType    string         `json:"evidence_type"`
	Description     string         `json:"description"`
	PublicData      map[string]any `json:"public_data"`
	InternalTruth   map[string]any `json:"internal_truth"`
	Question        string         `json:"question"`
	DiscoveredFacts []string       `json:"discovered_facts"`
	InspectionCount int            `json:"inspection_count"`
}

type Output struct {
	Analysis         string   `json:"analysis"`
	NewFacts         []string `json:"new_facts"`
	ReliabilityDelta int      `json:"reliability_delta"`
	// RevealRelated signals that this analysis justifies revealing the
	// locations/timeline events linked to the evidence. The service decides
	// what that concretely means.
	RevealRelated bool `json:"reveal_related"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Analyzes discovered evidence and yields player-safe forensic insights.",
		Capabilities: []string{"evidence_inspection"},
		AllowedTools: []string{},
		InputSchema:  "evidenceagent.Input",
		OutputSchema: "evidenceagent.Output",
		MaxRetries:   2,
		Timeout:      60 * time.Second,
		Temperature:  0.5,
		MaxTokens:    2000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("evidenceagent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

You are the EvidenceAgent, a forensic analyst. The detective inspects a piece of evidence and may ask a question about it.

Rules:
- Base your analysis on the public data; use the confidential internal truth ONLY to decide which player-safe insight to surface next.
- Never state the internal truth outright and never name the culprit. Give the detective a meaningful but partial step forward.
- Repeated inspections (inspection_count) should go deeper but with diminishing returns.
- "new_facts": at most 3 short player-safe facts unlocked by this analysis.
- "reliability_delta" between -20 and 20: how this analysis changes confidence in the evidence.
- "reveal_related": true if this analysis would naturally point the detective to the places/timeline moments linked to this evidence.

Respond with ONLY one JSON object:
{"analysis": string, "new_facts": [string], "reliability_delta": int, "reveal_related": bool}`,
		TaskType, runtime.SecurityPreamble)

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
	if out.Analysis == "" {
		return nil, fmt.Errorf("analysis is required")
	}
	if out.ReliabilityDelta < -20 || out.ReliabilityDelta > 20 {
		return nil, fmt.Errorf("reliability_delta out of range [-20,20]: %d", out.ReliabilityDelta)
	}
	if len(out.NewFacts) > 3 {
		out.NewFacts = out.NewFacts[:3]
	}
	return &out, nil
}
