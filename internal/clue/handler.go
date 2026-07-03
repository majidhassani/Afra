package clue

import (
	"encoding/json"
	"net/http"

	"casemind/internal/httpx"
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
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	clues, err := h.svc.List(r.Context(), userID, missionID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"clues": clues})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	clueID, ok := httpx.PathUUID(w, r, "clueID")
	if !ok {
		return
	}
	c, err := h.svc.Get(r.Context(), userID, missionID, clueID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"clue": c})
}

type inspectRequest struct {
	Question string `json:"question"`
}

func (h *Handler) Inspect(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	clueID, ok := httpx.PathUUID(w, r, "clueID")
	if !ok {
		return
	}
	var req inspectRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // question is optional
	}
	result, err := h.svc.Inspect(r.Context(), userID, missionID, clueID, req.Question)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Explain(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	clueID, ok := httpx.PathUUID(w, r, "clueID")
	if !ok {
		return
	}
	result, err := h.svc.Explain(r.Context(), userID, missionID, clueID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
