// Package gamemap owns mission map locations. The map is deliberately
// simple: Google Maps-compatible lat/lng markers with status, risk, badges,
// and available actions — no custom map engine.
package gamemap

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Location statuses.
const (
	StatusHidden     = "hidden"
	StatusDiscovered = "discovered"
	StatusLocked     = "locked"
	StatusVisited    = "visited"
)

type Location struct {
	ID               uuid.UUID       `json:"id"`
	MissionID        uuid.UUID       `json:"mission_id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	Latitude         float64         `json:"latitude"`
	Longitude        float64         `json:"longitude"`
	Status           string          `json:"status"`
	RiskLevel        int             `json:"risk_level"`
	Description      string          `json:"description"`
	// VisualPrompt is an image-generation input; never serialized to clients.
	VisualPrompt     string          `json:"-"`
	AvailableActions json.RawMessage `json:"available_actions"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// Visible reports whether the location may appear on the client map at all.
func (l *Location) Visible() bool {
	return l.Status == StatusDiscovered || l.Status == StatusVisited || l.Status == StatusLocked
}

// Marker is one Google Maps marker in the map view.
//
// The recommendation fields (ObjectiveStatus, HasRequiredAction, Recommended,
// Priority) let the UI communicate progress: highlight the next recommended
// location, flag spots with required actions or new clues, dim completed ones,
// and warn on high-risk locations.
type Marker struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	Status       string    `json:"status"`
	RiskLevel    int       `json:"risk_level"`
	HasNewClue   bool      `json:"has_new_clue"`
	HasCharacter bool      `json:"has_character"`
	IsLocked     bool      `json:"is_locked"`
	Badge        string    `json:"badge,omitempty"`

	// Progress/guidance signals.
	ObjectiveStatus   string `json:"objective_status"` // available | active | completed | high_risk
	HasRequiredAction bool   `json:"has_required_action"`
	Recommended       bool   `json:"recommended"`
	Priority          string `json:"priority"` // high | medium | low
}

// Objective statuses a marker can advertise.
const (
	MarkerObjectiveAvailable = "available"
	MarkerObjectiveActive    = "active"
	MarkerObjectiveCompleted = "completed"
	MarkerObjectiveHighRisk  = "high_risk"
)

// HighRiskThreshold is the risk_level at or above which a location is flagged
// as high-risk on the map.
const HighRiskThreshold = 60

// Annotate fills the recommendation fields from the marker's own state. It sets
// a location as recommended only when told to (the caller decides which single
// location is the next best step). hasMore reports whether the location still
// has undiscovered clues.
func (m *Marker) Annotate(hasMore, recommended bool) {
	switch {
	case m.RiskLevel >= HighRiskThreshold:
		m.ObjectiveStatus = MarkerObjectiveHighRisk
	case m.Status == StatusVisited && !hasMore:
		m.ObjectiveStatus = MarkerObjectiveCompleted
	case m.Status == StatusVisited:
		m.ObjectiveStatus = MarkerObjectiveActive
	default:
		m.ObjectiveStatus = MarkerObjectiveAvailable
	}
	// Something still to do here: never visited, or visited but more to find.
	m.HasRequiredAction = !m.IsLocked && (m.Status == StatusDiscovered || (m.Status == StatusVisited && hasMore))
	m.Recommended = recommended && m.HasRequiredAction
	switch {
	case m.Recommended:
		m.Priority = "high"
	case m.HasRequiredAction:
		m.Priority = "medium"
	default:
		m.Priority = "low"
	}
}

// MapView is the full map response for the Swift Google Maps view.
type MapView struct {
	MissionID uuid.UUID `json:"mission_id"`
	Center    LatLng    `json:"center"`
	Zoom      int       `json:"zoom"`
	Locations []Marker  `json:"locations"`
}

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
