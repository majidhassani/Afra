# AgentVerse Frontend — Completion Report

Date: 2026-07-04
Location: `web/`

## Stack

- React 18 + TypeScript (strict), Vite 6
- react-router-dom 6 (v7 future flags enabled)
- TanStack Query 5 for server state
- Zustand 5 for auth session + toasts
- lucide-react icons
- Local typed i18n system (no runtime dependency), English + Persian with full RTL
- Plain CSS design-token system (`src/styles/tokens.css`), no CSS framework
- Vitest + Testing Library (jsdom)

## Routes implemented (all with loading / empty / error / success states)

```
/login  /register
/app/dashboard  /app/profile  /app/wallet  /app/history
/app/settings   /app/diagnostics
/app/missions   /app/missions/new  /app/missions/:missionId
/app/missions/:missionId/map
/app/missions/:missionId/locations/:locationId
/app/missions/:missionId/characters
/app/missions/:missionId/characters/:characterId
/app/missions/:missionId/clues
/app/missions/:missionId/clues/:clueId
/app/missions/:missionId/journal
/app/missions/:missionId/events
/app/missions/:missionId/time
```

`/app` redirects to the dashboard; unknown routes render a translated 404.

## API modules (`src/shared/api/endpoints.ts`)

auth, me, profile (+stats/history/badges), wallet (+transactions/pricing/
rewarded-ad/purchase-verify), missions (create/list/dashboard/archive/events),
map (view/location/actions/ask-ai), characters (list/detail/chat), clues
(list/detail/inspect/explain), guidance, time (get/advance), journal (CRUD),
health/ready.

Client behavior:

- `{"data"}/{"error"}` envelope parsing with a uniform `ApiError`
- Bearer token injection; single transparent refresh+retry on 401, then
  logout redirect
- Error→translation-key mapping (network, unauthorized, insufficient balance,
  mission generating, not found, validation, conflict, rate limit, server)
- Mission generation: `POST /missions` → 202 → dashboard polls every 2.5s
  while `generating`, plus SSE-driven invalidation
- SSE via fetch streaming (EventSource cannot send Authorization); 3
  reconnect attempts with exponential backoff, then polling fallback; status
  surfaced on the Events screen (Live / Polling / Disconnected)

## Bilingual support

- ~230 translation keys per language, en/fa parity enforced by a unit test
- Language switcher on auth screens, top bar, and Settings; persisted in
  localStorage; updates `document.documentElement.lang` and `dir`
- Vazirmatn for Persian, Inter for English; RTL-aware CSS (logical
  properties, `.rtl-flip` for directional icons, `unicode-bidi: plaintext`
  for mixed-direction AI content)

## Design system

Dark postmodern-minimal console: ink/graphite surfaces with warm off-white
text; muted cyan (AI), oxidized green (mission), amber (wallet), signal red
(danger), quiet violet (rare) accents. Full-width bands and panels — no
nested cards; radius ≤ 8px; restrained motion (marker pulse, subtle skeleton
shimmer) with `prefers-reduced-motion` respected. Fallback tactical map:
dark grid board with coordinate-normalized markers, lock/visited states,
clue/character flag dots, and a side panel (desktop) / stacked sheet (mobile).

## Mock mode

`VITE_ENABLE_MOCKS=true` routes every request to an in-memory adapter that
preserves envelope semantics, seeds a playable detective mission (bilingual
content), simulates 8-second mission generation with events, charges/credits
mock coins, and enforces insufficient-balance and daily-ad-limit errors.
Isolated to `src/shared/api/mock/`.

## Verification

- `npm run build` (tsc -b + vite build): **passes**, bundle ~398 KB / 118 KB gzip
- `npm test`: **11/11 tests pass** (i18n parity/placeholders, envelope +
  error mapping, mock adapter behavior incl. truth-layer privacy scan)
- Runtime smoke test (Vite dev server, mock mode): login → mission dashboard
  → map marker selection → clues, in both English LTR and Persian RTL
- Viewports checked: 1440×900 desktop (side nav) and ~524px mobile (bottom
  nav); no horizontal overflow, no console errors (router future-flag
  warnings resolved)
- Privacy: `internal_truth`, `private_state`, `hidden_state`, WorldBible are
  absent from the TypeScript types and asserted absent from mock payloads by
  test

## Known limitations

- No Google Maps key integration; the tactical fallback board is the map
  (marker contract is Maps-compatible, so a real map layer can slot in)
- Character/clue visuals are prompt placeholders by design — no image
  generation assets exist yet
- SSE payload shape assumed to be JSON mission events per frame; verify
  against the Go stream handler when it stabilizes
- Journal has manual save (no autosave), per the "clear manual save" option
- Backend-integration flows (refresh rotation, real pricing keys such as
  `mission_creation`) were exercised against the contract + mocks, not a
  live backend in this session

## Run commands

```bash
cd web
npm install
npm run dev     # http://localhost:5173 (VITE_API_BASE_URL for backend)
npm run build
npm test
```
