package detective

import (
	"net/http"

	"casemind/internal/auth"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		response.Err(w, apperrors.Unauthorized("unauthenticated", "authentication required"))
		return
	}
	profile, err := h.svc.Profile(r.Context(), userID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"profile": profile})
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		response.Err(w, apperrors.Unauthorized("unauthenticated", "authentication required"))
		return
	}
	history, err := h.svc.History(r.Context(), userID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"history": history})
}
