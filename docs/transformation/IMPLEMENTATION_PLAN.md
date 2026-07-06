# AgentVerse Dashboard-to-Game Transformation — Implementation Plan

Source of truth: `AgentVerse_Dashboard_To_Game_Transformation` roadmap folder (00_MASTER_AGENT, 01_ROADMAP, phase docs 02–11, AVDS design docs, API_CONTRACTS, ACCEPTANCE_CRITERIA) plus the existing audit in `docs/audit/AGENTVERSE_CURRENT_STATUS_REPORT.md`.

Rule: do not rewrite the project. Extend it incrementally, phase by phase, keeping auth, mission engine, wallet guard, LLM runtime, agent runtime, profile, i18n, Docker, and the mock API intact.

## 1. Files Inspected

Backend:
- `internal/platform/http/router.go` — full route table (missions, map, characters, clues, guidance, time, wallet, profile, legacy cases).
- `internal/missionevent/missionevent.go` — mission events repository + Recorder (SSE bus).
- `internal/character/entity.go` — `PublicCharacter` still exposes `avatar_prompt`, `thumbnail_prompt`.
- `internal/clue/entity.go` — `PublicClue` still exposes `avatar_or_thumbnail_prompt`.
- `internal/mission/dashboard.go`, `internal/missioncomplete/service.go`, `internal/gamemap/*` (sizes reviewed; details per phase).
- `migrations/0001_init.sql`, `migrations/0002_agentverse.sql`.
- `docs/openapi.yaml`.

Frontend:
- `web/src/app/router.tsx` — routes; Diagnostics is a normal route; no result route; no timeline route.
- `web/src/shared/types/api.ts` — mirrors prompt fields (`avatar_prompt`, `thumbnail_prompt`, `avatar_or_thumbnail_prompt`, `visual_prompt`).
- `web/src/shared/api/endpoints.ts` — full API client; no timeline/result endpoints.
- `web/src/shared/config/env.ts` — env flags (no maps key, no diagnostics flag).
- `web/package.json` — no `lint` script; no ESLint dependency; no Google Maps dependency.
- Feature folders: `missions`, `map`, `characters`, `clues`, `guidance`, `wallet`, `profile`, `settings`, `diagnostics`, `events`, `journal`, `time`.

## 2. Current Working Features (keep)

- Auth + JWT refresh; player profile with rank/XP/history/badges.
- Wallet with server-side guard (reserve/settle/release), pricing, transactions.
- Mission CRUD + async generation, dashboard contract (`/missions/{id}/dashboard`), events + SSE stream.
- Mission completion check/complete with scored `MissionResult`.
- Map data endpoint (`/missions/{id}/map`) with Google Maps-compatible markers, location detail/actions.
- Character chat, clue inspect/explain, AI guidance — all paid via wallet guard, all locale-aware (`withLocale`).
- Persian/English i18n with RTL; mock API adapter; mock LLM; Docker + migrations; basic PWA.

## 3. Current Gaps (from roadmap + audit)

1. No `GET /api/v1/missions/{id}/timeline` (player-facing curated timeline).
2. Prompt fields cross the public DTO boundary (character/clue/location).
3. No `avatar_url` / `image_url` in public DTOs; image generation not wired to characters/clues.
4. Web map is a fallback tactical board, not Google Maps.
5. Mission result is modal-only; no `/missions/:id/result` route; result not reviewable from history.
6. Hub/wallet/profile/history/settings still read as dashboard; Diagnostics not dev-gated.
7. No `npm run lint` script.
8. OpenAPI schema detail is thin.
9. AI language contract not enforced server-side (frontend heuristic only).
10. Purchase/ad verification placeholders not clearly demo-gated by environment.
11. Mobile polish incomplete (bottom sheets, safe areas, SW caching policy).

## 4. Phase-by-Phase Plan and Exact Files to Change

### Phase 0 — Stabilize contracts and privacy
- New `internal/missiontimeline/` (service + handler): curates `missionevent` rows into player-facing items (type, title, description, mission_time, location_id, related ids, importance). Route `GET /api/v1/missions/{missionID}/timeline` in `router.go`.
- `/events` stays as raw feed (documented as dev/detail feed); `/timeline` is the official player timeline.
- `internal/character/entity.go`: drop `avatar_prompt`/`thumbnail_prompt` from `PublicCharacter`; add `avatar_url`, `avatar_status`.
- `internal/clue/entity.go`: drop `avatar_or_thumbnail_prompt` from `PublicClue`; add `image_url`, `image_status`.
- `internal/location` / `internal/gamemap`: stop exposing `visual_prompt` publicly.
- `web/src/shared/types/api.ts`, mock data: mirror the cleaned DTOs.
- `docs/openapi.yaml`: add schemas for dashboard, map, timeline, completion-check, complete, profile, wallet, guidance, chat, clue explain, result.
- `web/package.json`: add ESLint (flat config) + `"lint"` script.
- Diagnostics dev-gate: `VITE_ENABLE_DIAGNOSTICS` flag in `env.ts`, route guard in `router.tsx`, nav item hidden in production (`AppShell`).

### Phase 1 — Timeline and next-action roadmap
- Normalize event types (mission_started, location_visited, clue_discovered, character_talked, ai_guidance_received, time_advanced, risk_changed, objective_completed/failed, new_location_unlocked, mission_ready_to_complete, mission_completed/failed) in the timeline curator; emit missing ones from mission/gamemap/character/clue/timeengine/missioncomplete services where absent.
- Frontend: `timelineApi` in `endpoints.ts`; upgrade `TimelineLog.tsx` (icons by type, importance colors, location/clue links, reveal animation, compact mobile layout); new `NextActionCard.tsx` (title, reason, priority chip, cost hint, deep link to map marker); dashboard shows timeline preview + next action; dedicated timeline panel/page.

### Phase 2 — Real Google Maps
- Add `VITE_GOOGLE_MAPS_API_KEY` to `env.ts`; loader util (no heavy SDK dependency; use official JS API via script loader).
- `web/src/features/map/GoogleMissionMap.tsx`: real map, custom marker rendering with states (recommended pulse, new_clue badge, character, required_action, completed, high_risk, locked), dark game map style.
- `MapPage.tsx`: use Google map when key present, existing tactical board as fallback; mobile bottom sheet (`LocationBottomSheet`) and desktop side panel on marker tap; highlight recommended marker from dashboard next action.

### Phase 3 — AVDS UI revolution
- `web/src/styles/`: AVDS tokens (semantic gameplay colors, glass panels, tactical borders, glow, motion durations from MOTION_AND_ANIMATION.md, reduced-motion support).
- Rebuild: `DashboardPage.tsx` (Mission Hub → game lobby: active mission card, mission type cards, rank, wallet chip, recent missions), `MissionDashboardPage.tsx` (command center HUD), `WalletPage.tsx` (game economy), `ProfilePage.tsx` (agent progression: rank/XP/badges), `HistoryPage.tsx` (mission archive cards), `SettingsPage.tsx` (premium minimal).
- Shared AVDS components in `web/src/shared/ui/` (GamePanel, HUDChip, GameButton, ProgressMeter, RiskMeter, BottomSheet…).

### Phase 4 — Avatar/clue image pipeline
- Migration `0003_visual_assets.sql`: `characters.avatar_url`, `characters.avatar_status`, `clues.image_url`, `clues.image_status`.
- Repos/entities updated; character/clue generation stores prompts internally only.
- Wire `internal/avatar` generation to mission characters (endpoint `POST /missions/{id}/characters/{characterID}/avatar` or auto-wire); add clue image generation endpoint reusing `internal/llm/openaicompat/imagegen.go`; graceful `pending/unavailable` status when provider missing.
- Frontend: `Avatar.tsx` uses `avatar_url`; `ClueCard`/`ClueDetailPage` use `image_url`; premium placeholders; no prompt text anywhere.

### Phase 5 — AI guidance and language hardening
- Backend `internal/llm/promptcontract` (or in `internal/agent`): every outbound prompt includes locale/language/direction/response contract block; unit tests asserting Persian/English instruction injection and no-spoiler system rules.
- Structured guidance output (summary, current_goal, recommended_actions, warnings, spoiler_level) kept/extended in `internal/guidance`.
- Language mismatch guard server-side (heuristic script check + retry/annotate) complementing `web/src/shared/i18n/languageGuard.ts`.
- Optional `cmd/llmsmoke` (or make target) live-provider smoke test.

### Phase 6 — Mission result page
- Persist result at completion (mission.result already stored — verify) and add `GET /api/v1/missions/{missionID}/result` returning last result (404/409 when absent).
- Frontend route `/app/missions/:missionId/result`: cinematic banner, stars reveal, XP/coin count-up, objectives, missed clues, decisions, CTAs; `MissionResultModal` links to it; `HistoryPage` links per mission.

### Phase 7 — Wallet production/demo separation
- Backend config flag (e.g. `WALLET_DEMO_PURCHASES`, default enabled outside production): `VerifyPurchase`/`ClaimRewardedAd` rejected in production unless explicitly enabled; responses marked `"demo": true`.
- Frontend wallet page: coin packs grid, daily reward, watch-ad button, AI cost explanation, recent transactions; test receipt form dev-only.

### Phase 8 — Mobile/PWA polish
- Bottom nav (Mission | Map | AI | Clues | Profile) in `AppShell`; wallet chip in HUD.
- Safe-area CSS, ≥44px touch targets, full-screen map, keyboard-safe chat (visualViewport), reduced motion.
- `web/public/sw.js`: never cache `/api/` responses; manifest/theme/icons review.

### Phase 9 — QA and release
- `go test ./...`, `cd web && npm run build && npm run lint && npm test`, `docker compose config` (+ build if environment allows).
- Smoke flow with mock mode; RTL QA; prompt visibility grep QA; update docs (`docs/openapi.yaml`, README notes).

## 5. Risk List

- Removing prompt fields from public DTOs may break mock data/tests → update mocks + types in same change.
- New timeline endpoint must not leak hidden truth → curate strictly from player-safe `mission_events` payloads.
- Google Maps without a key must never break the app → runtime detection with fallback board (existing behavior preserved).
- UI overhaul risks regressing i18n/RTL → reuse `useI18n` everywhere; add fa/en strings together.
- DB migration risk → additive columns only, no destructive changes.
- Wallet demo gating must not disable real AI charging → only `VerifyPurchase`/`ClaimRewardedAd` are gated.

## 6. Build/Test Commands

- Backend: `go build ./... && go test ./...`
- Frontend: `cd web && npm run build && npm test && npm run lint`
- Docker: `docker compose config` (build where network/time allows)

## 7. Rollback Notes

- All work is incremental commits on `develop`; each phase is separately revertable.
- Migration 0003 is additive (new nullable columns) — rollback = drop columns.
- Google Maps is feature-flagged by env key — unset key restores fallback board.
- Diagnostics/demo purchase gates are env flags — flipping flags restores previous behavior.
