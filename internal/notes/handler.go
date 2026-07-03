package notes

import (
	"context"
	"encoding/json"
	"net/http"

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

type NoteInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (in *NoteInput) validate() error {
	return validator.New().
		Required("content", in.Content).MaxLen("content", in.Content, 10000).
		MaxLen("title", in.Title, 200).
		Err()
}

func (s *Service) List(ctx context.Context, userID, caseID uuid.UUID) ([]Note, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	return s.repo.ListByCase(ctx, caseID)
}

func (s *Service) Create(ctx context.Context, userID, caseID uuid.UUID, in NoteInput) (*Note, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	n := &Note{CaseID: caseID, Title: in.Title, Content: in.Content}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *Service) Update(ctx context.Context, userID, caseID, noteID uuid.UUID, in NoteInput) (*Note, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	n := &Note{ID: noteID, CaseID: caseID, Title: in.Title, Content: in.Content}
	if err := s.repo.Update(ctx, n); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, caseID, noteID)
}

func (s *Service) Delete(ctx context.Context, userID, caseID, noteID uuid.UUID) error {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, caseID, noteID)
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	items, err := h.svc.List(r.Context(), userID, caseID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"notes": items})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	var in NoteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	n, err := h.svc.Create(r.Context(), userID, caseID, in)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"note": n})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	noteID, ok := httpx.PathUUID(w, r, "noteID")
	if !ok {
		return
	}
	var in NoteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	n, err := h.svc.Update(r.Context(), userID, caseID, noteID, in)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"note": n})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	noteID, ok := httpx.PathUUID(w, r, "noteID")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), userID, caseID, noteID); err != nil {
		response.Err(w, err)
		return
	}
	response.NoContent(w)
}
