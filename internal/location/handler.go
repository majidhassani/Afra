package location

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"casemind/internal/httpx"
	"casemind/pkg/response"
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

// Map returns the discovered locations for the case map view.
func (s *Service) Map(ctx context.Context, userID, caseID uuid.UUID) ([]PublicLocation, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListDiscovered(ctx, caseID)
	if err != nil {
		return nil, err
	}
	out := make([]PublicLocation, 0, len(items))
	for i := range items {
		out = append(out, items[i].Public())
	}
	return out, nil
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Map(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	locations, err := h.svc.Map(r.Context(), userID, caseID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"locations": locations})
}
