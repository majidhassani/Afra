# AgentVerse Frontend Upgrade Plan

Turn the `web` app from a clean-but-admin-looking console into a cinematic,
map-based AI mission game — **incrementally**, keeping the existing architecture.

## 1. Inspection findings

| Area | Current state |
| --- | --- |
| **Framework** | React 18 + Vite 6 + TypeScript, `react-router-dom` v6, `@tanstack/react-query` v5, `zustand`, `lucide-react`. No Next/Vue/Svelte. |
| **Styling** | Plain CSS with design tokens (`src/styles/tokens.css` → `base.css` → `app.css`). No Tailwind/shadcn/MUI. Already dark-themed but reads as a minimal admin console. |
| **API client** | `src/shared/api/client.ts` (`apiRequest`) + typed endpoint modules in `endpoints.ts`. `{data}/{error}` envelope, bearer auth, refresh+retry. Mock adapter behind `VITE_ENABLE_MOCKS`. |
| **i18n** | `src/shared/i18n` — `I18nProvider` with `lang`, `dir`, `t`; `en.ts`/`fa.ts` dictionaries; sets `document.dir`. Solid foundation, RTL already wired. |
| **Mission pages** | `features/missions` (Dashboard/Hub, MissionDashboard, New, List), `features/map`, `features/characters`, `features/clues`, `features/wallet`, `features/profile`, `features/guidance`. |
| **Avatar/clue** | `AvatarPlaceholder` in `badges.tsx` renders initials but **leaks the raw prompt via `title`**. `ClueDetailPage` and `LocationPage` render `visual_description`/`avatar_or_thumbnail_prompt`/`visual_prompt` as **user-facing text**. |

### Problems confirmed in code
1. **Admin feel** — sidebar + topbar + flat panels, tables, stat blocks.
2. **Language** — every AI service on the backend hardcodes `Language: "en"` (guidance, clue, character chat, gamemap action, time). The client never sends language on AI calls. So a Persian UI still gets English AI text.
3. **Prompt leakage** — avatar/clue/location prompts shown to users (see above).
4/5. **Immersion** — no HUD, risk meter, objective progress, mission tiles, cinematic loading/empty/error states.

## 2. Implementation plan (incremental)

### A. Language consistency (correctness)
- **Client**: attach `Accept-Language`, `X-App-Language`, `X-UI-Direction` headers to *every* request; attach `locale`/`language`/`ui_direction`/`response_language`/`response_contract` to *AI* request bodies via a `withLocale()` helper. Add a `useLocale()` hook exposing `{ locale, language, direction, contract }`.
- **Backend (minimal, surgical)**: add `httpx.RequestLanguage(r)` (reads `X-App-Language` → `Accept-Language`, defaults `en`) and thread it into the AI service methods that hardcoded `"en"` (guidance, location ask, character chat, clue inspect/explain, location action, time advance).
- **Dev guard**: `useLanguageGuard()` detects an AI response whose script doesn't match the selected language; shows a **non-blocking dev-only** warning toast. Never blocks production UI.

### B. Avatar + prompt hiding (privacy)
- New `Avatar` component (`shared/ui/Avatar.tsx`): renders `imageUrl` if present; else a **premium placeholder** (category-tinted gradient + initials + tactical frame). Generation-ready hook `useCharacterPortrait()` that can call `avatarApi.generate` and shows a "Generating portrait…" state; falls back to placeholder. **Never** renders any `*_prompt`.
- Remove all prompt text from `ClueDetailPage`, `LocationPage`, and replace `AvatarPlaceholder` usages.
- `ClueCard` / evidence-card component: image if available, else premium evidence placeholder with title/type/reliability — never the visual prompt (except behind an explicit dev flag).

### C. Cinematic visual system (feel)
- Extend `tokens.css`: glass, glow, gradients, risk palette, HUD radii.
- Global overhaul in `app.css`: atmospheric background, HUD sidenav/topbar, glass panels, glowing active states, mission tiles, animated markers, cinematic loaders.
- Reusable game components: `GameButton`, `StatusChip`, `RiskMeter`, `ObjectiveProgress`, `MissionTile`, `MissionHUD`, `WalletBalance`, `CharacterCard`, `LoadingScreen`.

### D. Screen upgrades
- **Mission Hub** (`DashboardPage`): game lobby — agent summary, wallet, active mission hero, mission-type tiles, recent history.
- **Active Mission**: command center — HUD (objective, progress, risk, time), recommended action, CTAs (Open Map / Ask AI), event feed.
- **Map + Location bottom sheet**: marker states (recommended/new-clue/character/locked/completed/high-risk), immersive bottom sheet with actions.
- **Character chat**: in-game conversation with avatar, mood/trust chips, typing indicator ("Mission Control is analyzing…"), cost.
- **AI Assistant** (`GuidancePanel`): Mission Control with suggested questions (localized), typing state, referenced items.
- **Wallet/Profile**: game-economy + agent-dossier styling.
- Loading/empty/error states everywhere, localized (EN/FA).

### E. Verification
- `npm run build`, `npx vitest run`, backend `go build ./... && go test ./...`.
- Manual RTL (fa) / LTR (en) pass.

## 3. Non-goals
- No framework swap, no Tailwind migration, no full rewrite.
- No real Google Maps SDK swap (keep the existing tactical board; data contract already Maps-compatible).
