// Package character owns mission NPCs, generalized from suspects. Every
// character carries avatar/thumbnail prompts for later image generation and
// a private state that never reaches the client.
package character

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Categories.
const (
	CategoryGuide      = "guide"
	CategoryField      = "field"
	CategoryAntagonist = "antagonist"
	CategoryNeutral    = "neutral"
)

// Character is the full internal entity. PrivateState must never reach the
// client — always convert through Public().
type Character struct {
	ID                uuid.UUID
	MissionID         uuid.UUID
	Name              string
	Role              string
	Category          string
	Age               int
	PublicProfile     string
	Personality       json.RawMessage
	CurrentLocationID *uuid.UUID
	TrustLevel        int
	StressLevel       int
	Mood              string
	DialogueStyle     string
	AvatarPrompt      string
	ThumbnailPrompt   string
	VisualStyleTags   json.RawMessage

	// Internal only — Truth Layer.
	PrivateState json.RawMessage

	CreatedAt time.Time
	UpdatedAt time.Time
}

// PublicCharacter is the only character shape returned to clients.
type PublicCharacter struct {
	ID                uuid.UUID       `json:"id"`
	MissionID         uuid.UUID       `json:"mission_id"`
	Name              string          `json:"name"`
	Role              string          `json:"role"`
	Category          string          `json:"category"`
	Age               int             `json:"age,omitempty"`
	PublicProfile     string          `json:"public_profile"`
	Personality       json.RawMessage `json:"personality"`
	CurrentLocationID *uuid.UUID      `json:"current_location_id,omitempty"`
	TrustLevel        int             `json:"trust_level"`
	Mood              string          `json:"mood"`
	AvatarPrompt      string          `json:"avatar_prompt"`
	ThumbnailPrompt   string          `json:"thumbnail_prompt"`
	VisualStyleTags   json.RawMessage `json:"visual_style_tags"`
}

func (c *Character) Public() PublicCharacter {
	return PublicCharacter{
		ID:                c.ID,
		MissionID:         c.MissionID,
		Name:              c.Name,
		Role:              c.Role,
		Category:          c.Category,
		Age:               c.Age,
		PublicProfile:     c.PublicProfile,
		Personality:       c.Personality,
		CurrentLocationID: c.CurrentLocationID,
		TrustLevel:        c.TrustLevel,
		Mood:              c.Mood,
		AvatarPrompt:      c.AvatarPrompt,
		ThumbnailPrompt:   c.ThumbnailPrompt,
		VisualStyleTags:   c.VisualStyleTags,
	}
}

func PublicList(list []Character) []PublicCharacter {
	out := make([]PublicCharacter, 0, len(list))
	for i := range list {
		out = append(out, list[i].Public())
	}
	return out
}
