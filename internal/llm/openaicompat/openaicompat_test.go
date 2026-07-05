package openaicompat

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"casemind/internal/config"
	"casemind/internal/llm"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(config.LLM{
		Provider:    "gemini",
		BaseURL:     srv.URL + "/v1",
		APIKey:      "test-key",
		Model:       "test-model",
		Timeout:     5 * time.Second,
		MaxTokens:   256,
		Temperature: 0.7,
	}, discardLogger())
}

func okBody(content string) string {
	return `{"model":"test-model","choices":[{"message":{"content":` +
		string(mustJSON(content)) + `}}],"usage":{"prompt_tokens":7,"completion_tokens":2,"total_tokens":9,"cost":0.00000475}}`
}

func TestChatSuccess(t *testing.T) {
	var gotAuth string
	p := testProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(okBody("OK.")))
	})
	resp, err := p.Chat(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "OK." {
		t.Errorf("content = %q", resp.Content)
	}
	if resp.Usage.Cost == 0 {
		t.Errorf("expected provider-reported cost to be parsed")
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth header = %q", gotAuth)
	}
}

func TestRetryOn500ThenSuccess(t *testing.T) {
	calls := 0
	p := testProvider(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(okBody("recovered")))
	})
	resp, err := p.Chat(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
	if resp.Content != "recovered" {
		t.Errorf("content = %q", resp.Content)
	}
}

func TestNoRetryOnAuthError(t *testing.T) {
	calls := 0
	p := testProvider(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"bad key"}}`))
	})
	_, err := p.Chat(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("auth errors must not be retried; got %d calls", calls)
	}
}

func TestNoRetryOnMalformedResponse(t *testing.T) {
	calls := 0
	p := testProvider(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{not json`))
	})
	_, err := p.Chat(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("malformed bodies must not be retried; got %d calls", calls)
	}
}

func TestVisionPayloadUsesContentParts(t *testing.T) {
	var body map[string]any
	p := testProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Write([]byte(okBody("seen")))
	})
	_, err := p.Chat(context.Background(), llm.Request{
		Messages: []llm.Message{{
			Role:    llm.RoleUser,
			Content: "what is this?",
			Images:  []llm.Image{{Data: []byte{1, 2, 3}, MIME: "image/png"}},
		}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	messages := body["messages"].([]any)
	content := messages[0].(map[string]any)["content"]
	parts, ok := content.([]any)
	if !ok {
		t.Fatalf("expected multimodal content parts, got %T", content)
	}
	if len(parts) != 2 {
		t.Fatalf("expected text + image parts, got %d", len(parts))
	}
	img := parts[1].(map[string]any)
	if img["type"] != "image_url" {
		t.Errorf("part type = %v", img["type"])
	}
	url := img["image_url"].(map[string]any)["url"].(string)
	if !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Errorf("image url = %q", url)
	}
}

func TestPingReportsStatusAndBody(t *testing.T) {
	p := testProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"message":"key expired"}}`))
	})
	err := p.Ping(context.Background())
	if err == nil {
		t.Fatal("expected ping failure")
	}
	msg := err.Error()
	for _, want := range []string{"403", "key expired", "gemini", "test-model"} {
		if !strings.Contains(msg, want) {
			t.Errorf("ping error missing %q: %s", want, msg)
		}
	}
}

func TestJSONModeSetsResponseFormat(t *testing.T) {
	var body map[string]any
	p := testProvider(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Write([]byte(okBody(`{"ok":true}`)))
	})
	_, err := p.Chat(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "json please"}},
		JSONMode: true,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	rf, ok := body["response_format"].(map[string]any)
	if !ok || rf["type"] != "json_object" {
		t.Errorf("response_format = %v", body["response_format"])
	}
}
