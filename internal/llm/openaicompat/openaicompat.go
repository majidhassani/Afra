// Package openaicompat implements llm.Client against any OpenAI-compatible
// chat-completions API. It is the production provider for Gemini through the
// ArvanCloud gateway and works unchanged with other compatible backends
// (OpenAI, GLM, ...): the provider identity comes from configuration, not
// from code.
package openaicompat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"casemind/internal/config"
	"casemind/internal/llm"
	apperrors "casemind/pkg/errors"
)

const (
	maxAttempts     = 4
	baseBackoff     = 500 * time.Millisecond
	maxBackoff      = 8 * time.Second
	maxResponseSize = 4 << 20
)

type Provider struct {
	name        string
	baseURL     string
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpc       *http.Client
	log         *slog.Logger
}

func New(cfg config.LLM, log *slog.Logger) *Provider {
	return &Provider{
		name:        cfg.Provider,
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		httpc:       &http.Client{Timeout: cfg.Timeout},
		log:         log.With("component", "llm", "provider", cfg.Provider, "model", cfg.Model),
	}
}

func (p *Provider) Name() string { return p.name + ":" + p.model }

// Ping verifies connectivity, credentials and model availability with the
// smallest possible request. It returns a descriptive error (HTTP status and
// response body included) so startup failures are diagnosable from logs.
func (p *Provider) Ping(ctx context.Context) error {
	start := time.Now()
	resp, err := p.do(ctx, mustJSON(chatRequest{
		Model:       p.model,
		Messages:    []wireMessage{{Role: llm.RoleUser, Content: "Hello\n\nReply only:\n\nOK"}},
		Temperature: 0,
		MaxTokens:   16,
	}))
	if err != nil {
		return fmt.Errorf("llm connection check failed (provider=%s model=%s base_url=%s): %w",
			p.name, p.model, p.baseURL, err)
	}
	p.log.Info("llm connection verified",
		"latency_ms", time.Since(start).Milliseconds(),
		"reply", truncate(resp.Content, 40))
	return nil
}

// --- wire types ---

// wireMessage carries either a plain string content or multimodal parts.
type wireMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []wireMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	MaxTokens      int           `json:"max_tokens,omitempty"`
	ResponseFormat *respFormat   `json:"response_format,omitempty"`
}

type respFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage llm.Usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func toWire(messages []llm.Message) []wireMessage {
	out := make([]wireMessage, 0, len(messages))
	for _, m := range messages {
		if len(m.Images) == 0 {
			out = append(out, wireMessage{Role: m.Role, Content: m.Content})
			continue
		}
		parts := make([]contentPart, 0, len(m.Images)+1)
		if m.Content != "" {
			parts = append(parts, contentPart{Type: "text", Text: m.Content})
		}
		for _, img := range m.Images {
			parts = append(parts, contentPart{Type: "image_url", ImageURL: &imageURL{
				URL: "data:" + img.MIME + ";base64," + base64.StdEncoding.EncodeToString(img.Data),
			}})
		}
		out = append(out, wireMessage{Role: m.Role, Content: parts})
	}
	return out
}

// Chat sends the request with production retry semantics: transient failures
// (429, 5xx, timeouts, network errors) are retried with exponential backoff
// and jitter; auth errors and invalid requests fail immediately.
func (p *Provider) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	temperature := req.Temperature
	if temperature <= 0 {
		temperature = p.temperature
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = p.maxTokens
	}
	body := chatRequest{
		Model:       p.model,
		Messages:    toWire(req.Messages),
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	if req.JSONMode {
		body.ResponseFormat = &respFormat{Type: "json_object"}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, apperrors.Internal(err, "marshal llm request")
	}

	start := time.Now()
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, apperrors.Unavailable(ctx.Err(), "llm request cancelled")
			case <-time.After(backoff(attempt, lastErr)):
			}
		}
		resp, err := p.do(ctx, payload)
		if err == nil {
			p.log.Info("llm chat completed",
				"latency_ms", time.Since(start).Milliseconds(),
				"attempts", attempt+1,
				"prompt_tokens", resp.Usage.PromptTokens,
				"completion_tokens", resp.Usage.CompletionTokens,
				"cost", resp.Usage.Cost,
				"json_mode", req.JSONMode)
			return resp, nil
		}
		lastErr = err
		if !retryable(ctx, err) {
			break
		}
		p.log.Warn("llm chat attempt failed, retrying",
			"attempt", attempt+1, "error", err)
	}
	p.log.Error("llm chat failed",
		"latency_ms", time.Since(start).Milliseconds(), "error", lastErr)
	return nil, apperrors.Unavailable(lastErr, "llm provider unavailable")
}

type httpStatusError struct {
	status     int
	body       string
	retryAfter time.Duration
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("llm http status %d: %s", e.status, e.body)
}

// retryable reports whether the error is worth another attempt: rate limits,
// server errors, timeouts and transport failures are; auth errors, invalid
// requests and context cancellation are not.
func retryable(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	var se *httpStatusError
	if errors.As(err, &se) {
		return se.status == http.StatusTooManyRequests || se.status >= 500
	}
	// Response decode errors are not transport failures; a malformed body
	// will be malformed again.
	var de *decodeError
	if errors.As(err, &de) {
		return false
	}
	// Everything else at this level is a transport failure (DNS, reset,
	// per-request timeout) and is retryable.
	return true
}

// backoff computes exponential backoff with jitter, honoring a server
// Retry-After when present.
func backoff(attempt int, lastErr error) time.Duration {
	var se *httpStatusError
	if errors.As(lastErr, &se) && se.retryAfter > 0 && se.retryAfter <= 30*time.Second {
		return se.retryAfter
	}
	d := baseBackoff << (attempt - 1)
	if d > maxBackoff {
		d = maxBackoff
	}
	// Full jitter: uniform in [d/2, d).
	return d/2 + time.Duration(rand.Int64N(int64(d/2)))
}

type decodeError struct{ err error }

func (e *decodeError) Error() string { return "decode llm response: " + e.err.Error() }
func (e *decodeError) Unwrap() error { return e.err }

func (p *Provider) do(ctx context.Context, payload []byte) (*llm.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.httpc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, maxResponseSize))
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, &httpStatusError{
			status:     httpResp.StatusCode,
			body:       truncate(string(raw), 500),
			retryAfter: parseRetryAfter(httpResp.Header.Get("Retry-After")),
		}
	}
	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, &decodeError{err: err}
	}
	if parsed.Error != nil {
		return nil, &decodeError{err: fmt.Errorf("llm error: %s", parsed.Error.Message)}
	}
	if len(parsed.Choices) == 0 {
		return nil, &decodeError{err: fmt.Errorf("llm returned no choices: %s", truncate(string(raw), 300))}
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return nil, &decodeError{err: fmt.Errorf("llm returned empty content")}
	}
	model := parsed.Model
	if model == "" {
		model = p.model
	}
	return &llm.Response{
		Content: content,
		Usage:   parsed.Usage,
		Model:   model,
	}, nil
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
