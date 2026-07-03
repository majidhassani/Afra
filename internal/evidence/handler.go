package evidence

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
	response.JSON(w, http.StatusOK, map[string]any{"evidence": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	evidenceID, ok := httpx.PathUUID(w, r, "evidenceID")
	if !ok {
		return
	}
	detail, err := h.svc.Get(r.Context(), userID, caseID, evidenceID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, detail)
}

type inspectRequest struct {
	Question string `json:"question"`
}

func (h *Handler) Inspect(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	evidenceID, ok := httpx.PathUUID(w, r, "evidenceID")
	if !ok {
		return
	}
	var req inspectRequest
	if r.Body != nil {
		// Question is optional; an empty body means a general inspection.
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
			return
		}
	}
	result, err := h.svc.Inspect(r.Context(), userID, caseID, evidenceID, req.Question)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
