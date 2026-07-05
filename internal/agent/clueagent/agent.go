// Package clueagent implements the ClueAgent. It handles three task types:
// generating and distributing clues across map locations, inspecting a clue
// (deeper paid analysis), and explaining a clue to the player without
// revealing hidden truth.
package clueagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name         = "clue"
	TaskGenerate = "clue_generation"
	TaskInspect  = "clue_inspection"
	TaskExplain  = "clue_explanation"
)

// --- generation ---

type LocationRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type CharacterRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type GenerateInput struct {
	MissionType string         `json:"mission_type"`
	Summary     string         `json:"summary"`
	Truth       map[string]any `json:"truth"` // confidential
	Locations   []LocationRef  `json:"locations"`
	Characters  []CharacterRef `json:"characters"`
	Objectives  []string       `json:"objectives"`
	Language    string         `json:"language"`
}

type Clue struct {
	Key                     string         `json:"key"`
	LocationKey             string         `json:"location_key"`
	Title                   string         `json:"title"`
	Type                    string         `json:"type"`
	ShortDescription        string         `json:"short_description"`
	DetailedDescription     string         `json:"detailed_description"`
	VisualDescription       string         `json:"visual_description"`
	AvatarOrThumbnailPrompt string         `json:"avatar_or_thumbnail_prompt"`
	Discovered              bool           `json:"discovered"`
	Reliability             int            `json:"reliability"`
	Importance              string         `json:"importance"`
	RelatedCharacterKeys    []string       `json:"related_character_keys"`
	PublicData              map[string]any `json:"public_data"`
	InternalTruth           map[string]any `json:"internal_truth"`
}

type GenerateOutput struct {
	Clues []Clue `json:"clues"`
}

// --- inspection ---

type InspectInput struct {
	MissionSummary  string         `json:"mission_summary"`
	Title           string         `json:"title"`
	Type            string         `json:"type"`
	ShortDesc       string         `json:"short_description"`
	DetailedDesc    string         `json:"detailed_description"`
	PublicData      map[string]any `json:"public_data"`
	InternalTruth   map[string]any `json:"internal_truth"` // confidential
	Question        string         `json:"question"`
	DiscoveredFacts []string       `json:"discovered_facts"`
	Language        string         `json:"language"`
}

type InspectOutput struct {
	Analysis         string   `json:"analysis"`
	NewFacts         []string `json:"new_facts"`
	ReliabilityDelta int      `json:"reliability_delta"`
}

// --- explanation ---

type ExplainInput struct {
	MissionSummary  string         `json:"mission_summary"`
	Title           string         `json:"title"`
	Type            string         `json:"type"`
	ShortDesc       string         `json:"short_description"`
	DetailedDesc    string         `json:"detailed_description"`
	PublicData      map[string]any `json:"public_data"`
	DiscoveredClues []string       `json:"discovered_clues"`
	Objectives      []string       `json:"objectives"`
	Language        string         `json:"language"`
}

type ExplainOutput struct {
	Explanation string   `json:"explanation"`
	NextSteps   []string `json:"next_steps"`
	CompareWith []string `json:"compare_with"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Creates, inspects, and explains clues without revealing hidden truth.",
		Capabilities: []string{TaskGenerate, TaskInspect, TaskExplain},
		InputSchema:  "clueagent.*Input",
		OutputSchema: "clueagent.*Output",
		MaxRetries:   2,
		Timeout:      90 * time.Second,
		Temperature:  0.7,
		MaxTokens:    3500,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	switch task.Type {
	case TaskGenerate:
		in, ok := task.Input.(GenerateInput)
		if !ok {
			return nil, fmt.Errorf("clueagent: unexpected input type %T", task.Input)
		}
		return promptFor(TaskGenerate, in, fmt.Sprintf(`You are the ClueAgent of AgentVerse. Create 6-10 clues for this %q mission and distribute them across the provided locations. Language: %s.

Rules:
- Every clue's "location_key" must be one of the provided location keys; spread clues over at least 3 locations.
- 2-3 clues start "discovered": true; the rest are found through play.
- Descriptions are rich enough to later generate evidence-card images: "visual_description" and "avatar_or_thumbnail_prompt" describe visible appearance ONLY — no hidden truth, no solution.
- "internal_truth" (backend only) holds the true meaning, hidden relation, and false-lead flag; include 1-2 false leads.
- "type" is a short kind such as footprint, document, gps_signal, radio_log, witness_statement, photo, camera_footage, vehicle_trace, personal_item.
- "importance" is low|medium|high. "reliability" 0-100.

Respond with ONLY one JSON object:
{"clues": [{"key": string, "location_key": string, "title": string, "type": string,
  "short_description": string, "detailed_description": string, "visual_description": string,
  "avatar_or_thumbnail_prompt": string, "discovered": bool, "reliability": int, "importance": string,
  "related_character_keys": [string], "public_data": object, "internal_truth": object}]}`,
			in.MissionType, in.Language))
	case TaskInspect:
		in, ok := task.Input.(InspectInput)
		if !ok {
			return nil, fmt.Errorf("clueagent: unexpected input type %T", task.Input)
		}
		return promptFor(TaskInspect, in, fmt.Sprintf(`You are the ClueAgent of AgentVerse performing a deep inspection of a discovered clue. Language: %s.

Rules:
- Use the confidential internal truth to steer the analysis, but never state it outright — surface at most one careful, partial new insight.
- "new_facts": at most 2 short player-safe facts earned by this inspection (empty if none).
- "reliability_delta" between -10 and 10.

Respond with ONLY one JSON object:
{"analysis": string, "new_facts": [string], "reliability_delta": int}`, in.Language))
	case TaskExplain:
		in, ok := task.Input.(ExplainInput)
		if !ok {
			return nil, fmt.Errorf("clueagent: unexpected input type %T", task.Input)
		}
		return promptFor(TaskExplain, in, fmt.Sprintf(`You are the in-mission AI assistant of AgentVerse explaining a discovered clue to the player. Language: %s.

Rules:
- You only know public clue data and what the player has discovered. Explain what the clue appears to be, why it might matter, what to compare it with, and possible next steps.
- Do NOT invent hidden truth and do NOT solve the mission.

Respond with ONLY one JSON object:
{"explanation": string, "next_steps": [string], "compare_with": [string]}`, in.Language))
	default:
		return nil, fmt.Errorf("clueagent: unknown task type %q", task.Type)
	}
}

func promptFor(taskType string, input any, body string) ([]llm.Message, error) {
	ctxJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	system := fmt.Sprintf("TASK_TYPE: %s\n%s\n\n%s", taskType, runtime.MissionSecurityPreamble, body)
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: "CONFIDENTIAL CONTEXT:\n" + string(ctxJSON)},
	}, nil
}

// Parse handles the generation schema (default when ParseTask is bypassed).
func (a *Agent) Parse(raw []byte) (any, error) {
	return a.ParseTask(TaskGenerate, raw)
}

func (a *Agent) ParseTask(taskType string, raw []byte) (any, error) {
	switch taskType {
	case TaskGenerate:
		var out GenerateOutput
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		if len(out.Clues) < 3 {
			return nil, fmt.Errorf("expected at least 3 clues, got %d", len(out.Clues))
		}
		discovered := 0
		for i := range out.Clues {
			c := &out.Clues[i]
			if c.Key == "" || c.Title == "" || c.LocationKey == "" {
				return nil, fmt.Errorf("clue %d missing key, title, or location_key", i)
			}
			if c.Reliability <= 0 || c.Reliability > 100 {
				c.Reliability = 50
			}
			switch c.Importance {
			case "low", "medium", "high":
			default:
				c.Importance = "medium"
			}
			if c.Discovered {
				discovered++
			}
		}
		if discovered == 0 {
			out.Clues[0].Discovered = true
		}
		return &out, nil
	case TaskInspect:
		var out InspectOutput
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		if out.Analysis == "" {
			return nil, fmt.Errorf("analysis is required")
		}
		if out.ReliabilityDelta < -10 || out.ReliabilityDelta > 10 {
			out.ReliabilityDelta = 0
		}
		if len(out.NewFacts) > 2 {
			out.NewFacts = out.NewFacts[:2]
		}
		return &out, nil
	case TaskExplain:
		var out ExplainOutput
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		if out.Explanation == "" {
			return nil, fmt.Errorf("explanation is required")
		}
		return &out, nil
	default:
		return nil, fmt.Errorf("clueagent: unknown task type %q", taskType)
	}
}
