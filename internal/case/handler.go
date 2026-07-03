package cases

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

type createCaseRequest struct {
	Type       string `json:"type"`
	Difficulty string `json:"difficulty"`
	Language   string `json:"language"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	var req createCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	c, err := h.svc.Create(r.Context(), userID, req.Type, req.Difficulty, req.Language)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]any{"case": c})
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
	response.JSON(w, http.StatusOK, map[string]any{"cases": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	dashboard, err := h.svc.Get(r.Context(), userID, caseID)
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
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	c, err := h.svc.Archive(r.Context(), userID, caseID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"case": c})
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := h.svc.Events(r.Context(), userID, caseID, limit)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"events": events})
}

// Stream pushes live case events over Server-Sent Events.
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	caseID, ok := httpx.PathUUID(w, r, "caseID")
	if !ok {
		return
	}
	if err := h.svc.EnsureOwned(r.Context(), userID, caseID); err != nil {
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
	fmt.Fprintf(w, "event: connected\ndata: {\"case_id\":%q}\n\n", caseID)
	flusher.Flush()

	events, unsubscribe := h.bus.Subscribe(caseID)
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
