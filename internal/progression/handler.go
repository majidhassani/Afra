package progression

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"casemind/internal/httpx"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ConfirmEvidence handles
// POST /api/v1/missions/{missionID}/clues/{clueID}/confirm.
func (h *Handler) ConfirmEvidence(w http.ResponseWriter, r *http.Request) {
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
	env, err := h.svc.ConfirmEvidence(r.Context(), userID, missionID, clueID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, env)
}

type hypothesisRequest struct {
	Answer  string   `json:"answer"`
	ClueIDs []string `json:"clue_ids"`
}

// SubmitHypothesis handles POST /api/v1/missions/{missionID}/hypothesis.
func (h *Handler) SubmitHypothesis(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	var req hypothesisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	clueIDs := make([]uuid.UUID, 0, len(req.ClueIDs))
	for _, s := range req.ClueIDs {
		if id, err := uuid.Parse(s); err == nil {
			clueIDs = append(clueIDs, id)
		}
	}
	res, err := h.svc.SubmitHypothesis(r.Context(), userID, missionID, req.Answer, clueIDs)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}
