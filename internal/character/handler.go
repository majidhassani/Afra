package character

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"casemind/internal/httpx"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/images"
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
	characters, err := h.svc.List(r.Context(), userID, missionID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"characters": characters})
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
	characterID, ok := httpx.PathUUID(w, r, "characterID")
	if !ok {
		return
	}
	detail, err := h.svc.Get(r.Context(), userID, missionID, characterID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, detail)
}

type chatRequest struct {
	Message    string     `json:"message"`
	LocationID *uuid.UUID `json:"location_id,omitempty"`
	// Images optionally attaches photos the player shows to the character
	// (vision). Validated for MIME, size and dimensions before use.
	Images []images.Payload `json:"images,omitempty"`
}

func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	characterID, ok := httpx.PathUUID(w, r, "characterID")
	if !ok {
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	attachments, err := images.DecodeAndValidate(req.Images)
	if err != nil {
		response.Err(w, err)
		return
	}
	result, err := h.svc.Chat(r.Context(), userID, missionID, characterID, req.Message, httpx.RequestLanguage(r), attachments, req.LocationID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
