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
	VisualPrompt     string          `json:"visual_prompt"`
	AvailableActions json.RawMessage `json:"available_actions"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// Visible reports whether the location may appear on the client map at all.
func (l *Location) Visible() bool {
	return l.Status == StatusDiscovered || l.Status == StatusVisited || l.Status == StatusLocked
}

// Marker is one Google Maps marker in the map view.
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
