package cases

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"casemind/internal/agent/casegenerator"
	mapagent "casemind/internal/agent/map"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	timelineagent "casemind/internal/agent/timeline"
	"casemind/internal/casebible"
	"casemind/internal/evidence"
	"casemind/internal/history"
	"casemind/internal/location"
	"casemind/internal/suspect"
	"casemind/internal/timeline"
	apperrors "casemind/pkg/errors"
)

// Generator runs the multi-agent case generation pipeline and persists the
// validated result: Case Bible (private), suspects, evidence, locations,
// real timeline, and initial discoveries.
type Generator struct {
	orch      *orchestrator.Orchestrator
	cases     Repository
	bibles    casebible.Repository
	suspects  suspect.Repository
	evidences evidence.Repository
	locations location.Repository
	timelines timeline.Repository
	recorder  *history.Recorder
	log       *slog.Logger
}

func NewGenerator(
	orch *orchestrator.Orchestrator,
	cases Repository,
	bibles casebible.Repository,
	suspects suspect.Repository,
	evidences evidence.Repository,
	locations location.Repository,
	timelines timeline.Repository,
	recorder *history.Recorder,
	log *slog.Logger,
) *Generator {
	return &Generator{
		orch: orch, cases: cases, bibles: bibles, suspects: suspects,
		evidences: evidences, locations: locations, timelines: timelines,
		recorder: recorder, log: log,
	}
}

// Run generates the full case content. On success the case becomes `open`;
// on failure it becomes `failed`. Events are emitted either way.
func (g *Generator) Run(ctx context.Context, c *Case, languages ...string) {
	language := "en"
	if len(languages) > 0 {
		language = NormalizeLanguage(languages[0])
	}
	g.recorder.Emit(ctx, c.ID, "case_generation_started", map[string]any{
		"case_id": c.ID, "type": c.Type, "difficulty": c.Difficulty, "language": language,
	})
	if err := g.generate(ctx, c, language); err != nil {
		g.log.Error("case generation failed", "case_id", c.ID, "error", err)
		if uerr := g.cases.UpdateStatus(ctx, c.ID, StatusFailed); uerr != nil {
			g.log.Error("mark case failed", "case_id", c.ID, "error", uerr)
		}
		g.recorder.Emit(ctx, c.ID, "case_generation_failed", map[string]any{
			"case_id": c.ID, "reason": "generation error",
		})
		return
	}
	if err := g.cases.UpdateStatus(ctx, c.ID, StatusOpen); err != nil {
		g.log.Error("mark case open", "case_id", c.ID, "error", err)
		return
	}
	g.recorder.Emit(ctx, c.ID, "case_ready", map[string]any{"case_id": c.ID})
}

func (g *Generator) generate(ctx context.Context, c *Case, language string) error {
	// 1. Core bible, suspects, evidence.
	coreAny, err := g.orch.Run(ctx, casegenerator.Name, runtime.Task{
		Type:   casegenerator.TaskType,
		CaseID: c.ID,
		UserID: c.UserID,
		Input:  casegenerator.Input{CaseType: c.Type, Difficulty: c.Difficulty, Language: language},
	})
	if err != nil {
		return err
	}
	core := coreAny.(*casegenerator.Output)

	// 2. Real timeline from the bible.
	culpritKey := ""
	suspectRefs := make([]timelineagent.SuspectRef, 0, len(core.Suspects))
	for _, s := range core.Suspects {
		suspectRefs = append(suspectRefs, timelineagent.SuspectRef{Key: s.Key, Name: s.Name})
		if s.IsCulprit {
			culpritKey = s.Key
		}
	}
	tlAny, err := g.orch.Run(ctx, timelineagent.Name, runtime.Task{
		Type:   timelineagent.TaskType,
		CaseID: c.ID,
		UserID: c.UserID,
		Input: timelineagent.Input{
			CaseSummary: core.Summary, Motive: core.Motive, Truth: core.Truth,
			Suspects: suspectRefs, CulpritKey: culpritKey, FalseLeads: core.FalseLeads, Language: language,
		},
	})
	if err != nil {
		return err
	}
	tl := tlAny.(*timelineagent.Output)

	// 3. Locations from the evidence layout.
	evRefs := make([]mapagent.EvidenceRef, 0, len(core.Evidence))
	suspectKeys := make([]string, 0, len(core.Suspects))
	for _, s := range core.Suspects {
		suspectKeys = append(suspectKeys, s.Key)
	}
	for _, e := range core.Evidence {
		evRefs = append(evRefs, mapagent.EvidenceRef{Key: e.Key, Title: e.Title, LocationKey: e.LocationKey})
	}
	mpAny, err := g.orch.Run(ctx, mapagent.Name, runtime.Task{
		Type:   mapagent.TaskType,
		CaseID: c.ID,
		UserID: c.UserID,
		Input:  mapagent.Input{CaseSummary: core.Summary, Evidence: evRefs, SuspectKeys: suspectKeys, Language: language},
	})
	if err != nil {
		return err
	}
	mp := mpAny.(*mapagent.Output)

	// 4. Persist suspects; build key -> id map and find the culprit id.
	suspectIDs := map[string]uuid.UUID{}
	var culpritID uuid.UUID
	for _, gs := range core.Suspects {
		s := &suspect.Suspect{
			CaseID:           c.ID,
			Name:             gs.Name,
			Age:              gs.Age,
			Job:              gs.Job,
			RelationToVictim: gs.RelationToVictim,
			PublicProfile:    mustJSON(gs.PublicProfile, `{}`),
			Personality:      mustJSON(gs.Personality, `{}`),
			KnownFacts:       mustJSON(gs.KnownFacts, `[]`),
			StressLevel:      10,
			TrustLevel:       50,
			IsCulprit:        gs.IsCulprit,
			Secrets:          mustJSON(gs.Secrets, `[]`),
			LieProfile:       mustJSON(gs.LieProfile, `{}`),
			PrivateMemory:    []byte(`[]`),
		}
		if err := g.suspects.Create(ctx, s); err != nil {
			return err
		}
		suspectIDs[gs.Key] = s.ID
		if gs.IsCulprit {
			culpritID = s.ID
		}
	}
	if culpritID == uuid.Nil {
		return apperrors.Internal(nil, "generated case has no culprit")
	}

	// 5. Persist locations; key -> id map.
	locationIDs := map[string]uuid.UUID{}
	for _, gl := range mp.Locations {
		l := &location.Location{
			CaseID:            c.ID,
			Name:              gl.Name,
			Type:              gl.Type,
			Latitude:          gl.Latitude,
			Longitude:         gl.Longitude,
			Description:       gl.Description,
			Discovered:        gl.Discovered,
			RelatedSuspectIDs: idsJSON(gl.RelatedSuspectKeys, suspectIDs),
		}
		if err := g.locations.Create(ctx, l); err != nil {
			return err
		}
		locationIDs[gl.Key] = l.ID
	}

	// 6. Persist real timeline events.
	timelineIDs := map[string]uuid.UUID{}
	for _, ge := range tl.Events {
		e := &timeline.RealEvent{
			CaseID:       c.ID,
			OccurredAt:   ge.OccurredAt,
			Title:        ge.Title,
			Description:  ge.Description,
			Participants: idsJSON(ge.ParticipantKeys, suspectIDs),
			Revealed:     ge.Revealed,
		}
		if err := g.timelines.CreateReal(ctx, e); err != nil {
			return err
		}
		timelineIDs[ge.Key] = e.ID
	}

	// 7. Persist evidence, resolving suspect and location keys.
	evidenceTruth := map[string]any{}
	initialEvidence := []string{}
	for _, ge := range core.Evidence {
		relLoc := []uuid.UUID{}
		if id, ok := locationIDs[ge.LocationKey]; ok {
			relLoc = append(relLoc, id)
		}
		reliability := ge.Reliability
		if reliability <= 0 || reliability > 100 {
			reliability = 50
		}
		e := &evidence.Evidence{
			CaseID:             c.ID,
			Title:              ge.Title,
			Type:               ge.Type,
			Description:        ge.Description,
			PublicData:         mustJSON(ge.PublicData, `{}`),
			InternalTruth:      mustJSON(ge.InternalTruth, `{}`),
			Discovered:         ge.Discovered,
			Reliability:        reliability,
			RelatedSuspectIDs:  idsJSON(ge.RelatedSuspectKeys, suspectIDs),
			RelatedLocationIDs: mustJSON(relLoc, `[]`),
		}
		if err := g.evidences.Create(ctx, e); err != nil {
			return err
		}
		evidenceTruth[e.ID.String()] = ge.InternalTruth
		if ge.Discovered {
			initialEvidence = append(initialEvidence, ge.Title)
		}
	}

	// 8. Persist the private Case Bible.
	suspectSecrets := map[string]any{}
	for _, gs := range core.Suspects {
		suspectSecrets[suspectIDs[gs.Key].String()] = map[string]any{
			"secrets": gs.Secrets, "lie_profile": gs.LieProfile,
		}
	}
	realTimeline := make([]map[string]any, 0, len(tl.Events))
	for _, ge := range tl.Events {
		realTimeline = append(realTimeline, map[string]any{
			"id": timelineIDs[ge.Key].String(), "occurred_at": ge.OccurredAt,
			"title": ge.Title, "description": ge.Description,
		})
	}
	bible := &casebible.Bible{
		CaseID:         c.ID,
		CulpritID:      culpritID,
		Motive:         core.Motive,
		Truth:          mustJSON(core.Truth, `{}`),
		RealTimeline:   mustJSON(realTimeline, `[]`),
		HiddenFacts:    mustJSON(core.HiddenFacts, `[]`),
		FalseLeads:     mustJSON(core.FalseLeads, `[]`),
		EvidenceTruth:  mustJSON(evidenceTruth, `{}`),
		SuspectSecrets: mustJSON(suspectSecrets, `{}`),
	}
	if err := g.bibles.Create(ctx, bible); err != nil {
		return err
	}

	// 9. Player-facing case content.
	if err := g.cases.SetGeneratedContent(ctx, c.ID, core.Title, core.Summary); err != nil {
		return err
	}
	c.Title = core.Title
	c.Summary = core.Summary
	g.recorder.Emit(ctx, c.ID, "initial_evidence_available", map[string]any{
		"evidence_titles": initialEvidence,
	})
	return nil
}

func mustJSON(v any, fallback string) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return []byte(fallback)
	}
	return b
}

func idsJSON(keys []string, mapping map[string]uuid.UUID) json.RawMessage {
	ids := []string{}
	for _, k := range keys {
		if id, ok := mapping[k]; ok {
			ids = append(ids, id.String())
		}
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return []byte(`[]`)
	}
	return b
}
