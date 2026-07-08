# AgentVerse — Living World & Avatar/Image Fix Plan

Brief: the world feels static, reports are unclear, timeline is under-used, time
is abstract, the UI doesn't react to world changes, and avatar/clue images look
pixelated/stale/broken. Make the world **alive** and the images **crisp**.

Rule: do not rewrite. Extend the working engine. AI may *suggest* public-safe
events but **backend services apply state**; never expose the WorldBible.
Every meaningful update returns the envelope: `state_changes, timeline_events,
objective_updates, unlocked_locations, risk_update, time_update, theme_update,
next_recommended_actions, message` (extends the progression envelope already
shipped in the gameplay-state fix).

## 1. Current State (already shipped, reused here)

- **Progression envelope + engine** (`internal/progression`): confirm evidence
  → map unlock; hypothesis checkpoint. Reused as the base for reports.
- **Curated timeline** (`internal/missiontimeline` + `TimelineLog`): event
  types, importance, deep links, reveal animation. Base for the scrollable
  timeline everywhere.
- **6-biome theme system** (`shared/theme/environment.ts` + `cinematic.css`):
  `data-env` tints the whole shell. Base for dynamic world sub-themes.
- **Time engine** (`internal/timeengine`): advance-time emits events.
- **Image pipeline** (`internal/visualasset`): avatar_url/image_url + status.
- **Standalone result page** + modal. Base for debrief/replay.

## 2. Problems → Fixes

| Problem | Fix |
|---|---|
| World static | Deterministic **world state** (weather/time-of-day/visibility/phase/risk) derived each read; time-advance + reports mutate it |
| Reports unclear / no place | **Mission Report Center** `/missions/:id/report` + `POST /reports` with verdicts and state diff (generalizes the hypothesis endpoint) |
| Reports don't update state | Report envelope drives timeline/objective/unlock/risk/theme updates |
| Timeline weak | `ScrollableMissionTimeline` reused on dashboard/map/report/debrief; current-time indicator, deadlines, tap detail |
| Time abstract | **Time-based events**: advance-time triggers scheduled/conditional events (witness leaves, clues fade, risk rises, theme flips) |
| UI doesn't react | **Dynamic UI themes**: biome × {weather, time-of-day, danger} → rain overlay, night darken, danger red pulse |
| No debrief/replay | Debrief extends the result page with a timeline replay |
| Pixelated/stale images | **Versioned image URLs** (`?ver=`), width/height, no `image-rendering:pixelated`, premium placeholders, retry on failed, 1024² minimum target |

## 3. World State Model (`internal/mission/worldstate.go`)

Deterministic `DeriveWorldState(publicState, clock, risk, type) → WorldState`:
- `weather`: keyword-mapped from public_state (`clear|rain|snow|storm|fog`)
- `time_of_day`: parsed from the mission clock hour (`day|dusk|night`)
- `visibility`: from weather+time (`high|medium|low`)
- `risk_score`, `urgency` (from deadline proximity), `world_phase`
  (`opening|investigation|closing`), `active_events`
- `theme_id`: compact `"<biome>_<modifier>"` (e.g. `forest_rain`, `city_night`,
  `*_danger`) — the client composes overlays from the flags.

Exposed on the dashboard as `world_state`, and in the time-advance response as
`theme_update`. Pure function → unit-tested.

## 4. Endpoints

- `GET /missions/{id}/dashboard` → add `world_state`.
- `POST /missions/{id}/reports` — report submission (progress/evidence/
  hypothesis/objective/incident/final). Verdicts: `accepted |
  partially_accepted | unsupported | wrong | too_early | duplicate`. Returns the
  full envelope. (Phase 2; generalizes `/hypothesis`.)
- `POST /missions/{id}/time/advance` → enrich response with `theme_update` +
  `triggered_events`. (Phase 2.)

## 5. Dynamic UI Themes (frontend)

`AppShell` reads `world_state` from the cached dashboard and sets
`data-env` (biome) + `data-weather`, `data-tod`, `data-danger` on `<html>`.
`cinematic.css` adds:
- rain: animated diagonal streaks overlay
- night: darken + cool shift on the backdrop
- danger: red HUD pulse ring on the shell
Respects `prefers-reduced-motion`.

## 6. Avatar / Image Quality

- DTO adds `avatar_version/avatar_width/avatar_height` and
  `image_version/image_width/image_height` (additive).
- Frontend: append `?ver=` for cache-busting when a version is present; set
  intrinsic width/height; never use `image-rendering: pixelated`; crisp scaling;
  premium placeholder for `missing`; retry button for `failed`.
- Generation target 1024² (visualasset), thumbnails deferred to storage phase.

## 7. Tests

- worldstate: rain keyword → weather=rain; 21:00 → night; risk≥60 → danger;
  theme_id composition.
- report verdicts (when Phase 2 lands): too_early/unsupported/accepted.
- image DTO carries version; frontend cache-bust appends ?ver.
- `go test ./...`, web build/lint/test.

## 8. Delivery Order (each a tested, revertable commit)

1. **World state + dynamic UI themes** (this iteration): deterministic
   `world_state` on the dashboard + rain/night/danger overlays.
2. Time-based events + `theme_update` in advance-time.
3. Mission Report Center (`/reports`) generalizing hypothesis.
4. Scrollable timeline everywhere + journal pending-reports.
5. Debrief timeline replay.
6. AI Game Director pacing.
7. Image versioning/quality/cache-busting.

Honest scope note: this is a multi-iteration program. Each numbered item ships
independently with tests so nothing half-lands.
