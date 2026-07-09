package report

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"casemind/internal/httpx"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type submitRequest struct {
	Type               string   `json:"type"`
	Title              string   `json:"title"`
	Summary            string   `json:"summary"`
	LinkedClueIDs      []string `json:"linked_clue_ids"`
	SuspectCharacterID string   `json:"suspect_character_id"`
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	sub := Submission{Type: req.Type, Title: req.Title, Summary: req.Summary}
	for _, raw := range req.LinkedClueIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Err(w, apperrors.Invalid("invalid_clue_id", "linked_clue_ids must be UUIDs"))
			return
		}
		sub.LinkedClueIDs = append(sub.LinkedClueIDs, id)
	}
	if req.SuspectCharacterID != "" {
		id, err := uuid.Parse(req.SuspectCharacterID)
		if err != nil {
			response.Err(w, apperrors.Invalid("invalid_character_id", "suspect_character_id must be a UUID"))
			return
		}
		sub.SuspectCharacterID = &id
	}
	res, err := h.svc.Submit(r.Context(), userID, missionID, sub)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	reports, err := h.svc.List(r.Context(), userID, missionID, limit)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"reports": reports})
}
