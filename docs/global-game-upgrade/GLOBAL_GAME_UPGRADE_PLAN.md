# AgentVerse Global Game Upgrade Plan

## Current Architecture

- Backend: Go HTTP API in the repository root, with modular packages under `internal/*`.
- Frontend: Vite + React + TypeScript in `web`, using React Query, React Router, Zustand auth state, and a mock API adapter.
- Persistence and infra: PostgreSQL migrations, Redis-backed rate limiting, Dockerfiles for API and web, and docker-compose orchestration.
- API style: `/api/v1` JSON endpoints with authenticated mission, map, character, clue, guidance, wallet, and profile flows.

## Backend Modules

- Auth/session: `internal/auth`
- Mission lifecycle and dashboard: `internal/mission`, `internal/missioncomplete`
- Game world: `internal/gamemap`, `internal/location`, `internal/timeengine`, `internal/timeline`
- Investigation content: `internal/character`, `internal/clue`, `internal/evidence`, `internal/suspect`
- AI runtime: `internal/agent/*`, `internal/llm/*`, `internal/guidance`
- Economy/profile: `internal/wallet`, `internal/playerprofile`
- Private truth boundaries: `internal/worldbible`, `internal/casebible`

## Current Frontend Framework

- React 18 and React Router for app screens.
- React Query for API state and polling.
- CSS token system in `web/src/styles` with a dark, game-oriented layer already started.
- i18n support for English/Persian with RTL-aware document direction.
- Mock API for offline development.

## Design Problems

- Some screens still read as a functional dashboard instead of a mission command center.
- Mission dashboard needed stronger completion/readiness, wallet, next-action, and timeline visibility.
- PWA installability assets were missing.
- The frontend dashboard API used the legacy mission detail route instead of the new dashboard contract.

## Gameplay Problems

- The player could see briefing/objectives, but readiness and failure conditions were not visible enough.
- Completion flow existed server-side but was not surfaced as a clear CTA.
- Wallet balance was available elsewhere, but not always visible inside active mission context.
- Timeline preview existed as events, but not as part of the dashboard contract.

## Migration Plan

1. Keep all existing auth, wallet, mission, LLM, and Docker flows intact.
2. Align `GET /api/v1/missions/{missionID}/dashboard` with the AgentVerse contract while keeping old safe payload fields.
3. Update the web API client and mock adapter to use the contract route.
4. Upgrade the mission dashboard into the primary command center: objective, progress, risk, time, wallet, next action, timeline, win/loss, and final CTA.
5. Add PWA basics: manifest, icon, theme color, and offline shell service worker.
6. Continue polishing map, character, clue, wallet, and profile screens without exposing raw prompts or private truth.

## Implementation Order

1. Backend dashboard contract and wallet balance.
2. Frontend types, endpoints, and mock parity.
3. Mission dashboard HUD/readiness/result UI.
4. PWA metadata and offline shell.
5. Cross-platform visual QA and responsive tuning.
6. Build/test verification.

## Risk Areas

- Dashboard response shape drift between old `/missions/{id}` and new `/missions/{id}/dashboard`.
- Final completion can run paid judging, so UI must only enable it when backend says ready.
- Service worker caching must not cache authenticated API responses.
- Persian strings and RTL layout must remain complete as new UI labels are added.
- WorldBible, hidden state, prompts, and private truth must remain absent from frontend DTOs.

## Acceptance Checklist

- Mission hub and active mission feel game-like and mobile-first.
- Active mission shows objective, progress, risk, time, wallet, next action, timeline, and completion readiness.
- Map uses simple markers plus a detail side/bottom panel.
- Guidance requests include locale and do not reveal hidden truth.
- Paid AI actions keep using server-side wallet/cost guards.
- Persian/English labels are localized.
- PWA manifest and offline shell exist.
- Build and tests pass.
