// Package guidance is the "AI help at every stage" module: the player can
// ask the in-world AI assistant for hints from any screen, and ask
// location-scoped questions. The GuidanceAgent only ever sees discovered,
// player-safe state — never the World Bible.
package guidance

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"casemind/internal/agent/guidanceagent"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/character"
	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/interaction"
	"casemind/internal/missionevent"
	"casemind/internal/wallet"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

// MissionGateway is the slice of the mission module this service needs; it
// is implemented by mission.Service.
type MissionGateway interface {
	EnsureOwnedActive(ctx context.Context, userID, missionID uuid.UUID) error
	SummaryOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	TypeOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	ClockOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	ObjectiveTitles(ctx context.Context, userID, missionID uuid.UUID) ([]string, error)
}

// ProfileCounter is the slice of the player profile module this service uses.
type ProfileCounter interface {
	AddCounters(ctx context.Context, userID uuid.UUID, clues, aiInteractions, locations int) error
}

// Context passed with a guidance request from the client.
type RequestContext struct {
	Screen         string     `json:"screen"`
	LocationID     *uuid.UUID `json:"location_id,omitempty"`
	SelectedClueID *uuid.UUID `json:"selected_clue_id,omitempty"`
}

// Result is the player-safe guidance response.
type Result struct {
	Message         string                  `json:"message"`
	HintLevel       string                  `json:"hint_level"`
	ReferencedItems []guidanceagent.ItemRef `json:"referenced_items"`
	Cost            wallet.Cost             `json:"cost"`
}

type Service struct {
	guard      MissionGateway
	wallet     *wallet.Guard
	orch       *orchestrator.Orchestrator
	convs      interaction.Repository
	clues      clue.Repository
	locations  gamemap.Repository
	characters character.Repository
	events     missionevent.Repository
	recorder   *missionevent.Recorder
	profiles   ProfileCounter
	log        *slog.Logger
}

func NewService(
	guard MissionGateway,
	walletGuard *wallet.Guard,
	orch *orchestrator.Orchestrator,
	convs interaction.Repository,
	clues clue.Repository,
	locations gamemap.Repository,
	characters character.Repository,
	events missionevent.Repository,
	recorder *missionevent.Recorder,
	profiles ProfileCounter,
	log *slog.Logger,
) *Service {
	return &Service{
		guard: guard, wallet: walletGuard, orch: orch, convs: convs, clues: clues,
		locations: locations, characters: characters, events: events,
		recorder: recorder, profiles: profiles, log: log,
	}
}

// Guide answers a paid guidance request from any screen.
func (s *Service) Guide(ctx context.Context, userID, missionID uuid.UUID, message, language string, reqCtx RequestContext) (*Result, error) {
	return s.run(ctx, userID, missionID, message, language, reqCtx, guidanceagent.TaskGuide, wallet.ActionAIGuidance, nil)
}

// AskAtLocation answers a paid location-scoped AI question.
func (s *Service) AskAtLocation(ctx context.Context, userID, missionID, locationID uuid.UUID, message, language string) (*Result, error) {
	l, err := s.locations.GetByID(ctx, missionID, locationID)
	if err != nil {
		return nil, err
	}
	if !l.Visible() {
		return nil, apperrors.NotFound("location_not_found", "location not found")
	}
	return s.run(ctx, userID, missionID, message, language,
		RequestContext{Screen: "location", LocationID: &locationID},
		guidanceagent.TaskLocation, wallet.ActionLocationAsk, l)
}

func (s *Service) run(ctx context.Context, userID, missionID uuid.UUID, message, language string,
	reqCtx RequestContext, taskType, action string, location *gamemap.Location) (*Result, error) {

	if err := validator.New().
		Required("message", message).MaxLen("message", message, 2000).
		Err(); err != nil {
		return nil, err
	}
	if err := s.guard.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}

	input, err := s.buildContext(ctx, userID, missionID, message, language, reqCtx, location)
	if err != nil {
		return nil, err
	}

	conv, err := s.convs.FindOrCreate(ctx, missionID, userID, interaction.TypeGuidance, nil)
	if err != nil {
		return nil, err
	}
	recent, err := s.convs.Recent(ctx, conv.ID, 12)
	if err != nil {
		return nil, err
	}
	for _, m := range recent {
		input.RecentMessages = append(input.RecentMessages, guidanceagent.Turn{Role: m.Sender, Content: m.Content})
	}

	res, err := s.wallet.Reserve(ctx, userID, &missionID, action)
	if err != nil {
		return nil, err
	}
	outAny, meta, err := s.orch.RunWithMeta(ctx, guidanceagent.Name, runtime.Task{
		Type: taskType, MissionID: missionID, UserID: userID, Input: *input,
	})
	if err != nil {
		s.wallet.Release(ctx, res, guidanceagent.Name)
		return nil, err
	}
	charged := s.wallet.Settle(ctx, res, guidanceagent.Name, meta)
	out := outAny.(*guidanceagent.Output)

	// Store the exchange in the guidance thread.
	if err := s.convs.AddMessage(ctx, &interaction.Message{
		InteractionID: conv.ID, Sender: interaction.SenderPlayer, Content: message,
	}); err != nil {
		s.log.Error("store guidance question", "error", err)
	}
	if err := s.convs.AddMessage(ctx, &interaction.Message{
		InteractionID: conv.ID, Sender: interaction.SenderAI, Content: out.Message,
	}); err != nil {
		s.log.Error("store guidance answer", "error", err)
	}
	s.recorder.Emit(ctx, missionID, "ai_guidance", map[string]any{
		"screen": input.Screen, "hint_level": out.HintLevel,
	})
	if err := s.profiles.AddCounters(ctx, userID, 0, 1, 0); err != nil {
		s.log.Error("profile counter", "error", err)
	}

	return &Result{
		Message:         out.Message,
		HintLevel:       out.HintLevel,
		ReferencedItems: out.ReferencedItems,
		Cost:            wallet.Cost{CoinsCharged: charged},
	}, nil
}

// buildContext assembles the strictly player-visible world state.
func (s *Service) buildContext(ctx context.Context, userID, missionID uuid.UUID, message, language string,
	reqCtx RequestContext, location *gamemap.Location) (*guidanceagent.Input, error) {

	summary, err := s.guard.SummaryOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	missionType, err := s.guard.TypeOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	clock, err := s.guard.ClockOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	objectives, err := s.guard.ObjectiveTitles(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}

	discovered, err := s.clues.ListDiscovered(ctx, missionID)
	if err != nil {
		return nil, err
	}
	clueRefs := make([]guidanceagent.ItemRef, 0, len(discovered))
	for i := range discovered {
		clueRefs = append(clueRefs, guidanceagent.ItemRef{Type: "clue", ID: discovered[i].ID.String(), Name: discovered[i].Title})
	}

	locations, err := s.locations.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	visited := []guidanceagent.ItemRef{}
	unvisited := []guidanceagent.ItemRef{}
	for i := range locations {
		l := &locations[i]
		if !l.Visible() {
			continue // hidden locations never reach the guidance agent
		}
		ref := guidanceagent.ItemRef{Type: "location", ID: l.ID.String(), Name: l.Name}
		if l.Status == gamemap.StatusVisited {
			visited = append(visited, ref)
		} else {
			unvisited = append(unvisited, ref)
		}
	}

	characters, err := s.characters.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	charRefs := make([]guidanceagent.ItemRef, 0, len(characters))
	for i := range characters {
		charRefs = append(charRefs, guidanceagent.ItemRef{Type: "character", ID: characters[i].ID.String(), Name: characters[i].Name})
	}

	input := &guidanceagent.Input{
		Screen:             reqCtx.Screen,
		MissionType:        missionType,
		MissionSummary:     summary,
		MissionTime:        clock,
		Objectives:         objectives,
		DiscoveredClues:    clueRefs,
		VisitedLocations:   visited,
		UnvisitedLocations: unvisited,
		KnownCharacters:    charRefs,
		DiscoveredFacts:    s.factTexts(ctx, missionID),
		PlayerMessage:      message,
		InjectionDetected:  len(runtime.DetectInjection(message)) > 0,
		Language:           runtime.Language(language),
	}
	if location != nil {
		input.LocationName = location.Name
		input.LocationDescription = location.Description
	}
	return input, nil
}

func (s *Service) factTexts(ctx context.Context, missionID uuid.UUID) []string {
	events, err := s.events.ListByMission(ctx, missionID, 100)
	if err != nil {
		s.log.Error("list mission facts", "error", err)
		return nil
	}
	return missionevent.FactTexts(events)
}
