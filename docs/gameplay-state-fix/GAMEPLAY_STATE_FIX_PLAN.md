# AgentVerse — Gameplay State Fix Plan

Brief: the project works technically, but **gameplay progression is broken** —
the game doesn't clearly advance, locked locations never open, there's no way
to confirm evidence or submit a hypothesis, and completion is fuzzy. Fix the
progression loop so it is **state-driven**:

`player action → validation → state update → timeline event → objective update
→ map unlock → next action → UI feedback`

Rule: do not rewrite. Extend existing services. AI never mutates the DB — backend
services validate and apply state. Never expose the WorldBible.

## 1. Current State-Flow Audit

Backend today:
- **Locations** (`internal/gamemap`): statuses `hidden | discovered | locked |
  visited`. `Detail` marks discovered→visited. `Action` runs LocationAgent and
  can discover clues. **There is no code that ever transitions `locked →
  available`.** Locked locations are permanent dead ends → progression stalls.
- **Clues** (`internal/clue`): only a `discovered` bool + `image_status`. Inspect
  adds facts but does not change a lifecycle status. **No confirm / evidence
  board / lifecycle.**
- **Timeline** (`internal/missiontimeline`): curates emitted events; good. But
  no events exist for confirm / unlock / hypothesis (those actions don't exist).
- **Completion** (`internal/missioncomplete`): `completion-check` + `complete`
  (JudgeAgent). Readiness = discovered clues ≥ target, ≥1 location, ≥1 char.
  No "final decision with linked evidence"; no lightweight hypothesis checkpoint.
- **Time** (`internal/timeengine`): advance-time exists and emits events.
- **AI** (`internal/guidance`): context from discovered facts/locations/chars;
  structured but thin — no confirmed-evidence set, no "needs_more_evidence".

## 2. Broken Scenarios

1. Player discovers all clues at open locations but a `locked` location never
   opens → cannot progress, cannot finish.
2. Player has a strong clue but no way to "confirm" it as evidence → progress
   feels unaffected by their reasoning.
3. Player wants to say "I think X" mid-mission → no hypothesis endpoint; the
   only option is the final, charged JudgeAgent.
4. Confirming/among evidence does not visibly update the timeline or map.

## 3. State Model Changes

- **Clue lifecycle**: add `clues.status` = `discovered | inspected | confirmed`
  (migration 0004; `discovered` bool kept for back-compat). Inspect sets
  `inspected`; a new confirm action sets `confirmed`.
- **Location unlock**: keep the four statuses; add a deterministic **unlock
  engine** that transitions the earliest `locked` location → `discovered` on
  qualifying triggers, with a human reason.
- **Mission `decision_ready`**: represented by `can_complete = true` (already
  emitted as the `mission_ready_to_complete` milestone) — no new DB status, to
  avoid churn.

## 4. New/Changed Endpoints (state-changing → standard envelope)

Every state-changing action returns the **progression envelope**:
```json
{ "message": "...", "state_changes": [], "timeline_events": [],
  "objective_updates": [], "unlocked_locations": [],
  "next_recommended_actions": [] }
```

- `POST /api/v1/missions/{id}/clues/{clueID}/confirm` — confirm a discovered
  clue as evidence. Free (a player decision, not an AI call). Effects: status→
  confirmed, `evidence_confirmed` timeline event, **unlock engine** may open the
  next locked location (`location_unlocked` event), progress recompute.
- `POST /api/v1/missions/{id}/hypothesis` — submit a working theory
  `{answer, clue_ids}`. Deterministic verdict from confirmed-evidence coverage
  (`too_early | unsupported | partially_correct`); emits `hypothesis_submitted`;
  returns feedback + next actions. Never claims `correct`/`wrong` (that is the
  final JudgeAgent's job) and never reveals hidden truth.

## 5. Map Unlock Engine (`internal/progression`)

Deterministic, ordered by `created_at`:
- Trigger: a clue becomes **confirmed** (primary), or discovered-clue count
  crosses a threshold. On trigger, open the single earliest `locked` location
  and record a reason ("Unlocked after confirming evidence: <clue>").
- One unlock per confirm keeps pacing legible and prevents opening everything
  at once. Returns `unlocked_locations: [{id, name, reason}]`.
- Emits a `location_unlocked` timeline event per unlocked location.

## 6. Timeline Changes

New event types wired through the curator (`internal/missiontimeline`):
`evidence_confirmed`, `hypothesis_submitted`, `location_unlocked` (already
mapped) → curated into the player timeline with importance and related ids.

## 7. Completion Changes

Readiness continues to gate `complete`. Confirmed evidence is surfaced to the
player (evidence board) and feeds the AI context and hypothesis verdict. The
final decision remains `POST /complete` (JudgeAgent) — documented as the
"final decision" step; hypothesis is the mid-mission checkpoint before it.

## 8. AI Context Changes

Guidance context packet gains: confirmed-evidence titles and a
`needs_more_evidence` signal derived from readiness, so the assistant can say
"you still need to confirm N clues" instead of guessing. Output already
structured (message/hint_level/referenced_items); the language contract from
the prior phase stays.

## 9. Files to Modify

Backend:
- `migrations/0004_clue_lifecycle.sql` (new)
- `internal/clue/entity.go`, `repository.go`, `service.go` (status + confirm)
- `internal/gamemap/repository.go` (list locked / unlock helpers)
- `internal/progression/*` (new: unlock engine + confirm/hypothesis orchestration)
- `internal/missiontimeline/missiontimeline.go` (curate new event types)
- `internal/platform/http/router.go`, `cmd/api/main.go` (wire routes)
- `docs/openapi.yaml`

Frontend:
- `web/src/shared/types/api.ts`, `endpoints.ts`, mock adapter
- `web/src/features/clues/ClueDetailPage.tsx` (Confirm-evidence action)
- surface unlocked locations via toast + timeline refresh

## 10. Tests

- confirming a discovered clue sets status=confirmed and emits `evidence_confirmed`
- confirming unlocks the earliest locked location and emits `location_unlocked`
- confirming an already-confirmed clue is idempotent (no double unlock)
- hypothesis with too little evidence → `too_early`
- hypothesis with evidence but no linked clues → `unsupported`
- timeline curates the new event types
- `go test ./...`, `web` build/lint/test

## 11. Scope Note (honest)

This pass implements the **core progression loop**: clue→evidence lifecycle,
deterministic map unlock, hypothesis checkpoint, and the standard envelope, with
tests. Explicit `/visit` and `/inspect` endpoints (today folded into `/actions`
and `Detail`), the full time-consequence matrix, and a bespoke final-decision
payload with linked evidence are documented here and left as the next iteration
so each change ships tested and revertable.
