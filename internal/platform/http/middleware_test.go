package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPrivacyGuardBlocksTruthLayerLeaks: any handler that (incorrectly)
// serializes forbidden fields must be blocked with a 500 and the body
// replaced.
func TestPrivacyGuardBlocksTruthLayerLeaks(t *testing.T) {
	leaky := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"suspect":{"name":"Elena","is_culprit":true,"secrets":["forged paintings"]}}}`))
	})
	handler := PrivacyGuard(slog.Default())(leaky)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/test", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "is_culprit") || strings.Contains(body, "forged") {
		t.Fatalf("blocked response still leaks: %s", body)
	}
	var envelope map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("blocked response is not valid JSON: %v", err)
	}
}

func TestPrivacyGuardPassesCleanResponses(t *testing.T) {
	clean := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"suspect":{"name":"Elena","stress_level":40}}}`))
	})
	handler := PrivacyGuard(slog.Default())(clean)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/test", nil))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Elena") {
		t.Fatal("clean body was altered")
	}
}

// TestPrivacyGuardStreamsSSEThrough: non-JSON responses (the event stream)
// must pass through unbuffered.
func TestPrivacyGuardStreamsSSEThrough(t *testing.T) {
	sse := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: connected\ndata: {}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	handler := PrivacyGuard(slog.Default())(sse)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/stream", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "event: connected") {
		t.Fatal("SSE body did not pass through")
	}
	if !rec.Flushed {
		t.Fatal("SSE flush did not pass through")
	}
}
