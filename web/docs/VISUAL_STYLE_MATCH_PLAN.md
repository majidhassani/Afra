# AgentVerse Visual Style Match Plan

## Source Brief

- Read all files from `AgentVerse_Visual_Style_Match_Codex_Agent`.
- Target visual language: dark sci-fi Unity-style HUD, black glass panels, green/cyan glow, angular borders, scanlines, tactical map/grid/topographic texture.
- CSS-first requirement: the interface must still look premium if all imagery is removed.
- Keep backend APIs, routing, gameplay, queries, and existing feature behavior intact.

## Implementation Strategy

1. Add AVDS global tokens and CSS-only tactical background primitives.
2. Create reusable HUD components:
   - `GameTopBar`
   - `GameBottomNav`
   - `HudPanel`
   - `TacticalButton`
   - `ResourceChip`
   - `MissionDossierCard`
   - `TimelineRail`
   - `MissionMapPanel`
   - `SelectedMissionBar`
   - `ProgressRing`
   - `StatusChip`
3. Replace dashboard shell language with game HUD language:
   - Remove the persistent SaaS sidebar as the primary visual object.
   - Promote top command bar and bottom route rail.
   - Keep responsive/mobile navigation and active mission context.
4. Migrate screens in order:
   - Home / Mission Command
   - Mission Select
   - Active Mission / Gameplay
   - Map
   - Evidence / Clues
   - Characters
   - Report Center
   - Profile / History
   - Wallet / Resources
   - Settings and remaining player-facing screens
5. Use broad shared primitive styling so legacy panels, lists, forms, empty/loading/error states inherit the AVDS look.

## Screen Mapping

- Home: centered AgentVerse identity, circular deploy CTA, active mission dossier, tactical bottom nav.
- Mission Select: filter rail, dossier cards, progress rings, selected mission bar, deploy CTA.
- Active Mission: left mission panels, central tactical map/overview, right timeline/stage panel, bottom action bar.
- Map: CSS tactical board fallback remains first-class, side details become HUD panel.
- Evidence, Characters, Report, Profile, Wallet, Settings: shared HUD panels, chips, angular buttons, dense readable tactical layouts.

## Acceptance Checks

- No main screen should read as SaaS dashboard/admin UI.
- All player-facing screens must use dark glass HUD panels and tactical controls.
- Background must be CSS-only and survive image removal.
- RTL/LTR must remain safe through logical CSS and existing `dir` handling.
- Run:
  - `cd web && npm run build`
  - `cd web && npm run lint || true`
  - `cd web && npm test || true`
