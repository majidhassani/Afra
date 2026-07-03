package solve

import (
	"encoding/json"
	"net/http"

	"casemind/internal/httpx"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Solve(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	var acc Accusation
	if err := json.NewDecoder(r.Body).Decode(&acc); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	verdict, err := h.svc.Solve(r.Context(), userID, caseID, acc)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, verdict)
}

func (h *Handler) Attempts(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	attempts, err := h.svc.Attempts(r.Context(), userID, caseID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"attempts": attempts})
}
