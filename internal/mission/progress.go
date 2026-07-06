package mission

// This file holds the deterministic mission-progress and risk models. They are
// intentionally pure functions of already-collected, player-visible signals:
// the same inputs always yield the same score, so progress is never a vague
// AI guess and tests can assert exact behaviour.

// ProgressWeights is the explicit, stored progress formula for a mission. The
// five weights must sum to 100. The default matches the design spec; a mission
// type may override it, but the weights it used are always inspectable.
type ProgressWeights struct {
	Clues          int `json:"clues"`           // required clue discovery
	Interactions   int `json:"interactions"`    // required character interactions
	Locations      int `json:"locations"`       // map location exploration
	Decisions      int `json:"decisions"`       // correct player decisions
	FinalReadiness int `json:"final_readiness"` // final objective readiness
}

// DefaultProgressWeights is the baseline formula:
//
//	30% required clue discovery
//	25% required character interactions
//	20% map location exploration
//	15% correct player decisions
//	10% final objective readiness
var DefaultProgressWeights = ProgressWeights{
	Clues: 30, Interactions: 25, Locations: 20, Decisions: 15, FinalReadiness: 10,
}

// ProgressInputs are the deterministic signals the progress formula consumes.
type ProgressInputs struct {
	DiscoveredClues  int
	TotalClues       int
	InteractedChars  int
	TotalChars       int
	VisitedLocations int
	TotalLocations   int
	// CorrectDecisions is the count of confirmed facts the player has surfaced;
	// DecisionTarget is how many earn full credit for the "decisions" slice.
	CorrectDecisions int
	DecisionTarget   int
	// FinalReady is true once the mandatory clue coverage for the final
	// decision has been met.
	FinalReady bool
}

// ratio returns have/want clamped to [0,1] as a fraction, treating want<=0 as
// "nothing required here" (fully satisfied).
func ratio(have, want int) float64 {
	if want <= 0 {
		return 1
	}
	if have >= want {
		return 1
	}
	if have < 0 {
		return 0
	}
	return float64(have) / float64(want)
}

// MissionProgress computes the 0-100 progress score for the given weights.
func MissionProgress(in ProgressInputs, w ProgressWeights) int {
	final := 0.0
	if in.FinalReady {
		final = 1
	}
	score := ratio(in.DiscoveredClues, in.TotalClues)*float64(w.Clues) +
		ratio(in.InteractedChars, in.TotalChars)*float64(w.Interactions) +
		ratio(in.VisitedLocations, in.TotalLocations)*float64(w.Locations) +
		ratio(in.CorrectDecisions, in.DecisionTarget)*float64(w.Decisions) +
		final*float64(w.FinalReadiness)
	return clampScore(int(score + 0.5))
}

// RiskInputs are the deterministic signals the risk formula consumes.
type RiskInputs struct {
	// AmbientRisk is public_state.risk_level (0-100), driven by the world/time
	// engine.
	AmbientRisk int
	// ElapsedMinutes / DeadlineMinutes drive time pressure. DeadlineMinutes<=0
	// means the mission has no hard deadline.
	ElapsedMinutes  int
	DeadlineMinutes int
	// HighRiskLocations is the number of visible high-risk locations the player
	// has entered; danger accrues the longer they operate there.
	HighRiskLocations int
}

// RiskScore blends ambient danger, time pressure, and field exposure into a
// 0-100 risk score. It is monotonically non-decreasing in elapsed time and
// ambient risk, so advancing the clock always raises (or holds) risk.
func RiskScore(in RiskInputs) int {
	ambient := float64(clampScore(in.AmbientRisk))

	var timePressure float64
	switch {
	case in.DeadlineMinutes > 0:
		timePressure = float64(in.ElapsedMinutes) / float64(in.DeadlineMinutes) * 100
	default:
		// No hard deadline: risk still climbs with time, saturating at 24h.
		timePressure = float64(in.ElapsedMinutes) / float64(24*60) * 100
	}
	if timePressure > 100 {
		timePressure = 100
	}
	if timePressure < 0 {
		timePressure = 0
	}

	exposure := float64(in.HighRiskLocations) * 8
	if exposure > 100 {
		exposure = 100
	}

	score := 0.4*ambient + 0.4*timePressure + 0.2*exposure
	return clampScore(int(score + 0.5))
}

// TimeRemaining formats the minutes left before the mission deadline in a
// "N days M hours" shape. It returns ("", 0, false) when the mission has no
// deadline, and ("expired", 0, true) once the deadline has passed.
func TimeRemaining(elapsedMinutes, deadlineMinutes int) (label string, remaining int, hasDeadline bool) {
	if deadlineMinutes <= 0 {
		return "", 0, false
	}
	remaining = deadlineMinutes - elapsedMinutes
	if remaining <= 0 {
		return "expired", 0, true
	}
	days := remaining / (24 * 60)
	hours := (remaining % (24 * 60)) / 60
	minutes := remaining % 60
	switch {
	case days > 0:
		return plural(days, "day") + " " + plural(hours, "hour"), remaining, true
	case hours > 0:
		return plural(hours, "hour") + " " + plural(minutes, "minute"), remaining, true
	default:
		return plural(minutes, "minute"), remaining, true
	}
}

func plural(n int, unit string) string {
	s := itoa(n) + " " + unit
	if n != 1 {
		s += "s"
	}
	return s
}

func clampScore(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

// itoa avoids pulling strconv into a hot pure-function file.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
