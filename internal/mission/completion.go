package mission

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	apperrors "casemind/pkg/errors"
)

// CompletionSnapshot is the player-safe mission state the completion flow needs
// to gate and judge a final decision. It is assembled by the mission module so
// the completion service never has to reach into mission internals.
type CompletionSnapshot struct {
	Status             string
	Type               string
	Difficulty         string
	Summary            string
	Objectives         []Objective
	WinConditions      []string
	FailureConditions  []string
	DeadlineMinutes    int
	RiskLevel          int
	ElapsedMinutes     int
	RequiredClueTarget int
}

// CompletionSnapshot returns the state needed to evaluate mission completion.
// It enforces ownership and that the mission is actually in play.
func (s *Service) CompletionSnapshot(ctx context.Context, userID, missionID uuid.UUID) (*CompletionSnapshot, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	if m.Status != StatusActive && m.Status != StatusReady {
		return nil, apperrors.Conflict("mission_not_active",
			"this mission is not in progress (status: "+m.Status+")")
	}
	objectives := m.ParsedObjectives()
	public := parsePublicState(m.PublicState)
	return &CompletionSnapshot{
		Status:             m.Status,
		Type:               m.Type,
		Difficulty:         m.Difficulty,
		Summary:            m.Summary,
		Objectives:         objectives,
		WinConditions:      public.WinConditions,
		FailureConditions:  public.FailureConditions,
		DeadlineMinutes:    public.DeadlineMinutes,
		RiskLevel:          public.RiskLevel,
		ElapsedMinutes:     MinutesSinceStart(m.MissionTime),
		RequiredClueTarget: MandatoryClueTarget(objectives),
	}, nil
}

// ReadinessOf reports whether the mission can be completed and what is still
// missing — the same gate the completion flow uses, exposed for the report
// center so a rejected final report can explain exactly what is lacking.
func (s *Service) ReadinessOf(ctx context.Context, userID, missionID uuid.UUID) (bool, []string, error) {
	snap, err := s.CompletionSnapshot(ctx, userID, missionID)
	if err != nil {
		return false, nil, err
	}
	discovered, _, err := s.clues.Counts(ctx, missionID)
	if err != nil {
		return false, nil, err
	}
	visited, _, err := s.locations.CountVisited(ctx, missionID)
	if err != nil {
		return false, nil, err
	}
	interacted, err := s.interactions.CountInteractedCharacters(ctx, missionID)
	if err != nil {
		return false, nil, err
	}
	can, missing := EvaluateReadiness(ReadinessInputs{
		DiscoveredClues:    discovered,
		InteractedChars:    interacted,
		VisitedLocations:   visited,
		RequiredClueTarget: snap.RequiredClueTarget,
	})
	return can, missing, nil
}

// StoredResult returns the persisted end-of-mission result document for a
// completed or failed mission. It enforces ownership and reports a conflict
// when the mission has not reached a terminal state yet (no result to show).
func (s *Service) StoredResult(ctx context.Context, userID, missionID uuid.UUID) (json.RawMessage, string, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, "", err
	}
	if m.Status != StatusCompleted && m.Status != StatusFailed {
		return nil, m.Status, apperrors.Conflict("mission_not_finished",
			"this mission has no result yet (status: "+m.Status+")")
	}
	if len(m.Result) == 0 {
		return nil, m.Status, apperrors.NotFound("result_not_found", "mission result is unavailable")
	}
	return m.Result, m.Status, nil
}

// Finish records the terminal mission outcome: it resolves each objective's
// status from the judged per-objective results, stores the result document,
// and moves the mission to completed or failed.
func (s *Service) Finish(ctx context.Context, userID, missionID uuid.UUID, result json.RawMessage, success bool, completedKeys map[string]bool) error {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return err
	}
	objectives := m.ParsedObjectives()
	for i := range objectives {
		o := &objectives[i]
		switch {
		case completedKeys[o.ID]:
			o.Status = ObjStatusCompleted
			o.Progress = 100
		case o.Mandatory():
			o.Status = ObjStatusFailed
		default:
			o.Status = ObjStatusSkipped
		}
	}
	if err := s.repo.UpdateObjectives(ctx, missionID, mustJSONRaw(objectives, `[]`)); err != nil {
		return err
	}
	status := StatusFailed
	if success {
		status = StatusCompleted
	}
	return s.repo.SetResult(ctx, missionID, result, status)
}
