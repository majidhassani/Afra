package mission

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"casemind/internal/agent/characteragent"
	"casemind/internal/agent/clueagent"
	"casemind/internal/agent/missionagent"
	"casemind/internal/agent/missionmap"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/agent/worldagent"
	"casemind/internal/character"
	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/llm"
	"casemind/internal/missionevent"
	"casemind/internal/wallet"
	"casemind/internal/worldbible"
)

// Generator runs the multi-agent mission generation pipeline and persists
// the validated result: mission plan, private World Bible, map locations,
// characters, and clues.
type Generator struct {
	orch       *orchestrator.Orchestrator
	missions   Repository
	bibles     worldbible.Repository
	locations  gamemap.Repository
	characters character.Repository
	clues      clue.Repository
	recorder   *missionevent.Recorder
	wallet     *wallet.Guard
	log        *slog.Logger
}

func NewGenerator(
	orch *orchestrator.Orchestrator,
	missions Repository,
	bibles worldbible.Repository,
	locations gamemap.Repository,
	characters character.Repository,
	clues clue.Repository,
	recorder *missionevent.Recorder,
	walletGuard *wallet.Guard,
	log *slog.Logger,
) *Generator {
	return &Generator{
		orch: orch, missions: missions, bibles: bibles, locations: locations,
		characters: characters, clues: clues, recorder: recorder, wallet: walletGuard, log: log,
	}
}

// Run generates the full mission content. On success the mission becomes
// `ready` and the reservation is settled; on failure it becomes `failed` and
// the reserved coins are refunded. Events stream either way.
func (g *Generator) Run(ctx context.Context, m *Mission, language string, res *wallet.Reservation) {
	g.recorder.Emit(ctx, m.ID, "mission_generation_started", map[string]any{
		"mission_id": m.ID, "type": m.Type, "difficulty": m.Difficulty,
	})
	usage, err := g.generate(ctx, m, language)
	if err != nil {
		g.log.Error("mission generation failed", "mission_id", m.ID, "error", err)
		if res != nil {
			g.wallet.Release(ctx, res, "mission_generation")
		}
		if uerr := g.missions.UpdateStatus(ctx, m.ID, StatusFailed); uerr != nil {
			g.log.Error("mark mission failed", "mission_id", m.ID, "error", uerr)
		}
		g.recorder.Emit(ctx, m.ID, "mission_generation_failed", map[string]any{
			"mission_id": m.ID, "reason": "generation error",
		})
		return
	}
	if res != nil {
		g.wallet.Settle(ctx, res, "mission_generation", &runtime.Meta{Model: usage.model, Usage: usage.usage})
	}
	if err := g.missions.UpdateStatus(ctx, m.ID, StatusReady); err != nil {
		g.log.Error("mark mission ready", "mission_id", m.ID, "error", err)
		return
	}
	g.recorder.Emit(ctx, m.ID, "mission_ready", map[string]any{"mission_id": m.ID, "title": m.Title})
}

type usageTotal struct {
	model string
	usage llm.Usage
}

func (u *usageTotal) add(meta *runtime.Meta) {
	if meta == nil {
		return
	}
	if meta.Model != "" {
		u.model = meta.Model
	}
	u.usage.PromptTokens += meta.Usage.PromptTokens
	u.usage.CompletionTokens += meta.Usage.CompletionTokens
	u.usage.TotalTokens += meta.Usage.TotalTokens
}

func (g *Generator) generate(ctx context.Context, m *Mission, language string) (*usageTotal, error) {
	total := &usageTotal{}

	// 1. Mission plan.
	planAny, meta, err := g.orch.RunWithMeta(ctx, missionagent.Name, runtime.Task{
		Type: missionagent.TaskType, MissionID: m.ID, UserID: m.UserID,
		Input: missionagent.Input{MissionType: m.Type, Difficulty: m.Difficulty, RegionHint: m.Region, Language: language},
	})
	total.add(meta)
	if err != nil {
		return total, err
	}
	plan := planAny.(*missionagent.Output)
	g.recorder.Emit(ctx, m.ID, "generation_progress", map[string]any{"stage": "plan", "title": plan.Title})

	// 2. World Bible core.
	worldAny, meta, err := g.orch.RunWithMeta(ctx, worldagent.Name, runtime.Task{
		Type: worldagent.TaskType, MissionID: m.ID, UserID: m.UserID,
		Input: worldagent.Input{
			MissionType: m.Type, Title: plan.Title, Summary: plan.Summary,
			Briefing: plan.Briefing, Difficulty: m.Difficulty, Region: plan.Region, Language: language,
		},
	})
	total.add(meta)
	if err != nil {
		return total, err
	}
	world := worldAny.(*worldagent.Output)
	g.recorder.Emit(ctx, m.ID, "generation_progress", map[string]any{"stage": "world"})

	// 3. Map locations.
	mapAny, meta, err := g.orch.RunWithMeta(ctx, missionmap.Name, runtime.Task{
		Type: missionmap.TaskType, MissionID: m.ID, UserID: m.UserID,
		Input: missionmap.Input{MissionType: m.Type, Summary: plan.Summary, Region: plan.Region, Language: language},
	})
	total.add(meta)
	if err != nil {
		return total, err
	}
	worldMap := mapAny.(*missionmap.Output)

	locationIDs := map[string]uuid.UUID{}
	locationRefs := make([]characteragent.LocationRef, 0, len(worldMap.Locations))
	clueLocationRefs := make([]clueagent.LocationRef, 0, len(worldMap.Locations))
	hiddenLocations := []map[string]any{}
	for _, gl := range worldMap.Locations {
		l := &gamemap.Location{
			MissionID:        m.ID,
			Name:             gl.Name,
			Type:             gl.Type,
			Latitude:         gl.Latitude,
			Longitude:        gl.Longitude,
			Status:           gl.Status,
			RiskLevel:        gl.RiskLevel,
			Description:      gl.Description,
			VisualPrompt:     gl.VisualPrompt,
			AvailableActions: mustJSONRaw(gl.AvailableActions, `[]`),
		}
		if err := g.locations.Create(ctx, l); err != nil {
			return total, err
		}
		locationIDs[gl.Key] = l.ID
		locationRefs = append(locationRefs, characteragent.LocationRef{Key: gl.Key, Name: gl.Name})
		clueLocationRefs = append(clueLocationRefs, clueagent.LocationRef{Key: gl.Key, Name: gl.Name})
		if gl.Status != gamemap.StatusDiscovered {
			hiddenLocations = append(hiddenLocations, map[string]any{
				"id": l.ID.String(), "name": gl.Name, "status": gl.Status,
			})
		}
	}
	g.recorder.Emit(ctx, m.ID, "generation_progress", map[string]any{"stage": "map", "locations": len(worldMap.Locations)})

	// 4. Characters.
	charsAny, meta, err := g.orch.RunWithMeta(ctx, characteragent.Name, runtime.Task{
		Type: characteragent.TaskType, MissionID: m.ID, UserID: m.UserID,
		Input: characteragent.Input{
			MissionType: m.Type, Summary: plan.Summary, Truth: world.Truth,
			Locations: locationRefs, Language: language,
		},
	})
	total.add(meta)
	if err != nil {
		return total, err
	}
	chars := charsAny.(*characteragent.Output)

	characterIDs := map[string]uuid.UUID{}
	characterSecrets := map[string]any{}
	clueCharacterRefs := make([]clueagent.CharacterRef, 0, len(chars.Characters))
	for _, gc := range chars.Characters {
		var locID *uuid.UUID
		if id, ok := locationIDs[gc.LocationKey]; ok {
			locID = &id
		}
		c := &character.Character{
			MissionID:         m.ID,
			Name:              gc.Name,
			Role:              gc.Role,
			Category:          gc.Category,
			Age:               gc.Age,
			PublicProfile:     gc.PublicProfile,
			Personality:       mustJSONRaw(gc.Personality, `{}`),
			CurrentLocationID: locID,
			TrustLevel:        50,
			StressLevel:       10,
			Mood:              gc.Mood,
			DialogueStyle:     gc.DialogueStyle,
			AvatarPrompt:      gc.AvatarPrompt,
			ThumbnailPrompt:   gc.ThumbnailPrompt,
			VisualStyleTags:   mustJSONRaw(gc.VisualStyleTags, `[]`),
			PrivateState:      mustJSONRaw(gc.PrivateState, `{}`),
		}
		if err := g.characters.Create(ctx, c); err != nil {
			return total, err
		}
		characterIDs[gc.Key] = c.ID
		characterSecrets[c.ID.String()] = gc.PrivateState
		clueCharacterRefs = append(clueCharacterRefs, clueagent.CharacterRef{Key: gc.Key, Name: gc.Name})
	}
	g.recorder.Emit(ctx, m.ID, "generation_progress", map[string]any{"stage": "characters", "characters": len(chars.Characters)})

	// 5. Clues distributed across the map.
	objectiveTitles := make([]string, 0, len(plan.Objectives))
	for _, o := range plan.Objectives {
		objectiveTitles = append(objectiveTitles, o.Title)
	}
	cluesAny, meta, err := g.orch.RunWithMeta(ctx, clueagent.Name, runtime.Task{
		Type: clueagent.TaskGenerate, MissionID: m.ID, UserID: m.UserID,
		Input: clueagent.GenerateInput{
			MissionType: m.Type, Summary: plan.Summary, Truth: world.Truth,
			Locations: clueLocationRefs, Characters: clueCharacterRefs,
			Objectives: objectiveTitles, Language: language,
		},
	})
	total.add(meta)
	if err != nil {
		return total, err
	}
	generatedClues := cluesAny.(*clueagent.GenerateOutput)

	clueTruth := map[string]any{}
	initialClues := []string{}
	for _, gc := range generatedClues.Clues {
		var locID *uuid.UUID
		if id, ok := locationIDs[gc.LocationKey]; ok {
			locID = &id
		}
		relatedIDs := []string{}
		for _, key := range gc.RelatedCharacterKeys {
			if id, ok := characterIDs[key]; ok {
				relatedIDs = append(relatedIDs, id.String())
			}
		}
		c := &clue.Clue{
			MissionID:               m.ID,
			LocationID:              locID,
			Title:                   gc.Title,
			Type:                    gc.Type,
			ShortDescription:        gc.ShortDescription,
			DetailedDescription:     gc.DetailedDescription,
			VisualDescription:       gc.VisualDescription,
			AvatarOrThumbnailPrompt: gc.AvatarOrThumbnailPrompt,
			Discovered:              gc.Discovered,
			Reliability:             gc.Reliability,
			Importance:              gc.Importance,
			RelatedCharacterIDs:     mustJSONRaw(relatedIDs, `[]`),
			PublicData:              mustJSONRaw(gc.PublicData, `{}`),
			InternalTruth:           mustJSONRaw(gc.InternalTruth, `{}`),
		}
		if err := g.clues.Create(ctx, c); err != nil {
			return total, err
		}
		clueTruth[c.ID.String()] = gc.InternalTruth
		if gc.Discovered {
			initialClues = append(initialClues, gc.Title)
		}
	}
	g.recorder.Emit(ctx, m.ID, "generation_progress", map[string]any{"stage": "clues", "clues": len(generatedClues.Clues)})

	// 6. Private World Bible.
	failureRules := append([]string{}, world.FailureRules...)
	failureRules = append(failureRules, plan.FailConditions...)
	bible := &worldbible.Bible{
		MissionID:        m.ID,
		Truth:            mustJSONRaw(world.Truth, `{}`),
		HiddenState:      mustJSONRaw(world.HiddenState, `{}`),
		CharacterSecrets: mustJSONRaw(characterSecrets, `{}`),
		ClueTruth:        mustJSONRaw(clueTruth, `{}`),
		MapTruth:         mustJSONRaw(map[string]any{"hidden_locations": hiddenLocations}, `{}`),
		TimelineTruth:    mustJSONRaw(world.EventSchedule, `[]`),
		FailureRules:     mustJSONRaw(failureRules, `[]`),
	}
	if err := g.bibles.Create(ctx, bible); err != nil {
		return total, err
	}

	// 7. Player-facing mission content.
	objectives := make([]Objective, 0, len(plan.Objectives))
	for _, o := range plan.Objectives {
		objType := o.Type
		if objType == "" {
			objType = ObjectiveRequired
			if o.Optional {
				objType = ObjectiveOptional
			}
		}
		// Hidden objectives start locked; everything else is immediately active.
		status := ObjStatusActive
		if objType == ObjectiveHidden {
			status = ObjStatusLocked
		}
		objectives = append(objectives, Objective{
			ID: o.Key, Type: objType, Title: o.Title, Description: o.Description,
			Status: status, Progress: 0, RequiredClues: o.RequiredClues,
			Optional: o.Optional || objType == ObjectiveOptional,
		})
	}
	m.Title = plan.Title
	m.Summary = plan.Summary
	m.Briefing = plan.Briefing
	m.Region = plan.Region
	m.Objectives = mustJSONRaw(objectives, `[]`)

	// Stage template: the visible game-level structure, scaled to the real
	// generated content (clue counts, characters, locations).
	nonGuideChars := 0
	for _, gc := range chars.Characters {
		if gc.Category != character.CategoryGuide {
			nonGuideChars++
		}
	}
	m.Stages = mustJSONRaw(DefaultStages(DefaultStageInputs{
		MissionType:    m.Type,
		Difficulty:     m.Difficulty,
		TotalClues:     len(generatedClues.Clues),
		ClueTarget:     MandatoryClueTarget(objectives),
		TotalChars:     nonGuideChars,
		TotalLocations: len(worldMap.Locations),
	}), `[]`)

	// Fold the player-safe win/loss hints and the mission deadline into the
	// public state so the dashboard and completion checks can read them
	// without touching the private World Bible.
	publicState := map[string]any{}
	for k, v := range world.PublicState {
		publicState[k] = v
	}
	publicState[PublicKeyWinConditions] = plan.WinConditions
	publicState[PublicKeyFailureConditions] = plan.FailConditions
	publicState[PublicKeyDeadlineMinutes] = plan.DeadlineHours * 60
	m.PublicState = mustJSONRaw(publicState, `{}`)
	m.CenterLat = worldMap.CenterLat
	m.CenterLng = worldMap.CenterLng
	m.MapZoom = worldMap.Zoom
	if err := g.missions.SetGeneratedContent(ctx, m); err != nil {
		return total, err
	}
	g.recorder.Emit(ctx, m.ID, "initial_clues_available", map[string]any{"clue_titles": initialClues})
	return total, nil
}

func mustJSONRaw(v any, fallback string) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return []byte(fallback)
	}
	return b
}
