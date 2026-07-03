// Package characteragent implements the CharacterAgent: it creates the
// mission NPCs — public identity, private state, dialogue style, and
// avatar/thumbnail prompts for later image generation.
package characteragent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "character"
	TaskType = "character_generation"
)

type LocationRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Input struct {
	MissionType string         `json:"mission_type"`
	Summary     string         `json:"summary"`
	Truth       map[string]any `json:"truth"` // confidential
	Locations   []LocationRef  `json:"locations"`
	Language    string         `json:"language"`
}

type Character struct {
	Key             string         `json:"key"`
	Name            string         `json:"name"`
	Role            string         `json:"role"`
	Category        string         `json:"category"` // guide | field | antagonist | neutral
	Age             int            `json:"age"`
	PublicProfile   string         `json:"public_profile"`
	Personality     map[string]any `json:"personality"`
	LocationKey     string         `json:"location_key"`
	Mood            string         `json:"mood"`
	DialogueStyle   string         `json:"dialogue_style"`
	AvatarPrompt    string         `json:"avatar_prompt"`
	ThumbnailPrompt string         `json:"thumbnail_prompt"`
	VisualStyleTags []string       `json:"visual_style_tags"`
	PrivateState    map[string]any `json:"private_state"`
}

type Output struct {
	Characters []Character `json:"characters"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Creates mission NPCs with public identity, private state, and avatar prompts.",
		Capabilities: []string{TaskType},
		InputSchema:  "characteragent.Input",
		OutputSchema: "characteragent.Output",
		MaxRetries:   0,
		Timeout:      45 * time.Second,
		Temperature:  0.8,
		MaxTokens:    3500,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("characteragent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

You are the CharacterAgent of AgentVerse. Create 3-6 NPCs for this %q mission. Language: %s.

Rules:
- Exactly one character has category "guide": the always-available in-world AI/mission-control assistant (no location_key needed).
- Include at least one character whose "private_state" connects to the hidden truth (category "antagonist" or a "field"/"neutral" character hiding something).
- "public_profile" is 1-2 player-safe sentences. "private_state" holds hidden_knowledge, goals, and what they lie about — backend only.
- "avatar_prompt" and "thumbnail_prompt" describe visible appearance for image generation and must NOT hint at hidden truth.
- "location_key" must be one of the provided location keys (or "" for the guide).
- "personality" is a small object of trait scores 0-100 (e.g. openness, honesty, stress, patience).

Respond with ONLY one JSON object:
{"characters": [{"key": string, "name": string, "role": string, "category": "guide"|"field"|"antagonist"|"neutral",
  "age": int, "public_profile": string, "personality": object, "location_key": string, "mood": string,
  "dialogue_style": string, "avatar_prompt": string, "thumbnail_prompt": string,
  "visual_style_tags": [string], "private_state": object}]}`,
		TaskType, runtime.MissionSecurityPreamble, in.MissionType, in.Language)

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
	if len(out.Characters) < 2 {
		return nil, fmt.Errorf("expected at least 2 characters, got %d", len(out.Characters))
	}
	guides := 0
	for i := range out.Characters {
		c := &out.Characters[i]
		if c.Key == "" || c.Name == "" {
			return nil, fmt.Errorf("character %d missing key or name", i)
		}
		switch c.Category {
		case "guide":
			guides++
		case "field", "antagonist", "neutral":
		default:
			c.Category = "field"
		}
		if c.AvatarPrompt == "" {
			return nil, fmt.Errorf("character %s missing avatar_prompt", c.Key)
		}
		if c.Mood == "" {
			c.Mood = "neutral"
		}
		if c.PrivateState == nil {
			c.PrivateState = map[string]any{}
		}
	}
	if guides != 1 {
		return nil, fmt.Errorf("expected exactly one guide character, got %d", guides)
	}
	return &out, nil
}

// Fallback keeps mission generation playable when an external LLM returns
// empty or truncated JSON for the character stage. It uses only the task
// input and visible-safe prompt text; private state stays backend-only.
func (a *Agent) Fallback(task runtime.Task, _ []byte, _ error) (any, bool) {
	in, ok := task.Input.(Input)
	if !ok || task.Type != TaskType {
		return nil, false
	}
	loc := func(index int) string {
		if len(in.Locations) == 0 {
			return ""
		}
		if index >= len(in.Locations) {
			index = len(in.Locations) - 1
		}
		return in.Locations[index].Key
	}
	out := &Output{Characters: []Character{
		{
			Key:             "char_guide",
			Name:            "Mission Control",
			Role:            "Mission Control AI",
			Category:        "guide",
			Age:             0,
			PublicProfile:   "The operation's tactical assistant, always available on the mission channel.",
			Personality:     map[string]any{"openness": 90, "honesty": 95, "stress": 5, "patience": 95},
			LocationKey:     "",
			Mood:            "calm",
			DialogueStyle:   "concise, tactical, supportive",
			AvatarPrompt:    "futuristic mission control AI interface, calm tactical assistant, holographic face silhouette, dark command center UI, premium mobile game style",
			ThumbnailPrompt: "mission control AI avatar, holographic silhouette, dark UI card",
			VisualStyleTags: []string{"holographic", "tactical", "mobile-game"},
			PrivateState:    map[string]any{"notes": "fallback guide; knows only player-visible mission state"},
		},
		{
			Key:             "char_field_lead",
			Name:            "Field Lead",
			Role:            "Local Field Lead",
			Category:        "field",
			Age:             42,
			PublicProfile:   "An experienced local operator who knows the terrain and the people involved.",
			Personality:     map[string]any{"openness": 55, "honesty": 75, "stress": 45, "patience": 60},
			LocationKey:     loc(0),
			Mood:            "focused",
			DialogueStyle:   "practical, cautious, protective of the team",
			AvatarPrompt:    "realistic portrait of a focused field lead in rugged outdoor gear, weathered face, mission landscape background, cinematic mobile game character portrait",
			ThumbnailPrompt: "field lead portrait, rugged outdoor gear, realistic style, dark UI card",
			VisualStyleTags: []string{"realistic", "cinematic", "portrait"},
			PrivateState:    map[string]any{"hidden_knowledge": []string{"Has noticed inconsistencies in local reports but needs proof"}, "goals": []string{"protect the team", "resolve the mission"}},
		},
		{
			Key:             "char_specialist",
			Name:            "Mission Specialist",
			Role:            "Technical Specialist",
			Category:        "field",
			Age:             35,
			PublicProfile:   "A detail-oriented specialist who tracks evidence, signals, and field anomalies.",
			Personality:     map[string]any{"openness": 70, "honesty": 85, "stress": 35, "patience": 65},
			LocationKey:     loc(1),
			Mood:            "analytical",
			DialogueStyle:   "precise, evidence-driven, calm under pressure",
			AvatarPrompt:    "realistic portrait of a technical mission specialist with tablet and field equipment, focused expression, cinematic mobile game character portrait",
			ThumbnailPrompt: "technical specialist portrait, field equipment, realistic style, dark UI card",
			VisualStyleTags: []string{"realistic", "cinematic", "portrait"},
			PrivateState:    map[string]any{"hidden_knowledge": []string{"Has partial data that can connect later clues"}, "goals": []string{"verify the evidence trail"}},
		},
		{
			Key:             "char_local_contact",
			Name:            "Local Contact",
			Role:            "Local Contact",
			Category:        "neutral",
			Age:             50,
			PublicProfile:   "A well-connected local contact who hears rumors before officials do.",
			Personality:     map[string]any{"openness": 60, "honesty": 65, "stress": 20, "patience": 75},
			LocationKey:     loc(2),
			Mood:            "wary",
			DialogueStyle:   "observant, indirect, trades small details for trust",
			AvatarPrompt:    "realistic portrait of a wary local contact, practical clothing, local street or field background, cinematic mobile game character portrait",
			ThumbnailPrompt: "local contact portrait, practical clothing, realistic style, dark UI card",
			VisualStyleTags: []string{"realistic", "cinematic", "portrait"},
			PrivateState:    map[string]any{"hidden_knowledge": []string{"Knows a rumor that may become useful after the player finds supporting clues"}, "goals": []string{"avoid danger", "help without becoming exposed"}},
		},
	}}
	return out, true
}
