package missiontimeline

import (
	"net/http"
	"strconv"

	"casemind/internal/httpx"
	"casemind/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Get serves GET /api/v1/missions/{missionID}/timeline — the curated
// player-facing mission timeline, oldest first.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	view, err := h.svc.Timeline(r.Context(), userID, missionID, limit)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, view)
}
