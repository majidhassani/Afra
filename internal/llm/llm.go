// Package llm defines the LLM Runtime interface. No handler or domain code
// may call a provider directly — all calls go through the agent runtime,
// which uses this interface.
package llm

import "context"

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
)

// Image is an inline image attachment for multimodal (vision) requests.
// Data is the raw bytes; MIME must be one of the SupportedImageMIMEs.
type Image struct {
	Data []byte `json:"-"`
	MIME string `json:"mime"`
}

// SupportedImageMIMEs lists the image types accepted end-to-end.
var SupportedImageMIMEs = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/jpg":  true,
	"image/webp": true,
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// Images attaches inline images to this message for vision-capable
	// providers. Providers without vision support ignore them.
	Images []Image `json:"-"`
}

type Request struct {
	Messages    []Message
	Temperature float64
	MaxTokens   int
	// JSONMode asks the provider for a JSON-object response when supported.
	JSONMode bool
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	// Cost is the provider-reported cost for the call, when available
	// (e.g. the ArvanCloud gateway returns it in the usage block).
	Cost float64 `json:"cost,omitempty"`
}

type Response struct {
	Content string
	Usage   Usage
	Model   string
}

// Client is implemented by every chat provider (remote and mock).
type Client interface {
	Chat(ctx context.Context, req Request) (*Response, error)
	// Ping sends a minimal request to verify connectivity, credentials and
	// model availability. Used by the fail-fast startup check.
	Ping(ctx context.Context) error
	Name() string
}

// ImageRequest asks an image-capable provider to generate one image.
type ImageRequest struct {
	Prompt string
	// Size is "WxH", e.g. "512x512". Providers may round to supported sizes.
	Size string
	// Transparent requests a transparent background when supported.
	Transparent bool
}

// ImageGenerator is implemented by providers that can generate images.
// It is intentionally separate from Client: chat and image generation may be
// served by different backends (the current Gemini gateway is chat-only, so
// avatar generation plugs in its own provider without touching chat code).
type ImageGenerator interface {
	// GenerateImage returns PNG bytes.
	GenerateImage(ctx context.Context, req ImageRequest) ([]byte, error)
	Name() string
}
