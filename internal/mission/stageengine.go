package mission

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"casemind/internal/character"
	"casemind/internal/gamemap"
	"casemind/internal/wallet"
)

// StageUpdate is the player-safe outcome of a stage evaluation pass: the full
// stage list with derived counters, plus everything that changed this pass.
// State-changing endpoints fold it into their response envelopes; the client
// uses it to raise reward popups, stage banners, and the suspect reveal.
type StageUpdate struct {
	Stages       []Stage `json:"stages"`
	CurrentStage *Stage  `json:"current_stage,omitempty"`
	StageIndex   int     `json:"stage_index"` // 1-based, 0 when all done
	StageCount   int     `json:"stage_count"`

	// Changes made by this pass (empty on a no-op evaluation).
	CompletedStages   []Stage           `json:"completed_stages"`
	ActivatedStage    *Stage            `json:"activated_stage,omitempty"`
	Rewards           []StageReward     `json:"rewards"`
	UnlockedLocations []StageUnlockLoc  `json:"unlocked_locations"`
	SuspectRevealed   bool              `json:"suspect_revealed"`
	Suspect           *character.PublicCharacter `json:"suspect,omitempty"`
	TimelineEvents    []string          `json:"timeline_events"`
}

// StageUnlockLoc mirrors progression.UnlockedLoc without importing it.
type StageUnlockLoc struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Reason string    `json:"reason"`
}

// Changed reports whether the pass transitioned any stage.
func (u *StageUpdate) Changed() bool {
	return len(u.CompletedStages) > 0 || u.ActivatedStage != nil
}

// stageCounters are the cumulative mission-wide numbers stage requirements
// are checked against.
type stageCounters struct {
	visitedLocations int
	discoveredClues  int
	confirmedClues   int
	interviewedChars int
	missionCompleted bool
	acceptedReports  func(reportType string) int
}

// EvaluateStages runs the stage engine: it derives requirement progress from
// live gameplay state, completes the active stage when every requirement is
// met (cascading — one pass can complete several stages), grants each newly
// completed stage's reward exactly once, and persists transitions.
//
// The pass is idempotent: with no new player progress it changes nothing, so
// it is safe to run lazily from gameplay-status reads as well as from every
// state-changing action.
func (s *Service) EvaluateStages(ctx context.Context, userID, missionID uuid.UUID) (*StageUpdate, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	stages := m.ParsedStages()
	backfilled := false
	if len(stages) == 0 {
		stages = s.backfillStages(ctx, m)
		backfilled = true
	}

	counters, err := s.stageCounters(ctx, m)
	if err != nil {
		return nil, err
	}

	update := &StageUpdate{
		CompletedStages:   []Stage{},
		Rewards:           []StageReward{},
		UnlockedLocations: []StageUnlockLoc{},
		TimelineEvents:    []string{},
	}

	changed := backfilled
	for i := range stages {
		st := &stages[i]
		deriveStageProgress(st, counters)
		if st.Status != StageStatusActive {
			continue
		}
		if st.Progress < 100 {
			break // the active stage gates everything after it
		}
		// Transition: complete this stage, activate the next.
		st.Status = StageStatusCompleted
		changed = true
		update.CompletedStages = append(update.CompletedStages, *st)
		s.applyStageReward(ctx, userID, missionID, *st, update)
		s.applyStageUnlocks(ctx, missionID, *st, update)
		if i+1 < len(stages) {
			next := &stages[i+1]
			next.Status = StageStatusActive
			deriveStageProgress(next, counters)
			activated := *next
			update.ActivatedStage = &activated
			s.recorder.Emit(ctx, missionID, "stage_unlocked", map[string]any{
				"stage_id": next.ID, "title": next.Title,
			})
			update.TimelineEvents = append(update.TimelineEvents, "stage_unlocked")
		}
	}

	if changed {
		raw, err := json.Marshal(stages)
		if err == nil {
			if err := s.repo.UpdateStages(ctx, missionID, raw); err != nil {
				s.log.Error("persist stages", "error", err)
			}
		}
	}

	update.Stages = stages
	update.StageCount = len(stages)
	if cur := CurrentStage(stages); cur != nil {
		c := *cur
		update.CurrentStage = &c
		update.StageIndex = StageIndex(stages, cur.ID)
	}
	if !update.SuspectRevealed {
		// Reveal already happened in an earlier pass? Re-expose the suspect so
		// gameplay-status stays stable across reads.
		if identifyCompleted(stages) {
			if suspect := s.pickSuspect(ctx, missionID); suspect != nil {
				pub := suspect.Public()
				update.Suspect = &pub
			}
		}
	}
	return update, nil
}

// backfillStages builds and persists the default template for missions
// generated before the stage system existed.
func (s *Service) backfillStages(ctx context.Context, m *Mission) []Stage {
	_, totalClues, err := s.clues.Counts(ctx, m.ID)
	if err != nil {
		totalClues = 0
	}
	characters, err := s.characters.ListByMission(ctx, m.ID)
	if err != nil {
		characters = nil
	}
	_, totalLocations, err := s.locations.CountVisited(ctx, m.ID)
	if err != nil {
		totalLocations = 0
	}
	stages := DefaultStages(DefaultStageInputs{
		MissionType:    m.Type,
		Difficulty:     m.Difficulty,
		TotalClues:     totalClues,
		ClueTarget:     MandatoryClueTarget(m.ParsedObjectives()),
		TotalChars:     nonGuideCount(characters),
		TotalLocations: totalLocations,
	})
	return stages
}

// stageCounters gathers the cumulative numbers requirements check against.
func (s *Service) stageCounters(ctx context.Context, m *Mission) (stageCounters, error) {
	discovered, _, err := s.clues.Counts(ctx, m.ID)
	if err != nil {
		return stageCounters{}, err
	}
	confirmed, err := s.clues.CountConfirmed(ctx, m.ID)
	if err != nil {
		return stageCounters{}, err
	}
	visited, _, err := s.locations.CountVisited(ctx, m.ID)
	if err != nil {
		return stageCounters{}, err
	}
	interacted, err := s.interactions.CountInteractedCharacters(ctx, m.ID)
	if err != nil {
		return stageCounters{}, err
	}
	missionID := m.ID
	return stageCounters{
		visitedLocations: visited,
		discoveredClues:  discovered,
		confirmedClues:   confirmed,
		interviewedChars: interacted,
		missionCompleted: m.Status == StatusCompleted || m.Status == StatusFailed,
		acceptedReports: func(reportType string) int {
			n, err := s.repo.CountAcceptedReports(ctx, missionID, reportType)
			if err != nil {
				s.log.Error("count accepted reports", "error", err)
				return 0
			}
			return n
		},
	}, nil
}

// deriveStageProgress fills the derived counters of one stage in place.
func deriveStageProgress(st *Stage, c stageCounters) {
	if st.Status == StageStatusLocked {
		st.Progress = 0
		return
	}
	if st.Status == StageStatusCompleted {
		st.Progress = 100
		st.FoundClueCount = min(c.discoveredClues, max(st.RequiredClueCount, c.discoveredClues))
		for i := range st.RequiredActions {
			st.RequiredActions[i].Done = st.RequiredActions[i].Count
		}
		return
	}
	st.FoundClueCount = min(c.discoveredClues, st.RequiredClueCount)
	if st.RequiredClueCount == 0 {
		st.FoundClueCount = c.discoveredClues
	}
	totalNeed, totalDone := 0, 0
	for i := range st.RequiredActions {
		a := &st.RequiredActions[i]
		var have int
		switch a.Type {
		case StageActionVisitLocations:
			have = c.visitedLocations
		case StageActionFindClues:
			have = c.discoveredClues
		case StageActionConfirmEvidence:
			have = c.confirmedClues
		case StageActionInterviewCharacters:
			have = c.interviewedChars
		case StageActionSubmitReport:
			have = c.acceptedReports(a.Report)
		case StageActionFinalDecision:
			if c.missionCompleted {
				have = 1
			}
		}
		a.Done = min(have, a.Count)
		totalNeed += a.Count
		totalDone += a.Done
	}
	if totalNeed == 0 {
		// A stage with no requirements (debrief) completes with the mission.
		if c.missionCompleted {
			st.Progress = 100
		} else {
			st.Progress = 0
		}
		return
	}
	st.Progress = totalDone * 100 / totalNeed
}

// applyStageReward grants the reward exactly once (transition-time only),
// records it on the timeline, and folds it into the update.
func (s *Service) applyStageReward(ctx context.Context, userID, missionID uuid.UUID, st Stage, update *StageUpdate) {
	r := st.Reward
	if r.XP > 0 {
		if err := s.profiles.ApplyMissionResult(ctx, userID, false, r.XP); err != nil {
			s.log.Error("grant stage xp", "error", err)
		}
	}
	if r.Coins > 0 {
		if _, err := s.walletSvc.Credit(ctx, userID, &missionID, wallet.TxMissionReward, r.Coins,
			map[string]any{"stage_id": st.ID, "reason": "stage_reward"}); err != nil {
			s.log.Error("grant stage coins", "error", err)
		}
	}
	if r.XP > 0 || r.Coins > 0 || r.Badge != "" {
		update.Rewards = append(update.Rewards, r)
	}
	s.recorder.Emit(ctx, missionID, "stage_completed", map[string]any{
		"stage_id": st.ID, "title": st.Title,
		"xp": r.XP, "coins": r.Coins, "badge": r.Badge,
	})
	update.TimelineEvents = append(update.TimelineEvents, "stage_completed")
}

// applyStageUnlocks opens the next locked location and/or reveals the suspect.
func (s *Service) applyStageUnlocks(ctx context.Context, missionID uuid.UUID, st Stage, update *StageUpdate) {
	if st.UnlockOnComplete.UnlockLocation {
		if loc := s.unlockNextLocation(ctx, missionID, "Unlocked by completing stage: "+st.Title); loc != nil {
			update.UnlockedLocations = append(update.UnlockedLocations, *loc)
			update.TimelineEvents = append(update.TimelineEvents, "location_unlocked")
		}
	}
	if st.UnlockOnComplete.RevealSuspect {
		if suspect := s.pickSuspect(ctx, missionID); suspect != nil {
			pub := suspect.Public()
			update.Suspect = &pub
			update.SuspectRevealed = true
			s.recorder.Emit(ctx, missionID, "suspect_identified", map[string]any{
				"character_id": suspect.ID, "name": suspect.Name, "role": suspect.Role,
			})
			update.TimelineEvents = append(update.TimelineEvents, "suspect_identified")
		}
	}
}

// unlockNextLocation opens the earliest still-locked location, if any.
func (s *Service) unlockNextLocation(ctx context.Context, missionID uuid.UUID, reason string) *StageUnlockLoc {
	locked, err := s.locations.ListLocked(ctx, missionID)
	if err != nil || len(locked) == 0 {
		return nil
	}
	next := locked[0]
	if err := s.locations.UpdateStatus(ctx, next.ID, gamemap.StatusDiscovered); err != nil {
		s.log.Error("stage unlock location", "error", err)
		return nil
	}
	s.recorder.Emit(ctx, missionID, "location_unlocked", map[string]any{
		"location_id": next.ID, "name": next.Name, "reason": reason,
	})
	return &StageUnlockLoc{ID: next.ID, Name: next.Name, Reason: reason}
}

// pickSuspect deterministically selects the mission's prime suspect: the
// first antagonist-category character, else the first non-guide. Guilt is
// never asserted — the reveal only names a prime suspect; the JudgeAgent
// still decides the ending.
func (s *Service) pickSuspect(ctx context.Context, missionID uuid.UUID) *character.Character {
	chars, err := s.characters.ListByMission(ctx, missionID)
	if err != nil {
		s.log.Error("list characters for suspect pick", "error", err)
		return nil
	}
	var fallback *character.Character
	for i := range chars {
		switch chars[i].Category {
		case character.CategoryAntagonist:
			return &chars[i]
		case character.CategoryGuide:
			continue
		default:
			if fallback == nil {
				fallback = &chars[i]
			}
		}
	}
	return fallback
}

// identifyCompleted reports whether the suspect-reveal stage is done.
func identifyCompleted(stages []Stage) bool {
	for i := range stages {
		if stages[i].UnlockOnComplete.RevealSuspect {
			return stages[i].Status == StageStatusCompleted
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
