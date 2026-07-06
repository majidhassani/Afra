// Package clue owns discoverable mission clues, generalized from evidence.
// Clues carry rich visual descriptions and thumbnail prompts so evidence
// cards and images can be generated later.
package clue

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

var ValidTypes = []string{
	"footprint", "document", "gps_signal", "radio_log", "broken_trap", "blood_trace",
	"animal_track", "camera_footage", "phone_record", "witness_statement", "map_fragment",
	"tool_mark", "vehicle_trace", "chemical_sample", "weather_damage", "personal_item",
	"audio_recording", "photo", "satellite_hint",
}

// Clue is the full internal entity. InternalTruth must never reach the
// client — always convert through Public().
type Clue struct {
	ID                      uuid.UUID
	MissionID               uuid.UUID
	LocationID              *uuid.UUID
	Title                   string
	Type                    string
	ShortDescription        string
	DetailedDescription     string
	VisualDescription       string
	Discovered              bool
	Reliability             int
	Importance              string
	RelatedCharacterIDs     json.RawMessage
	PublicData              json.RawMessage

	// Visual asset delivered to clients. The generation prompt below must
	// never cross the public boundary.
	ImageURL    string
	ImageStatus string // none | pending | ready | unavailable

	// Internal only — generation inputs and Truth Layer.
	AvatarOrThumbnailPrompt string
	InternalTruth           json.RawMessage

	CreatedAt time.Time
	UpdatedAt time.Time
}

// PublicClue is the only clue shape returned to clients.
type PublicClue struct {
	ID                      uuid.UUID       `json:"id"`
	MissionID               uuid.UUID       `json:"mission_id"`
	LocationID              *uuid.UUID      `json:"location_id,omitempty"`
	Title                   string          `json:"title"`
	Type                    string          `json:"type"`
	ShortDescription        string          `json:"short_description"`
	DetailedDescription     string          `json:"detailed_description"`
	VisualDescription       string          `json:"visual_description"`
	ImageURL                string          `json:"image_url"`
	ImageStatus             string          `json:"image_status"`
	Discovered              bool            `json:"discovered"`
	Reliability             int             `json:"reliability"`
	Importance              string          `json:"importance"`
	RelatedCharacterIDs     json.RawMessage `json:"related_character_ids"`
	PublicData              json.RawMessage `json:"public_data"`
	CreatedAt               time.Time       `json:"created_at"`
}

func (c *Clue) Public() PublicClue {
	return PublicClue{
		ID:                      c.ID,
		MissionID:               c.MissionID,
		LocationID:              c.LocationID,
		Title:                   c.Title,
		Type:                    c.Type,
		ShortDescription:        c.ShortDescription,
		DetailedDescription:     c.DetailedDescription,
		VisualDescription:       c.VisualDescription,
		ImageURL:                c.ImageURL,
		ImageStatus:             c.PublicImageStatus(),
		Discovered:              c.Discovered,
		Reliability:             c.Reliability,
		Importance:              c.Importance,
		RelatedCharacterIDs:     c.RelatedCharacterIDs,
		PublicData:              c.PublicData,
		CreatedAt:               c.CreatedAt,
	}
}

// PublicImageStatus derives the client-visible image status: an existing URL
// always means ready; otherwise the stored status (or "none") is used.
func (c *Clue) PublicImageStatus() string {
	if c.ImageURL != "" {
		return "ready"
	}
	if c.ImageStatus != "" {
		return c.ImageStatus
	}
	return "none"
}

func PublicList(list []Clue) []PublicClue {
	out := make([]PublicClue, 0, len(list))
	for i := range list {
		out = append(out, list[i].Public())
	}
	return out
}
