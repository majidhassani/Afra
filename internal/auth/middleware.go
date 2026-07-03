package auth

import (
	"net/http"
	"strings"

	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

// RequireAuth validates the Bearer access token and stores the user id in
// the request context.
func RequireAuth(tm *TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, found := strings.CutPrefix(header, "Bearer ")
			if !found || token == "" {
				response.Err(w, apperrors.Unauthorized("unauthenticated", "missing bearer token"))
				return
			}
			userID, err := tm.Parse(token)
			if err != nil {
				response.Err(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
		})
	}
}
