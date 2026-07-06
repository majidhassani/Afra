package missioncomplete

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"casemind/internal/agent/missionjudge"
	"casemind/internal/mission"
)

// buildResult turns the JudgeAgent verdict plus deterministic domain facts into
// the player-safe end-of-mission result.
func (s *Service) buildResult(ctx context.Context, missionID uuid.UUID, snap *mission.CompletionSnapshot, judged *missionjudge.Output, discoveredClues int) *Result {
	// Domain rules cap the score so a "success" is never a weak pass and a
	// failure never scores high.
	score := judged.Score
	if judged.Success && score < 60 {
		score = 60
	}
	if !judged.Success && score > 55 {
		score = 55
	}

	completedByKey := completedKeys(judged)
	titlesByKey := objectiveTitles(snap.Objectives)

	completed := []string{}
	failed := []string{}
	missedOptional := []string{}
	good := []string{}
	bad := []string{}
	for _, o := range snap.Objectives {
		title := titlesByKey[o.ID]
		if title == "" {
			title = o.Title
		}
		note := judgedNote(judged, o.ID)
		if completedByKey[o.ID] {
			completed = append(completed, title)
			if note != "" {
				good = append(good, note)
			}
			continue
		}
		if o.Mandatory() {
			failed = append(failed, title)
			if note != "" {
				bad = append(bad, note)
			}
		} else {
			missedOptional = append(missedOptional, title)
		}
	}

	found, missed := s.criticalClues(ctx, missionID)

	xp := mission.XPForDifficulty(snap.Difficulty) * score / 100
	coins := mission.CoinRewardForScore(score)

	return &Result{
		CanComplete:              true,
		Success:                  judged.Success,
		Score:                    score,
		Stars:                    starsForScore(score, judged.Success),
		ResultTitle:              resultTitle(judged.Success),
		ResultSummary:            judged.Feedback,
		CompletedObjectives:      completed,
		FailedObjectives:         failed,
		MissedOptionalObjectives: missedOptional,
		CriticalCluesFound:       found,
		CriticalCluesMissed:      missed,
		GoodDecisions:            good,
		BadDecisions:             bad,
		XPReward:                 xp,
		CoinReward:               coins,
	}
}

// criticalClues splits the mission's high-importance clues into those the
// player found and those they missed. Revealing missed titles is safe here: the
// mission is over and the JudgeAgent feedback already discloses the truth.
func (s *Service) criticalClues(ctx context.Context, missionID uuid.UUID) (found, missed []string) {
	found, missed = []string{}, []string{}
	critical, err := s.clues.ListCritical(ctx, missionID)
	if err != nil {
		s.log.Error("list critical clues", "error", err)
		return found, missed
	}
	for i := range critical {
		if critical[i].Discovered {
			found = append(found, critical[i].Title)
		} else {
			missed = append(missed, critical[i].Title)
		}
	}
	return found, missed
}

func starsForScore(score int, success bool) int {
	if !success {
		// A failed mission earns at most one star, and only for a near miss.
		if score >= 45 {
			return 1
		}
		return 0
	}
	switch {
	case score >= 90:
		return 5
	case score >= 75:
		return 4
	case score >= 60:
		return 3
	default:
		return 2
	}
}

func resultTitle(success bool) string {
	if success {
		return "Mission Successful"
	}
	return "Mission Failed"
}

func notReadyReason(canComplete bool, missing []string) string {
	if canComplete {
		return "The mission is ready for your final decision."
	}
	if len(missing) > 0 {
		return "You are not ready to finish: " + missing[0] + "."
	}
	return "You have not met the requirements to complete this mission yet."
}

func missionCompletedEvent(success bool) string {
	if success {
		return "mission_completed"
	}
	return "mission_failed"
}

// judgeObjectives maps stored objectives into the JudgeAgent input shape.
func judgeObjectives(objectives []mission.Objective) []missionjudge.ObjectiveState {
	out := make([]missionjudge.ObjectiveState, 0, len(objectives))
	for _, o := range objectives {
		out = append(out, missionjudge.ObjectiveState{
			Key: o.ID, Title: o.Title, RequiredClues: o.RequiredClues,
			Optional: !o.Mandatory(),
		})
	}
	return out
}

func objectiveTitles(objectives []mission.Objective) map[string]string {
	m := map[string]string{}
	for _, o := range objectives {
		m[o.ID] = o.Title
	}
	return m
}

// completedKeys collects the objective keys the judge marked complete.
func completedKeys(judged *missionjudge.Output) map[string]bool {
	keys := map[string]bool{}
	for _, r := range judged.ObjectiveResults {
		if r.Completed {
			keys[r.Key] = true
		}
	}
	return keys
}

func judgedNote(judged *missionjudge.Output, key string) string {
	for _, r := range judged.ObjectiveResults {
		if r.Key == key {
			return r.Note
		}
	}
	return ""
}

func rawToMap(raw json.RawMessage) map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func rawToStrings(raw json.RawMessage) []string {
	var s []string
	_ = json.Unmarshal(raw, &s)
	return s
}
