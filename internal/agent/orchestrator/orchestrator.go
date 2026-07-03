// Package orchestrator routes agent tasks to registered agents through the
// agent runtime. Application services depend on this — never on providers.
package orchestrator

import (
	"context"

	"casemind/internal/agent/runtime"
	apperrors "casemind/pkg/errors"
)

type Orchestrator struct {
	rt     *runtime.Runtime
	agents map[string]runtime.Agent
}

func New(rt *runtime.Runtime) *Orchestrator {
	return &Orchestrator{rt: rt, agents: map[string]runtime.Agent{}}
}

func (o *Orchestrator) Register(agent runtime.Agent) {
	o.agents[agent.Manifest().Name] = agent
}

// Run executes the named agent and returns its parsed, validated intention.
func (o *Orchestrator) Run(ctx context.Context, agentName string, task runtime.Task) (any, error) {
	out, _, err := o.RunWithMeta(ctx, agentName, task)
	return out, err
}

// RunWithMeta is Run plus execution metadata (model, token usage) so callers
// can settle wallet charges and write LLM usage logs.
func (o *Orchestrator) RunWithMeta(ctx context.Context, agentName string, task runtime.Task) (any, *runtime.Meta, error) {
	agent, ok := o.agents[agentName]
	if !ok {
		return nil, nil, apperrors.Internal(nil, "unknown agent: "+agentName)
	}
	return o.rt.ExecuteMeta(ctx, agent, task)
}
