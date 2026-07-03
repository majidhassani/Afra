package wallet

import (
	"encoding/json"
	"net/http"
	"strconv"

	"casemind/internal/httpx"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	wallet, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"wallet": wallet})
}

func (h *Handler) Transactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.svc.Transactions(r.Context(), userID, limit)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"transactions": items})
}

func (h *Handler) Pricing(w http.ResponseWriter, r *http.Request) {
	if _, ok := httpx.RequestUser(w, r); !ok {
		return
	}
	pricing, err := h.svc.Pricing(r.Context())
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"pricing": pricing})
}

func (h *Handler) ClaimRewardedAd(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	txn, err := h.svc.ClaimRewardedAd(r.Context(), userID)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"transaction": txn})
}

type verifyPurchaseRequest struct {
	Platform  string `json:"platform"`
	ProductID string `json:"product_id"`
	Receipt   string `json:"receipt"`
}

func (h *Handler) VerifyPurchase(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.RequestUser(w, r)
	if !ok {
		return
	}
	var req verifyPurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	txn, err := h.svc.VerifyPurchase(r.Context(), userID, req.Platform, req.ProductID, req.Receipt)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"transaction": txn})
}
