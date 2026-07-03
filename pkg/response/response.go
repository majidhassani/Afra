// Package response writes the uniform JSON envelope for all API responses:
// success: {"data": ...}, failure: {"error": {"code": ..., "message": ...}}.
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	apperrors "casemind/pkg/errors"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Envelope struct {
	Data  any        `json:"data,omitempty"`
	Error *ErrorBody `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Envelope{Data: data}); err != nil {
		slog.Error("response encode failed", "error", err)
	}
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Err(w http.ResponseWriter, err error) {
	e := apperrors.AsError(err)
	status := statusFor(e.Kind)
	msg := e.Message
	if e.Kind == apperrors.KindInternal {
		// Never leak internal error details to the client.
		msg = "internal server error"
		slog.Error("internal error", "code", e.Code, "error", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Error: &ErrorBody{Code: e.Code, Message: msg}})
}

func statusFor(kind apperrors.Kind) int {
	switch kind {
	case apperrors.KindInvalid:
		return http.StatusBadRequest
	case apperrors.KindUnauthorized:
		return http.StatusUnauthorized
	case apperrors.KindForbidden:
		return http.StatusForbidden
	case apperrors.KindNotFound:
		return http.StatusNotFound
	case apperrors.KindConflict:
		return http.StatusConflict
	case apperrors.KindRateLimited:
		return http.StatusTooManyRequests
	case apperrors.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
