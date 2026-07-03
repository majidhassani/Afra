// Package missionmap implements the mission MapAgent: it creates Google
// Maps-compatible mission locations (lat/lng markers, no custom map engine)
// with risk, status, visual prompts, and available actions.
package missionmap

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "mission_map"
	TaskType = "mission_map_generation"
)

type Input struct {
	MissionType string `json:"mission_type"`
	Summary     string `json:"summary"`
	Region      string `json:"region"`
	Language    string `json:"language"`
}

type Location struct {
	Key              string   `json:"key"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	Status           string   `json:"status"` // discovered | hidden | locked
	RiskLevel        int      `json:"risk_level"`
	Description      string   `json:"description"`
	VisualPrompt     string   `json:"visual_prompt"`
	AvailableActions []string `json:"available_actions"`
}

type Output struct {
	CenterLat float64    `json:"center_lat"`
	CenterLng float64    `json:"center_lng"`
	Zoom      int        `json:"zoom"`
	Locations []Location `json:"locations"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Creates Google Maps-compatible mission locations with markers, risk, and actions.",
		Capabilities: []string{TaskType},
		InputSchema:  "missionmap.Input",
		OutputSchema: "missionmap.Output",
		MaxRetries:   2,
		Timeout:      90 * time.Second,
		Temperature:  0.7,
		MaxTokens:    3000,
	}
}

var allowedActions = map[string]bool{
	"inspect_area": true, "talk_to_character": true, "ask_ai": true,
	"view_clues": true, "review_documents": true, "scan_environment": true,
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("missionmap: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
%s

You are the MapAgent of AgentVerse. Create 4-7 mission locations as Google Maps markers for this %q mission set in %q. Language: %s.

Rules:
- Real-world-plausible lat/lng values clustered within roughly 10km of a sensible center; "zoom" 11-14.
- 2-3 locations start "discovered"; the rest are "hidden" (revealed through play) or "locked".
- "risk_level" 0-100. "type" is a short kind like operations_base, river_crossing, camp, village, research_center.
- "visual_prompt" describes the visible scene for later image generation — it must NOT hint at the hidden solution.
- "available_actions" from: inspect_area, talk_to_character, ask_ai, view_clues, review_documents, scan_environment.

Respond with ONLY one JSON object:
{"center_lat": number, "center_lng": number, "zoom": int,
 "locations": [{"key": string, "name": string, "type": string, "latitude": number, "longitude": number,
   "status": "discovered"|"hidden"|"locked", "risk_level": int, "description": string,
   "visual_prompt": string, "available_actions": [string]}]}`,
		TaskType, runtime.MissionSecurityPreamble, in.MissionType, in.Region, in.Language)

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
	if len(out.Locations) < 2 {
		return nil, fmt.Errorf("expected at least 2 locations, got %d", len(out.Locations))
	}
	if out.Zoom < 3 || out.Zoom > 20 {
		out.Zoom = 12
	}
	discovered := 0
	for i := range out.Locations {
		l := &out.Locations[i]
		if l.Key == "" || l.Name == "" {
			return nil, fmt.Errorf("location %d missing key or name", i)
		}
		switch l.Status {
		case "discovered":
			discovered++
		case "hidden", "locked":
		default:
			l.Status = "hidden"
		}
		if l.RiskLevel < 0 || l.RiskLevel > 100 {
			l.RiskLevel = 20
		}
		actions := make([]string, 0, len(l.AvailableActions))
		for _, act := range l.AvailableActions {
			if allowedActions[act] {
				actions = append(actions, act)
			}
		}
		if len(actions) == 0 {
			actions = []string{"inspect_area", "ask_ai", "view_clues"}
		}
		l.AvailableActions = actions
	}
	if discovered == 0 {
		out.Locations[0].Status = "discovered"
	}
	return &out, nil
}
