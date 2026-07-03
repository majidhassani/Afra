// Package httpx holds small HTTP handler helpers shared by all modules.
package httpx

import (
	"net/http"

	"github.com/google/uuid"

	"casemind/internal/auth"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

// RequestUser extracts the authenticated user or writes a 401.
func RequestUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		response.Err(w, apperrors.Unauthorized("unauthenticated", "authentication required"))
		return uuid.Nil, false
	}
	return userID, true
}

// PathUUID parses a UUID path parameter or writes a 400.
func PathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		response.Err(w, apperrors.Invalid("invalid_id", name+" must be a valid UUID"))
		return uuid.Nil, false
	}
	return id, true
}
