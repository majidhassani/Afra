package mission

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"casemind/internal/httpx"
	"casemind/internal/notification"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Handler struct {
	svc *Service
	bus *notification.Bus
}

func NewHandler(svc *Service, bus *notification.Bus) *Handler {
	return &Handler{svc: svc, bus: bus}
}

type createMissionRequest struct {
	Type       string `json:"type"`
	Difficulty string `json:"difficulty"`
	Region     string `json:"region"`
	Language   string `json:"language"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	var req createMissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	m, err := h.svc.Create(r.Context(), userID, req.Type, req.Difficulty, req.Region, req.Language)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]any{"mission": m})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.List(r.Context(), userID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"missions": items})
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
	dashboard, err := h.svc.Get(r.Context(), userID, missionID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	m, err := h.svc.Archive(r.Context(), userID, missionID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"mission": m})
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := h.svc.Events(r.Context(), userID, missionID, limit)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	missionID, ok := httpx.PathUUID(w, r, "missionID")
	if !ok {
		return
	}
	if err := h.svc.EnsureOwned(r.Context(), userID, missionID); err != nil {
		response.Err(w, err)
		return
	}
	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		response.Err(w, apperrors.Internal(nil, "streaming unsupported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "event: connected\ndata: {\"mission_id\":%q}\n\n", missionID)
	flusher.Flush()

	events, unsubscribe := h.bus.Subscribe(missionID)
	defer unsubscribe()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, open := <-events:
			if !open {
				return
			}
			payload, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, payload)
			flusher.Flush()
		}
	}
}
