package mock

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Mission-mode canned responses: a wildlife-rescue mission ("Operation
// Silent Plains") whose keys are consistent across the plan, world, map,
// characters, and clues so the full generation pipeline links up.

func missionType(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if t, ok := strings.CutPrefix(strings.TrimSpace(line), "MISSION_TYPE: "); ok {
			return strings.TrimSpace(t)
		}
	}
	return "wildlife_rescue"
}

// confidentialContext extracts the JSON object following the
// "CONFIDENTIAL CONTEXT:" / "PLAYER-VISIBLE CONTEXT:" marker so canned
// responses can echo real titles back (exact-match intentions).
func confidentialContext(text string) map[string]any {
	for _, marker := range []string{"CONFIDENTIAL CONTEXT:", "PLAYER-VISIBLE CONTEXT:"} {
		if _, after, found := strings.Cut(text, marker); found {
			start := strings.IndexByte(after, '{')
			if start < 0 {
				continue
			}
			dec := json.NewDecoder(strings.NewReader(after[start:]))
			var m map[string]any
			if err := dec.Decode(&m); err == nil {
				return m
			}
		}
	}
	return nil
}

var missionTitles = map[string]string{
	"detective":         "The Gallery After Dark",
	"wildlife_rescue":   "Operation Silent Plains",
	"disaster_response": "Floodline: Karun Delta",
	"exploration":       "The Unmapped Valley",
	"survival":          "Seventy Hours on the Ridge",
	"diplomacy":         "The Two Rivers Accord",
	"medical_mystery":   "The Quiet Village Fever",
}

func missionPlanJSON(mt string) string {
	title := missionTitles[mt]
	if title == "" {
		title = "Operation Silent Plains"
	}
	return fmt.Sprintf(`{
  "title": %q,
  "summary": "Cheetah sightings in the Kavir Reserve have dropped to zero in three weeks. GPS collars are going dark one by one and the local rangers cannot explain why.",
  "briefing": "Agent, you are deploying to the Kavir Reserve field sector. In the last three weeks every collared cheetah has vanished from tracking, and the reserve's radio net has been unreliable. Work the map, talk to the people on the ground, and find out what is really happening before the population is lost. Mission Control will assist you at every step.",
  "region": "Kavir Reserve, Iran",
  "objectives": [
    {"key": "obj_primary", "type": "primary", "title": "Save the remaining cheetahs", "description": "Find the cause of the disappearances and act before the reserve loses its last cheetahs.", "required_clues": 3, "optional": false},
    {"key": "obj_source", "type": "required", "title": "Find the source of the cheetah disappearance", "description": "Investigate map locations and gather enough clues to explain the vanishing collars.", "required_clues": 3, "optional": false},
    {"key": "obj_network", "type": "required", "title": "Identify who is behind it", "description": "Collect evidence pointing to the people or forces responsible.", "required_clues": 2, "optional": false},
    {"key": "obj_trust", "type": "optional", "title": "Win the rangers' trust", "description": "Talk with the field staff and corroborate their accounts.", "required_clues": 1, "optional": true},
    {"key": "obj_final", "type": "final", "title": "Close the migration corridor", "description": "Convince a ranger or official to act on your findings before time runs out.", "required_clues": 4, "optional": false}
  ],
  "win_conditions": ["Discover the cause of the disappearances", "Gather enough evidence", "Protect the remaining animals before time runs out"],
  "fail_conditions": ["The remaining population disperses before the cause is found", "Accusing the wrong party without evidence", "The mission deadline passes"],
  "deadline_hours": 120
}`, title)
}

const worldJSON = `{
  "truth": {
    "core_problem": "poachers are using cloned conservation permits to move through checkpoints",
    "real_cause": "an organized hunting network is jamming collar frequencies and baiting the migration corridor",
    "hidden_antagonist": "the regional logistics officer, Farid Kia, who sells patrol schedules"
  },
  "hidden_state": {
    "network_activity": "escalating",
    "jammer_location": "old quarry east of the hunter camp",
    "next_shipment_minutes": 420,
    "ranger_morale": 40
  },
  "public_state": {"weather": "dry and windy", "risk_level": 35},
  "event_schedule": [
    {"at_minutes": 60, "type": "map_activity", "title": "Fresh tire tracks near the corridor", "description": "A vehicle crossed the migration corridor before dawn."},
    {"at_minutes": 180, "type": "weather_changed", "title": "Dust storm building", "description": "Visibility begins to drop across the eastern sector."},
    {"at_minutes": 420, "type": "npc_movement", "title": "Activity at the hunter camp", "description": "Someone is packing crates at the camp."},
    {"at_minutes": 600, "type": "world_state_change", "title": "Collar signal flicker", "description": "A dead collar pings once near the quarry."}
  ],
  "failure_rules": ["If the shipment leaves the reserve, the network goes dark", "Accusing the head ranger without evidence collapses local cooperation"]
}`

const missionMapJSON = `{
  "center_lat": 34.712,
  "center_lng": 52.101,
  "zoom": 12,
  "locations": [
    {"key": "loc_ranger_station", "name": "Ranger Station", "type": "operations_base", "latitude": 34.7301, "longitude": 52.0899, "status": "discovered", "risk_level": 15, "description": "A dusty field station with a cracked radio mast and wall maps of the reserve.", "visual_prompt": "small remote wildlife ranger station, dusty equipment, cracked radio, wall maps, realistic mobile game background", "available_actions": ["inspect_area", "talk_to_character", "ask_ai", "view_clues", "review_documents"]},
    {"key": "loc_river_crossing", "name": "River Crossing", "type": "river_crossing", "latitude": 34.6987, "longitude": 52.1245, "status": "discovered", "risk_level": 30, "description": "A shallow ford on the seasonal river where animal tracks converge.", "visual_prompt": "shallow desert river crossing, muddy banks, animal tracks, sparse reeds, morning haze, realistic mobile game background", "available_actions": ["inspect_area", "scan_environment", "ask_ai", "view_clues"]},
    {"key": "loc_hunter_camp", "name": "Hunter Camp", "type": "camp", "latitude": 34.6702, "longitude": 52.1568, "status": "hidden", "risk_level": 70, "description": "An improvised camp tucked behind a rock spur, recently used.", "visual_prompt": "abandoned hunter camp in desert scrubland, tarps, cold fire pit, crates, harsh light, realistic mobile game background", "available_actions": ["inspect_area", "scan_environment", "ask_ai", "view_clues"]},
    {"key": "loc_research_center", "name": "Research Center", "type": "research_center", "latitude": 34.7452, "longitude": 52.1301, "status": "discovered", "risk_level": 10, "description": "The conservation project's small research outpost with collar telemetry racks.", "visual_prompt": "small wildlife research outpost interior, telemetry racks, monitors, specimen charts, realistic mobile game background", "available_actions": ["inspect_area", "talk_to_character", "ask_ai", "view_clues", "review_documents"]},
    {"key": "loc_village_market", "name": "Village Market", "type": "village", "latitude": 34.7203, "longitude": 52.0653, "status": "hidden", "risk_level": 25, "description": "The nearest village's weekly market, where fuel and rumors change hands.", "visual_prompt": "small desert village market street, stalls, fuel cans, midday shade, realistic mobile game background", "available_actions": ["talk_to_character", "ask_ai", "view_clues"]}
  ]
}`

const charactersJSON = `{
  "characters": [
    {"key": "char_guide", "name": "KAVIR Control", "role": "Mission Control AI", "category": "guide", "age": 0, "public_profile": "The reserve operation's tactical assistant, always on the radio channel.", "personality": {"openness": 90, "honesty": 95, "stress": 5, "patience": 95}, "location_key": "", "mood": "calm", "dialogue_style": "concise, tactical, supportive", "avatar_prompt": "futuristic mission control AI interface, calm tactical assistant, holographic blue-gray face silhouette, dark command center UI, premium mobile game style", "thumbnail_prompt": "mission control AI avatar, holographic blue silhouette, dark UI card", "visual_style_tags": ["holographic", "tactical", "mobile-game"], "private_state": {"notes": "knows only public mission state"}},
    {"key": "char_ranger", "name": "Thomas Reed", "role": "Head Ranger", "category": "field", "age": 47, "public_profile": "A tired but respected ranger who has worked the reserve for 18 years.", "personality": {"openness": 40, "honesty": 70, "stress": 65, "patience": 45, "protectiveness": 90}, "location_key": "loc_ranger_station", "mood": "exhausted", "dialogue_style": "short, direct, defensive when questioned about missing patrols", "avatar_prompt": "realistic portrait of a tired male wildlife ranger in his late 40s, weathered face, dusty khaki uniform, sunburned skin, tired eyes, cinematic mobile game character portrait", "thumbnail_prompt": "head ranger portrait, khaki uniform, rugged realistic style, dark UI card", "visual_style_tags": ["realistic", "cinematic", "portrait"], "private_state": {"hidden_knowledge": ["He skipped two night patrols to cover for his sick deputy", "He suspects the logistics officer but has no proof"], "lies_about": ["the skipped patrols"], "goals": ["protect his staff", "save the cheetahs"]}},
    {"key": "char_scientist", "name": "Dr. Leila Arman", "role": "Wildlife Biologist", "category": "field", "age": 36, "public_profile": "The collar telemetry lead at the research center, meticulous and worried.", "personality": {"openness": 75, "honesty": 90, "stress": 55, "patience": 60}, "location_key": "loc_research_center", "mood": "worried", "dialogue_style": "precise, data-driven, quick to show charts", "avatar_prompt": "realistic portrait of a focused female wildlife biologist in her mid 30s, field jacket, tablet in hand, research outpost background, cinematic mobile game character portrait", "thumbnail_prompt": "wildlife biologist portrait, field jacket, realistic style, dark UI card", "visual_style_tags": ["realistic", "cinematic", "portrait"], "private_state": {"hidden_knowledge": ["The collar failures cluster along the east corridor at night", "A signed equipment requisition she filed was altered"], "goals": ["find the telemetry pattern"]}},
    {"key": "char_logistics", "name": "Farid Kia", "role": "Regional Logistics Officer", "category": "antagonist", "age": 44, "public_profile": "The regional office's logistics coordinator, polished and helpful on the surface.", "personality": {"openness": 55, "honesty": 20, "stress": 30, "patience": 80}, "location_key": "loc_village_market", "mood": "smooth", "dialogue_style": "friendly, evasive on specifics, redirects blame subtly", "avatar_prompt": "realistic portrait of a composed middle-aged logistics officer, pressed shirt, subtle smile, market street background, cinematic mobile game character portrait", "thumbnail_prompt": "logistics officer portrait, pressed shirt, realistic style, dark UI card", "visual_style_tags": ["realistic", "cinematic", "portrait"], "private_state": {"is_antagonist": true, "hidden_knowledge": ["He sells patrol schedules to the hunting network", "He arranged the frequency jammer at the quarry"], "lies_about": ["his contact with the hunter camp", "the altered requisitions"], "goals": ["keep the shipment schedule moving", "stay above suspicion"]}},
    {"key": "char_merchant", "name": "Golnar Rahimi", "role": "Market Merchant", "category": "neutral", "age": 58, "public_profile": "Runs the fuel and dry-goods stall; sees everyone who passes through.", "personality": {"openness": 65, "honesty": 75, "stress": 20, "patience": 70}, "location_key": "loc_village_market", "mood": "wry", "dialogue_style": "chatty, trades information for goodwill", "avatar_prompt": "realistic portrait of a sharp-eyed older market merchant woman, patterned headscarf, market stall background, warm light, cinematic mobile game character portrait", "thumbnail_prompt": "market merchant portrait, patterned headscarf, realistic style, dark UI card", "visual_style_tags": ["realistic", "cinematic", "portrait"], "private_state": {"hidden_knowledge": ["A city buyer paid cash for 200 liters of diesel twice this month", "The logistics officer meets a stranger behind the mosque on Thursdays"], "goals": ["stay out of trouble"]}}
  ]
}`

const cluesJSON = `{
  "clues": [
    {"key": "clue_collar", "location_key": "loc_research_center", "title": "Broken GPS Collar", "type": "gps_signal", "short_description": "A damaged GPS collar recovered near the dry riverbed.", "detailed_description": "The collar is scratched, partly crushed, and still blinking red. Dried mud is packed under the signal casing and the antenna lead is cut cleanly, not worn.", "visual_description": "broken wildlife GPS collar on dry cracked desert soil, small red blinking light, dust, animal hair, realistic evidence photo", "avatar_or_thumbnail_prompt": "close-up forensic evidence card of a broken wildlife GPS collar on desert soil, realistic mobile game UI, cinematic lighting", "discovered": true, "reliability": 85, "importance": "high", "related_character_keys": ["char_scientist"], "public_data": {"note": "antenna lead severed cleanly"}, "internal_truth": {"actual": "The collar was cut off by the network and dumped; the clean cut proves human interference.", "false_lead": false}},
    {"key": "clue_radio_log", "location_key": "loc_ranger_station", "title": "Broken Radio Log", "type": "radio_log", "short_description": "The station log shows gaps in the night radio checks.", "detailed_description": "Three consecutive nights are missing their 02:00 check-in entries. The pages are otherwise complete and initialed.", "visual_description": "worn paper radio logbook on a dusty desk, missing entries circled, handheld radio beside it, realistic evidence photo", "avatar_or_thumbnail_prompt": "evidence card of an open worn radio logbook with missing entries, realistic mobile game UI, cinematic lighting", "discovered": true, "reliability": 75, "importance": "medium", "related_character_keys": ["char_ranger"], "public_data": {"missing_nights": 3}, "internal_truth": {"actual": "Reed skipped patrols covering for his deputy; unrelated to the network but it delayed detection.", "false_lead": true}},
    {"key": "clue_boot_prints", "location_key": "loc_river_crossing", "title": "Fresh Boot Prints Near Riverbank", "type": "footprint", "short_description": "Several deep boot prints appear in wet soil near the river crossing.", "detailed_description": "The prints are recent, sharply edged, and point away from the river. One sole pattern has a triangular notch on the heel.", "visual_description": "wet riverbank soil, clear boot prints, small broken reeds, morning mist, realistic investigation photo", "avatar_or_thumbnail_prompt": "close-up forensic style photo of fresh boot prints in muddy riverbank soil, realistic mobile game evidence card, cinematic lighting", "discovered": true, "reliability": 80, "importance": "medium", "related_character_keys": [], "public_data": {"sole_mark": "triangular notch on heel"}, "internal_truth": {"actual": "Prints belong to a network scout who baits the corridor; the notch matches boots sold at the market.", "false_lead": false}},
    {"key": "clue_tire_tracks", "location_key": "loc_river_crossing", "title": "Heavy Tire Tracks", "type": "vehicle_trace", "short_description": "Wide tire tracks cut across the migration corridor.", "detailed_description": "Dual rear wheels, heavily loaded, heading east before dawn based on the moisture line in the tread marks.", "visual_description": "heavy truck tire tracks across cracked desert ground, dawn shadows, realistic investigation photo", "avatar_or_thumbnail_prompt": "evidence card of heavy tire tracks across desert ground, realistic mobile game UI, cinematic lighting", "discovered": false, "reliability": 70, "importance": "medium", "related_character_keys": ["char_logistics"], "public_data": {"direction": "east"}, "internal_truth": {"actual": "The shipment truck route toward the quarry staging point.", "false_lead": false}},
    {"key": "clue_torn_permit", "location_key": "loc_hunter_camp", "title": "Torn Conservation Permit", "type": "document", "short_description": "Half of an official-looking conservation transit permit, torn and burned at one corner.", "detailed_description": "The serial number is intact. The stamp is correct but the paper stock is wrong — thinner than official issue.", "visual_description": "torn official permit document on scorched campfire stones, official stamp visible, realistic evidence photo", "avatar_or_thumbnail_prompt": "evidence card of a torn burned permit document with official stamp, realistic mobile game UI, cinematic lighting", "discovered": false, "reliability": 90, "importance": "high", "related_character_keys": ["char_logistics"], "public_data": {"serial_visible": true}, "internal_truth": {"actual": "A cloned permit from the batch Kia's contact prints; the serial ties to the regional office.", "false_lead": false}},
    {"key": "clue_requisition", "location_key": "loc_research_center", "title": "Altered Equipment Requisition", "type": "document", "short_description": "A filed requisition form whose quantities look overwritten.", "detailed_description": "The original ink lists two frequency scanners; the altered copy signs out four. The approving signature is the logistics office's.", "visual_description": "official requisition form with visible ink alterations under desk lamp light, realistic evidence photo", "avatar_or_thumbnail_prompt": "evidence card of an altered requisition form, ink corrections visible, realistic mobile game UI, cinematic lighting", "discovered": false, "reliability": 85, "importance": "high", "related_character_keys": ["char_scientist", "char_logistics"], "public_data": {"item": "frequency scanners"}, "internal_truth": {"actual": "Kia diverted scanners to build the collar jammer.", "false_lead": false}},
    {"key": "clue_fuel_purchase", "location_key": "loc_village_market", "title": "Suspicious Fuel Purchase", "type": "witness_statement", "short_description": "The merchant recalls a stranger buying unusual amounts of diesel, twice.", "detailed_description": "Cash payment, city plates, two hundred liters each time, always the day before the wind turns east.", "visual_description": "market fuel stall with diesel cans, receipt book, late afternoon light, realistic investigation photo", "avatar_or_thumbnail_prompt": "evidence card of a market fuel stall with stacked diesel cans, realistic mobile game UI, cinematic lighting", "discovered": false, "reliability": 65, "importance": "medium", "related_character_keys": ["char_merchant"], "public_data": {"volume_liters": 200}, "internal_truth": {"actual": "Fuel for the jammer generator at the quarry.", "false_lead": false}}
  ]
}`

func dialogueJSON(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "injection_detected: true"):
		return `{"reply": "Let's stay on mission, agent. Ask me something about the field and I'll answer it straight.", "emotion": "guarded", "mood": "neutral", "trust_delta": -2, "stress_delta": 2, "unlock_clue_titles": [], "new_facts": []}`
	case strings.Contains(lower, "collar") || strings.Contains(lower, "gps"):
		return `{"reply": "The collars didn't fail on their own. Look at the cut on the one we recovered — that's a blade, not a rock. Whoever did it knew our telemetry windows.", "emotion": "worried", "mood": "focused", "trust_delta": 3, "stress_delta": 4, "unlock_clue_titles": ["Altered Equipment Requisition"], "new_facts": ["The collar failures cluster along the east corridor at night"]}`
	case strings.Contains(lower, "patrol") || strings.Contains(lower, "radio"):
		return `{"reply": "I was checking the north fence. The radio was down, so I couldn't report it. If you're implying something, say it plainly.", "emotion": "defensive", "mood": "tense", "trust_delta": -2, "stress_delta": 6, "unlock_clue_titles": ["Broken Radio Log"], "new_facts": []}`
	case strings.Contains(lower, "fuel") || strings.Contains(lower, "market") || strings.Contains(lower, "stranger"):
		return `{"reply": "You buy tea, I talk. Twice this month a man with city plates took two hundred liters of diesel, cash, no receipt. Out here, nobody burns that much for nothing.", "emotion": "wry", "mood": "chatty", "trust_delta": 2, "stress_delta": 0, "unlock_clue_titles": ["Suspicious Fuel Purchase"], "new_facts": ["A city buyer paid cash for large diesel purchases twice this month"]}`
	default:
		return `{"reply": "Ask around, agent. The reserve keeps its secrets, but the people who work it see more than they say.", "emotion": "neutral", "mood": "neutral", "trust_delta": 1, "stress_delta": 0, "unlock_clue_titles": [], "new_facts": []}`
	}
}

func guidanceJSON(text string) string {
	ctx := confidentialContext(text)
	// Point at the first unvisited location when one exists.
	if ctx != nil {
		if unvisited, ok := ctx["unvisited_locations"].([]any); ok && len(unvisited) > 0 {
			if loc, ok := unvisited[0].(map[string]any); ok {
				id, _ := loc["id"].(string)
				name, _ := loc["name"].(string)
				return fmt.Sprintf(`{"message": "Based on what you've gathered so far, %s is your strongest unexplored lead. Cross-check any documents you find there against the collar timeline.", "hint_level": "low", "referenced_items": [{"type": "location", "id": %q, "name": %q}]}`, name, id, name)
			}
		}
	}
	return `{"message": "Review your discovered clues side by side: the clean antenna cut and the missing radio checks don't tell the same story. Decide which timeline you trust, then test it against the people who were on duty.", "hint_level": "medium", "referenced_items": []}`
}

func locationActionJSON(text string) string {
	ctx := confidentialContext(text)
	if ctx != nil {
		if undiscovered, ok := ctx["undiscovered_clues"].([]any); ok && len(undiscovered) > 0 {
			if clue, ok := undiscovered[0].(map[string]any); ok {
				if title, ok := clue["title"].(string); ok && title != "" {
					return fmt.Sprintf(`{"narrative": "You work the area methodically, checking shadows and disturbed ground. Behind a low ridge of stones, something out of place catches the light.", "discover_clue_titles": [%q], "new_facts": ["This spot was visited recently by someone careful"]}`, title)
				}
			}
		}
	}
	return `{"narrative": "You sweep the area carefully but find nothing new. The ground has been picked over — or there was nothing here to begin with.", "discover_clue_titles": [], "new_facts": []}`
}

const clueInspectionJSON = `{
  "analysis": "Close inspection supports deliberate interference: the damage pattern is too clean for wear and the residue suggests handling within the last week. Compare this with movement patterns near the east corridor.",
  "new_facts": ["The damage was inflicted deliberately and recently"],
  "reliability_delta": 5
}`

const clueExplanationJSON = `{
  "explanation": "On its face this clue shows recent human activity where there should be none. It matters because it anchors a timeline: whatever happened here happened deliberately and recently. On its own it proves nothing about who is responsible.",
  "next_steps": ["Compare it with the other clues from nearby locations", "Ask the people stationed closest to where it was found"],
  "compare_with": ["Fresh Boot Prints Near Riverbank", "Broken Radio Log"]
}`

const timeAdvanceJSON = `{
  "summary": "The wind picks up from the east and visibility drops along the corridor. A ranger radio check mentions engine noise near the hunter camp just before the channel cuts out.",
  "events": [
    {"type": "weather_changed", "title": "Dust rising in the east"},
    {"type": "map_activity", "title": "Engine noise reported near the hunter camp"}
  ],
  "public_state_changes": {"weather": "dusty, wind rising", "risk_level": 45},
  "hidden_state_changes": {"network_activity": "staging", "next_shipment_minutes": 300}
}`

const missionDirectorJSON = `{
  "events": [{"type": "radio_chatter", "title": "A half-heard voice on the ranger channel mentions the old quarry"}],
  "hint": "You have threads pointing east: tracks, fuel, and jammed signals. Consider what they have in common.",
  "tone": "encouraging"
}`

func missionJudgeJSON(text string) string {
	ctx := confidentialContext(text)
	discovered, total := 0.0, 1.0
	outcome := ""
	if ctx != nil {
		if v, ok := ctx["discovered_clues"].(float64); ok {
			discovered = v
		}
		if v, ok := ctx["total_clues"].(float64); ok && v > 0 {
			total = v
		}
		outcome, _ = ctx["player_outcome"].(string)
	}
	lower := strings.ToLower(outcome)
	strongOutcome := strings.Contains(lower, "logistics") || strings.Contains(lower, "kia") ||
		strings.Contains(lower, "jammer") || strings.Contains(lower, "network") || strings.Contains(lower, "poach")
	if strongOutcome && discovered/total >= 0.4 {
		return `{"success": true, "score": 86, "feedback": "A solid operation. You connected the severed collar, the altered requisition, and the fuel purchases into a coherent picture of an organized network — and you were right about who fed them the patrol schedules.", "objective_results": [{"key": "obj_source", "completed": true, "note": "Jamming and baiting identified"}, {"key": "obj_network", "completed": true, "note": "The logistics connection was named with evidence"}, {"key": "obj_trust", "completed": true, "note": "Field staff corroborated your findings"}], "reasoning_assessment": "Evidence-first reasoning with a correct causal chain.", "world_impact": "The shipment was intercepted and the corridor secured before the population dispersed."}`
	}
	return `{"success": false, "score": 42, "feedback": "The mission ends without a supported conclusion. Several high-value clues were never found, and the stated outcome does not match the evidence trail you gathered.", "objective_results": [{"key": "obj_source", "completed": false, "note": "The jamming mechanism was never established"}, {"key": "obj_network", "completed": false, "note": "No responsible party was tied to evidence"}, {"key": "obj_trust", "completed": true, "note": "The rangers cooperated with you"}], "reasoning_assessment": "The conclusion outran the evidence.", "world_impact": "Collar losses continue; the network adapts its routes."}`
}
