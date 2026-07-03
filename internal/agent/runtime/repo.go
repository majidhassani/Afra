package runtime

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type PGRunRepository struct{ pool *pgxpool.Pool }

func NewPGRunRepository(pool *pgxpool.Pool) *PGRunRepository {
	return &PGRunRepository{pool: pool}
}

func (r *PGRunRepository) Insert(ctx context.Context, run *Run) error {
	input := run.Input
	if len(input) == 0 {
		input = []byte(`{}`)
	}
	output := run.Output
	if len(output) == 0 {
		output = []byte(`{}`)
	}
	tokenUsage := run.TokenUsage
	if len(tokenUsage) == 0 {
		tokenUsage = []byte(`{}`)
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO agent_runs
		 (case_id, mission_id, agent_name, task_type, input_hash, input, output, model, status, latency_ms, token_usage, error)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at`,
		run.CaseID, run.MissionID, run.AgentName, run.TaskType, run.InputHash, input, output,
		run.Model, run.Status, run.LatencyMS, tokenUsage, run.Error,
	).Scan(&run.ID, &run.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "insert agent run")
	}
	return nil
}
