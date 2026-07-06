# AgentVerse Current Status Report

## 1. Executive Summary

AgentVerse is a substantial working prototype with real backend modules, authenticated gameplay APIs, wallet charging, mission generation/completion, a React web app, mock frontend mode, Persian/English i18n, and a game-oriented UI layer. It is not production-ready yet: Google Maps is still a tactical fallback board in web, image/avatar generation is not fully wired into character/clue content, mission timeline has events and dashboard preview but no dedicated `/missions/{id}/timeline` API, and several production concerns remain around secrets, Docker build verification, observability, and end-to-end QA.

- Overall readiness percentage: 72%
- Backend readiness percentage: 82%
- Frontend readiness percentage: 76%
- Gameplay readiness percentage: 70%
- UI/game-feel readiness percentage: 68%
- Production readiness percentage: 55%

Evidence: backend router and module wiring in `internal/platform/http/router.go` and `cmd/api/main.go`; frontend routing in `web/src/app/router.tsx`; successful `go test ./...`, `web npm run build`, and `web npm test`; package scripts in `web/package.json`.

## 2. What Already Works

Feature: Auth/session
Status: implemented and tested
Evidence: endpoints in `internal/platform/http/router.go` (`POST /api/v1/auth/register`, `login`, `refresh`, `logout`, `GET /api/v1/me`); handlers in `internal/auth/handler.go`; tests in `internal/auth/service_test.go`.
Notes: frontend has login/register routes in `web/src/app/router.tsx` and auth store in `web/src/features/auth/authStore.ts`.

Feature: Player profile
Status: implemented and wired
Evidence: profile routes in `internal/platform/http/router.go` (`GET/PUT /api/v1/profile`, `stats`, `history`, `badges`); frontend API in `web/src/shared/api/endpoints.ts` (`profileApi`); UI in `web/src/features/profile/ProfilePage.tsx` and `HistoryPage.tsx`.
Notes: profile progression is updated by mission completion through `internal/playerprofile/repository.go` `ApplyMissionResult`.

Feature: Wallet and paid AI action guard
Status: implemented and wired
Evidence: wallet routes in `internal/platform/http/router.go`; wallet guard in `internal/wallet/guard.go`; wallet service/repository in `internal/wallet/service.go` and `internal/wallet/repository.go`; frontend wallet UI in `web/src/features/wallet/WalletPage.tsx`.
Notes: server-side reserve/settle/release flow exists. Insufficient balance maps to UI error via `web/src/shared/api/client.ts` `errorKey`.

Feature: Mission CRUD/lifecycle
Status: implemented and wired
Evidence: mission routes in `internal/platform/http/router.go`; handlers in `internal/mission/handler.go`; service in `internal/mission/service.go`; frontend API in `web/src/shared/api/endpoints.ts` `missionsApi`; mission screens in `web/src/features/missions/*`.
Notes: creation uses wallet reserve and async generator in `internal/mission/service.go` `Create`.

Feature: Mission dashboard
Status: implemented and wired
Evidence: backend `GET /api/v1/missions/{missionID}/dashboard` in `internal/platform/http/router.go`; DTO/service in `internal/mission/dashboard.go` `MissionDashboard` and `Dashboard`; frontend query in `web/src/features/missions/missionQueries.ts`; UI in `web/src/features/missions/MissionDashboardPage.tsx`.
Notes: dashboard includes objective, progress, risk, time, wallet balance, next action, win/loss, timeline preview, completion readiness.

Feature: Mission completion and result
Status: implemented and wired
Evidence: routes `GET /completion-check` and `POST /complete` in `internal/platform/http/router.go`; logic in `internal/missioncomplete/service.go` `Check` and `Complete`; result DTO in `internal/missioncomplete/service.go`; tests in `internal/missioncomplete/service_test.go`; modal in `web/src/features/missions/MissionResultModal.tsx`.
Notes: completion is gated and charged only when judged.

Feature: Map/location gameplay data
Status: implemented, web uses fallback tactical map
Evidence: backend map route `GET /api/v1/missions/{missionID}/map` in `internal/platform/http/router.go`; map service in `internal/gamemap/service.go`; map entities in `internal/gamemap/entity.go`; web map in `web/src/features/map/MapPage.tsx`.
Notes: data is Google Maps-compatible, but web does not use Google Maps SDK.

Feature: Characters and chat
Status: implemented and wired
Evidence: character routes in `internal/platform/http/router.go`; chat service in `internal/character/service.go` `Chat`; UI in `web/src/features/characters/CharactersPage.tsx` and `CharacterChatPage.tsx`; frontend API in `web/src/shared/api/endpoints.ts` `charactersApi`.
Notes: chat supports image attachments in frontend and LLM image payloads in API types.

Feature: Clues and clue analysis
Status: implemented and wired
Evidence: clue routes in `internal/platform/http/router.go`; handlers in `internal/clue/handler.go`; services in `internal/clue/service.go`; UI in `web/src/features/clues/CluesPage.tsx` and `ClueDetailPage.tsx`.
Notes: inspect/explain actions are paid and use locale metadata through frontend `withLocale`.

Feature: AI guidance
Status: implemented and wired
Evidence: `POST /api/v1/missions/{missionID}/guidance` in `internal/platform/http/router.go`; service in `internal/guidance/service.go` `Guide`; UI in `web/src/features/guidance/GuidancePanel.tsx`.
Notes: guidance is framed as no-spoiler and uses wallet guard.

Feature: LLM provider abstraction
Status: implemented
Evidence: OpenAI-compatible provider in `internal/llm/openaicompat/openaicompat.go`; mock provider in `internal/llm/mock/mock.go`; config validation in `internal/config/config.go`.
Notes: any non-mock provider label uses OpenAI-compatible chat completions. There is no provider-specific GLM SDK.

Feature: Persian/English i18n and RTL
Status: implemented and wired
Evidence: translations in `web/src/shared/i18n/en.ts` and `fa.ts`; document `lang`/`dir` set in `web/src/shared/i18n/index.tsx`; RTL CSS in `web/src/styles/base.css`, `tokens.css`, `app.css`, and `game.css`.
Notes: AI request locale headers are set in `web/src/shared/api/client.ts` via `localeHeaders`.

Feature: PWA basics
Status: implemented, basic only
Evidence: manifest and service worker links in `web/index.html`; assets in `web/public/manifest.webmanifest`, `web/public/sw.js`, `web/public/pwa-icon.svg`.
Notes: offline shell exists, but no deep offline gameplay strategy.

## 3. What Is Partially Implemented

Feature: Google Maps marker UX
Current state: backend returns Google Maps-compatible marker data; frontend renders a custom tactical board.
Missing: real Google Maps SDK integration, marker clustering/viewport behavior, mobile bottom-sheet over actual map.
Evidence: backend comments and DTOs in `internal/gamemap/entity.go`; web fallback comment in `web/src/features/map/MapPage.tsx`.
Risk: product promise says Google Maps, but current web UX is not actual Google Maps.

Feature: Mission timeline
Current state: mission events and dashboard timeline preview exist; web has `TimelineLog`; case timeline endpoints exist.
Missing: dedicated `GET /api/v1/missions/{missionID}/timeline` endpoint requested by product contract.
Evidence: mission events route in `internal/platform/http/router.go` (`/missions/{missionID}/events`); case timeline route only under `/api/v1/cases/{caseID}/timeline`; `TimelineLog` in `web/src/features/missions/TimelineLog.tsx`; no `/missions/{missionID}/timeline` route found.
Risk: API contract mismatch for clients expecting mission timeline endpoint.

Feature: Avatar/image pipeline
Current state: avatar generation endpoint exists and fallback avatars render; external image generator exists.
Missing: character `avatar_url` persistence/wiring; clue image URL support; generated images are not attached to generated characters/clues in normal mission flow.
Evidence: avatar endpoint in `internal/avatar/handler.go`; image generator in `internal/llm/openaicompat/imagegen.go`; frontend placeholder in `web/src/shared/ui/Avatar.tsx`; prompt fields still present in `web/src/shared/types/api.ts`.
Risk: visual content remains placeholder-heavy.

Feature: Mission result UX
Current state: result modal exists and is wired from mission dashboard completion.
Missing: standalone result route/history detail, replay/report page, more robust conflict handling for unready completion.
Evidence: `MissionResultModal` in `web/src/features/missions/MissionResultModal.tsx`; dashboard wiring in `web/src/features/missions/MissionDashboardPage.tsx`; backend completion in `internal/missioncomplete/service.go`.
Risk: completed mission result review depends on result payload shape and dashboard route.

Feature: OpenAPI/Swagger
Current state: `docs/openapi.yaml` exists and is served.
Missing: detailed schemas for new dashboard/result fields; route descriptions are high-level.
Evidence: Swagger route in `internal/platform/http/router.go`; docs in `docs/openapi.yaml`.
Risk: clients may not know exact response contracts.

Feature: Production deployment
Current state: Dockerfiles and compose exist; `docker compose config` succeeds.
Missing: docker compose build not verified in this audit; secrets are resolved into compose config from current environment; no CI evidence found.
Evidence: `docker-compose.yml`, `Dockerfile`, `web/Dockerfile`; command result for `docker compose config`.
Risk: build/release surprises and secret-handling risk.

## 4. What Is Mocked Only

Feature: Frontend mock API
Mock location: `web/src/shared/api/mock/mockClient.ts` and `web/src/shared/api/mock/mockData.ts`.
What is missing for real implementation: nothing for backend parity in core flows, but mock data is not evidence of production behavior.
Evidence: `VITE_ENABLE_MOCKS` in `web/src/shared/config/env.ts`; mock tests in `web/src/shared/api/mock/mockClient.test.ts`.

Feature: Mock LLM
Mock location: `internal/llm/mock/mock.go` and canned mission content in `internal/llm/mock/mission.go`.
What is missing for real implementation: real provider credentials and live model QA; provider-specific GLM behavior is not implemented.
Evidence: config `ProviderMock` in `internal/config/config.go`; mock provider `Name/Ping/Chat` in `internal/llm/mock/mock.go`.

Feature: Purchase verification
Mock location: `internal/wallet/service.go` `VerifyPurchase`.
What is missing for real implementation: real App Store / Google Play receipt validation.
Evidence: comment in `VerifyPurchase` says store-side receipt verification is a placeholder hook.

Feature: Rewarded ads
Mock location: `internal/wallet/service.go` `ClaimRewardedAd`.
What is missing for real implementation: ad network callback validation.
Evidence: comment in `ClaimRewardedAd` says real ad-network callback validation is a placeholder.

Feature: Web map rendering
Mock location: not mock data, but fallback UI in `web/src/features/map/MapPage.tsx`.
What is missing for real implementation: Google Maps SDK render layer.
Evidence: component comment says fallback tactical map board is swappable for real Maps.

## 5. What Is Missing

Backend:
- Dedicated `GET /api/v1/missions/{missionID}/timeline` endpoint: not found; evidence `internal/platform/http/router.go`.
- Persisted `avatar_url` / clue image URL fields: not found in public frontend DTOs or DB evidence; prompt fields exist in `internal/character/entity.go`, `internal/clue/entity.go`.
- Provider-specific GLM SDK: not found; evidence `internal/config/config.go` says non-mock providers use OpenAI-compatible client.
- Full OpenAPI schemas for dashboard/result: partial only in `docs/openapi.yaml`.

Frontend:
- Real Google Maps SDK screen: not found; fallback exists in `web/src/features/map/MapPage.tsx`.
- Dedicated mission result route/history report page: not found; modal exists.
- Dedicated mission timeline API page using `/missions/{id}/timeline`: not found; events page exists.
- Lint script: not found in `web/package.json`; `npm run lint` fails.

AI/LLM:
- Live provider integration test against configured GLM/Gemini gateway: not found.
- Strong runtime validation that AI responses match selected language beyond frontend heuristic guard: partial; evidence `web/src/shared/i18n/languageGuard.ts`.

Gameplay:
- Real map travel/route mechanics: not found.
- Rich win/loss result screen as standalone flow: partial via modal.
- Full tutorial/onboarding for mission rules: not found.

Design:
- Some utility/diagnostic/settings screens still read as app dashboard rather than game command center.
- Real generated character/clue imagery: not found.

Wallet/economy:
- Real ad network and store verification: placeholder.
- Product catalog/store UI beyond testing verification form: partial; evidence `web/src/features/wallet/WalletPage.tsx`.

Mobile/PWA:
- PWA installability basics exist; offline gameplay/cache strategy is minimal.
- No Playwright/mobile visual regression tests found.

Testing:
- No frontend lint script.
- No end-to-end browser tests found.
- No Docker build result in this audit.

## 6. Backend Audit

| Area | Status | Evidence | Notes |
|---|---|---|---|
| Auth | Ready | `internal/auth/handler.go`, `service.go`, `service_test.go`; router auth endpoints | Implemented with JWT refresh flow. |
| User/Profile | Ready | `GET /api/v1/me`; `internal/auth/context.go` | Basic user identity works. |
| Player Profile | Ready | `internal/playerprofile/*`; `/api/v1/profile*` routes | Stats/history/badges present. |
| Wallet | Ready | `internal/wallet/*`; `/api/v1/wallet*` routes | Paid action guard implemented; store/ad validation placeholders. |
| Mission CRUD | Ready | `internal/mission/handler.go`, `service.go`; `/api/v1/missions` routes | Async generation and archive/list/get exist. |
| Mission Dashboard | Ready | `internal/mission/dashboard.go`; `/dashboard` route | Strong structured dashboard. |
| Mission Completion | Ready | `internal/missioncomplete/service.go`, `service_test.go` | Gated result and rewards exist. |
| Mission Objectives | Ready | `internal/mission/objective.go`; dashboard objective fields | Structured objective types/status/progress. |
| Progress/Risk | Ready | `internal/mission/progress.go`, `progress_test.go` | Deterministic formulas tested. |
| Timeline | Partial | `internal/missionevent/missionevent.go`; `/missions/{id}/events`; `internal/timeline/*` for cases | Mission events exist; mission timeline endpoint missing. |
| Time Engine | Ready | `internal/timeengine/service.go`, `handler.go`; `/time` and `/time/advance` | Time advance is paid and can change state. |
| Map/Locations | Partial | `internal/gamemap/*`; `/missions/{id}/map` | Backend marker system ready; web is not real Google Maps. |
| Characters | Ready | `internal/character/*`; character routes | Public/private state separation exists. |
| Clues/Evidence | Ready/Partial | mission clues in `internal/clue/*`; legacy case evidence in `internal/evidence/*` | Mission clue flow ready; image rendering incomplete. |
| AI Guidance | Ready | `internal/guidance/service.go`, `handler.go` | No-spoiler guidance with wallet guard. |
| Character Chat | Ready | `internal/character/service.go` `Chat`; chat route | Paid LLM action; image attachments supported by API types. |
| Agent Runtime | Ready | `internal/agent/runtime/*`, `runtime_test.go`; many `internal/agent/*` packages | Multi-agent generation framework exists. |
| LLM Provider | Ready/Partial | `internal/llm/openaicompat/openaicompat.go`; tests | OpenAI-compatible only; provider-specific SDK absent. |
| GLM integration | Partial | config accepts any provider label in `internal/config/config.go` | No GLM-specific adapter found. |
| Mock LLM | Ready | `internal/llm/mock/*`, tests | Useful offline path. |
| WorldBible privacy | Ready/Verify | `internal/worldbible/worldbible.go`; `internal/privacy/privacy.go`, `privacy_test.go` | Privacy guard blocks forbidden keys; verify prompt fields separately. |
| Migrations | Ready | `migrations/0001_init.sql`, `0002_agentverse.sql` | AgentVerse schema exists. |
| Docker | Partial | `docker-compose.yml`, `Dockerfile`, `web/Dockerfile`; `docker compose config` passed | Build not run; config resolves secrets from environment. |
| Swagger | Partial | `docs/openapi.yaml`; served in router | Present but schema detail incomplete. |
| Tests | Partial/Good | `go test ./...` passed; many packages have no test files | Core tests pass; coverage uneven. |

## 7. Frontend Audit

| Area | Status | Evidence | Notes |
|---|---|---|---|
| Framework | Ready | `web/package.json` uses React, Vite, TypeScript | Build passes. |
| Routing | Ready | `web/src/app/router.tsx` | Auth and mission routes wired. |
| API Client | Ready | `web/src/shared/api/client.ts` | Envelope, bearer auth, refresh retry, locale headers. |
| Mock Adapter | Ready | `web/src/shared/api/mock/mockClient.ts`, tests | Core flows covered by mock tests. |
| Auth UI | Ready | `LoginPage.tsx`, `RegisterPage.tsx`, `guards.tsx` | Connected to auth API. |
| Mission Hub | Ready/Partial | `DashboardPage.tsx`, `MissionsListPage.tsx` | Functional, still can be more game-like. |
| Mission Dashboard | Ready | `MissionDashboardPage.tsx`, `hud.ts`, `TimelineLog.tsx` | Strongest game screen. |
| Map Screen | Partial | `MapPage.tsx` | Playable fallback; not Google Maps SDK. |
| Location Panel | Ready | `LocationPage.tsx`; selected marker panel in `MapPage.tsx` | Actions and guidance wired. |
| Character UI | Ready/Partial | `CharactersPage.tsx`, `Avatar.tsx` | Placeholder avatars; no real avatar_url data yet. |
| Character Chat | Ready | `CharacterChatPage.tsx` | Paid chat, attachments, transcript. |
| Clue UI | Ready/Partial | `CluesPage.tsx`, `ClueDetailPage.tsx` | Evidence-style placeholder; generated clue images absent. |
| AI Guidance UI | Ready | `GuidancePanel.tsx` | Locale-aware requests and cost display. |
| Wallet UI | Ready/Partial | `WalletPage.tsx` | Balance/pricing/transactions; purchase form is test-like. |
| Profile UI | Ready | `ProfilePage.tsx`, `HistoryPage.tsx` | Stats and history wired. |
| Timeline UI | Partial/Ready | `TimelineLog.tsx`, `EventsPage.tsx` | Uses events, not dedicated timeline endpoint. |
| Mission Result UI | Ready/Partial | `MissionResultModal.tsx` | Modal wired; no standalone result route. |
| i18n | Ready | `en.ts`, `fa.ts`, `index.tsx`, tests | 11 frontend tests pass including i18n. |
| Persian RTL | Ready | `index.tsx` sets `dir`; RTL CSS files | Good baseline. |
| Mobile responsiveness | Partial | CSS media queries in `web/src/styles/app.css`; bottom nav | Needs device QA screenshots. |
| PWA | Partial | `web/index.html`, `web/public/manifest.webmanifest`, `sw.js` | Basic install/offline shell only. |
| Game-like design system | Partial/Good | `web/src/styles/game.css`, `tokens.css`, `game.tsx` | Good direction; some screens still utilitarian. |
| Avatar rendering | Partial | `Avatar.tsx` | Placeholder and imageUrl support; no backend avatar_url feed. |
| Clue image rendering | Partial | `ClueDetailPage.tsx` evidence placeholder | No actual clue image URL. |

## 8. API Contract Audit

Endpoint: `/api/v1/auth/register`
Method: POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: `internal/platform/http/router.go`, `web/src/shared/api/endpoints.ts` `authApi.register`, `web/src/shared/api/mock/mockClient.ts`.
Notes: public route rate-limited.

Endpoint: `/api/v1/auth/login`
Method: POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `authApi.login`, mock adapter.
Notes: works with token store.

Endpoint: `/api/v1/me`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `authApi.me`, mock adapter.
Notes: protected.

Endpoint: `/api/v1/profile`
Method: GET/PUT
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router profile routes; `profileApi.get/update`; mock adapter.
Notes: profile update supports display name.

Endpoint: `/api/v1/profile/stats`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `profileApi.stats`, `ProfilePage.tsx`.
Notes: returns profile + coins spent/earned.

Endpoint: `/api/v1/profile/history`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `profileApi.history`, `HistoryPage.tsx`.
Notes: mission result detail is limited.

Endpoint: `/api/v1/wallet`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `walletApi.get`, `WalletPage.tsx`, mock adapter.
Notes: dashboard also receives wallet balance.

Endpoint: `/api/v1/wallet/transactions`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `walletApi.transactions`, `WalletPage.tsx`.
Notes: limit query supported.

Endpoint: `/api/v1/wallet/pricing`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `walletApi.pricing`, wallet and paid action UIs.
Notes: server-side source of prices.

Endpoint: `/api/v1/wallet/rewarded-ad/claim`
Method: POST
Implemented: partial
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `walletApi.claimRewardedAd`, `internal/wallet/service.go`.
Notes: backend comment says ad-network validation is placeholder.

Endpoint: `/api/v1/wallet/purchase/verify`
Method: POST
Implemented: partial
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `walletApi.verifyPurchase`, `internal/wallet/service.go`.
Notes: receipt verification is placeholder.

Endpoint: `/api/v1/missions`
Method: GET/POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `missionsApi.list/create`, `NewMissionPage.tsx`.
Notes: creation charges `mission_start`.

Endpoint: `/api/v1/missions/{id}`
Method: GET
Implemented: yes
Used by frontend: partial/no for main dashboard
Mocked in frontend: yes
Evidence: router `Missions.Get`; frontend dashboard now uses `/dashboard`.
Notes: legacy dashboard-like payload still exists.

Endpoint: `/api/v1/missions/{id}/dashboard`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `internal/mission/dashboard.go`, `missionsApi.dashboard`, `mockClient.ts`.
Notes: key gameplay contract.

Endpoint: `/api/v1/missions/{id}/map`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `gamemap.Handler.Map`, `mapApi.view`, `MapPage.tsx`.
Notes: Google Maps-compatible data, fallback render.

Endpoint: `/api/v1/missions/{id}/timeline`
Method: GET
Implemented: no
Used by frontend: no
Mocked in frontend: no
Evidence: not found in `internal/platform/http/router.go`; case timeline exists under `/api/v1/cases/{caseID}/timeline`.
Notes: should be built or documented as `/events`.

Endpoint: `/api/v1/missions/{id}/events`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `missionsApi.events`, `EventsPage.tsx`, `TimelineLog.tsx`.
Notes: currently fills mission timeline role.

Endpoint: `/api/v1/missions/{id}/guidance`
Method: POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `guidanceApi.ask`, `GuidancePanel.tsx`, `internal/guidance/service.go`.
Notes: paid AI action.

Endpoint: `/api/v1/missions/{id}/completion-check`
Method: GET
Implemented: yes
Used by frontend: partial
Mocked in frontend: yes
Evidence: router, `missionsApi.completionCheck`, mock adapter.
Notes: dashboard mostly uses `can_complete` from dashboard response; direct API exists.

Endpoint: `/api/v1/missions/{id}/complete`
Method: POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `missionsApi.complete`, `MissionDashboardPage.tsx`, `MissionResultModal.tsx`.
Notes: paid final judgment when ready.

Endpoint: `/api/v1/missions/{id}/locations/{locationID}`
Method: GET
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `mapApi.location`, `LocationPage.tsx`.
Notes: location detail with characters/clues.

Endpoint: `/api/v1/missions/{id}/locations/{locationID}/actions`
Method: POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `mapApi.runAction`, `LocationPage.tsx`.
Notes: paid location action.

Endpoint: `/api/v1/missions/{id}/characters/{characterID}/chat`
Method: POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `charactersApi.chat`, `CharacterChatPage.tsx`.
Notes: paid AI action.

Endpoint: `/api/v1/missions/{id}/clues/{clueID}/inspect`
Method: POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `cluesApi.inspect`, `ClueDetailPage.tsx`.
Notes: paid.

Endpoint: `/api/v1/missions/{id}/clues/{clueID}/explain`
Method: POST
Implemented: yes
Used by frontend: yes
Mocked in frontend: yes
Evidence: router, `cluesApi.explain`, `ClueDetailPage.tsx`.
Notes: paid.

Endpoint: `/api/v1/avatars/generate`
Method: POST
Implemented: yes
Used by frontend: partial/not in mission character flow
Mocked in frontend: no dedicated mock path found
Evidence: router, `internal/avatar/handler.go`, `avatarApi.generate`.
Notes: endpoint exists but not wired to generated character cards.

## 9. Gameplay Readiness Audit

| Question | Status | Evidence | Notes |
|---|---|---|---|
| Does the player know the goal? | Ready | `MissionDashboardPage.tsx` shows primary objective and briefing | Good. |
| Does the player know where to go next? | Ready/Partial | dashboard uses `next_recommended_actions`; map markers have `recommended` | Good but needs richer roadmap. |
| Is progress visible? | Ready | `MissionDashboardPage.tsx`, `internal/mission/progress.go` | Deterministic and visible. |
| Is risk visible? | Ready | dashboard `risk_score`; `RiskMeter`; map marker risk | Good. |
| Is timeline visible? | Partial | `TimelineLog.tsx`, `/events` route | Visible, but no `/missions/{id}/timeline` endpoint. |
| Is win/loss clear? | Partial/Ready | win/failure conditions and result modal | Result exists, but no full result page. |
| Are objectives structured? | Ready | `internal/mission/objective.go`; frontend `Objective` type | Good. |
| Is AI guidance useful? | Partial/Ready | `internal/guidance/service.go`, `GuidancePanel.tsx` | Needs live model QA. |
| Is the map playable? | Partial | `MapPage.tsx`, `LocationPage.tsx` | Playable board, not Google Maps. |
| Are clues meaningful? | Ready/Partial | clue agent/service/UI | Textual clues strong; image pipeline incomplete. |
| Are characters interactive? | Ready | character chat route and UI | Good. |
| Is wallet integrated into gameplay? | Ready | wallet guard, cost badges, dashboard wallet balance | Good; monetization validation incomplete. |

## 10. Design and UX Audit

Overall UI feel:
- Dashboard: partial; some hub/profile/wallet/settings screens still have dashboard patterns.
- Game: partial/good; mission dashboard, map board, cards, result modal, HUD are game-like.
- Command center: good on mission dashboard; partial elsewhere.
- Mobile app: partial; responsive CSS exists, but no screenshot/device QA evidence.
- PWA: partial; install/offline shell exists.
- International product: partial/good; Persian/English and RTL exist.

Problematic screen: Wallet
Problem: still partly a testing/admin-like monetization page, especially purchase verification.
Evidence: `web/src/features/wallet/WalletPage.tsx`; backend `VerifyPurchase` placeholder in `internal/wallet/service.go`.
Recommended fix: convert to player economy screen with coin packs, reward state, and hide raw receipt testing in production.

Problematic screen: Diagnostics
Problem: intentionally diagnostic/admin-like.
Evidence: route `diagnostics` in `web/src/app/router.tsx`; component `web/src/features/diagnostics/DiagnosticsPage.tsx`.
Recommended fix: keep behind dev flag or remove from production nav.

Problematic screen: Map
Problem: gameplay board is useful but not actual Google Maps; side panel on desktop, not full mobile map bottom-sheet UX.
Evidence: `web/src/features/map/MapPage.tsx` comment says fallback tactical map board.
Recommended fix: integrate Google Maps SDK and convert selected marker panel into bottom sheet on mobile.

Problematic screen: Profile/History
Problem: functional stats/history, less game-like progression presentation.
Evidence: `web/src/features/profile/ProfilePage.tsx`, `HistoryPage.tsx`.
Recommended fix: add rank progression, badges, mission result cards, and stronger visual hierarchy.

## 11. Language and Localization Audit

- English support: Ready. Evidence: `web/src/shared/i18n/en.ts`.
- Persian support: Ready. Evidence: `web/src/shared/i18n/fa.ts`.
- RTL layout: Ready. Evidence: document direction in `web/src/shared/i18n/index.tsx`; RTL CSS selectors in `web/src/styles/base.css`, `tokens.css`, `app.css`, `game.css`.
- AI request locale metadata: Ready. Evidence: `web/src/shared/api/client.ts` adds `localeHeaders`; endpoint payload helpers use `withLocale` in `web/src/shared/api/endpoints.ts`.
- AI response language consistency: Partial. Evidence: frontend checks in `web/src/shared/i18n/languageGuard.ts`; backend passes language in handlers such as `internal/guidance/handler.go` and `internal/character/handler.go`. Risk: no comprehensive live LLM tests.
- Hardcoded strings: Partial. Evidence: most UI uses `useI18n`; some operational strings remain in mock data and hardcoded completion request text in `MissionDashboardPage.tsx`. These are not prominent UI labels, but should be reviewed.

## 12. Avatar and Image Pipeline Audit

- avatar_url support: Partial. Evidence: `web/src/shared/ui/Avatar.tsx` accepts `imageUrl`; no `avatar_url` field found in `PublicCharacter` type in `web/src/shared/types/api.ts`.
- avatar_prompt exposure: Backend/public DTO exposes prompt fields. Evidence: `internal/character/entity.go` `PublicCharacter` includes `avatar_prompt` and `thumbnail_prompt`; frontend type mirrors them in `web/src/shared/types/api.ts`. UI mostly does not render them; `Avatar.tsx` explicitly avoids prompts.
- character placeholder UI: Ready. Evidence: `web/src/shared/ui/Avatar.tsx`.
- avatar generation endpoint: Ready/Partial. Evidence: `GET /api/v1/avatars/options`, `POST /api/v1/avatars/generate` in router; handler in `internal/avatar/handler.go`; frontend `avatarApi` exists. Not wired to mission characters.
- clue image support: Partial/missing. Evidence: clue prompt fields exist in `internal/clue/entity.go` and `web/src/shared/types/api.ts`; UI uses `evidence-figure` placeholder in `ClueDetailPage.tsx`.
- clue visual prompt exposure: Mostly hidden in UI, but prompt fields are in frontend DTO and mock data. Evidence: `web/src/shared/types/api.ts`; prompt key filter in `ClueDetailPage.tsx`.
- image fallback UI: Ready. Evidence: `Avatar.tsx` and clue evidence placeholder in `ClueDetailPage.tsx`.

Conclusion: raw prompt text is not intentionally rendered in current main UI, but prompt fields still cross the API/frontend boundary. For stricter privacy/product quality, remove prompt fields from public DTOs or replace with generated image URLs/placeholders.

## 13. Wallet and Monetization Audit

- wallet balance: Ready. Evidence: `internal/wallet/service.go` `Get/Balance`; UI `WalletPage.tsx`; mission dashboard `wallet_balance`.
- transactions: Ready. Evidence: `/api/v1/wallet/transactions`, `walletApi.transactions`.
- pricing rules: Ready. Evidence: `/api/v1/wallet/pricing`, `wallet.Service.PriceOf`.
- AI usage charging: Ready. Evidence: `wallet.Guard Reserve/Settle/Release`; usage log in `internal/wallet/guard.go`.
- rewarded ads: Partial/mock-like. Evidence: `ClaimRewardedAd` comment in `internal/wallet/service.go`.
- purchase verification: Partial/mock-like. Evidence: `VerifyPurchase` comment in `internal/wallet/service.go`.
- insufficient balance handling: Ready. Evidence: `wallet.Repository.Reserve` conflict; frontend `errorKey` maps insufficient balance.
- UI integration: Ready/Partial. Evidence: wallet page, cost badges, dashboard balance; purchase UI still testing-oriented.

## 14. Test and Build Audit

| Command | Result | Failure reason | Relevant logs |
|---|---|---|---|
| `go test ./...` with `GOCACHE=/private/tmp/afra-go-cache` | Passed | none | All listed Go test packages passed; many packages report `[no test files]`. |
| `docker compose config` | Passed | none | Compose resolved services for api, web, postgres, redis. It also resolved environment secrets from the current shell, which is a security risk; secrets are intentionally not copied here. |
| `docker compose build` | Not run | Could pull/build images and require network/time beyond audit; no user approval requested because config already validated structure | Evidence inspected: `docker-compose.yml`, `Dockerfile`, `web/Dockerfile`. |
| `cd web && npm install` | Not run | Avoided mutating `package-lock.json`/`node_modules` during audit; existing install is sufficient because build/test ran | Evidence: `web/package-lock.json`; build/test passed. |
| `cd web && npm run build` | Passed | none | `tsc -b && vite build`; Vite built production assets successfully. |
| `cd web && npm run lint` | Failed | Missing script | `web/package.json` has `dev`, `build`, `preview`, `test`, `test:watch`; npm reported `Missing script: "lint"`. |
| `cd web && npm test` | Passed | none | Vitest: 3 files, 11 tests passed. |

## 15. Technical Debt and Risks

Architecture risks:
- Mixed legacy case routes and AgentVerse mission routes may confuse future API clients. Evidence: both `/api/v1/cases/*` and `/api/v1/missions/*` in `internal/platform/http/router.go`.
- Public prompt fields are still present in DTOs. Evidence: `internal/character/entity.go`, `internal/clue/entity.go`, `web/src/shared/types/api.ts`.

UI risks:
- Real Google Maps not implemented in web. Evidence: fallback comment in `MapPage.tsx`.
- Several screens remain utilitarian. Evidence: Wallet/Profile/Diagnostics components.

Gameplay risks:
- Mission timeline endpoint mismatch. Evidence: no `/missions/{id}/timeline` route.
- Completion UX depends on modal, not durable result page. Evidence: `MissionResultModal.tsx` only.

AI risks:
- Live provider QA not evidenced. Evidence: tests are mock/OpenAI-compatible unit tests, not live GLM/Gemini integration.
- Language consistency is guarded but not fully guaranteed. Evidence: `languageGuard.ts`.

Wallet/security risks:
- Store/ad verification placeholder. Evidence: comments in `internal/wallet/service.go`.
- `docker compose config` resolves secrets into output from environment; audit did not reproduce values in this report.

Production risks:
- No lint script.
- No E2E/mobile visual tests.
- Docker build not verified in this audit.
- No CI pipeline evidence found in inspected files.

## 16. Recommended Roadmap

### Phase 0 — Stabilize

Goal: remove contract/security ambiguity before new feature work.
Tasks: add lint script; expand OpenAPI schemas; decide whether `/missions/{id}/timeline` is new endpoint or `/events` is official; remove prompt fields from public API responses or guarantee they are never rendered.
Files likely affected: `web/package.json`, `docs/openapi.yaml`, `internal/platform/http/router.go`, `internal/mission/handler.go`, `internal/character/entity.go`, `internal/clue/entity.go`, `web/src/shared/types/api.ts`.
Acceptance criteria: lint passes; OpenAPI documents dashboard/result; no prompt fields in public frontend DTOs unless explicitly justified.

### Phase 1 — Gameplay Clarity

Goal: make every active mission self-explanatory.
Tasks: formalize mission timeline endpoint; improve next-action roadmap; add explicit readiness reasons and final result route.
Files likely affected: `internal/mission/dashboard.go`, `internal/missionevent/*`, `web/src/features/missions/*`, `web/src/shared/api/endpoints.ts`.
Acceptance criteria: player always sees objective, progress, risk, next action, timeline, and finish requirements.

### Phase 2 — UI/Game Design Overhaul

Goal: make all main screens feel like one premium command-center game.
Tasks: redesign wallet/profile/history/settings states; hide diagnostics in production; improve mobile navigation and dense HUD layout.
Files likely affected: `web/src/features/wallet/WalletPage.tsx`, `ProfilePage.tsx`, `HistoryPage.tsx`, `SettingsPage.tsx`, `DiagnosticsPage.tsx`, `web/src/styles/*`.
Acceptance criteria: no player-facing screen feels like an admin console.

### Phase 3 — AI Guidance and Language Consistency

Goal: make AI guidance reliable, localized, and no-spoiler.
Tasks: add backend language assertions/tests where feasible; add structured guidance schema validation; add live provider smoke test command.
Files likely affected: `internal/guidance/*`, `internal/character/*`, `internal/agent/*`, `web/src/shared/i18n/languageGuard.ts`.
Acceptance criteria: Persian UI produces Persian AI output in smoke tests; guidance never leaks hidden truth.

### Phase 4 — Map and Mission Flow

Goal: deliver the promised map-based gameplay.
Tasks: integrate Google Maps SDK in web; implement marker badges; mobile bottom sheet; route from next action to marker/location.
Files likely affected: `web/src/features/map/MapPage.tsx`, `LocationPage.tsx`, `web/src/styles/*`, `web/package.json`.
Acceptance criteria: markers render on real map; tapping marker opens bottom sheet; mobile flow is touch-friendly.

### Phase 5 — Avatar / Clue Visual Pipeline

Goal: replace prompt placeholders with images or polished safe placeholders.
Tasks: add `avatar_url`/`image_url` fields; persist generated images; wire avatar generation to character creation; add clue image generation/storage.
Files likely affected: migrations, `internal/avatar/*`, `internal/character/*`, `internal/clue/*`, `internal/platform/storage/*`, `web/src/shared/ui/Avatar.tsx`, clue UI.
Acceptance criteria: no raw prompts shown or shipped to UI; characters/clues display real image URLs or premium placeholders.

### Phase 6 — Wallet and Monetization

Goal: make economy production-grade.
Tasks: implement real store receipt validation; real rewarded ad callbacks; product catalog UI; transaction audit views.
Files likely affected: `internal/wallet/*`, `web/src/features/wallet/WalletPage.tsx`, `docs/openapi.yaml`.
Acceptance criteria: paid actions remain server-priced; fake receipt/ad paths are disabled in production.

### Phase 7 — Mobile/PWA Polish

Goal: make it installable and reliable on phones.
Tasks: improve service worker strategy; add safe-area QA; Playwright mobile screenshots; app icons at required sizes.
Files likely affected: `web/public/*`, `web/index.html`, `web/src/styles/*`, test config.
Acceptance criteria: install prompt works; offline shell is stable; mobile screenshots pass.

### Phase 8 — QA and Release Readiness

Goal: establish release confidence.
Tasks: docker compose build in CI; E2E smoke flows; API contract tests; secret scanning; observability.
Files likely affected: CI config (not found), Dockerfiles, tests, docs.
Acceptance criteria: repeatable build/test pipeline; no secrets in logs; release checklist passes.

## 17. Keep / Refactor / Remove / Build Matrix

| Item | Decision | Reason | Priority |
|---|---|---|---|
| Auth/JWT flow | KEEP | Implemented, tested, wired | High |
| Wallet Guard | KEEP | Correct server-side paid action boundary | High |
| Mission dashboard contract | KEEP | Strong gameplay clarity foundation | High |
| `/missions/{id}/events` as timeline source | VERIFY | Works, but contract asks `/timeline` | High |
| Public prompt fields | REFACTOR | Product says raw prompts should not reach users | High |
| Google Maps fallback board | REFACTOR | Useful now, but not promised Google Maps UX | High |
| MissionResultModal | KEEP | Good result reveal, wired | Medium |
| Standalone result page | BUILD | Needed for history/review | Medium |
| Mock API adapter | KEEP | Good offline dev/test path | Medium |
| Mock LLM | KEEP | Essential for tests/dev | Medium |
| Purchase verification placeholder | REFACTOR | Not production monetization | High |
| Rewarded ad placeholder | REFACTOR | Not production monetization | High |
| Diagnostics page in production nav | REMOVE/DEFER | Useful dev tool, not player-facing | Medium |
| Lint script | BUILD | Missing quality gate | High |
| OpenAPI schemas | REFACTOR | Current docs are too thin for clients | Medium |
| Docker build verification | VERIFY | Config passes; build not audited | High |
| PWA service worker | VERIFY | Basic shell exists, needs QA | Medium |
| Avatar generation endpoint | KEEP | Good base; needs character wiring | Medium |
| Clue image pipeline | BUILD | Prompt fields exist but image delivery missing | Medium |

## 18. Final Recommendation

Do first: stabilize contracts and privacy boundaries. Add/decide the mission timeline endpoint, expand OpenAPI schemas, add a lint script, and stop exposing prompt fields to public frontend types before building more features.

Do not touch yet: do not rewrite the mission engine, auth, wallet guard, agent runtime, or dashboard foundation. These are valuable and working.

Biggest blocker: the gap between promised premium map/visual gameplay and current fallback visuals: no real Google Maps SDK on web, no persisted character/clue image URLs, and prompt fields still crossing the public boundary.

Fastest path to a playable global-quality demo: keep the current backend and mission dashboard, add real Google Maps rendering with mobile bottom sheet, remove prompt exposure, wire generated/persisted avatar and clue image URLs, polish wallet/profile/history into game screens, and add one E2E mobile smoke test for create mission → map → clue/chat → guidance → complete.
