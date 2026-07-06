package timeengine

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

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	info, err := h.svc.Get(r.Context(), userID, missionID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, info)
}

type advanceRequest struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

func (h *Handler) Advance(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	var req advanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	result, err := h.svc.Advance(r.Context(), userID, missionID, req.Amount, req.Unit, httpx.RequestLanguage(r))
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
