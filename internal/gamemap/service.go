package gamemap

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"casemind/internal/agent/locationagent"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/character"
	"casemind/internal/clue"
	"casemind/internal/missionevent"
	"casemind/internal/wallet"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

// MissionGateway is the slice of the mission module this service needs; it
// is implemented by mission.Service.
type MissionGateway interface {
	EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error
	EnsureOwnedActive(ctx context.Context, userID, missionID uuid.UUID) error
	SummaryOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	ClockOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	MapCenterOf(ctx context.Context, userID, missionID uuid.UUID) (lat, lng float64, zoom int, err error)
}

// ProfileCounter is the slice of the player profile module this service uses.
type ProfileCounter interface {
	AddCounters(ctx context.Context, userID uuid.UUID, clues, aiInteractions, locations int) error
}

type Service struct {
	repo       Repository
	guard      MissionGateway
	wallet     *wallet.Guard
	orch       *orchestrator.Orchestrator
	clues      clue.Repository
	characters character.Repository
	events     missionevent.Repository
	recorder   *missionevent.Recorder
	profiles   ProfileCounter
	log        *slog.Logger
}

func NewService(
	repo Repository,
	guard MissionGateway,
	walletGuard *wallet.Guard,
	orch *orchestrator.Orchestrator,
	clues clue.Repository,
	characters character.Repository,
	events missionevent.Repository,
	recorder *missionevent.Recorder,
	profiles ProfileCounter,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo, guard: guard, wallet: walletGuard, orch: orch, clues: clues,
		characters: characters, events: events, recorder: recorder, profiles: profiles, log: log,
	}
}

// Map returns the Google Maps view: center, zoom, and visible markers with
// clue/character badges. Hidden locations are omitted entirely.
func (s *Service) Map(ctx context.Context, userID, missionID uuid.UUID) (*MapView, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	lat, lng, zoom, err := s.guard.MapCenterOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	locations, err := s.repo.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	characters, err := s.characters.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	charactersAt := map[uuid.UUID]bool{}
	for i := range characters {
		if characters[i].CurrentLocationID != nil {
			charactersAt[*characters[i].CurrentLocationID] = true
		}
	}

	markers := []Marker{}
	recommendedChosen := false
	for i := range locations {
		l := &locations[i]
		if !l.Visible() {
			continue
		}
		discovered, err := s.clues.ListDiscoveredAtLocation(ctx, missionID, l.ID)
		if err != nil {
			return nil, err
		}
		undiscovered, err := s.clues.ListUndiscoveredAtLocation(ctx, missionID, l.ID)
		if err != nil {
			return nil, err
		}
		hasNewClue := len(discovered) > 0
		hasMore := len(undiscovered) > 0
		m := Marker{
			ID: l.ID, Name: l.Name, Type: l.Type, Lat: l.Latitude, Lng: l.Longitude,
			Status: l.Status, RiskLevel: l.RiskLevel,
			HasNewClue: hasNewClue, HasCharacter: charactersAt[l.ID],
			IsLocked: l.Status == StatusLocked,
		}
		switch {
		case m.IsLocked:
			m.Badge = "locked"
		case hasNewClue:
			m.Badge = "new_clue"
		case m.HasCharacter:
			m.Badge = "character"
		}
		// Recommend the first not-yet-visited, accessible location as the
		// player's next best step.
		recommend := !recommendedChosen && l.Status == StatusDiscovered
		m.Annotate(hasMore, recommend)
		if m.Recommended {
			recommendedChosen = true
		}
		markers = append(markers, m)
	}
	return &MapView{
		MissionID: missionID,
		Center:    LatLng{Lat: lat, Lng: lng},
		Zoom:      zoom,
		Locations: markers,
	}, nil
}

// LocationDetail is the bottom-sheet payload for one tapped marker.
type LocationDetail struct {
	Location   *Location                   `json:"location"`
	Characters []character.PublicCharacter `json:"characters"`
	Clues      []clue.PublicClue           `json:"discovered_clues"`
}

func (s *Service) Detail(ctx context.Context, userID, missionID, locationID uuid.UUID) (*LocationDetail, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	l, err := s.repo.GetByID(ctx, missionID, locationID)
	if err != nil {
		return nil, err
	}
	if !l.Visible() {
		// Hidden locations do not exist as far as the client is concerned.
		return nil, apperrors.NotFound("location_not_found", "location not found")
	}
	characters, err := s.characters.ListAtLocation(ctx, missionID, locationID)
	if err != nil {
		return nil, err
	}
	discovered, err := s.clues.ListDiscoveredAtLocation(ctx, missionID, locationID)
	if err != nil {
		return nil, err
	}
	// Visiting a location marks it visited (first tap on a discovered spot).
	if l.Status == StatusDiscovered {
		if err := s.repo.UpdateStatus(ctx, locationID, StatusVisited); err != nil {
			s.log.Error("mark location visited", "error", err)
		} else {
			l.Status = StatusVisited
			s.recorder.Emit(ctx, missionID, "location_visited", map[string]any{
				"location_id": l.ID, "name": l.Name,
			})
			if err := s.profiles.AddCounters(ctx, userID, 0, 0, 1); err != nil {
				s.log.Error("profile counter", "error", err)
			}
		}
	}
	return &LocationDetail{Location: l, Characters: character.PublicList(characters), Clues: clue.PublicList(discovered)}, nil
}

// ActionResult is the player-safe outcome of a paid location action.
type ActionResult struct {
	Narrative       string            `json:"narrative"`
	DiscoveredClues []clue.PublicClue `json:"discovered_clues"`
	NewFacts        []string          `json:"new_facts"`
	Cost            wallet.Cost       `json:"cost"`
}

// Action executes a location action (search/inspect/scan) through the
// LocationAgent, unlocking any earned clues.
func (s *Service) Action(ctx context.Context, userID, missionID, locationID uuid.UUID, action, language string) (*ActionResult, error) {
	if err := validator.New().
		Required("action", action).MaxLen("action", action, 60).
		Err(); err != nil {
		return nil, err
	}
	if err := s.guard.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}
	l, err := s.repo.GetByID(ctx, missionID, locationID)
	if err != nil {
		return nil, err
	}
	if !l.Visible() {
		return nil, apperrors.NotFound("location_not_found", "location not found")
	}
	if l.Status == StatusLocked {
		return nil, apperrors.Conflict("location_locked", "this location is not accessible yet")
	}
	summary, err := s.guard.SummaryOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	clock, err := s.guard.ClockOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	undiscovered, err := s.clues.ListUndiscoveredAtLocation(ctx, missionID, locationID)
	if err != nil {
		return nil, err
	}
	clueRefs := make([]locationagent.ClueRef, 0, len(undiscovered))
	for i := range undiscovered {
		clueRefs = append(clueRefs, locationagent.ClueRef{Title: undiscovered[i].Title, Type: undiscovered[i].Type})
	}

	res, err := s.wallet.Reserve(ctx, userID, &missionID, wallet.ActionLocationSearch)
	if err != nil {
		return nil, err
	}
	outAny, meta, err := s.orch.RunWithMeta(ctx, locationagent.Name, runtime.Task{
		Type: locationagent.TaskType, MissionID: missionID, UserID: userID,
		Input: locationagent.Input{
			MissionSummary:    summary,
			MissionTime:       clock,
			LocationName:      l.Name,
			LocationType:      l.Type,
			Description:       l.Description,
			RiskLevel:         l.RiskLevel,
			Action:            action,
			UndiscoveredClues: clueRefs,
			DiscoveredFacts:   s.factTexts(ctx, missionID),
			Language:          runtime.Language(language),
		},
	})
	if err != nil {
		s.wallet.Release(ctx, res, locationagent.Name)
		return nil, err
	}
	charged := s.wallet.Settle(ctx, res, locationagent.Name, meta)
	out := outAny.(*locationagent.Output)

	// Validate discovery intentions against the actual undiscovered set.
	discovered := []clue.PublicClue{}
	for _, title := range out.DiscoverClueTitles {
		cl, err := s.clues.FindUndiscoveredByTitle(ctx, missionID, title)
		if err != nil || cl.LocationID == nil || *cl.LocationID != locationID {
			continue
		}
		if err := s.clues.MarkDiscovered(ctx, cl.ID); err != nil {
			s.log.Error("mark clue discovered", "error", err)
			continue
		}
		cl.Discovered = true
		discovered = append(discovered, cl.Public())
		s.recorder.Emit(ctx, missionID, "clue_discovered", map[string]any{
			"clue_id": cl.ID, "title": cl.Title, "location_id": locationID,
		})
	}
	for _, fact := range out.NewFacts {
		s.recorder.Emit(ctx, missionID, "fact_discovered", map[string]any{
			"fact": fact, "source": "location_action:" + l.Name,
		})
	}
	s.recorder.Emit(ctx, missionID, "player_action", map[string]any{
		"location_id": l.ID, "location_name": l.Name, "action": action,
	})
	if err := s.profiles.AddCounters(ctx, userID, len(discovered), 1, 0); err != nil {
		s.log.Error("profile counter", "error", err)
	}

	return &ActionResult{
		Narrative:       out.Narrative,
		DiscoveredClues: discovered,
		NewFacts:        out.NewFacts,
		Cost:            wallet.Cost{CoinsCharged: charged},
	}, nil
}

func (s *Service) factTexts(ctx context.Context, missionID uuid.UUID) []string {
	events, err := s.events.ListByMission(ctx, missionID, 100)
	if err != nil {
		s.log.Error("list mission facts", "error", err)
		return nil
	}
	return missionevent.FactTexts(events)
}
