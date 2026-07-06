package wallet

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

// Guard enforces the mandatory LLM cost flow:
//
//	estimate -> reserve -> run agent -> settle (actual) / release (failure)
//
// Every service that triggers a paid agent action takes a *Guard and wraps
// the orchestrator call with Reserve + Settle/Release. Usage is always
// logged to llm_usage_logs, success or failure.
type Guard struct {
	svc *Service
	log *slog.Logger
}

func NewGuard(svc *Service, log *slog.Logger) *Guard {
	return &Guard{svc: svc, log: log}
}

// Reserve prices the action server-side and reserves the coins.
// It returns apperrors.Conflict("insufficient_balance") when the player
// cannot afford the action.
func (g *Guard) Reserve(ctx context.Context, userID uuid.UUID, missionID *uuid.UUID, action string) (*Reservation, error) {
	return g.svc.Reserve(ctx, userID, missionID, action)
}

// Balance exposes the current spendable balance for player-safe HUD views.
func (g *Guard) Balance(ctx context.Context, userID uuid.UUID) (int, error) {
	return g.svc.Balance(ctx, userID)
}

// Settle charges the reserved coins after a successful agent run and writes
// the usage log. Returns the coins actually charged.
func (g *Guard) Settle(ctx context.Context, res *Reservation, agentName string, meta *runtime.Meta) int {
	model := ""
	usage := llm.Usage{}
	if meta != nil {
		model = meta.Model
		usage = meta.Usage
	}
	if _, err := g.svc.Settle(ctx, res, map[string]any{
		"agent": agentName, "model": model, "total_tokens": usage.TotalTokens,
	}); err != nil {
		g.log.Error("wallet settle failed", "reservation", res.ID, "error", err)
		return 0
	}
	g.logUsage(ctx, res, agentName, model, usage, res.Amount, "success")
	return res.Amount
}

// Release refunds the reserved coins after a failed agent run and logs the
// failed usage with zero charge.
func (g *Guard) Release(ctx context.Context, res *Reservation, agentName string) {
	if err := g.svc.Release(ctx, res); err != nil {
		g.log.Error("wallet release failed", "reservation", res.ID, "error", err)
	}
	g.logUsage(ctx, res, agentName, "", llm.Usage{}, 0, "failed_refunded")
}

func (g *Guard) logUsage(ctx context.Context, res *Reservation, agentName, model string, usage llm.Usage, charged int, status string) {
	if err := g.svc.InsertUsageLog(ctx, &UsageLog{
		UserID:       res.UserID,
		MissionID:    res.MissionID,
		AgentName:    agentName,
		ActionType:   res.ActionType,
		Model:        model,
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		CoinsCharged: charged,
		Status:       status,
	}); err != nil {
		g.log.Error("llm usage log failed", "reservation", res.ID, "error", err)
	}
}
