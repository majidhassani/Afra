package missionevent

// TimeUpdate is the player-safe "time passed" envelope every clocked gameplay
// action returns: the new mission clock, how many minutes the action cost,
// and any world events the passage of time triggered. It lives here (the
// shared leaf package) so gamemap/character/clue/report can all return it
// without importing the mission package.
type TimeUpdate struct {
	NewTime         string       `json:"new_time"`
	MinutesAdvanced int          `json:"minutes_advanced"`
	TriggeredEvents []WorldEvent `json:"triggered_events"`
}

// WorldEvent is a player-visible event triggered by the passage of time
// (from the mission's scheduled timeline). Descriptions are player-safe.
type WorldEvent struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}
