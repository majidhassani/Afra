package timeline

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"casemind/internal/httpx"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
	"casemind/pkg/validator"
)

// CaseGateway is implemented by cases.Service.
type CaseGateway interface {
	EnsureOwned(ctx context.Context, userID, caseID uuid.UUID) error
}

type Service struct {
	repo  Repository
	guard CaseGateway
}

func NewService(repo Repository, guard CaseGateway) *Service {
	return &Service{repo: repo, guard: guard}
}

// View is the player-facing timeline: confirmed (revealed real) events and
// the player's own events.
type View struct {
	ConfirmedEvents []ConfirmedEvent `json:"confirmed_events"`
	PlayerEvents    []PlayerEvent    `json:"player_events"`
}

func (s *Service) Get(ctx context.Context, userID, caseID uuid.UUID) (*View, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	revealed, err := s.repo.ListRevealed(ctx, caseID)
	if err != nil {
		return nil, err
	}
	confirmed := make([]ConfirmedEvent, 0, len(revealed))
	for i := range revealed {
		confirmed = append(confirmed, revealed[i].Confirmed())
	}
	player, err := s.repo.ListPlayer(ctx, caseID)
	if err != nil {
		return nil, err
	}
	return &View{ConfirmedEvents: confirmed, PlayerEvents: player}, nil
}

type PlayerEventInput struct {
	OccurredAt        time.Time `json:"occurred_at"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	LinkedEvidenceIDs []string  `json:"linked_evidence_ids"`
}

func (in *PlayerEventInput) validate() error {
	if in.OccurredAt.IsZero() {
		return apperrors.Invalid("validation_failed", "occurred_at is required")
	}
	return validator.New().
		Required("title", in.Title).MaxLen("title", in.Title, 200).
		MaxLen("description", in.Description, 2000).
		Err()
}

func (s *Service) CreatePlayerEvent(ctx context.Context, userID, caseID uuid.UUID, in PlayerEventInput) (*PlayerEvent, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	e := &PlayerEvent{
		CaseID:            caseID,
		OccurredAt:        in.OccurredAt,
		Title:             in.Title,
		Description:       in.Description,
		LinkedEvidenceIDs: marshalStrings(in.LinkedEvidenceIDs),
	}
	if err := s.repo.CreatePlayer(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Service) UpdatePlayerEvent(ctx context.Context, userID, caseID, eventID uuid.UUID, in PlayerEventInput) (*PlayerEvent, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	e := &PlayerEvent{
		ID:                eventID,
		CaseID:            caseID,
		OccurredAt:        in.OccurredAt,
		Title:             in.Title,
		Description:       in.Description,
		LinkedEvidenceIDs: marshalStrings(in.LinkedEvidenceIDs),
	}
	if err := s.repo.UpdatePlayer(ctx, e); err != nil {
		return nil, err
	}
	return s.repo.GetPlayer(ctx, caseID, eventID)
}

func (s *Service) DeletePlayerEvent(ctx context.Context, userID, caseID, eventID uuid.UUID) error {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return err
	}
	return s.repo.DeletePlayer(ctx, caseID, eventID)
}

func marshalStrings(s []string) json.RawMessage {
	if s == nil {
		s = []string{}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return []byte(`[]`)
	}
	return b
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	view, err := h.svc.Get(r.Context(), userID, caseID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, view)
}

func (h *Handler) CreatePlayerEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	var in PlayerEventInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	e, err := h.svc.CreatePlayerEvent(r.Context(), userID, caseID, in)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"event": e})
}

func (h *Handler) UpdatePlayerEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	eventID, ok := httpx.PathUUID(w, r, "eventID")
	if !ok {
		return
	}
	var in PlayerEventInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	e, err := h.svc.UpdatePlayerEvent(r.Context(), userID, caseID, eventID, in)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"event": e})
}

func (h *Handler) DeletePlayerEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	eventID, ok := httpx.PathUUID(w, r, "eventID")
	if !ok {
		return
	}
	if err := h.svc.DeletePlayerEvent(r.Context(), userID, caseID, eventID); err != nil {
		response.Err(w, err)
		return
	}
	response.NoContent(w)
}
