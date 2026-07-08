package mission

import (
	"encoding/json"
	"regexp"
	"strings"
)

// WorldState is the player-safe snapshot of the living world the client uses to
// drive dynamic UI themes (weather overlays, night darkening, danger pulse) and
// to communicate that the world is alive. It is derived deterministically from
// the mission's public state, clock, and risk — never from the WorldBible.
type WorldState struct {
	MissionTime  string   `json:"mission_time"`
	Weather      string   `json:"weather"`     // clear | rain | snow | storm | fog
	TimeOfDay    string   `json:"time_of_day"` // day | dusk | night
	Visibility   string   `json:"visibility"`  // high | medium | low
	RiskScore    int      `json:"risk_score"`
	Urgency      string   `json:"urgency"`     // calm | rising | critical
	WorldPhase   string   `json:"world_phase"` // opening | investigation | closing
	Danger       bool     `json:"danger"`      // risk high enough to warn the HUD
	ThemeID      string   `json:"theme_id"`    // e.g. forest_rain, city_night, desert_danger
	ActiveEvents []string `json:"active_events"`
}

// DangerThreshold is the risk score at or above which the HUD warns.
const DangerThreshold = 60

// biomeByType mirrors the client's coarse mission-type → biome mapping so the
// server can compose a ready-to-use theme_id. The client remains the source of
// truth for region-keyword refinements; this is a sensible default.
var biomeByType = map[string]string{
	"wildlife_rescue":   "forest",
	"exploration":       "space",
	"survival":          "snow",
	"disaster_response": "desert",
	"diplomacy":         "city",
	"detective":         "city",
	"medical_mystery":   "horror",
}

// BiomeForType returns the default biome for a mission type.
func BiomeForType(missionType string) string {
	if b, ok := biomeByType[missionType]; ok {
		return b
	}
	return "city"
}

var clockHour = regexp.MustCompile(`(\d{1,2}):(\d{2})`)

// DeriveWorldState computes the living-world snapshot. It is a pure function so
// the same inputs always yield the same world — deterministic gameplay.
//
// biome is the coarse environment (forest/desert/city/snow/horror/space) the
// client already derives; passing it here lets theme_id be a ready-to-use
// "<biome>_<modifier>" string. progress/deadline drive the world phase.
func DeriveWorldState(publicState json.RawMessage, clock, biome string, risk, progress int, hasDeadline, deadlineApproaching bool) WorldState {
	m := map[string]any{}
	_ = json.Unmarshal(publicState, &m)

	ws := WorldState{
		MissionTime:  clock,
		RiskScore:    risk,
		Danger:       risk >= DangerThreshold,
		ActiveEvents: []string{},
	}
	ws.Weather = weatherFrom(m)
	ws.TimeOfDay = timeOfDay(clock)
	ws.Visibility = visibility(ws.Weather, ws.TimeOfDay)
	ws.Urgency = urgency(hasDeadline, deadlineApproaching, risk)
	ws.WorldPhase = worldPhase(progress)
	ws.ThemeID = themeID(biome, ws.Weather, ws.TimeOfDay, ws.Danger)

	if ws.Weather != "clear" {
		ws.ActiveEvents = append(ws.ActiveEvents, "weather_"+ws.Weather)
	}
	if ws.Danger {
		ws.ActiveEvents = append(ws.ActiveEvents, "high_risk")
	}
	return ws
}

// weatherFrom reads a free-text weather field from public state and maps it to a
// known category. Persian and English keywords are both recognised.
func weatherFrom(m map[string]any) string {
	text := ""
	for _, k := range []string{"weather", "جو", "آب‌وهوا", "conditions"} {
		if v, ok := m[k].(string); ok {
			text += " " + v
		}
	}
	text = strings.ToLower(text)
	switch {
	case matches(text, `storm|thunder|طوفان|رعد`):
		return "storm"
	case matches(text, `snow|blizzard|برف|یخبندان`):
		return "snow"
	case matches(text, `rain|drizzle|wet|باران|بارانی|باران`):
		return "rain"
	case matches(text, `fog|mist|haze|مه|غبار`):
		return "fog"
	default:
		return "clear"
	}
}

// timeOfDay parses the hour out of a clock string like "Day 1 — 21:30".
func timeOfDay(clock string) string {
	match := clockHour.FindStringSubmatch(clock)
	if len(match) < 2 {
		return "day"
	}
	h := 0
	for _, r := range match[1] {
		h = h*10 + int(r-'0')
	}
	switch {
	case h >= 20 || h < 5:
		return "night"
	case h >= 17:
		return "dusk"
	default:
		return "day"
	}
}

func visibility(weather, tod string) string {
	if weather == "storm" || weather == "fog" || tod == "night" {
		return "low"
	}
	if weather == "rain" || weather == "snow" || tod == "dusk" {
		return "medium"
	}
	return "high"
}

func urgency(hasDeadline, approaching bool, risk int) string {
	switch {
	case (hasDeadline && approaching) || risk >= 80:
		return "critical"
	case risk >= DangerThreshold:
		return "rising"
	default:
		return "calm"
	}
}

func worldPhase(progress int) string {
	switch {
	case progress >= 70:
		return "closing"
	case progress >= 25:
		return "investigation"
	default:
		return "opening"
	}
}

// themeID composes a client theme token. Danger overrides weather so the HUD
// always signals threat; otherwise weather, then night, modify the biome.
func themeID(biome, weather, tod string, danger bool) string {
	if biome == "" {
		biome = "city"
	}
	switch {
	case danger:
		return biome + "_danger"
	case weather == "storm" || weather == "rain":
		return biome + "_rain"
	case weather == "snow" || weather == "storm":
		return biome + "_snow"
	case tod == "night":
		return biome + "_night"
	default:
		return biome + "_day"
	}
}

func matches(s, pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(s)
}
