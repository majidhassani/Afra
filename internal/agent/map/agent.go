// Package mapagent implements the MapAgent: during case generation it
// produces case locations and links them to evidence and suspects.
package mapagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "map"
	TaskType = "map_generation"
)

type EvidenceRef struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	LocationKey string `json:"location_key"`
}

type Input struct {
	CaseSummary string        `json:"case_summary"`
	Evidence    []EvidenceRef `json:"evidence"`
	SuspectKeys []string      `json:"suspect_keys"`
	Language    string        `json:"language"`
}

type GenLocation struct {
	Key                 string   `json:"key"`
	Name                string   `json:"name"`
	Type                string   `json:"type"`
	Latitude            float64  `json:"latitude"`
	Longitude           float64  `json:"longitude"`
	Description         string   `json:"description"`
	Discovered          bool     `json:"discovered"`
	RelatedEvidenceKeys []string `json:"related_evidence_keys"`
	RelatedSuspectKeys  []string `json:"related_suspect_keys"`
}

type Output struct {
	Locations []GenLocation `json:"locations"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Creates the case map: locations linked to evidence and suspects.",
		Capabilities: []string{"map_generation"},
		AllowedTools: []string{},
		InputSchema:  "mapagent.Input",
		OutputSchema: "mapagent.Output",
		MaxRetries:   2,
		Timeout:      90 * time.Second,
		Temperature:  0.6,
		MaxTokens:    3000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("mapagent: unexpected input type %T", task.Input)
	}
	language := normalizeLanguage(in.Language)
	system := fmt.Sprintf(`TASK_TYPE: %s
OUTPUT_LANGUAGE: %s
%s

You are the MapAgent. Create the locations for this case. Every "location_key" referenced by the evidence list MUST exist in your output. Locations tied to already-discovered evidence should be "discovered": true; others hidden.

Generate all player-facing location names and descriptions in %s. Keep JSON keys, location/evidence/suspect keys, and booleans exactly as specified.
Use plausible nearby latitude/longitude coordinates (same city).

Respond with ONLY one JSON object:
{"locations": [{"key","name","type","latitude","longitude","description","discovered","related_evidence_keys","related_suspect_keys"}]}`,
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
	if len(out.Locations) == 0 {
		return nil, fmt.Errorf("expected at least one location")
	}
	keys := map[string]bool{}
	for _, l := range out.Locations {
		if l.Key == "" || l.Name == "" {
			return nil, fmt.Errorf("every location needs a key and a name")
		}
		if keys[l.Key] {
			return nil, fmt.Errorf("duplicate location key %q", l.Key)
		}
		keys[l.Key] = true
		if l.Latitude < -90 || l.Latitude > 90 || l.Longitude < -180 || l.Longitude > 180 {
			return nil, fmt.Errorf("location %q has invalid coordinates", l.Key)
		}
	}
	return &out, nil
}
