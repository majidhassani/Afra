# Playable Mission Implementation Plan

> **Status: implemented** (2026-07). All backend + frontend phases below are
> in the tree: migration `0006_playable_mission.sql`, stage engine in
> `internal/mission/{stage,stageengine,timecost,gameplaystatus}.go`, report
> center in `internal/report/`, board art in `internal/visualasset/board.go`,
> routes in `internal/platform/http/router.go`, spec in `docs/openapi.yaml`,
> and the game UI in `web/src/features/game/` + `features/reports/`.
> `go test ./...` (97) and `vitest` (13 files) pass; verified in the browser
> against the mock API. One deliberate deviation: the mission board art is
> generated lazily by the client on first mission load (best effort) instead
> of inside the generation pipeline, keeping visualasset and the generator
> decoupled.

Goal: make every AgentVerse mission a clearly playable game level — visible
stages, a required-clue goal, a suspect-reveal moment, per-stage rewards,
progress the player can feel, a report center that drives state, time as a
real cost, AI-generated scenario board art, and a Unity-style game UI.

This plan builds on what already exists. It does **not** rewrite the engine.

## What already exists (reused, not rebuilt)

| Spec requirement | Existing foundation |
| --- | --- |
| Progress % / risk | `internal/mission/progress.go` + `dashboard.go` (deterministic `MissionProgress`, `RiskScore`) |
| Required clues | `Objective.RequiredClues`, `MandatoryClueTarget`, `EvaluateReadiness` |
| Timeline | `missionevent` recorder + `missiontimeline` curated view |
| Time engine | `timeengine` (paid advance, TimeAgent, DirectorAgent), mission clock (`FormatClock`) |
| World-driven UI themes | `mission.WorldState` (`theme_id`, weather, time-of-day, danger) |
| Final judgment / debrief data | `missioncomplete` (judge, XP, coins, stars, result JSON) |
| Evidence lifecycle + unlock engine | `progression` (confirm → unlock next location, uniform `Envelope`) |
| Image generation | `avatar` + `visualasset` (API + procedural fallback, data-URL persistence) |
| Rewards plumbing | `wallet` (credits), `playerprofile.ApplyMissionResult` |

## What is missing (the work)

1. **Mission stage system** — no explicit stages; objectives are flat.
2. **Report center** — no `/reports` endpoint, no report-driven state updates.
3. **Suspect reveal** — no defined stage where the suspect unlocks.
4. **Stage rewards** — rewards only exist at final completion.
5. **Time as action cost** — location actions/chats/inspects don't advance the clock; no preview.
6. **Scenario board art** — avatars/clue images exist, but no mission board backgrounds.
7. **Unified gameplay-status payload** — clients assemble state from many calls.
8. **Unity-style UI** — web app still reads as pages/cards, not a game scene.

---

## Backend design

### B1. Stage model (`internal/mission/stage.go` + migration `0006_mission_stages.sql`)

Stages are stored like objectives: a JSONB `stages` column on `missions`.

```go
type Stage struct {
    ID                string      `json:"id"`     // arrival, first_clues, identify_suspect, confirm_evidence, final_report, debrief
    Title             string      `json:"title"`
    Description       string      `json:"description"`
    Status            string      `json:"status"`   // locked | active | completed
    Progress          int         `json:"progress"` // 0-100, derived
    RequiredClueCount int         `json:"required_clue_count"`
    FoundClueCount    int         `json:"found_clue_count"` // derived at read time
    RequiredActions   []StageAction `json:"required_actions"`
    Reward            StageReward `json:"reward"`
    UnlockOnComplete  StageUnlock `json:"unlock_on_complete"`
}

type StageAction struct { // e.g. {type:"interview_characters", count:2}
    Type  string `json:"type"`  // find_clues | confirm_evidence | interview_characters | visit_locations | submit_report:<type> | final_decision
    Count int    `json:"count"`
    Done  int    `json:"done"`  // derived
}

type StageReward struct {
    XP              int    `json:"xp"`
    Coins           int    `json:"coins"`
    Badge           string `json:"badge,omitempty"`
    UnlockHint      bool   `json:"unlock_hint,omitempty"`
}

type StageUnlock struct {
    NextStageID     string `json:"next_stage_id,omitempty"`
    UnlockLocation  bool   `json:"unlock_location,omitempty"`  // open next locked location
    RevealSuspect   bool   `json:"reveal_suspect,omitempty"`   // suspect-reveal stage
}
```

**Default template** (deterministic, scaled by difficulty + actual clue/character
counts, no LLM): Arrival → First Clues → Identify Suspect (non-detective types:
"Identify Cause/Source") → Confirm Evidence → Final Report → Debrief.

- Generated at mission-generation time (`generator.go` step 7).
- **Backfill**: `Mission.ParsedStages()` lazily derives the default template for
  pre-stage missions so old missions stay playable (persisted on first
  state-changing evaluation).

### B2. Stage engine (`internal/mission/stageengine.go`)

`EvaluateStages(ctx, missionID) (*StageUpdate, error)`:
- computes derived counters (found clues, confirmed evidence, interviews,
  visits, accepted reports) from existing repositories;
- completes the active stage when all requirements are met; activates the next;
- persists the stages JSONB **only on transition** (idempotent otherwise);
- on transition emits timeline events (`stage_completed`, `stage_unlocked`,
  `suspect_identified` when `RevealSuspect`), credits stage rewards
  (wallet + profile XP), unlocks the next locked location, reveals the
  suspect character (`characters.role='suspect'` marked visible + event);
- returns a `StageUpdate{Completed []Stage, Activated *Stage, Rewards []StageReward, SuspectRevealed bool}`
  that handlers fold into their response envelopes.

Called from every state-changing flow: report submit, evidence confirm,
location action, character chat, time advance, and (lazily) gameplay-status.

Rewards are granted exactly once because a transition persists the new stage
status transactionally before crediting.

**Suspect reveal**: the suspect is the character the WorldBible truth points to —
never exposed directly. The reveal stage flips the *stage-designated* character
(highest `Importance` non-guide with `role` suspect-like, chosen at generation
and stored as `suspect_character_id` in the mission `stages` metadata) to
visible/unlocked, emits `suspect_identified`, and the HUD switches suspect
status: `hidden → identified`. Hidden truth (guilt) stays server-side.

### B3. Report center (`internal/report/`, migration table `mission_reports`)

- `POST /api/v1/missions/{missionID}/reports`
  body: `{type, title?, summary, linked_clue_ids[], suspect_character_id?}`
  types: `clue_report | suspect_report | progress_report | incident_report | final_report`
- Deterministic backend validation (no AI mutation):
  - `clue_report`: ≥1 linked discovered clue → accepted.
  - `suspect_report`: requires the suspect-reveal stage active/completed **and**
    ≥ stage's required confirmed evidence linking; otherwise rejected with the
    exact missing requirements ("Confirm 2 more pieces of evidence").
  - `progress_report`: always accepted; snapshots progress.
  - `incident_report`: accepted; records a world event.
  - `final_report`: gate = `EvaluateReadiness`; when ready it routes through the
    existing `missioncomplete.Complete` judge flow (charged), else rejected with
    `missing_requirements`.
- Response (single envelope, superset of `progression.Envelope`):
  `{verdict, feedback, report, stage_updates, state_changes, timeline_events,
    rewards, unlocked_locations, unlocked_characters, time_update,
    next_recommended_actions}`
- Submitting a report costs mission time (+15 min, see B5) and emits
  `report_submitted` / `report_accepted` / `report_rejected` timeline events.
- `GET /api/v1/missions/{missionID}/reports` lists past reports for the
  Report Terminal UI.

### B4. Gameplay status + action preview

- `GET /api/v1/missions/{missionID}/gameplay-status` — one payload the HUD can
  live on: mission, stages + `current_stage`, clue counter
  (`found/required` for the current stage and mission-wide), suspect status
  (`hidden|identified` + character id once revealed), progress %, risk,
  world state/theme, current time, next reward (the active stage's reward),
  pending-report flag (report CTA glow = a stage requires `submit_report:*`
  that isn't done), board art URLs, next recommended actions.
  Implementation: wraps `Dashboard()` + `EvaluateStages` (lazy, idempotent).
- `POST /api/v1/missions/{missionID}/actions/preview`
  body `{action_type, target_id?}` → `{time_cost_minutes, coin_cost, risk_note}`.
  Deterministic table (B5) + location `RiskLevel` → risk note. No state change.

### B5. Time as action cost (`internal/mission/timecost.go`)

Fixed, previewable costs (minutes of mission time):

| action | cost |
| --- | --- |
| travel / first visit of a location | 15 |
| location action (search/inspect/scan) | 20 |
| character chat message | 10 |
| clue inspect / explain | 10 |
| report submit | 15 |
| intentional wait | player-chosen (existing advance flow) |

Each flow commits the clock (`CommitClock`) after success, emits
`time_advanced`, and runs a **deterministic due-event sweep**: scheduled
WorldBible `TimelineTruth` events whose `at_minutes` fall inside the advance
window are surfaced as `world_event` timeline entries and merged into public
state via the existing sanitizer (weather/risk only). The paid TimeAgent flow
stays for intentional waiting; per-action costs are free and deterministic.
Response of every gameplay action gains `time_update: {new_time, minutes}` and
triggered `world_events` so the client can show "time passed" + popups.

### B6. Scenario board art (`internal/visualasset` extension, migration column `missions.board_art JSONB`)

- `POST /api/v1/missions/{missionID}/art/board/generate` body `{board_type}`:
  `mission_board_background | map_board_background | report_center_background |
  debrief_background | loading_screen`.
- Prompt built server-side from mission type, region/environment,
  world-state time-of-day + weather, risk mood; hard constraints appended:
  "wide empty sky/foreground areas for HUD, no text, no logos, no UI labels,
  no watermarks". Prompts never leave the server (privacy rule preserved).
- Rendering: `llm.ImageGenerator` (1024×576-ish), fallback = procedural
  theme-gradient board so the feature always works.
- Persisted per board type as data-URL + status + version in `board_art`;
  returned in gameplay-status; auto-generated for `mission_board_background`
  at the end of mission generation (best effort, async).

### B7. Routes added

```
GET  /api/v1/missions/{id}/gameplay-status
POST /api/v1/missions/{id}/reports
GET  /api/v1/missions/{id}/reports
POST /api/v1/missions/{id}/actions/preview
POST /api/v1/missions/{id}/art/board/generate
```

Guard rails kept: WorldBible never exposed; AI never mutates DB; all stage /
report / completion logic validated by backend; wallet flows untouched.

---

## Frontend design (web/src)

### F1. GameSceneShell (`features/game/GameSceneShell.tsx`)

Wraps all `/app/missions/:missionId/*` routes:
- full-screen board background (board art from gameplay-status, fallback:
  existing `theme_id` gradient from `game.css`), dark scrim, no page scroll
  chrome — HUD overlays instead of cards;
- polls/queries `gameplay-status` (react-query) and provides it via context;
- world-state theme classes already defined in `cinematic.css`/`game.css` are
  applied at the shell level (night darkening, rain overlay, danger pulse).

### F2. MissionHUD (always visible inside the shell)

- top bar: StageTracker (stage n/6 + title), mission clock, risk chip;
- left/bottom: ClueCounter ("2 / 5 clues"), suspect status chip
  (hidden → **SUSPECT IDENTIFIED** flash via SuspectRevealPanel), progress ring;
- right/bottom: next reward chip, glowing **Report CTA** (pulses when a report
  is pending for the current stage), floating action buttons (Map, Timeline, AI);
- ScrollableTimeline: bottom drawer available on every mission screen with
  current-time indicator (reuses `missiontimeline` data).

### F3. Flows

- **ActionTimePreview**: before location actions / chat / report submit, a
  bottom-sheet confirm shows `+20 min · low risk` from the preview endpoint.
- **After actions**: toast/popup shows time passed; `world_events` in the
  response raise WorldEventPopup; `stage_updates` raise RewardPopup
  ("STAGE COMPLETE — +50 XP") and unlock animations.
- **ReportTerminal** (`/app/missions/:id/report`): report-type picker, summary
  field, evidence linker (confirmed clues), submit → typed verdict panel
  (ACCEPTED / REJECTED with missing-evidence list), then state updates cascade.
- **DebriefScene**: MissionResultPage restyled as full-screen debrief board
  (uses `debrief_background`), stars, per-stage recap, rewards tally.

### F4. Files

- new `features/game/`: `GameSceneShell.tsx`, `MissionHUD.tsx`,
  `StageTracker.tsx`, `ClueCounter.tsx`, `SuspectRevealPanel.tsx`,
  `RewardPopup.tsx`, `ActionTimePreview.tsx`, `WorldEventPopup.tsx`,
  `ScrollableTimeline.tsx`;
- new `features/reports/ReportTerminalPage.tsx`;
- `shared/api/endpoints.ts`: `gameplayApi` (status, preview, reports, board art);
- `shared/types/api.ts`: Stage/Report/GameplayStatus types;
- router: mission routes nest under GameSceneShell; add `/report`;
- styles: extend `game.css` (HUD grid, glow, popups) — no dashboard grids.

---

## Delivery order

1. Migration 0006 (stages column, board_art column, mission_reports table).
2. `mission` stage model + default template + generator seeding + backfill.
3. Stage engine + wiring into progression/gamemap/character/clue/timeengine.
4. Time costs + due-event sweep + `time_update` in action responses.
5. Report package + routes.
6. Gameplay-status + action preview + board art endpoints.
7. OpenAPI + docs update.
8. Frontend: types + API, GameSceneShell + HUD, ReportTerminal, popups,
   timeline drawer, debrief restyle.
9. Tests: stage engine unit tests (template, transitions, rewards-once),
   report verdict tests, time-cost sweep tests; `go test ./...`, `go build`,
   web `tsc && vite build`.

## Acceptance mapping

Every item in `11_ACCEPTANCE_TESTS.md` maps to: stages visible from Stage 1
(gameplay-status), stage 1 completion unlocks stage 2 (engine), clue goal 2/5
(ClueCounter), suspect unlock at defined stage (RevealSuspect), glowing report
CTA (pending_report flag), accepted report updates state (report envelope),
wrong report explains missing evidence (deterministic verdicts), actions
consume time (B5), time triggers events (due-event sweep), board image
generated and used (B6 + shell), UI no longer dashboard (F1–F4).
