// Package casegenerator implements the CaseGeneratorAgent: it produces the
// core Case Bible, suspects, and evidence as a structured intention. It
// never persists anything itself.
package casegenerator

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/evidence"
	"casemind/internal/llm"
)

const (
	Name     = "case_generator"
	TaskType = "case_generation"
)

type Input struct {
	CaseType   string `json:"case_type"`
	Difficulty string `json:"difficulty"`
	Language   string `json:"language"`
}

type GenSuspect struct {
	Key              string         `json:"key"`
	Name             string         `json:"name"`
	Age              int            `json:"age"`
	Job              string         `json:"job"`
	RelationToVictim string         `json:"relation_to_victim"`
	PublicProfile    map[string]any `json:"public_profile"`
	Personality      map[string]any `json:"personality"`
	KnownFacts       []string       `json:"known_facts"`
	IsCulprit        bool           `json:"is_culprit"`
	Secrets          []string       `json:"secrets"`
	LieProfile       map[string]any `json:"lie_profile"`
}

type GenEvidence struct {
	Key                string         `json:"key"`
	Title              string         `json:"title"`
	Type               string         `json:"type"`
	Description        string         `json:"description"`
	PublicData         map[string]any `json:"public_data"`
	InternalTruth      map[string]any `json:"internal_truth"`
	Discovered         bool           `json:"discovered"`
	Reliability        int            `json:"reliability"`
	RelatedSuspectKeys []string       `json:"related_suspect_keys"`
	LocationKey        string         `json:"location_key"`
}

type Output struct {
	Title       string         `json:"title"`
	Summary     string         `json:"summary"`
	Motive      string         `json:"motive"`
	Truth       map[string]any `json:"truth"`
	HiddenFacts []string       `json:"hidden_facts"`
	FalseLeads  []string       `json:"false_leads"`
	Suspects    []GenSuspect   `json:"suspects"`
	Evidence    []GenEvidence  `json:"evidence"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Generates the private Case Bible, suspects, and evidence for a new case.",
		Capabilities: []string{"case_generation"},
		AllowedTools: []string{},
		InputSchema:  "casegenerator.Input",
		OutputSchema: "casegenerator.Output",
		MaxRetries:   2,
		Timeout:      120 * time.Second,
		Temperature:  0.9,
		MaxTokens:    8000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("casegenerator: unexpected input type %T", task.Input)
	}
	language := normalizeLanguage(in.Language)
	system := fmt.Sprintf(`TASK_TYPE: %s
CASE_TYPE: %s
OUTPUT_LANGUAGE: %s
%s

You are the CaseGeneratorAgent for a detective game. Design a complete, internally consistent %s case at %s difficulty.

Requirements:
- Generate all player-facing narrative text in %s. Keep JSON keys, enum values, suspect/evidence/location keys, timestamps, and booleans exactly as specified.
- Fields declared as object MUST be JSON objects, never strings. For example: "personality": {"temperament":"guarded","traits":["observant"]}.
- Fields declared as [string] MUST be JSON arrays of strings, never comma-separated text.
- 3 to 6 suspects, EXACTLY ONE with "is_culprit": true.
- 4 to 8 pieces of evidence; 2 or 3 with "discovered": true (the initial scene evidence), the rest hidden.
- Every suspect gets a "key" (s1, s2, ...) and every evidence a "key" (e1, e2, ...) plus a "location_key" (l1, l2, ...).
- Evidence "public_data" is what a detective sees; "internal_truth" is what it really proves. They must differ meaningfully.
- Include misleading "false_leads" that implicate innocent suspects.
- Harder difficulties: more suspects, subtler lies, less initially discovered evidence.

Respond with ONLY one JSON object:
{"title": string, "summary": string, "motive": string, "truth": object, "hidden_facts": [string], "false_leads": [string],
 "suspects": [{"key","name","age","job","relation_to_victim","public_profile","personality","known_facts","is_culprit","secrets","lie_profile"}],
 "evidence": [{"key","title","type","description","public_data","internal_truth","discovered","reliability","related_suspect_keys","location_key"}]}

Valid evidence types: %v`,
		TaskType, in.CaseType, language, runtime.SecurityPreamble, in.CaseType, in.Difficulty, languageName(language), evidence.ValidTypes)

	userJSON, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: string(userJSON)},
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
	raw, err := normalizeShape(raw)
	if err != nil {
		return nil, err
	}
	var out Output
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if out.Title == "" || out.Summary == "" || out.Motive == "" {
		return nil, fmt.Errorf("title, summary, and motive are required")
	}
	if len(out.Suspects) < 3 || len(out.Suspects) > 8 {
		return nil, fmt.Errorf("expected 3-8 suspects, got %d", len(out.Suspects))
	}
	if len(out.Evidence) < 3 {
		return nil, fmt.Errorf("expected at least 3 pieces of evidence, got %d", len(out.Evidence))
	}
	validType := map[string]bool{}
	for _, t := range evidence.ValidTypes {
		validType[t] = true
	}
	culprits := 0
	keys := map[string]bool{}
	for _, s := range out.Suspects {
		if s.Key == "" || s.Name == "" {
			return nil, fmt.Errorf("every suspect needs a key and a name")
		}
		if keys[s.Key] {
			return nil, fmt.Errorf("duplicate suspect key %q", s.Key)
		}
		keys[s.Key] = true
		if s.IsCulprit {
			culprits++
		}
	}
	if culprits != 1 {
		return nil, fmt.Errorf("expected exactly one culprit, got %d", culprits)
	}
	discovered := 0
	for _, e := range out.Evidence {
		if e.Key == "" || e.Title == "" {
			return nil, fmt.Errorf("every evidence needs a key and a title")
		}
		if !validType[e.Type] {
			return nil, fmt.Errorf("invalid evidence type %q", e.Type)
		}
		for _, sk := range e.RelatedSuspectKeys {
			if !keys[sk] {
				return nil, fmt.Errorf("evidence %q references unknown suspect key %q", e.Key, sk)
			}
		}
		if e.Discovered {
			discovered++
		}
	}
	if discovered == 0 {
		return nil, fmt.Errorf("at least one piece of evidence must start discovered")
	}
	return &out, nil
}

func normalizeShape(raw []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	normalizeObjectField(doc, "truth")
	normalizeStringSliceField(doc, "hidden_facts")
	normalizeStringSliceField(doc, "false_leads")

	normalizeObjectList(doc, "suspects", func(item map[string]any) {
		normalizeObjectField(item, "public_profile")
		normalizeObjectField(item, "personality")
		normalizeObjectField(item, "lie_profile")
		normalizeStringSliceField(item, "known_facts")
		normalizeStringSliceField(item, "secrets")
	})

	normalizeObjectList(doc, "evidence", func(item map[string]any) {
		normalizeObjectField(item, "public_data")
		normalizeObjectField(item, "internal_truth")
		normalizeStringSliceField(item, "related_suspect_keys")
	})

	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("normalize JSON: %w", err)
	}
	return out, nil
}

func normalizeObjectList(doc map[string]any, key string, normalize func(map[string]any)) {
	items, ok := doc[key].([]any)
	if !ok {
		return
	}
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			normalize(m)
		}
	}
}

func normalizeObjectField(doc map[string]any, key string) {
	switch v := doc[key].(type) {
	case nil:
		doc[key] = map[string]any{}
	case string:
		if v == "" {
			doc[key] = map[string]any{}
			return
		}
		doc[key] = map[string]any{"description": v}
	case []any:
		doc[key] = map[string]any{"items": v}
	}
}

func normalizeStringSliceField(doc map[string]any, key string) {
	switch v := doc[key].(type) {
	case nil:
		doc[key] = []string{}
	case string:
		if v == "" {
			doc[key] = []string{}
			return
		}
		doc[key] = []string{v}
	case []any:
		items := make([]string, 0, len(v))
		for _, item := range v {
			switch s := item.(type) {
			case string:
				items = append(items, s)
			default:
				b, err := json.Marshal(s)
				if err == nil {
					items = append(items, string(b))
				}
			}
		}
		doc[key] = items
	}
}
