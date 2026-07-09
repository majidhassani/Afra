package visualasset

import (
	"encoding/json"
	"net/http"

	"casemind/internal/httpx"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// GenerateCharacterAvatar handles
// POST /api/v1/missions/{missionID}/characters/{characterID}/avatar.
func (h *Handler) GenerateCharacterAvatar(w http.ResponseWriter, r *http.Request) {
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
	c, err := h.svc.CharacterAvatar(r.Context(), userID, missionID, characterID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"character": c})
}

// GenerateBoard handles
// POST /api/v1/missions/{missionID}/art/board/generate.
func (h *Handler) GenerateBoard(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	var req struct {
		BoardType string `json:"board_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	board, err := h.svc.Board(r.Context(), userID, missionID, req.BoardType)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"board": board})
}

// GenerateClueImage handles
// POST /api/v1/missions/{missionID}/clues/{clueID}/image.
func (h *Handler) GenerateClueImage(w http.ResponseWriter, r *http.Request) {
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
	c, err := h.svc.ClueImage(r.Context(), userID, missionID, clueID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"clue": c})
}
