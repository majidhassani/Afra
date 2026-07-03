// Package timelineagent implements the TimelineAgent: during case
// generation it produces the private real timeline from the Case Bible.
package timelineagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "timeline"
	TaskType = "timeline_generation"
)

type SuspectRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Input struct {
	CaseSummary string         `json:"case_summary"`
	Motive      string         `json:"motive"`
	Truth       map[string]any `json:"truth"`
	Suspects    []SuspectRef   `json:"suspects"`
	CulpritKey  string         `json:"culprit_key"`
	FalseLeads  []string       `json:"false_leads"`
	Language    string         `json:"language"`
}

type GenEvent struct {
	Key             string    `json:"key"`
	OccurredAt      time.Time `json:"occurred_at"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	ParticipantKeys []string  `json:"participant_keys"`
	Revealed        bool      `json:"revealed"`
}

type Output struct {
	Events []GenEvent `json:"events"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Builds the private real timeline of the case from the Case Bible.",
		Capabilities: []string{"timeline_generation"},
		AllowedTools: []string{},
		InputSchema:  "timelineagent.Input",
		OutputSchema: "timelineagent.Output",
		MaxRetries:   2,
		Timeout:      90 * time.Second,
		Temperature:  0.6,
		MaxTokens:    4000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("timelineagent: unexpected input type %T", task.Input)
	}
	language := normalizeLanguage(in.Language)
	system := fmt.Sprintf(`TASK_TYPE: %s
OUTPUT_LANGUAGE: %s
%s

You are the TimelineAgent. Build the REAL timeline of what actually happened in this case, consistent with the confidential truth below.

Requirements:
- Generate all player-facing event titles and descriptions in %s. Keep JSON keys, timestamps, suspect keys, and booleans exactly as specified.
- 4 to 10 events in chronological order, each with a unique "key" (t1, t2, ...).
- "occurred_at" must be RFC3339 timestamps on plausible dates.
- Events involving the crime itself must have "revealed": false. Publicly known framing events (body found, public arguments) may be "revealed": true. At least one event revealed, at least two hidden.
- "participant_keys" reference suspect keys.

Respond with ONLY one JSON object:
{"events": [{"key","occurred_at","title","description","participant_keys","revealed"}]}`,
		TaskType, language, runtime.SecurityPreamble, languageName(language))

	userJSON, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: "CONFIDENTIAL CONTEXT:\n" + string(userJSON)},
	}, nil
}

func normalizeLanguage(language string) string {
	if language == "fa" {
		return "fa"
	}
	return "en"
}

func languageName(language string) string {
	if language == "fa" {
		return "Persian (Farsi)"
	}
	return "English"
}

func (a *Agent) Parse(raw []byte) (any, error) {
	var out Output
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if len(out.Events) < 3 {
		return nil, fmt.Errorf("expected at least 3 timeline events, got %d", len(out.Events))
	}
	revealed := 0
	keys := map[string]bool{}
	for _, e := range out.Events {
		if e.Key == "" || e.Title == "" {
			return nil, fmt.Errorf("every event needs a key and a title")
		}
		if keys[e.Key] {
			return nil, fmt.Errorf("duplicate event key %q", e.Key)
		}
		keys[e.Key] = true
		if e.OccurredAt.IsZero() {
			return nil, fmt.Errorf("event %q has no occurred_at timestamp", e.Key)
		}
		if e.Revealed {
			revealed++
		}
	}
	if revealed == 0 {
		return nil, fmt.Errorf("at least one event must start revealed")
	}
	return &out, nil
}
