// Package llm defines the LLM Runtime interface. No handler or domain code
// may call a provider directly — all calls go through the agent runtime,
// which uses this interface.
package llm

import "context"

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
}

type Response struct {
	Content string
	Usage   Usage
	Model   string
}

// Client is implemented by the GLM provider and the mock provider.
type Client interface {
	Chat(ctx context.Context, req Request) (*Response, error)
	Name() string
}
