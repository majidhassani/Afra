// Package locationagent implements location actions: when the player
// searches or inspects a map location, this agent narrates the scene and
// decides — as a structured intention — which undiscovered clues at that
// location the action plausibly uncovers.
package locationagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "location"
	TaskType = "location_action"
)

type ClueRef struct {
	Title string `json:"title"`
	Type  string `json:"type"`
}

type Input struct {
	MissionSummary string `json:"mission_summary"`
	MissionTime    string `json:"mission_time"`
	LocationName   string `json:"location_name"`
	LocationType   string `json:"location_type"`
	Description    string `json:"description"`
	RiskLevel      int    `json:"risk_level"`
	Action         string `json:"action"`

	// Confidential: what is actually hidden here.
	UndiscoveredClues []ClueRef `json:"undiscovered_clues"`

	DiscoveredFacts []string `json:"discovered_facts"`
	Language        string   `json:"language"`
}

type Output struct {
	Narrative          string   `json:"narrative"`
	DiscoverClueTitles []string `json:"discover_clue_titles"`
	NewFacts           []string `json:"new_facts"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Executes location actions, narrating the scene and unlocking earned clues.",
		Capabilities: []string{TaskType},
		InputSchema:  "locationagent.Input",
		OutputSchema: "locationagent.Output",
		MaxRetries:   2,
		Timeout:      60 * time.Second,
		Temperature:  0.7,
		MaxTokens:    1500,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("locationagent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

%s

You are the field narrator of AgentVerse. The player performs action %q at %q.

Behavior rules:
- Narrate the action's outcome in 2-4 atmospheric, player-safe sentences consistent with the location description and risk level.
- "discover_clue_titles": subset of the provided undiscovered clue titles that this action plausibly uncovers (0-2). A thorough search may find one or two; a casual look may find none. Titles must match EXACTLY.
- Never mention clues that are not discovered by this action. Never reveal hidden truth.
- "new_facts": at most 2 short player-safe facts learned from the scene.

Respond with ONLY one JSON object:
{"narrative": string, "discover_clue_titles": [string], "new_facts": [string]}`,
		TaskType, runtime.MissionSecurityPreamble, runtime.LanguageDirective(in.Language), in.Action, in.LocationName)

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
	if out.Narrative == "" {
		return nil, fmt.Errorf("narrative is required")
	}
	if len(out.DiscoverClueTitles) > 2 {
		out.DiscoverClueTitles = out.DiscoverClueTitles[:2]
	}
	if len(out.NewFacts) > 2 {
		out.NewFacts = out.NewFacts[:2]
	}
	return &out, nil
}
