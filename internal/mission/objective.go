package mission

// Objective types. Every mission has exactly one primary objective; the rest
// are a mix of the remaining kinds.
const (
	ObjectivePrimary  = "primary"  // the headline goal — winning requires it
	ObjectiveRequired = "required" // mandatory sub-goal
	ObjectiveOptional = "optional" // bonus goal, never blocks a win
	ObjectiveHidden   = "hidden"   // not shown until discovered/unlocked
	ObjectiveDynamic  = "dynamic"  // spawned by world events during play
	ObjectiveFinal    = "final"    // the closing intervention/decision
)

// Objective statuses.
const (
	ObjStatusLocked    = "locked"
	ObjStatusActive    = "active"
	ObjStatusCompleted = "completed"
	ObjStatusFailed    = "failed"
	ObjStatusSkipped   = "skipped"
)

// ValidObjectiveTypes / ValidObjectiveStatuses back agent validation.
var (
	ValidObjectiveTypes = []string{
		ObjectivePrimary, ObjectiveRequired, ObjectiveOptional,
		ObjectiveHidden, ObjectiveDynamic, ObjectiveFinal,
	}
	ValidObjectiveStatuses = []string{
		ObjStatusLocked, ObjStatusActive, ObjStatusCompleted,
		ObjStatusFailed, ObjStatusSkipped,
	}
)

// Public-safe keys stored inside a mission's public_state JSONB by the
// generator. Win/failure hints are player-safe (the private win logic lives in
// the World Bible); the deadline drives the mission timer.
const (
	PublicKeyWinConditions     = "win_conditions"
	PublicKeyFailureConditions = "failure_conditions"
	PublicKeyDeadlineMinutes   = "deadline_minutes"
	PublicKeyRiskLevel         = "risk_level"
)
