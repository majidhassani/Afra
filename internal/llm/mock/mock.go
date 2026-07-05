// Package mock implements llm.Client with deterministic, schema-valid
// responses so the full game loop works without a real model. The agent
// prompts embed a "TASK_TYPE: <type>" marker which selects the canned
// response.
package mock

import (
	"context"
	"fmt"
	"strings"

	"casemind/internal/llm"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Name() string { return "mock" }

// Ping always succeeds: the mock needs no connectivity.
func (p *Provider) Ping(_ context.Context) error { return nil }

func (p *Provider) Chat(_ context.Context, req llm.Request) (*llm.Response, error) {
	all := strings.Builder{}
	for _, m := range req.Messages {
		all.WriteString(m.Content)
		all.WriteString("\n")
	}
	text := all.String()

	content := `{"error":"mock: unknown task type"}`
	switch taskType(text) {
	case "case_generation":
		content = caseGenerationJSON(caseType(text))
	case "timeline_generation":
		content = timelineJSON
	case "map_generation":
		content = mapJSON
	case "interrogation":
		content = interrogationJSON(text)
	case "evidence_inspection":
		content = evidenceJSON
	case "judge_verdict":
		content = judgeJSON(text)
	case "psychology_profile":
		content = psychologyJSON
	case "director_hint":
		content = directorJSON
	// --- AgentVerse mission task types ---
	case "mission_plan":
		content = missionPlanJSON(missionType(text))
	case "world_generation":
		content = worldJSON
	case "mission_map_generation":
		content = missionMapJSON
	case "character_generation":
		content = charactersJSON
	case "clue_generation":
		content = cluesJSON
	case "character_dialogue":
		content = dialogueJSON(text)
	case "ai_guidance", "location_ask":
		content = guidanceJSON(text)
	case "location_action":
		content = locationActionJSON(text)
	case "clue_inspection":
		content = clueInspectionJSON
	case "clue_explanation":
		content = clueExplanationJSON
	case "time_advance":
		content = timeAdvanceJSON
	case "mission_director":
		content = missionDirectorJSON
	case "mission_judgment":
		content = missionJudgeJSON(text)
	}

	return &llm.Response{
		Content: content,
		Usage:   llm.Usage{PromptTokens: len(text) / 4, CompletionTokens: len(content) / 4, TotalTokens: (len(text) + len(content)) / 4},
		Model:   "mock",
	}, nil
}

func taskType(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if t, ok := strings.CutPrefix(strings.TrimSpace(line), "TASK_TYPE: "); ok {
			return strings.TrimSpace(t)
		}
	}
	return ""
}

func caseType(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if t, ok := strings.CutPrefix(strings.TrimSpace(line), "CASE_TYPE: "); ok {
			return strings.TrimSpace(t)
		}
	}
	return "murder"
}

var caseTitles = map[string]string{
	"murder":         "The Gallery After Dark",
	"kidnapping":     "The Vanishing at Pier Nine",
	"missing_person": "The Empty Studio",
	"robbery":        "The Vault on Meridian Street",
	"fraud":          "The Phantom Ledger",
}

func caseGenerationJSON(ct string) string {
	title := caseTitles[ct]
	if title == "" {
		title = "The Gallery After Dark"
	}
	return fmt.Sprintf(`{
  "title": %q,
  "summary": "Gallery owner Victor Hale was found dead in his own exhibition hall the morning after a private showing. The room was locked from the inside, the night's sales ledger is missing, and four people had keys to the building.",
  "motive": "Elena Voss was about to be exposed by Victor for forging two paintings sold through the gallery. She confronted him after the showing; when he refused to stay quiet, she killed him and took the ledger that proved the forged sales.",
  "truth": {
    "narrative": "After the private showing ended at 22:30, Elena Voss stayed behind claiming to pack her materials. She confronted Victor about the forgery audit at 23:10 in the main hall. The argument escalated; she struck him with a bronze maquette from the pedestal, wiped it, and left through the service corridor at 23:40, taking the sales ledger. She re-locked the hall using the spare key from the restoration room and dropped it in the courtyard drain.",
    "murder_weapon": "bronze maquette from pedestal 4",
    "time_of_death": "23:10-23:25",
    "escape_route": "service corridor to the courtyard"
  },
  "hidden_facts": [
    "The main hall spare key is missing from the restoration room hook",
    "The courtyard drain contains a brass key wiped of prints",
    "Victor had scheduled a forgery audit for Monday morning",
    "The sales ledger for the last quarter is missing from the office"
  ],
  "false_leads": [
    "Marcus Reed argued loudly with Victor about money earlier that evening",
    "Dana Cole's coat was found near the scene with a wine stain that looks like blood",
    "A window latch in the office was broken weeks ago, suggesting an intruder"
  ],
  "suspects": [
    {
      "key": "s1",
      "name": "Marcus Reed",
      "age": 47,
      "job": "Gallery co-investor",
      "relation_to_victim": "Business partner",
      "public_profile": {"appearance": "Silver-haired, sharply dressed", "demeanor": "Impatient, defensive about money"},
      "personality": {"traits": ["blunt", "proud", "quick-tempered"], "tell": "taps his ring when cornered"},
      "known_facts": ["Argued with Victor about a delayed payout at 21:40", "Left the gallery at 22:15 by taxi"],
      "is_culprit": false,
      "secrets": ["He has been quietly moving gallery funds to cover personal debts"],
      "lie_profile": {"will_lie_about": ["the missing funds"], "style": "deflects with anger"}
    },
    {
      "key": "s2",
      "name": "Dana Cole",
      "age": 29,
      "job": "Gallery assistant",
      "relation_to_victim": "Employee",
      "public_profile": {"appearance": "Nervous, quick-spoken", "demeanor": "Eager to help, easily flustered"},
      "known_facts": ["Served wine during the showing", "Was the one who found the body at 08:05"],
      "personality": {"traits": ["anxious", "observant", "loyal"], "tell": "voice rises when stressed"},
      "is_culprit": false,
      "secrets": ["She saw Elena near the service corridor after closing but Elena begged her to stay quiet"],
      "lie_profile": {"will_lie_about": ["seeing Elena after hours"], "style": "omits rather than invents"}
    },
    {
      "key": "s3",
      "name": "Elena Voss",
      "age": 38,
      "job": "Art restorer",
      "relation_to_victim": "Contract restorer for the gallery",
      "public_profile": {"appearance": "Composed, precise", "demeanor": "Calm, professional, controlled"},
      "known_facts": ["Restored two of the paintings in the current exhibition", "Claims she left at 22:45"],
      "personality": {"traits": ["meticulous", "cold under pressure", "articulate"], "tell": "over-explains small details"},
      "is_culprit": true,
      "secrets": ["She forged two paintings sold through the gallery", "Victor discovered the forgeries and scheduled an audit", "She took the sales ledger after killing him"],
      "lie_profile": {"will_lie_about": ["when she left", "the forgery audit", "the ledger"], "style": "precise, rehearsed alternative timeline"}
    },
    {
      "key": "s4",
      "name": "Tomas Iker",
      "age": 55,
      "job": "Night security guard",
      "relation_to_victim": "Employee",
      "public_profile": {"appearance": "Weathered, heavy-set", "demeanor": "Terse but straightforward"},
      "known_facts": ["Made rounds at 23:00 and 01:00", "Reports the main hall door was locked at 01:00"],
      "personality": {"traits": ["routine-bound", "honest", "slow to volunteer"], "tell": "checks his logbook constantly"},
      "is_culprit": false,
      "secrets": ["He skipped the 23:00 round to watch a match in the break room"],
      "lie_profile": {"will_lie_about": ["the skipped round"], "style": "sticks to the logbook version"}
    }
  ],
  "evidence": [
    {
      "key": "e1",
      "title": "Crime scene report",
      "type": "document",
      "description": "Initial report on the state of the main exhibition hall and the victim.",
      "public_data": {"summary": "Victim found near pedestal 4, blunt force trauma, hall locked from inside, no forced entry"},
      "internal_truth": {"actual": "The bronze maquette on pedestal 4 is the murder weapon; it was wiped but repositioned slightly off its dust ring"},
      "discovered": true,
      "reliability": 90,
      "related_suspect_keys": [],
      "location_key": "l1"
    },
    {
      "key": "e2",
      "title": "Guest sign-out sheet",
      "type": "document",
      "description": "Sign-out times for everyone who attended the private showing.",
      "public_data": {"entries": ["Marcus Reed 22:15", "guests 22:20-22:30", "Dana Cole 22:50", "Elena Voss 22:45 (self-noted)"]},
      "internal_truth": {"actual": "Elena's sign-out time is self-written and false; she left after 23:40"},
      "discovered": true,
      "reliability": 60,
      "related_suspect_keys": ["s1", "s2", "s3"],
      "location_key": "l2"
    },
    {
      "key": "e3",
      "title": "Courtyard camera footage",
      "type": "camera_footage",
      "description": "Low-quality camera covering the courtyard behind the service corridor.",
      "public_data": {"summary": "A figure crosses the courtyard at 23:41; face not visible; pauses near the drain grate"},
      "internal_truth": {"actual": "The figure is Elena Voss dropping the spare key into the drain"},
      "discovered": false,
      "reliability": 75,
      "related_suspect_keys": ["s3"],
      "location_key": "l3"
    },
    {
      "key": "e4",
      "title": "Bronze maquette",
      "type": "weapon",
      "description": "Small bronze sculpture from pedestal 4, recently cleaned.",
      "public_data": {"observation": "Sits slightly off its dust ring; smells faintly of solvent"},
      "internal_truth": {"actual": "Murder weapon; traces of blood in the base seam despite wiping; solvent matches restoration supplies"},
      "discovered": false,
      "reliability": 85,
      "related_suspect_keys": ["s3"],
      "location_key": "l1"
    },
    {
      "key": "e5",
      "title": "Victor's desk calendar",
      "type": "document",
      "description": "Open on his office desk to next week.",
      "public_data": {"entry": "Monday 09:00 — 'audit w/ independent appraiser — QUIET'"},
      "internal_truth": {"actual": "The audit was to verify two suspected forgeries restored by Elena"},
      "discovered": false,
      "reliability": 80,
      "related_suspect_keys": ["s3"],
      "location_key": "l2"
    },
    {
      "key": "e6",
      "title": "Break room television log",
      "type": "phone_record",
      "description": "Smart TV viewing history from the staff break room.",
      "public_data": {"entry": "Streaming active 22:50-00:10"},
      "internal_truth": {"actual": "Confirms Tomas skipped his 23:00 round; nobody watched the corridors during the murder window"},
      "discovered": false,
      "reliability": 70,
      "related_suspect_keys": ["s4"],
      "location_key": "l4"
    }
  ]
}`, title)
}

const timelineJSON = `{
  "events": [
    {"key": "t1", "occurred_at": "2026-06-30T21:40:00Z", "title": "Argument over money", "description": "Marcus Reed and Victor Hale argue loudly about a delayed payout in the office.", "participant_keys": ["s1"], "revealed": true},
    {"key": "t2", "occurred_at": "2026-06-30T22:30:00Z", "title": "Private showing ends", "description": "Guests leave; staff begin closing the gallery.", "participant_keys": ["s2", "s3"], "revealed": true},
    {"key": "t3", "occurred_at": "2026-06-30T22:50:00Z", "title": "Dana signs out", "description": "Dana Cole signs out and leaves through the front entrance.", "participant_keys": ["s2"], "revealed": false},
    {"key": "t4", "occurred_at": "2026-06-30T23:10:00Z", "title": "Confrontation in the main hall", "description": "A confrontation takes place in the main exhibition hall.", "participant_keys": ["s3"], "revealed": false},
    {"key": "t5", "occurred_at": "2026-06-30T23:41:00Z", "title": "Figure crosses courtyard", "description": "An unidentified figure crosses the courtyard toward the drain grate.", "participant_keys": ["s3"], "revealed": false},
    {"key": "t6", "occurred_at": "2026-07-01T08:05:00Z", "title": "Body discovered", "description": "Dana Cole finds Victor Hale dead in the main hall and calls the police.", "participant_keys": ["s2"], "revealed": true}
  ]
}`

const mapJSON = `{
  "locations": [
    {"key": "l1", "name": "Main exhibition hall", "type": "crime_scene", "latitude": 52.3702, "longitude": 4.8952, "description": "The locked hall where the body was found.", "discovered": true, "related_evidence_keys": ["e1", "e4"], "related_suspect_keys": []},
    {"key": "l2", "name": "Gallery office", "type": "interior", "latitude": 52.3703, "longitude": 4.8954, "description": "Victor's office, where the ledger was kept.", "discovered": true, "related_evidence_keys": ["e2", "e5"], "related_suspect_keys": ["s1"]},
    {"key": "l3", "name": "Courtyard", "type": "exterior", "latitude": 52.3701, "longitude": 4.8949, "description": "Rear courtyard behind the service corridor.", "discovered": false, "related_evidence_keys": ["e3"], "related_suspect_keys": ["s3"]},
    {"key": "l4", "name": "Staff break room", "type": "interior", "latitude": 52.3704, "longitude": 4.8955, "description": "Break room used by gallery staff and security.", "discovered": false, "related_evidence_keys": ["e6"], "related_suspect_keys": ["s4"]}
  ]
}`

func interrogationJSON(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "injection_detected: true"):
		return `{"reply": "I don't know what you're playing at, detective, but I'm here to answer questions about that night — nothing else.", "emotion": "irritated", "stress_delta": 3, "trust_delta": -3, "unlocked_clues": [], "reveal_evidence_titles": [], "contradiction_detected": false}`
	case strings.Contains(lower, "ledger"):
		return `{"reply": "The ledger? I wouldn't know. Victor handled the sales book personally. If it's missing, perhaps you should ask whoever handled the money.", "emotion": "guarded", "stress_delta": 8, "trust_delta": -2, "unlocked_clues": ["The suspect deflected quickly when the ledger was mentioned"], "reveal_evidence_titles": ["Victor's desk calendar"], "contradiction_detected": false}`
	case strings.Contains(lower, "alibi") || strings.Contains(lower, "where were you"):
		return `{"reply": "I signed out at a quarter to eleven, as the sheet shows. I packed my materials, said goodnight, and went straight home. You can check the sign-out sheet yourself.", "emotion": "composed", "stress_delta": 4, "trust_delta": 1, "unlocked_clues": ["The suspect leans heavily on the sign-out sheet as an alibi"], "reveal_evidence_titles": ["Courtyard camera footage"], "contradiction_detected": true}`
	default:
		return `{"reply": "I've told you what I know. It was a normal evening until it wasn't — and I'd like to help, but I can't invent details I don't have.", "emotion": "neutral", "stress_delta": 2, "trust_delta": 0, "unlocked_clues": [], "reveal_evidence_titles": [], "contradiction_detected": false}`
	}
}

const evidenceJSON = `{
  "analysis": "Close examination supports the working theory: the item's condition is inconsistent with the official account of that evening, and the traces found suggest deliberate cleanup rather than routine handling.",
  "new_facts": ["The item was cleaned with a professional-grade solvent shortly before discovery"],
  "reliability_delta": 5,
  "reveal_related": true
}`

func judgeJSON(text string) string {
	if strings.Contains(text, "ACCUSATION_CORRECT: true") {
		return `{"score": 88, "feedback": "A sound accusation. You connected the falsified sign-out time, the courtyard footage, and the scheduled audit into a coherent chain of motive and opportunity. The case holds.", "motive_assessment": "The stated motive matches the evidence trail.", "reasoning_assessment": "Strong use of the timeline contradictions."}`
	}
	return `{"score": 35, "feedback": "The accusation does not survive scrutiny. The evidence you cite is circumstantial and key contradictions in the timeline remain unexplained. Re-examine who could actually reach the hall in the murder window.", "motive_assessment": "The proposed motive is not supported by discovered evidence.", "reasoning_assessment": "The reasoning overlooks at least one alibi inconsistency."}`
}

const psychologyJSON = `{
  "insight": "The suspect's composure is performative; their answers grow more rehearsed as questions approach the missing hour of the evening.",
  "stress_assessment": "Elevated and climbing — pressure on timeline details is effective.",
  "recommended_approach": "Present physical evidence and ask them to walk the timeline backwards."
}`

const directorJSON = `{
  "hint": "Someone's account of when they left does not match a document you already have. Compare statements against the sign-out sheet.",
  "tone": "encouraging"
}`
