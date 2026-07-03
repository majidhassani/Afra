// Package glm implements the llm.Client against GLM 5.2 through an
// OpenAI-compatible chat-completions API.
package glm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"casemind/internal/config"
	"casemind/internal/llm"
	apperrors "casemind/pkg/errors"
)

type Provider struct {
	baseURL     string
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpc       *http.Client
}

func New(cfg config.LLM) *Provider {
	return &Provider{
		baseURL:     cfg.BaseURL,
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		httpc:       &http.Client{Timeout: cfg.Timeout},
	}
}

func (p *Provider) Name() string { return "glm:" + p.model }

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []llm.Message `json:"messages"`
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
		Messages:    req.Messages,
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

	// Retry transient failures (network errors, 429, 5xx) with backoff.
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, apperrors.Unavailable(ctx.Err(), "llm request cancelled")
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		resp, err := p.do(ctx, payload)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !retryable(err) {
			break
		}
	}
	return nil, apperrors.Unavailable(lastErr, "llm provider unavailable")
}

type httpStatusError struct {
	status int
	body   string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("llm http status %d: %s", e.status, e.body)
}

func retryable(err error) bool {
	if se, ok := err.(*httpStatusError); ok {
		return se.status == http.StatusTooManyRequests || se.status >= 500
	}
	// Network-level errors are retryable.
	return true
}

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
	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, &httpStatusError{status: httpResp.StatusCode, body: truncate(string(raw), 500)}
	}
	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode llm response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("llm error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("llm returned no choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return nil, fmt.Errorf("llm returned empty content")
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
