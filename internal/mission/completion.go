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
