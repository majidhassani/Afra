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

// supportedLanguages is the set of languages the AI agents localize to.
var supportedLanguages = map[string]bool{"en": true, "fa": true}

// RequestLanguage resolves the caller's preferred AI-response language from the
// request. It prefers the explicit X-App-Language header the game client sends,
// then falls back to the primary Accept-Language tag, and finally defaults to
// English. Only languages the agents actually support are honored.
func RequestLanguage(r *http.Request) string {
	if lang := normalizeLang(r.Header.Get("X-App-Language")); lang != "" {
		return lang
	}
	if accept := r.Header.Get("Accept-Language"); accept != "" {
		// Take the first tag ("fa-IR,en;q=0.8" -> "fa-IR" -> "fa").
		primary := accept
		if i := indexAny(accept, ",;"); i >= 0 {
			primary = accept[:i]
		}
		if lang := normalizeLang(primary); lang != "" {
			return lang
		}
	}
	return "en"
}

func normalizeLang(tag string) string {
	tag = trimSpace(tag)
	if len(tag) >= 2 {
		tag = tag[:2]
	}
	tag = toLower(tag)
	if supportedLanguages[tag] {
		return tag
	}
	return ""
}

func indexAny(s, chars string) int {
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(chars); j++ {
			if s[i] == chars[j] {
				return i
			}
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
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
