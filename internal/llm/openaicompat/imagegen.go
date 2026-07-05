package openaicompat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"casemind/internal/llm"
)

// ImageGen implements llm.ImageGenerator against an OpenAI-compatible
// POST /images/generations endpoint. It is configured independently from
// the chat client so image generation can point at a different gateway or
// model than chat.
type ImageGen struct {
	name    string
	baseURL string
	apiKey  string
	model   string
	httpc   *http.Client
	log     *slog.Logger
}

type ImageGenConfig struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

func NewImageGen(cfg ImageGenConfig, log *slog.Logger) *ImageGen {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &ImageGen{
		name:    cfg.Provider,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		httpc:   &http.Client{Timeout: timeout},
		log:     log.With("component", "imagegen", "provider", cfg.Provider, "model", cfg.Model),
	}
}

func (g *ImageGen) Name() string { return g.name + ":" + g.model }

type imageGenRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format"`
	// Background is honored by providers that support transparency
	// (e.g. gpt-image); others ignore it.
	Background string `json:"background,omitempty"`
}

type imageGenResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (g *ImageGen) GenerateImage(ctx context.Context, req llm.ImageRequest) ([]byte, error) {
	body := imageGenRequest{
		Model:          g.model,
		Prompt:         req.Prompt,
		N:              1,
		Size:           req.Size,
		ResponseFormat: "b64_json",
	}
	if req.Transparent {
		body.Background = "transparent"
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.baseURL+"/images/generations", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	start := time.Now()
	httpResp, err := g.httpc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, &httpStatusError{status: httpResp.StatusCode, body: truncate(string(raw), 500)}
	}
	var parsed imageGenResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, &decodeError{err: err}
	}
	if parsed.Error != nil {
		return nil, &decodeError{err: fmt.Errorf("image API error: %s", parsed.Error.Message)}
	}
	if len(parsed.Data) == 0 || parsed.Data[0].B64JSON == "" {
		return nil, &decodeError{err: fmt.Errorf("image API returned no image data")}
	}
	img, err := base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
	if err != nil {
		return nil, &decodeError{err: fmt.Errorf("decode image base64: %w", err)}
	}
	g.log.Info("image generated", "latency_ms", time.Since(start).Milliseconds(), "bytes", len(img))
	return img, nil
}
