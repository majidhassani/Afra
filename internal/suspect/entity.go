package suspect

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Suspect is the full internal entity. Internal-only fields (IsCulprit,
// Secrets, LieProfile, PrivateMemory) must never reach the client — always
// convert through Public() before serializing.
type Suspect struct {
	ID                 uuid.UUID
	CaseID             uuid.UUID
	Name               string
	Age                int
	Job                string
	RelationToVictim   string
	PublicProfile      json.RawMessage
	Personality        json.RawMessage
	KnownFacts         json.RawMessage
	StressLevel        int
	TrustLevel         int
	InterrogationCount int

	// Internal only — Truth Layer.
	IsCulprit     bool
	Secrets       json.RawMessage
	LieProfile    json.RawMessage
	PrivateMemory json.RawMessage

	CreatedAt time.Time
	UpdatedAt time.Time
}

// PublicSuspect is the only suspect shape returned to clients.
type PublicSuspect struct {
	ID                 uuid.UUID       `json:"id"`
	CaseID             uuid.UUID       `json:"case_id"`
	Name               string          `json:"name"`
	Age                int             `json:"age"`
	Job                string          `json:"job"`
	RelationToVictim   string          `json:"relation_to_victim"`
	PublicProfile      json.RawMessage `json:"public_profile"`
	Personality        json.RawMessage `json:"personality"`
	KnownFacts         json.RawMessage `json:"known_facts"`
	StressLevel        int             `json:"stress_level"`
	TrustLevel         int             `json:"trust_level"`
	InterrogationCount int             `json:"interrogation_count"`
	CreatedAt          time.Time       `json:"created_at"`
}

func (s *Suspect) Public() PublicSuspect {
	return PublicSuspect{
		ID:                 s.ID,
		CaseID:             s.CaseID,
		Name:               s.Name,
		Age:                s.Age,
		Job:                s.Job,
		RelationToVictim:   s.RelationToVictim,
		PublicProfile:      s.PublicProfile,
		Personality:        s.Personality,
		KnownFacts:         s.KnownFacts,
		StressLevel:        s.StressLevel,
		TrustLevel:         s.TrustLevel,
		InterrogationCount: s.InterrogationCount,
		CreatedAt:          s.CreatedAt,
	}
}

func PublicList(list []Suspect) []PublicSuspect {
	out := make([]PublicSuspect, 0, len(list))
	for i := range list {
		out = append(out, list[i].Public())
	}
	return out
}
