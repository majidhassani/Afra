# AgentVerse — Global Game Polish Pass

A targeted product-quality pass on top of the prior upgrades. **No rewrites**;
extend what exists and keep every backend/frontend flow working.

## What prior passes already delivered

- **Mission command HUD** ([MissionDashboardPage.tsx](../src/features/missions/MissionDashboardPage.tsx)):
  objective, progress meter, risk meter, time remaining, wallet balance,
  recommended-next banner, win/failure conditions, completion-readiness panel
  with missing requirements + Complete CTA, entity summary cards.
- **Rich dashboard DTO** ([internal/mission/dashboard.go](../../internal/mission/dashboard.go))
  wired to `GET /api/v1/missions/{id}/dashboard`, plus the mock adapter path.
- **Map markers** with recommended / new-clue / character / locked / visited /
  high-risk states; Mission Control guidance with localized suggested questions.
- **Avatar & clue prompt safety** (premium placeholders, no raw prompts).
- **Wallet coin visuals**, game buttons, glass/HUD CSS system.
- **i18n en/fa with RTL**, locale metadata on every AI request, dev language guard.
- **PWA basics**: `manifest.webmanifest`, `sw.js`, `pwa-icon.svg`, theme-color.

## Remaining UX problems (this pass)

1. **Recommended-action contract mismatch (bug).** The frontend + mock use
   `{type,title,description,target_type,target_id,priority,cost_hint}`, but the
   Go dashboard emits the internal guidance shape `{action,reason,location_id,
   character_id,priority}`. Against the real API the recommended banner loses its
   description and location link (falls back to a heuristic). Must be aligned.
2. **Timeline is thin.** The events panel shows only `type` + a wall-clock time.
   The spec wants a real **TimelineLog**: mission time, event title, type,
   importance, related location/clue.
3. **No mobile safe-area / touch polish.** `viewport-fit=cover` is missing and no
   CSS uses `env(safe-area-inset-*)`, so the bottom nav / chat composer collide
   with the home indicator in PWA/standalone. Touch targets are small.
4. **Map detail is a side panel, not a mobile bottom sheet.** On phones it stacks
   but doesn't read as a premium sheet.
5. **No win/loss reveal.** Completing a mission returns a rich `MissionResult`
   (score, stars, summary, objectives, clues, rewards) but the UI only toasts.

## Exact files to modify

| Area | File |
| --- | --- |
| Recommended-action DTO | `internal/mission/dashboard.go` |
| TimelineLog | new `web/src/features/missions/TimelineLog.tsx`; use in `MissionDashboardPage.tsx` |
| Result reveal | new `web/src/features/missions/MissionResultModal.tsx`; use in `MissionDashboardPage.tsx` |
| Mobile safe-area / sheet | `web/index.html`, `web/src/styles/game.css`, `web/src/styles/app.css`, `web/src/features/map/MapPage.tsx` |
| Strings | `web/src/shared/i18n/en.ts`, `web/src/shared/i18n/fa.ts` |
| Mock parity | `web/src/shared/api/mock/mockClient.ts` (event importance/title fields) |

## Implementation order

1. Backend recommended-action contract alignment (+ `go build`/`go test`).
2. i18n strings (timeline + result reveal) in en **and** fa.
3. `TimelineLog` component + integrate.
4. `MissionResultModal` + integrate with the `complete` mutation.
5. Mobile: `viewport-fit=cover`, safe-area insets, touch sizing, map bottom sheet.
6. Verify: build, test, RTL/LTR visual smoke.

## Risks

- Changing the dashboard DTO could drift from the mock. Mitigation: the frontend
  already tolerates both shapes; align the Go output to the shape the TS types +
  mock already use, and keep fields optional.
- i18n parity is enforced by a test (`i18n.test.ts`): every new `en` key needs a
  matching `fa` key. Add both together.
- Safe-area/CSS changes must not regress desktop; scope them to mobile media
  queries and additive `env()` padding.

## Verify

```bash
# frontend
cd web && npm run build && npx vitest run
# backend (DTO change)
go build ./... && go test ./...
```
Plus a manual RTL (fa) / LTR (en) smoke test of the mission command center,
timeline, result reveal, and mobile viewport.
