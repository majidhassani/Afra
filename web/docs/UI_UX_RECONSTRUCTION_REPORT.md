# AgentVerse UI/UX Reconstruction Report

## Changed Screens

Home / Mission Command, Mission Select, Active Mission / Gameplay, Map, Evidence, Characters, Report Center, Profile, Wallet / Resources, Settings, and shared mission detail surfaces now consume the AVDS shell and primitives.

## Created Components

`GameTopBar`, `GameBottomNav`, `HudPanel`, `TacticalButton`, `ResourceChip`, `MissionDossierCard`, `TimelineRail`, `MissionMapPanel`, `SelectedMissionBar`, `ProgressRing`, `StatusChip`, shared loading/empty/error states, and the responsive `Avatar` renderer.

## Removed or Replaced

The visible legacy sidebar/topbar composition and generic dashboard emphasis were replaced by the tactical HUD shell. Existing routes, backend APIs, and gameplay actions were retained.

## Avatar and Image Pipeline

Avatar and evidence generation now request 1024px source assets with explicit portrait/evidence composition, consistent AgentVerse lighting and palette, no pixel-art language, and no low-resolution fallback being presented as a generated portrait. Client rendering uses bounded dimensions, lazy decoding, `object-fit: cover`, and versioned asset URLs.

## Responsive and Accessibility

Desktop uses a full-width HUD with floating panels; tablet collapses instrumentation into stacked mission surfaces; phone uses compact top resources and bottom navigation. Focus-visible styles, semantic labels, reduced-motion rules, and RTL-safe logical properties are part of AVDS.

## Remaining Issues

Actual image quality depends on configuring `IMAGE_API_BASE_URL`, `IMAGE_API_KEY`, and `IMAGE_API_MODEL`. Without an image provider the backend intentionally keeps a deterministic non-network fallback. Visual QA should be repeated against the deployed provider and with both Persian and English locales.

## Screenshots Checklist

- [ ] Home at 1440px LTR and RTL
- [ ] Mission Select at 1440px and 768px
- [ ] Active Mission at 1440px, 1024px, and 390px
- [ ] Map marker and bottom-sheet states
- [ ] Evidence and Characters with generated assets
- [ ] Report Center complete and pending states
- [ ] Profile, Wallet, Settings
- [ ] Reduced motion and keyboard focus pass

