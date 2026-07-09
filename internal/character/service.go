package character

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"casemind/internal/agent/dialogueagent"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/clue"
	"casemind/internal/interaction"
	"casemind/internal/llm"
	"casemind/internal/missionevent"
	"casemind/internal/wallet"
	"casemind/pkg/validator"
)

// MissionGateway is the slice of the mission module this service needs; it
// is implemented by mission.Service.
type MissionGateway interface {
	EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error
	EnsureOwnedActive(ctx context.Context, userID, missionID uuid.UUID) error
	SummaryOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	ClockOf(ctx context.Context, userID, missionID uuid.UUID) (string, error)
	ApplyActionTime(ctx context.Context, missionID uuid.UUID, action string) (*missionevent.TimeUpdate, error)
}

// ProfileCounter is the slice of the player profile module this service uses.
type ProfileCounter interface {
	AddCounters(ctx context.Context, userID uuid.UUID, clues, aiInteractions, locations int) error
}

type Service struct {
	repo     Repository
	guard    MissionGateway
	wallet   *wallet.Guard
	orch     *orchestrator.Orchestrator
	convs    interaction.Repository
	clues    clue.Repository
	events   missionevent.Repository
	recorder *missionevent.Recorder
	profiles ProfileCounter
	log      *slog.Logger
}

func NewService(
	repo Repository,
	guard MissionGateway,
	walletGuard *wallet.Guard,
	orch *orchestrator.Orchestrator,
	convs interaction.Repository,
	clues clue.Repository,
	events missionevent.Repository,
	recorder *missionevent.Recorder,
	profiles ProfileCounter,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo, guard: guard, wallet: walletGuard, orch: orch, convs: convs,
		clues: clues, events: events, recorder: recorder, profiles: profiles, log: log,
	}
}

func (s *Service) List(ctx context.Context, userID, missionID uuid.UUID) ([]PublicCharacter, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	characters, err := s.repo.ListByMission(ctx, missionID)
	if err != nil {
		return nil, err
	}
	return PublicList(characters), nil
}

// Detail is the character plus their conversation transcript.
type Detail struct {
	Character PublicCharacter       `json:"character"`
	Messages  []interaction.Message `json:"messages"`
}

func (s *Service) Get(ctx context.Context, userID, missionID, characterID uuid.UUID) (*Detail, error) {
	if err := s.guard.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	c, err := s.repo.GetByID(ctx, missionID, characterID)
	if err != nil {
		return nil, err
	}
	conv, err := s.convs.FindOrCreate(ctx, missionID, userID, interaction.TypeCharacterChat, &c.ID)
	if err != nil {
		return nil, err
	}
	messages, err := s.convs.Recent(ctx, conv.ID, 50)
	if err != nil {
		return nil, err
	}
	return &Detail{Character: c.Public(), Messages: messages}, nil
}

// ChatResult is the player-safe outcome of one dialogue exchange.
type ChatResult struct {
	Message       string            `json:"message"`
	Emotion       string            `json:"emotion"`
	Mood          string            `json:"mood"`
	TrustLevel    int               `json:"trust_level"`
	TrustDelta    int               `json:"trust_delta"`
	StressDelta   int               `json:"stress_delta"`
	UnlockedClues []clue.PublicClue `json:"unlocked_clues"`
	NewFacts      []string          `json:"new_facts"`
	Cost          wallet.Cost       `json:"cost"`
	TimeUpdate    *missionevent.TimeUpdate `json:"time_update,omitempty"`
}

// Chat runs one paid dialogue exchange with an NPC through the DialogueAgent.
func (s *Service) Chat(ctx context.Context, userID, missionID, characterID uuid.UUID, message, language string, attachments []llm.Image, locationID *uuid.UUID) (*ChatResult, error) {
	if err := validator.New().
		Required("message", message).MaxLen("message", message, 2000).
		Err(); err != nil {
		return nil, err
	}
	if err := s.guard.EnsureOwnedActive(ctx, userID, missionID); err != nil {
		return nil, err
	}
	c, err := s.repo.GetByID(ctx, missionID, characterID)
	if err != nil {
		return nil, err
	}
	conv, err := s.convs.FindOrCreate(ctx, missionID, userID, interaction.TypeCharacterChat, &c.ID)
	if err != nil {
		return nil, err
	}
	recent, err := s.convs.Recent(ctx, conv.ID, 20)
	if err != nil {
		return nil, err
	}
	summary, err := s.guard.SummaryOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	clock, err := s.guard.ClockOf(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}
	facts := s.factTexts(ctx, missionID)
	undiscovered := s.undiscoveredTitlesNear(ctx, missionID, c)

	injection := runtime.DetectInjection(message)
	input := dialogueagent.Input{
		MissionSummary: summary,
		MissionTime:    clock,
		CharacterName:  c.Name,
		Role:           c.Role,
		Category:       c.Category,
		Age:            c.Age,
		PublicProfile:  c.PublicProfile,
		Personality:    rawToMap(c.Personality),
		Mood:           c.Mood,
		DialogueStyle:  c.DialogueStyle,
		TrustLevel:     c.TrustLevel,
		StressLevel:    c.StressLevel,

		PrivateState: rawToMap(c.PrivateState),

		DiscoveredFacts:        facts,
		UndiscoveredClueTitles: undiscovered,
		RecentMessages:         toTurns(recent),
		Language:               language,
		PlayerMessage:          message,
		Images:                 attachments,
		InjectionDetected:      len(injection) > 0,
	}

	res, err := s.wallet.Reserve(ctx, userID, &missionID, wallet.ActionCharacterChat)
	if err != nil {
		return nil, err
	}
	outAny, meta, err := s.orch.RunWithMeta(ctx, dialogueagent.Name, runtime.Task{
		Type: dialogueagent.TaskType, MissionID: missionID, UserID: userID, Input: input,
	})
	if err != nil {
		s.wallet.Release(ctx, res, dialogueagent.Name)
		return nil, err
	}
	charged := s.wallet.Settle(ctx, res, dialogueagent.Name, meta)
	out := outAny.(*dialogueagent.Output)

	// Domain output guard: the reply must never contain hidden knowledge
	// verbatim.
	out.Reply = redactPrivate(out.Reply, c.PrivateState)

	// Store the exchange.
	if err := s.convs.AddMessage(ctx, &interaction.Message{
		InteractionID: conv.ID, Sender: interaction.SenderPlayer, Content: message,
	}); err != nil {
		return nil, err
	}
	if err := s.convs.AddMessage(ctx, &interaction.Message{
		InteractionID: conv.ID, Sender: interaction.SenderCharacter, Content: out.Reply,
		Metadata: mustJSON(map[string]any{"emotion": out.Emotion}),
	}); err != nil {
		return nil, err
	}

	// Apply bounded state deltas.
	trust := clamp(c.TrustLevel+out.TrustDelta, 0, 100)
	stress := clamp(c.StressLevel+out.StressDelta, 0, 100)
	mood := out.Mood
	if mood == "" {
		mood = c.Mood
	}
	if err := s.repo.UpdateDialogueState(ctx, c.ID, trust, stress, mood); err != nil {
		return nil, err
	}

	// Validate clue-unlock intentions against the actual undiscovered set.
	unlocked := s.unlockClues(ctx, missionID, out.UnlockClueTitles)
	for _, fact := range out.NewFacts {
		s.recorder.Emit(ctx, missionID, "fact_discovered", map[string]any{
			"fact": fact, "source": "dialogue:" + c.Name,
		})
	}
	s.recorder.Emit(ctx, missionID, "dialogue", map[string]any{
		"character_id": c.ID, "character_name": c.Name,
		"trust_level": trust, "mood": mood,
	})
	if err := s.profiles.AddCounters(ctx, userID, len(unlocked), 1, 0); err != nil {
		s.log.Error("profile counter", "error", err)
	}

	// Talking costs mission time and may trigger world events.
	var timeUpdate *missionevent.TimeUpdate
	if tu, err := s.guard.ApplyActionTime(ctx, missionID, "character_chat"); err == nil {
		timeUpdate = tu
	} else {
		s.log.Error("chat time cost", "error", err)
	}

	return &ChatResult{
		TimeUpdate:    timeUpdate,
		Message:       out.Reply,
		Emotion:       out.Emotion,
		Mood:          mood,
		TrustLevel:    trust,
		TrustDelta:    out.TrustDelta,
		StressDelta:   out.StressDelta,
		UnlockedClues: unlocked,
		NewFacts:      out.NewFacts,
		Cost:          wallet.Cost{CoinsCharged: charged},
	}, nil
}

// unlockClues marks agent-suggested clues discovered, ignoring titles that
// don't exist (invalid intentions are dropped, not errors).
func (s *Service) unlockClues(ctx context.Context, missionID uuid.UUID, titles []string) []clue.PublicClue {
	unlocked := []clue.PublicClue{}
	for _, title := range titles {
		cl, err := s.clues.FindUndiscoveredByTitle(ctx, missionID, title)
		if err != nil {
			continue
		}
		if err := s.clues.MarkDiscovered(ctx, cl.ID); err != nil {
			s.log.Error("mark clue discovered", "error", err)
			continue
		}
		cl.Discovered = true
		unlocked = append(unlocked, cl.Public())
		s.recorder.Emit(ctx, missionID, "clue_discovered", map[string]any{
			"clue_id": cl.ID, "title": cl.Title,
		})
	}
	return unlocked
}

// undiscoveredTitlesNear lists undiscovered clue titles related to this
// character or its location — the set the dialogue may naturally unlock.
func (s *Service) undiscoveredTitlesNear(ctx context.Context, missionID uuid.UUID, c *Character) []string {
	titles := []string{}
	if c.CurrentLocationID != nil {
		clues, err := s.clues.ListUndiscoveredAtLocation(ctx, missionID, *c.CurrentLocationID)
		if err == nil {
			for i := range clues {
				titles = append(titles, clues[i].Title)
			}
		}
	}
	return titles
}

func (s *Service) factTexts(ctx context.Context, missionID uuid.UUID) []string {
	events, err := s.events.ListByMission(ctx, missionID, 100)
	if err != nil {
		s.log.Error("list mission facts", "error", err)
		return nil
	}
	return missionevent.FactTexts(events)
}

// --- helpers ---

func toTurns(messages []interaction.Message) []dialogueagent.Turn {
	turns := make([]dialogueagent.Turn, 0, len(messages))
	for _, m := range messages {
		turns = append(turns, dialogueagent.Turn{Role: m.Sender, Content: m.Content})
	}
	return turns
}

func rawToMap(raw json.RawMessage) map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func mustJSON(m map[string]any) json.RawMessage {
	b, err := json.Marshal(m)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

// redactPrivate replaces any verbatim leak of a private-state string with an
// in-character deflection. Hard domain rule on top of prompt discipline.
func redactPrivate(reply string, privateState json.RawMessage) string {
	var state map[string]any
	if err := json.Unmarshal(privateState, &state); err != nil {
		return reply
	}
	lower := strings.ToLower(reply)
	for _, v := range state {
		for _, secret := range stringsIn(v) {
			if len(secret) >= 12 && strings.Contains(lower, strings.ToLower(secret)) {
				return "I'm not going to talk about that."
			}
		}
	}
	return reply
}

func stringsIn(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		out := []string{}
		for _, item := range t {
			out = append(out, stringsIn(item)...)
		}
		return out
	case map[string]any:
		out := []string{}
		for _, item := range t {
			out = append(out, stringsIn(item)...)
		}
		return out
	default:
		return nil
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
