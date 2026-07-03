// Package judgeagent implements the JudgeAgent: it evaluates a solve
// attempt. Correctness is decided deterministically by the solve service
// (accused id vs. Case Bible culprit id); the judge produces the score and
// narrative feedback without ever revealing the true culprit on failure.
package judgeagent

import (
	"encoding/json"
	"fmt"
	"time"

	"casemind/internal/agent/runtime"
	"casemind/internal/llm"
)

const (
	Name     = "judge"
	TaskType = "judge_verdict"
)

type Input struct {
	CaseSummary        string   `json:"case_summary"`
	AccusedName        string   `json:"accused_name"`
	AccusedMotive      string   `json:"accused_motive"`
	Reasoning          string   `json:"reasoning"`
	AccusationCorrect  bool     `json:"accusation_correct"`
	RealMotive         string   `json:"real_motive"`
	DiscoveredFacts    []string `json:"discovered_facts"`
	AttemptNumber      int      `json:"attempt_number"`
	AttemptsRemaining  int      `json:"attempts_remaining"`
}

type Output struct {
	Score               int    `json:"score"`
	Feedback            string `json:"feedback"`
	MotiveAssessment    string `json:"motive_assessment"`
	ReasoningAssessment string `json:"reasoning_assessment"`
}

type Agent struct{}

func New() *Agent { return &Agent{} }

func (a *Agent) Manifest() runtime.Manifest {
	return runtime.Manifest{
		Name:         Name,
		Version:      "1.0.0",
		Description:  "Scores solve attempts and writes safe verdict feedback.",
		Capabilities: []string{"judge_verdict"},
		AllowedTools: []string{},
		InputSchema:  "judgeagent.Input",
		OutputSchema: "judgeagent.Output",
		MaxRetries:   2,
		Timeout:      60 * time.Second,
		Temperature:  0.3,
		MaxTokens:    2000,
	}
}

func (a *Agent) Prompt(task runtime.Task) ([]llm.Message, error) {
	in, ok := task.Input.(Input)
	if !ok {
		return nil, fmt.Errorf("judgeagent: unexpected input type %T", task.Input)
	}
	system := fmt.Sprintf(`TASK_TYPE: %s
ACCUSATION_CORRECT: %t
%s

You are the JudgeAgent evaluating a detective's accusation. The backend has already determined whether the accusation is correct (ACCUSATION_CORRECT above). Your job:
- Score the quality of the accusation 0-100 (evidence use, reasoning coherence, motive accuracy). A correct accusation with weak reasoning still scores at least 60; an incorrect one never scores above 50.
- Write feedback for the detective. If the accusation is WRONG: do NOT reveal who the real culprit is, do NOT confirm or deny anyone else's guilt; point at the weakest link in their reasoning instead.
- If CORRECT: acknowledge the solved case and highlight what clinched it.

Respond with ONLY one JSON object:
{"score": int, "feedback": string, "motive_assessment": string, "reasoning_assessment": string}`,
		TaskType, in.AccusationCorrect, runtime.SecurityPreamble)

	ctxJSON, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: "CONFIDENTIAL CONTEXT:\n" + string(ctxJSON)},
	}, nil
}

func (a *Agent) Parse(raw []byte) (any, error) {
	var out Output
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if out.Feedback == "" {
		return nil, fmt.Errorf("feedback is required")
	}
	if out.Score < 0 || out.Score > 100 {
		return nil, fmt.Errorf("score out of range [0,100]: %d", out.Score)
	}
	return &out, nil
}
