// Package runtime is the agent execution engine. Agents are declarative
// components (manifest + prompt builder + output parser); the runtime owns
// LLM calls, JSON validation, retries with repair prompts, and audit
// logging. Agents never mutate domain state — they return structured
// intentions that application services validate.
package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"casemind/internal/llm"
	apperrors "casemind/pkg/errors"
)

// Manifest declares an agent's identity, contract, and execution policy.
type Manifest struct {
	Name         string
	Version      string
	Description  string
	Capabilities []string
	AllowedTools []string
	InputSchema  string
	OutputSchema string
	MaxRetries   int // retries after the first attempt
	Timeout      time.Duration
	Temperature  float64
	MaxTokens    int
}

// Task is one unit of agent work.
type Task struct {
	Type      string
	CaseID    uuid.UUID
	MissionID uuid.UUID
	UserID    uuid.UUID
	Input     any
}

// Agent is implemented by every agent (legacy case agents and mission agents).
type Agent interface {
	Manifest() Manifest
	// Prompt builds the LLM messages for the task.
	Prompt(task Task) ([]llm.Message, error)
	// Parse validates raw model output against the agent's output schema
	// and returns the typed intention.
	Parse(raw []byte) (any, error)
}

// TaskParser is optionally implemented by agents that handle multiple task
// types with different output schemas. When implemented, it takes precedence
// over Parse.
type TaskParser interface {
	ParseTask(taskType string, raw []byte) (any, error)
}

// Fallbacker is implemented by agents that can produce a safe deterministic
// output when the provider repeatedly returns malformed JSON.
type Fallbacker interface {
	Fallback(task Task, raw []byte, err error) (any, bool)
}

// Meta carries execution metadata (model, token usage) for billing.
type Meta struct {
	Model string
	Usage llm.Usage
}

// Run is the persisted audit record of one agent execution.
type Run struct {
	ID         uuid.UUID
	CaseID     *uuid.UUID
	MissionID  *uuid.UUID
	AgentName  string
	TaskType   string
	InputHash  string
	Input      []byte
	Output     []byte
	Model      string
	Status     string // "success" | "failed"
	LatencyMS  int
	TokenUsage []byte
	Error      string
	CreatedAt  time.Time
}

type RunRepository interface {
	Insert(ctx context.Context, run *Run) error
}

type Runtime struct {
	llm  llm.Client
	runs RunRepository
	log  *slog.Logger
}

func New(client llm.Client, runs RunRepository, log *slog.Logger) *Runtime {
	return &Runtime{llm: client, runs: runs, log: log}
}

// Execute runs the agent for the task: prompt -> LLM -> parse, retrying
// invalid output with a repair prompt, and always logging an AgentRun.
func (r *Runtime) Execute(ctx context.Context, agent Agent, task Task) (any, error) {
	out, _, err := r.ExecuteMeta(ctx, agent, task)
	return out, err
}

// ExecuteMeta is Execute plus execution metadata (model, aggregated token
// usage across attempts) so callers can settle wallet charges and write LLM
// usage logs.
func (r *Runtime) ExecuteMeta(ctx context.Context, agent Agent, task Task) (any, *Meta, error) {
	manifest := agent.Manifest()
	start := time.Now()

	inputJSON, err := json.Marshal(task.Input)
	if err != nil {
		return nil, nil, apperrors.Internal(err, "marshal agent input")
	}
	sum := sha256.Sum256(inputJSON)
	inputHash := hex.EncodeToString(sum[:])

	messages, err := agent.Prompt(task)
	if err != nil {
		return nil, nil, apperrors.Internal(err, "build agent prompt")
	}

	parse := agent.Parse
	if tp, ok := agent.(TaskParser); ok {
		parse = func(raw []byte) (any, error) { return tp.ParseTask(task.Type, raw) }
	}

	timeout := manifest.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}

	attempts := manifest.MaxRetries + 1
	var lastErr error
	var usage llm.Usage
	var model string
	var rawOutput string

	for attempt := 0; attempt < attempts; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		resp, callErr := r.llm.Chat(callCtx, llm.Request{
			Messages:    messages,
			Temperature: manifest.Temperature,
			MaxTokens:   manifest.MaxTokens,
			JSONMode:    true,
		})
		cancel()
		if callErr != nil {
			lastErr = callErr
			continue
		}
		usage.PromptTokens += resp.Usage.PromptTokens
		usage.CompletionTokens += resp.Usage.CompletionTokens
		usage.TotalTokens += resp.Usage.TotalTokens
		model = resp.Model
		rawOutput = resp.Content

		out, parseErr := parse([]byte(ExtractJSON(resp.Content)))
		if parseErr == nil {
			r.logRun(ctx, &Run{
				CaseID: idPtr(task.CaseID), MissionID: idPtr(task.MissionID),
				AgentName: manifest.Name, TaskType: task.Type,
				InputHash: inputHash, Input: inputJSON, Output: []byte(ExtractJSON(resp.Content)),
				Model: model, Status: "success", LatencyMS: int(time.Since(start).Milliseconds()),
				TokenUsage: usageJSON(usage),
			})
			return out, &Meta{Model: model, Usage: usage}, nil
		}
		lastErr = parseErr
		// Repair prompt: feed the invalid output back (truncated — the model
		// only needs to see what was wrong, not pay for the full echo) and
		// demand valid JSON.
		messages = append(messages,
			llm.Message{Role: llm.RoleAssistant, Content: truncate(resp.Content, 1500)},
			llm.Message{Role: llm.RoleUser, Content: fmt.Sprintf(
				"Your previous output was invalid: %v. Respond again with ONLY a valid JSON object matching the required schema. No prose, no markdown fences.", parseErr)},
		)
	}

	r.logRun(ctx, &Run{
		CaseID: idPtr(task.CaseID), MissionID: idPtr(task.MissionID),
		AgentName: manifest.Name, TaskType: task.Type,
		InputHash: inputHash, Input: inputJSON, Output: []byte(safeRawJSON(rawOutput)),
		Model: model, Status: "failed", LatencyMS: int(time.Since(start).Milliseconds()),
		TokenUsage: usageJSON(usage), Error: fmt.Sprintf("%v", lastErr),
	})
	if fb, ok := agent.(Fallbacker); ok {
		if out, recovered := fb.Fallback(task, []byte(rawOutput), lastErr); recovered {
			r.log.Warn("agent fallback used", "agent", manifest.Name, "task", task.Type, "error", lastErr)
			return out, &Meta{Model: model, Usage: usage}, nil
		}
	}
	return nil, &Meta{Model: model, Usage: usage}, apperrors.Unavailable(lastErr, fmt.Sprintf("agent %s failed after %d attempts", manifest.Name, attempts))
}

func (r *Runtime) logRun(ctx context.Context, run *Run) {
	// Audit logging must not be cancelled along with the request.
	logCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := r.runs.Insert(logCtx, run); err != nil {
		r.log.Error("persist agent run", "agent", run.AgentName, "error", err)
	}
	r.log.Info("agent run", "agent", run.AgentName, "task", run.TaskType,
		"status", run.Status, "latency_ms", run.LatencyMS)
}

func idPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func usageJSON(u llm.Usage) []byte {
	b, _ := json.Marshal(u)
	return b
}

func safeRawJSON(raw string) string {
	extracted := ExtractJSON(raw)
	if json.Valid([]byte(extracted)) {
		return extracted
	}
	b, _ := json.Marshal(map[string]string{"raw": truncate(raw, 4000)})
	return string(b)
}

// ExtractJSON strips markdown fences and surrounding prose, returning the
// first top-level JSON object in the text.
func ExtractJSON(s string) string {
	s = strings.TrimSpace(s)
	if fenced, ok := strings.CutPrefix(s, "```json"); ok {
		s = fenced
	} else if fenced, ok := strings.CutPrefix(s, "```"); ok {
		s = fenced
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	s = strings.TrimSpace(s)
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return s
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s[start:]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
