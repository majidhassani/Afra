package guidance

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

type guidanceRequest struct {
	Message string         `json:"message"`
	Context RequestContext `json:"context"`
}

func (h *Handler) Guide(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	var req guidanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	result, err := h.svc.Guide(r.Context(), userID, missionID, req.Message, httpx.RequestLanguage(r), req.Context)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

type askAIRequest struct {
	Message string `json:"message"`
}

func (h *Handler) AskAtLocation(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	locationID, ok := httpx.PathUUID(w, r, "locationID")
	if !ok {
		return
	}
	var req askAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	result, err := h.svc.AskAtLocation(r.Context(), userID, missionID, locationID, req.Message, httpx.RequestLanguage(r))
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
