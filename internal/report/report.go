// Package report owns the mission report center: player-submitted reports
// validated deterministically by the backend. An accepted report is a real
// game move — it advances stages, unlocks content, costs mission time, and
// lands on the timeline. The AI never mutates state here and hidden truth is
// never consulted: every verdict is computed from player-visible state only.
package report

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

// Report types.
const (
	TypeClue     = "clue_report"
	TypeSuspect  = "suspect_report"
	TypeProgress = "progress_report"
	TypeIncident = "incident_report"
	TypeFinal    = "final_report"
)

// Verdicts.
const (
	VerdictAccepted = "accepted"
	VerdictRejected = "rejected"
)

var ValidTypes = []string{TypeClue, TypeSuspect, TypeProgress, TypeIncident, TypeFinal}

// Report is one submitted report with its verdict.
type Report struct {
	ID                 uuid.UUID       `json:"id"`
	MissionID          uuid.UUID       `json:"mission_id"`
	Type               string          `json:"type"`
	Title              string          `json:"title"`
	Summary            string          `json:"summary"`
	LinkedClueIDs      json.RawMessage `json:"linked_clue_ids"`
	SuspectCharacterID *uuid.UUID      `json:"suspect_character_id,omitempty"`
	Verdict            string          `json:"verdict"`
	Feedback           string          `json:"feedback"`
	CreatedAt          time.Time       `json:"created_at"`
}

type Repository interface {
	Insert(ctx context.Context, r *Report, userID uuid.UUID) error
	ListByMission(ctx context.Context, missionID uuid.UUID, limit int) ([]Report, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Insert(ctx context.Context, rep *Report, userID uuid.UUID) error {
	linked := rep.LinkedClueIDs
	if len(linked) == 0 {
		linked = []byte(`[]`)
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO mission_reports
		   (mission_id, user_id, type, title, summary, linked_clue_ids, suspect_character_id, verdict, feedback)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at`,
		rep.MissionID, userID, rep.Type, rep.Title, rep.Summary, linked,
		rep.SuspectCharacterID, rep.Verdict, rep.Feedback,
	).Scan(&rep.ID, &rep.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "insert mission report")
	}
	return nil
}

func (r *PGRepository) ListByMission(ctx context.Context, missionID uuid.UUID, limit int) ([]Report, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, mission_id, type, title, summary, linked_clue_ids, suspect_character_id,
		        verdict, feedback, created_at
		 FROM mission_reports WHERE mission_id = $1
		 ORDER BY created_at DESC LIMIT $2`, missionID, limit)
	if err != nil {
		return nil, apperrors.Internal(err, "list mission reports")
	}
	defer rows.Close()
	reports := []Report{}
	for rows.Next() {
		var rep Report
		if err := scanReport(rows, &rep); err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate mission reports")
	}
	return reports, nil
}

func scanReport(row pgx.Row, rep *Report) error {
	err := row.Scan(&rep.ID, &rep.MissionID, &rep.Type, &rep.Title, &rep.Summary,
		&rep.LinkedClueIDs, &rep.SuspectCharacterID, &rep.Verdict, &rep.Feedback, &rep.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "scan mission report")
	}
	return nil
}

func validType(t string) bool {
	for _, v := range ValidTypes {
		if v == t {
			return true
		}
	}
	return false
}

func invalidTypeErr() error {
	return apperrors.Invalid("invalid_report_type",
		fmt.Sprintf("type must be one of: %s, %s, %s, %s, %s",
			TypeClue, TypeSuspect, TypeProgress, TypeIncident, TypeFinal))
}
