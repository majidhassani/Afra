package casegenerator

import (
	"context"
	"testing"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
	llmmock "casemind/internal/llm/mock"
)

// TestMockProviderOutputValidates guarantees the mock provider always
// produces output that survives the agent's schema validation — the game
// loop must work end to end without a real model.
func TestMockProviderOutputValidates(t *testing.T) {
	agent := New()
	provider := llmmock.New()

	for _, caseType := range []string{"murder", "kidnapping", "missing_person", "robbery", "fraud"} {
		messages, err := agent.Prompt(runtime.Task{
			Type:  TaskType,
			Input: Input{CaseType: caseType, Difficulty: "medium"},
		})
		if err != nil {
			t.Fatal(err)
		}
		resp, err := provider.Chat(context.Background(), llm.Request{Messages: messages})
		if err != nil {
			t.Fatal(err)
		}
		outAny, err := agent.Parse([]byte(runtime.ExtractJSON(resp.Content)))
		if err != nil {
			t.Fatalf("mock output invalid for %s: %v", caseType, err)
		}
		out := outAny.(*Output)
		if out.Title == "" {
			t.Fatalf("empty title for %s", caseType)
		}
		culprits := 0
		for _, s := range out.Suspects {
			if s.IsCulprit {
				culprits++
			}
		}
		if culprits != 1 {
			t.Fatalf("expected exactly one culprit for %s, got %d", caseType, culprits)
		}
	}
}

func TestParseRejectsInvalidOutput(t *testing.T) {
	agent := New()
	invalid := []string{
		`not json`,
		`{"title":"x","summary":"y","motive":"z","suspects":[],"evidence":[]}`,
		// Two culprits.
		`{"title":"x","summary":"y","motive":"z",
		  "suspects":[
		   {"key":"s1","name":"A","is_culprit":true},
		   {"key":"s2","name":"B","is_culprit":true},
		   {"key":"s3","name":"C","is_culprit":false}],
		  "evidence":[
		   {"key":"e1","title":"E1","type":"document","discovered":true},
		   {"key":"e2","title":"E2","type":"document"},
		   {"key":"e3","title":"E3","type":"document"}]}`,
		// Invalid evidence type.
		`{"title":"x","summary":"y","motive":"z",
		  "suspects":[
		   {"key":"s1","name":"A","is_culprit":true},
		   {"key":"s2","name":"B","is_culprit":false},
		   {"key":"s3","name":"C","is_culprit":false}],
		  "evidence":[
		   {"key":"e1","title":"E1","type":"hologram","discovered":true},
		   {"key":"e2","title":"E2","type":"document"},
		   {"key":"e3","title":"E3","type":"document"}]}`,
	}
	for i, raw := range invalid {
		if _, err := agent.Parse([]byte(raw)); err == nil {
			t.Errorf("case %d: expected parse error", i)
		}
	}
}

func TestParseNormalizesCommonLLMShapeDrift(t *testing.T) {
	agent := New()
	raw := `{"title":"x","summary":"y","motive":"z","truth":"culprit hid the ledger","hidden_facts":"hidden fact","false_leads":"false lead",
	  "suspects":[
	   {"key":"s1","name":"A","is_culprit":true,"public_profile":"quiet curator","personality":"guarded","known_facts":"worked late","secrets":"stole ledger","lie_profile":"deflects money questions"},
	   {"key":"s2","name":"B","is_culprit":false,"public_profile":{"role":"assistant"},"personality":{"temperament":"calm"},"known_facts":["signed out"],"secrets":[],"lie_profile":{}},
	   {"key":"s3","name":"C","is_culprit":false,"public_profile":"collector","personality":"proud","known_facts":["argued"],"secrets":"in debt","lie_profile":"exaggerates"}],
	  "evidence":[
	   {"key":"e1","title":"E1","type":"document","public_data":"visible note","internal_truth":"points to s1","discovered":true,"related_suspect_keys":"s1"},
	   {"key":"e2","title":"E2","type":"document","public_data":{},"internal_truth":{},"related_suspect_keys":[]},
	   {"key":"e3","title":"E3","type":"document","public_data":"receipt","internal_truth":"misdirection","related_suspect_keys":["s2"]}]}`

	outAny, err := agent.Parse([]byte(raw))
	if err != nil {
		t.Fatalf("expected normalized output to parse: %v", err)
	}
	out := outAny.(*Output)
	if got := out.Suspects[0].Personality["description"]; got != "guarded" {
		t.Fatalf("personality was not normalized, got %#v", got)
	}
	if len(out.Suspects[0].KnownFacts) != 1 || out.Suspects[0].KnownFacts[0] != "worked late" {
		t.Fatalf("known_facts was not normalized: %#v", out.Suspects[0].KnownFacts)
	}
	if got := out.Evidence[0].PublicData["description"]; got != "visible note" {
		t.Fatalf("public_data was not normalized, got %#v", got)
	}
	if len(out.Evidence[0].RelatedSuspectKeys) != 1 || out.Evidence[0].RelatedSuspectKeys[0] != "s1" {
		t.Fatalf("related_suspect_keys was not normalized: %#v", out.Evidence[0].RelatedSuspectKeys)
	}
}
