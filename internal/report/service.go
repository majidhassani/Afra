package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"casemind/internal/clue"
	"casemind/internal/mission"
	"casemind/internal/missionevent"
	"casemind/pkg/validator"
)

// MissionGateway is the slice of the mission module the report center needs.
// Implemented by mission.Service.
type MissionGateway interface {
	EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error
	EnsureOwnedActive(ctx context.Context, userID, missionID uuid.UUID) error
	EvaluateStages(ctx context.Context, userID, missionID uuid.UUID) (*mission.StageUpdate, error)
	ApplyActionTime(ctx context.Context, missionID uuid.UUID, action string) (*missionevent.TimeUpdate, error)
	ReadinessOf(ctx context.Context, userID, missionID uuid.UUID) (bool, []string, error)
}

type Service struct {
	repo     Repository
	clues    clue.Repository
	missions MissionGateway
	recorder *missionevent.Recorder
	log      *slog.Logger
}

func NewService(repo Repository, clues clue.Repository, missions MissionGateway, recorder *missionevent.Recorder, log *slog.Logger) *Service {
	return &Service{repo: repo, clues: clues, missions: missions, recorder: recorder, log: log}
}

// Submission is the player's report input.
type Submission struct {
	Type               string
	Title              string
	Summary            string
	LinkedClueIDs      []uuid.UUID
	SuspectCharacterID *uuid.UUID
}

// Result is the uniform report-center response: the verdict plus every state
// change the submission caused, so the client can react in one pass
// (REPORT ACCEPTED → rewards → stage completed → timeline updated → unlocks).
type Result struct {
	Verdict             string                   `json:"verdict"`
	Feedback            string                   `json:"feedback"`
	MissingRequirements []string                 `json:"missing_requirements"`
	Report              *Report                  `json:"report"`
	StageUpdate         *mission.StageUpdate     `json:"stage_update,omitempty"`
	TimeUpdate          *missionevent.TimeUpdate `json:"time_update,omitempty"`
	TimelineEvents      []string                 `json:"timeline_events"`
	NextActions         []NextAction             `json:"next_recommended_actions"`
}

type NextAction struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

// Submit validates the report deterministically, persists it with its
// verdict, charges the mission-time cost, and runs the stage engine so an
// accepted report immediately advances the mission.
func (s *Service) Submit(ctx context.Context, userID, missionID uuid.UUID, sub Submission) (*Result, error) {
	if !validType(sub.Type) {
		return nil, invalidTypeErr()
	}
	if err := validator.New().
		Required("summary", sub.Summary).MaxLen("summary", sub.Summary, 4000).
		MaxLen("title", sub.Title, 200).
		Err(); err != nil {
		return nil, err
	}
	if err := s.missions.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}

	verdict, feedback, missing := s.judge(ctx, userID, missionID, sub)

	linked, _ := json.Marshal(idStrings(sub.LinkedClueIDs))
	rep := &Report{
		MissionID:          missionID,
		Type:               sub.Type,
		Title:              sub.Title,
		Summary:            sub.Summary,
		LinkedClueIDs:      linked,
		SuspectCharacterID: sub.SuspectCharacterID,
		Verdict:            verdict,
		Feedback:           feedback,
	}
	if err := s.repo.Insert(ctx, rep, userID); err != nil {
		return nil, err
	}

	eventType := "report_rejected"
	if verdict == VerdictAccepted {
		eventType = "report_accepted"
	}
	s.recorder.Emit(ctx, missionID, "report_submitted", map[string]any{
		"report_id": rep.ID, "type": rep.Type, "title": rep.Title,
	})
	s.recorder.Emit(ctx, missionID, eventType, map[string]any{
		"report_id": rep.ID, "type": rep.Type, "feedback": feedback,
	})

	res := &Result{
		Verdict:             verdict,
		Feedback:            feedback,
		MissingRequirements: missing,
		Report:              rep,
		TimelineEvents:      []string{"report_submitted", eventType},
		NextActions:         []NextAction{},
	}

	// Filing a report takes time, accepted or not.
	if tu, err := s.missions.ApplyActionTime(ctx, missionID, mission.ActionReportSubmit); err == nil {
		res.TimeUpdate = tu
	} else {
		s.log.Error("report time cost", "error", err)
	}

	// Stage engine pass: an accepted report may complete the active stage.
	if su, err := s.missions.EvaluateStages(ctx, userID, missionID); err == nil {
		if su.Changed() {
			res.StageUpdate = su
			res.TimelineEvents = append(res.TimelineEvents, su.TimelineEvents...)
		}
		res.NextActions = append(res.NextActions, nextActions(su, verdict, sub.Type)...)
	} else {
		s.log.Error("report stage evaluation", "error", err)
	}
	return res, nil
}

// judge computes the deterministic verdict. It never consults hidden truth:
// acceptance is about process (enough evidence linked, right stage), so a
// "wrong" theory is caught later by the final JudgeAgent, not here.
func (s *Service) judge(ctx context.Context, userID, missionID uuid.UUID, sub Submission) (verdict, feedback string, missing []string) {
	missing = []string{}
	switch sub.Type {
	case TypeProgress:
		return VerdictAccepted, "Progress report filed. Command has an up-to-date picture of your mission.", missing
	case TypeIncident:
		return VerdictAccepted, "Incident report filed. The event is now on the official record.", missing

	case TypeClue:
		linked := s.countLinkedDiscovered(ctx, missionID, sub.LinkedClueIDs)
		if linked == 0 {
			missing = append(missing, "Link at least 1 discovered clue to the report")
			return VerdictRejected, "A clue report must document actual evidence — link at least one discovered clue.", missing
		}
		return VerdictAccepted, fmt.Sprintf("Clue report accepted: %d piece(s) of evidence are now on record.", linked), missing

	case TypeSuspect:
		confirmed := s.countLinkedConfirmed(ctx, missionID, sub.LinkedClueIDs)
		const need = 2
		if sub.SuspectCharacterID == nil {
			missing = append(missing, "Name a suspect character")
		}
		if confirmed < need {
			missing = append(missing, fmt.Sprintf("Link %d more confirmed piece(s) of evidence", need-confirmed))
		}
		if len(missing) > 0 {
			return VerdictRejected,
				"A suspect report needs a named suspect backed by confirmed evidence. " + strings.Join(missing, "; ") + ".",
				missing
		}
		return VerdictAccepted, "Suspect report accepted. Your named suspect is now the official line of investigation.", missing

	case TypeFinal:
		ready, reqMissing, err := s.missions.ReadinessOf(ctx, userID, missionID)
		if err != nil {
			s.log.Error("final report readiness", "error", err)
			return VerdictRejected, "Could not evaluate mission readiness — try again.", missing
		}
		if !ready {
			return VerdictRejected,
				"The mission is not ready for a final report. Missing: " + strings.Join(reqMissing, "; ") + ".",
				reqMissing
		}
		return VerdictAccepted,
			"Final report accepted. Submit your final decision to close the mission — command is waiting.",
			missing
	}
	return VerdictRejected, "Unknown report type.", missing
}

func nextActions(su *mission.StageUpdate, verdict, reportType string) []NextAction {
	actions := []NextAction{}
	if verdict == VerdictAccepted && reportType == TypeFinal {
		actions = append(actions, NextAction{
			Type: "submit_final_decision", Title: "Submit your final decision",
		})
		return actions
	}
	if su != nil && su.CurrentStage != nil {
		for _, a := range su.CurrentStage.RequiredActions {
			if a.Done < a.Count {
				actions = append(actions, NextAction{Type: a.Type, Title: a.Label()})
				break
			}
		}
	}
	return actions
}

func (s *Service) countLinkedDiscovered(ctx context.Context, missionID uuid.UUID, ids []uuid.UUID) int {
	n := 0
	for _, id := range ids {
		c, err := s.clues.GetByID(ctx, missionID, id)
		if err == nil && c.Discovered {
			n++
		}
	}
	return n
}

func (s *Service) countLinkedConfirmed(ctx context.Context, missionID uuid.UUID, ids []uuid.UUID) int {
	n := 0
	for _, id := range ids {
		c, err := s.clues.GetByID(ctx, missionID, id)
		if err == nil && c.LifecycleStatus() == clue.StatusConfirmed {
			n++
		}
	}
	return n
}

// List returns the mission's report history, newest first. Ownership only —
// finished missions can still review their reports in the debrief.
func (s *Service) List(ctx context.Context, userID, missionID uuid.UUID, limit int) ([]Report, error) {
	if err := s.missions.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	return s.repo.ListByMission(ctx, missionID, limit)
}

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
