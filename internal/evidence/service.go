package evidence

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	evidenceagent "casemind/internal/agent/evidence"
	"casemind/internal/agent/orchestrator"
	"casemind/internal/agent/runtime"
	"casemind/internal/history"
	"casemind/internal/location"
	"casemind/internal/memory"
	"casemind/internal/timeline"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

// CaseGateway is implemented by cases.Service.
type CaseGateway interface {
	EnsureOwned(ctx context.Context, userID, caseID uuid.UUID) error
	EnsureOwnedOpen(ctx context.Context, userID, caseID uuid.UUID) error
	SummaryOf(ctx context.Context, userID, caseID uuid.UUID) (string, error)
}

type Service struct {
	repo      Repository
	guard     CaseGateway
	locations location.Repository
	timelines timeline.Repository
	memory    *memory.Service
	orch      *orchestrator.Orchestrator
	recorder  *history.Recorder
	log       *slog.Logger
}

func NewService(
	repo Repository,
	guard CaseGateway,
	locations location.Repository,
	timelines timeline.Repository,
	mem *memory.Service,
	orch *orchestrator.Orchestrator,
	recorder *history.Recorder,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo, guard: guard, locations: locations, timelines: timelines,
		memory: mem, orch: orch, recorder: recorder, log: log,
	}
}

func (s *Service) List(ctx context.Context, userID, caseID uuid.UUID) ([]PublicEvidence, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListDiscovered(ctx, caseID)
	if err != nil {
		return nil, err
	}
	return PublicList(items), nil
}

// EvidenceDetail is one discovered piece of evidence plus its inspections.
type EvidenceDetail struct {
	Evidence    PublicEvidence `json:"evidence"`
	Inspections []Inspection   `json:"inspections"`
}

func (s *Service) Get(ctx context.Context, userID, caseID, evidenceID uuid.UUID) (*EvidenceDetail, error) {
	if err := s.guard.EnsureOwned(ctx, userID, caseID); err != nil {
		return nil, err
	}
	e, err := s.getDiscovered(ctx, caseID, evidenceID)
	if err != nil {
		return nil, err
	}
	inspections, err := s.repo.ListInspections(ctx, caseID, evidenceID)
	if err != nil {
		return nil, err
	}
	return &EvidenceDetail{Evidence: e.Public(), Inspections: inspections}, nil
}

// InspectionResult is the player-safe outcome of one inspection.
type InspectionResult struct {
	Analysis          string          `json:"analysis"`
	NewFacts          []string        `json:"new_facts"`
	Evidence          PublicEvidence  `json:"evidence"`
	RevealedLocations []uuid.UUID     `json:"revealed_location_ids"`
	RevealedTimeline  int             `json:"revealed_timeline_events"`
}

// Inspect implements the evidence inspection flow: EvidenceAgent analyzes
// the item; the service validates and applies its intentions (facts,
// reliability, related reveals).
func (s *Service) Inspect(ctx context.Context, userID, caseID, evidenceID uuid.UUID, question string) (*InspectionResult, error) {
	if err := validator.New().MaxLen("question", question, 1000).Err(); err != nil {
		return nil, err
	}
	if err := s.guard.EnsureOwnedOpen(ctx, userID, caseID); err != nil {
		return nil, err
	}
	e, err := s.getDiscovered(ctx, caseID, evidenceID)
	if err != nil {
		return nil, err
	}
	summary, err := s.guard.SummaryOf(ctx, userID, caseID)
	if err != nil {
		return nil, err
	}
	prior, err := s.repo.ListInspections(ctx, caseID, evidenceID)
	if err != nil {
		return nil, err
	}

	outAny, err := s.orch.Run(ctx, evidenceagent.Name, runtime.Task{
		Type: evidenceagent.TaskType, CaseID: caseID, UserID: userID,
		Input: evidenceagent.Input{
			CaseSummary:     summary,
			EvidenceTitle:   e.Title,
			EvidenceType:    e.Type,
			Description:     e.Description,
			PublicData:      rawToMap(e.PublicData),
			InternalTruth:   rawToMap(e.InternalTruth),
			Question:        question,
			DiscoveredFacts: s.memory.FactTexts(ctx, caseID),
			InspectionCount: len(prior),
		},
	})
	if err != nil {
		return nil, err
	}
	out := outAny.(*evidenceagent.Output)

	if err := s.repo.AddInspection(ctx, &Inspection{
		EvidenceID: e.ID, CaseID: caseID, Question: question, Analysis: out.Analysis,
	}); err != nil {
		return nil, err
	}
	if out.ReliabilityDelta != 0 {
		if err := s.repo.AdjustReliability(ctx, e.ID, out.ReliabilityDelta); err != nil {
			s.log.Error("adjust reliability", "error", err)
		}
	}
	added := s.memory.AddFacts(ctx, caseID, "inspection:"+e.Title, out.NewFacts)
	facts := make([]string, 0, len(added))
	for _, f := range added {
		facts = append(facts, f.Fact)
	}

	revealedLocations := []uuid.UUID{}
	revealedTimeline := 0
	if out.RevealRelated {
		locIDs := rawToUUIDs(e.RelatedLocationIDs)
		if n, err := s.locations.RevealByIDs(ctx, caseID, locIDs); err != nil {
			s.log.Error("reveal locations", "error", err)
		} else if n > 0 {
			revealedLocations = locIDs
			s.recorder.Emit(ctx, caseID, "locations_discovered", map[string]any{
				"location_ids": locIDs, "via_evidence": e.Title,
			})
		}
		tlIDs := rawToUUIDs(e.RelatedTimelineEventIDs)
		if n, err := s.timelines.RevealByIDs(ctx, caseID, tlIDs); err != nil {
			s.log.Error("reveal timeline events", "error", err)
		} else {
			revealedTimeline = n
			if n > 0 {
				s.recorder.Emit(ctx, caseID, "timeline_events_revealed", map[string]any{
					"count": n, "via_evidence": e.Title,
				})
			}
		}
	}

	s.recorder.Emit(ctx, caseID, "evidence_inspected", map[string]any{
		"evidence_id": e.ID, "title": e.Title, "new_facts": facts,
	})

	updated, err := s.repo.GetByID(ctx, caseID, evidenceID)
	if err != nil {
		updated = e
	}
	return &InspectionResult{
		Analysis:          out.Analysis,
		NewFacts:          facts,
		Evidence:          updated.Public(),
		RevealedLocations: revealedLocations,
		RevealedTimeline:  revealedTimeline,
	}, nil
}

// getDiscovered hides undiscovered evidence completely (404), so the player
// cannot probe for hidden items.
func (s *Service) getDiscovered(ctx context.Context, caseID, evidenceID uuid.UUID) (*Evidence, error) {
	e, err := s.repo.GetByID(ctx, caseID, evidenceID)
	if err != nil {
		return nil, err
	}
	if !e.Discovered {
		return nil, apperrors.NotFound("evidence_not_found", "evidence not found")
	}
	return e, nil
}

func rawToMap(raw json.RawMessage) map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func rawToUUIDs(raw json.RawMessage) []uuid.UUID {
	var strs []string
	if err := json.Unmarshal(raw, &strs); err != nil {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(strs))
	for _, s := range strs {
		if id, err := uuid.Parse(s); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
