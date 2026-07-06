package missioncomplete

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

// Check answers whether the mission can be completed yet (no charge).
func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	check, err := h.svc.Check(r.Context(), userID, missionID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, check)
}

// Result returns the persisted result of a finished mission so it can be
// reviewed after the completion modal closes and from history. 409 when the
// mission is not finished yet, 404 when no result was stored.
func (h *Handler) Result(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	result, status, err := h.svc.StoredResult(r.Context(), userID, missionID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"mission_id":     missionID,
		"mission_status": status,
		"result":         result,
	})
}

// Complete submits the player's final decision. If the mission is not ready it
// returns 409 with the CompletionCheck explaining what is missing; otherwise it
// returns the judged mission result.
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	var dec Decision
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&dec); err != nil {
			response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
			return
		}
	}
	dec.Language = httpx.RequestLanguage(r)
	result, check, err := h.svc.Complete(r.Context(), userID, missionID, dec)
	if err != nil {
		response.Err(w, err)
		return
	}
	if check != nil {
		// Not ready: explain why, do not treat as an error the client must retry.
		response.JSON(w, http.StatusConflict, check)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
