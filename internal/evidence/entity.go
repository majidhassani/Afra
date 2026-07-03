package evidence

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

var ValidTypes = []string{
	"phone_record", "gps_location", "camera_footage", "fingerprint", "dna", "blood",
	"weapon", "message", "bank_transaction", "witness_statement", "lab_report",
	"map_marker", "document", "image", "audio",
}

// Evidence is the full internal entity. InternalTruth must never reach the
// client — always convert through Public().
type Evidence struct {
	ID          uuid.UUID
	CaseID      uuid.UUID
	Title       string
	Type        string
	Description string
	PublicData  json.RawMessage

	// Internal only — Truth Layer.
	InternalTruth json.RawMessage

	Discovered              bool
	Reliability             int
	RelatedSuspectIDs       json.RawMessage
	RelatedLocationIDs      json.RawMessage
	RelatedTimelineEventIDs json.RawMessage
	AttachmentURL           string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// PublicEvidence is the only evidence shape returned to clients.
type PublicEvidence struct {
	ID                 uuid.UUID       `json:"id"`
	CaseID             uuid.UUID       `json:"case_id"`
	Title              string          `json:"title"`
	Type               string          `json:"type"`
	Description        string          `json:"description"`
	PublicData         json.RawMessage `json:"public_data"`
	Discovered         bool            `json:"discovered"`
	Reliability        int             `json:"reliability"`
	RelatedSuspectIDs  json.RawMessage `json:"related_suspect_ids"`
	RelatedLocationIDs json.RawMessage `json:"related_location_ids"`
	AttachmentURL      string          `json:"attachment_url,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

func (e *Evidence) Public() PublicEvidence {
	return PublicEvidence{
		ID:                 e.ID,
		CaseID:             e.CaseID,
		Title:              e.Title,
		Type:               e.Type,
		Description:        e.Description,
		PublicData:         e.PublicData,
		Discovered:         e.Discovered,
		Reliability:        e.Reliability,
		RelatedSuspectIDs:  e.RelatedSuspectIDs,
		RelatedLocationIDs: e.RelatedLocationIDs,
		AttachmentURL:      e.AttachmentURL,
		CreatedAt:          e.CreatedAt,
	}
}

func PublicList(list []Evidence) []PublicEvidence {
	out := make([]PublicEvidence, 0, len(list))
	for i := range list {
		out = append(out, list[i].Public())
	}
	return out
}

// Inspection is one stored inspection of a piece of evidence.
type Inspection struct {
	ID         uuid.UUID `json:"id"`
	EvidenceID uuid.UUID `json:"evidence_id"`
	CaseID     uuid.UUID `json:"case_id"`
	Question   string    `json:"question"`
	Analysis   string    `json:"analysis"`
	CreatedAt  time.Time `json:"created_at"`
}
