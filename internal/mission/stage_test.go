package mission

import "testing"

func TestDefaultStagesTemplate(t *testing.T) {
	stages := DefaultStages(DefaultStageInputs{
		MissionType: "detective",
		Difficulty:  "medium",
		TotalClues:  6,
		ClueTarget:  5,
		TotalChars:  3,
	})
	if len(stages) != 6 {
		t.Fatalf("expected 6 stages, got %d", len(stages))
	}
	if stages[0].ID != StageArrival || stages[0].Status != StageStatusActive {
		t.Fatalf("stage 1 must be arrival and active, got %s/%s", stages[0].ID, stages[0].Status)
	}
	for i := 1; i < len(stages); i++ {
		if stages[i].Status != StageStatusLocked {
			t.Fatalf("stage %d (%s) must start locked", i+1, stages[i].ID)
		}
	}
	identify := stages[2]
	if identify.ID != StageIdentifySuspect || !identify.UnlockOnComplete.RevealSuspect {
		t.Fatalf("stage 3 must reveal the suspect")
	}
	if identify.RequiredClueCount != 5 {
		t.Fatalf("identify stage clue target = %d, want 5", identify.RequiredClueCount)
	}
	foundReport := false
	for _, a := range identify.RequiredActions {
		if a.Type == StageActionSubmitReport && a.Report == "suspect_report" {
			foundReport = true
		}
	}
	if !foundReport {
		t.Fatalf("identify stage must require a suspect report")
	}
	if stages[4].ID != StageFinalReport || stages[4].RequiredActions[0].Type != StageActionFinalDecision {
		t.Fatalf("stage 5 must require the final decision")
	}
}

func TestDefaultStagesScalesToTinyMissions(t *testing.T) {
	stages := DefaultStages(DefaultStageInputs{
		MissionType: "wildlife_rescue",
		TotalClues:  1,
		TotalChars:  0,
	})
	identify := stages[2]
	if identify.RequiredClueCount != 1 {
		t.Fatalf("clue target must clamp to total clues, got %d", identify.RequiredClueCount)
	}
	for _, a := range identify.RequiredActions {
		if a.Type == StageActionInterviewCharacters {
			t.Fatalf("no interview requirement when the mission has no characters")
		}
	}
	if identify.Title != "Identify the Cause" {
		t.Fatalf("non-detective identify title = %q", identify.Title)
	}
}

func TestDeriveStageProgress(t *testing.T) {
	c := stageCounters{
		visitedLocations: 1,
		discoveredClues:  2,
		confirmedClues:   1,
		interviewedChars: 1,
		acceptedReports:  func(string) int { return 0 },
	}
	st := Stage{
		ID: StageIdentifySuspect, Status: StageStatusActive, RequiredClueCount: 5,
		RequiredActions: []StageAction{
			{Type: StageActionFindClues, Count: 5},
			{Type: StageActionInterviewCharacters, Count: 2},
			{Type: StageActionSubmitReport, Report: "suspect_report", Count: 1},
		},
	}
	deriveStageProgress(&st, c)
	if st.FoundClueCount != 2 {
		t.Fatalf("found clues = %d, want 2", st.FoundClueCount)
	}
	if st.Progress != (2+1+0)*100/8 {
		t.Fatalf("progress = %d, want %d", st.Progress, (2+1+0)*100/8)
	}

	// All requirements met → 100.
	c.discoveredClues = 5
	c.interviewedChars = 2
	c.acceptedReports = func(rt string) int {
		if rt == "suspect_report" {
			return 1
		}
		return 0
	}
	deriveStageProgress(&st, c)
	if st.Progress != 100 {
		t.Fatalf("progress = %d, want 100", st.Progress)
	}

	// Locked stages never show progress.
	locked := Stage{Status: StageStatusLocked, RequiredActions: []StageAction{{Type: StageActionFindClues, Count: 2}}}
	deriveStageProgress(&locked, c)
	if locked.Progress != 0 {
		t.Fatalf("locked stage progress = %d, want 0", locked.Progress)
	}

	// Requirement-free stages complete with the mission.
	debrief := Stage{ID: StageDebrief, Status: StageStatusActive, RequiredActions: []StageAction{}}
	deriveStageProgress(&debrief, c)
	if debrief.Progress != 0 {
		t.Fatalf("debrief progress before completion = %d, want 0", debrief.Progress)
	}
	c.missionCompleted = true
	deriveStageProgress(&debrief, c)
	if debrief.Progress != 100 {
		t.Fatalf("debrief progress after completion = %d, want 100", debrief.Progress)
	}
}

func TestCurrentStageAndIndex(t *testing.T) {
	stages := []Stage{
		{ID: "a", Status: StageStatusCompleted},
		{ID: "b", Status: StageStatusActive},
		{ID: "c", Status: StageStatusLocked},
	}
	cur := CurrentStage(stages)
	if cur == nil || cur.ID != "b" {
		t.Fatalf("current stage = %v, want b", cur)
	}
	if StageIndex(stages, "b") != 2 {
		t.Fatalf("stage index = %d, want 2", StageIndex(stages, "b"))
	}
	all := []Stage{{ID: "a", Status: StageStatusCompleted}}
	if CurrentStage(all) != nil {
		t.Fatalf("all-completed missions have no current stage")
	}
}
