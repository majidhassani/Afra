package mission

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"casemind/internal/character"
	"casemind/internal/clue"
	"casemind/internal/gamemap"
	"casemind/internal/interaction"
	"casemind/internal/missionevent"
	"casemind/internal/wallet"
	"casemind/internal/worldbible"
	apperrors "casemind/pkg/errors"
)

// ProfileCounter is the slice of the player profile module this service uses.
// ApplyMissionResult with completed=false is a pure XP grant (stage rewards).
type ProfileCounter interface {
	IncrementTotalMissions(ctx context.Context, userID uuid.UUID) error
	ApplyMissionResult(ctx context.Context, userID uuid.UUID, completed bool, xpDelta int) error
}

type Service struct {
	repo         Repository
	characters   character.Repository
	clues        clue.Repository
	locations    gamemap.Repository
	interactions interaction.Repository
	events       missionevent.Repository
	bibles       worldbible.Repository
	generator    *Generator
	wallet       *wallet.Guard
	walletSvc    *wallet.Service
	profiles     ProfileCounter
	recorder     *missionevent.Recorder
	log          *slog.Logger
}

func NewService(
	repo Repository,
	characters character.Repository,
	clues clue.Repository,
	locations gamemap.Repository,
	interactions interaction.Repository,
	events missionevent.Repository,
	bibles worldbible.Repository,
	generator *Generator,
	walletGuard *wallet.Guard,
	walletSvc *wallet.Service,
	profiles ProfileCounter,
	recorder *missionevent.Recorder,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo, characters: characters, clues: clues, locations: locations,
		interactions: interactions, events: events, bibles: bibles, generator: generator,
		wallet: walletGuard, walletSvc: walletSvc, profiles: profiles, recorder: recorder, log: log,
	}
}

func NormalizeLanguage(language string) string {
	if language == "" {
		return "en"
	}
	return language
}

// Create creates a mission in generating status, reserves the server-side
// mission-start cost, and runs the multi-agent generation pipeline detached
// from the request context.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, missionType, difficulty, region, language string) (*Mission, error) {
	language = NormalizeLanguage(language)
	if err := ValidateNewMission(missionType, difficulty, language); err != nil {
		return nil, err
	}
	res, err := s.wallet.Reserve(ctx, userID, nil, wallet.ActionMissionStart)
	if err != nil {
		return nil, err
	}
	m := &Mission{
		UserID:     userID,
		Type:       missionType,
		Title:      "Generating mission...",
		Status:     StatusGenerating,
		Difficulty: difficulty,
		Region:     region,
		Summary:    "",
		Briefing:   "",
	}
	if err := s.repo.Create(ctx, m); err != nil {
		s.wallet.Release(ctx, res, "mission_create")
		return nil, err
	}
	res.MissionID = &m.ID
	if err := s.profiles.IncrementTotalMissions(ctx, userID); err != nil {
		s.log.Error("increment total missions", "error", err)
	}

	genMission := *m
	go func() {
		genCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.generator.Run(genCtx, &genMission, language, res)
	}()
	return m, nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Mission, error) {
	return s.repo.ListByUser(ctx, userID)
}

type Dashboard struct {
	Mission    *Mission                    `json:"mission"`
	Characters []character.PublicCharacter `json:"characters"`
	Clues      []clue.PublicClue           `json:"clues"`
	Locations  []gamemap.Marker            `json:"locations"`
}

func (s *Service) Get(ctx context.Context, userID, missionID uuid.UUID) (*Dashboard, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	characters, err := s.characters.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	discovered, err := s.clues.ListDiscovered(ctx, missionID)
	if err != nil {
		return nil, err
	}
	locations, err := s.locations.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	markers := make([]gamemap.Marker, 0, len(locations))
	for i := range locations {
		l := &locations[i]
		if !l.Visible() {
			continue
		}
		markers = append(markers, gamemap.Marker{
			ID: l.ID, Name: l.Name, Type: l.Type, Lat: l.Latitude, Lng: l.Longitude,
			Status: l.Status, RiskLevel: l.RiskLevel, IsLocked: l.Status == gamemap.StatusLocked,
		})
	}
	return &Dashboard{
		Mission: m, Characters: character.PublicList(characters),
		Clues: clue.PublicList(discovered), Locations: markers,
	}, nil
}

func (s *Service) Archive(ctx context.Context, userID, missionID uuid.UUID) (*Mission, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	if m.Status == StatusGenerating {
		return nil, apperrors.Conflict("mission_generating", "cannot archive a mission while it is generating")
	}
	if err := s.repo.UpdateStatus(ctx, missionID, StatusArchived); err != nil {
		return nil, err
	}
	m.Status = StatusArchived
	return m, nil
}

func (s *Service) Events(ctx context.Context, userID, missionID uuid.UUID, limit int) ([]missionevent.Event, error) {
	if err := s.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	return s.events.ListByMission(ctx, missionID, limit)
}

func (s *Service) EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error {
	_, err := s.repo.GetForUser(ctx, userID, missionID)
	return err
}

func (s *Service) EnsureOwnedActive(ctx context.Context, userID, missionID uuid.UUID) error {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return err
	}
	if m.Status != StatusReady && m.Status != StatusActive {
		return apperrors.Conflict("mission_not_active", "this mission is not ready for play (status: "+m.Status+")")
	}
	return nil
}

func (s *Service) SummaryOf(ctx context.Context, userID, missionID uuid.UUID) (string, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return "", err
	}
	return m.Summary, nil
}

func (s *Service) TypeOf(ctx context.Context, userID, missionID uuid.UUID) (string, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return "", err
	}
	return m.Type, nil
}

func (s *Service) ClockOf(ctx context.Context, userID, missionID uuid.UUID) (string, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return "", err
	}
	return m.CurrentTime, nil
}

func (s *Service) MapCenterOf(ctx context.Context, userID, missionID uuid.UUID) (float64, float64, int, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return 0, 0, 0, err
	}
	return m.CenterLat, m.CenterLng, m.MapZoom, nil
}

func (s *Service) ObjectiveTitles(ctx context.Context, userID, missionID uuid.UUID) ([]string, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	titles := []string{}
	for _, objective := range m.ParsedObjectives() {
		titles = append(titles, objective.Title)
	}
	return titles, nil
}

func (s *Service) PreviewClock(ctx context.Context, userID, missionID uuid.UUID, minutes int) (string, int, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return "", 0, err
	}
	next := m.MissionTime.Add(time.Duration(minutes) * time.Minute)
	return FormatClock(next), MinutesSinceStart(next), nil
}

func (s *Service) CommitClock(ctx context.Context, missionID uuid.UUID, minutes int) (string, error) {
	next, err := s.repo.AdvanceMissionTime(ctx, missionID, minutes)
	if err != nil {
		return "", err
	}
	return FormatClock(next), nil
}

func (s *Service) PublicStateOf(ctx context.Context, userID, missionID uuid.UUID) (map[string]any, error) {
	m, err := s.repo.GetForUser(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	state := map[string]any{}
	_ = json.Unmarshal(m.PublicState, &state)
	return state, nil
}

func (s *Service) MergePublicState(ctx context.Context, missionID uuid.UUID, changes map[string]any) error {
	m, err := s.repo.GetByID(ctx, missionID)
	if err != nil {
		return err
	}
	state := map[string]any{}
	_ = json.Unmarshal(m.PublicState, &state)
	for k, v := range changes {
		state[k] = v
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return apperrors.Internal(err, "merge public state")
	}
	return s.repo.UpdatePublicState(ctx, missionID, raw)
}
