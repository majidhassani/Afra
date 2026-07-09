# AgentVerse UI/UX Reconstruction Plan

## Direction

AgentVerse is treated as a premium tactical mission game, not as a SaaS dashboard. The visual reference is a restrained sci-fi HUD: generous negative space, graphite glass, emerald/cyan instrumentation, amber rewards, sharp tactical borders, and a persistent sense of mission state.

## Audit Findings

- Visual: the current shell can collapse into a narrow RTL-aligned content column, leaving most of the desktop viewport empty.
- UX: mission context, current objective, time, risk, and next action are split across too many sections and are not consistently present.
- Typography: the display and body roles are not separated; remote font loading can leave Persian in an unstyled system fallback.
- Avatars: the procedural fallback is an 8x8 mirrored identicon. It is technically sharp but visually reads as pixel art, and regenerated data URLs need stronger cache and sizing rules.
- Image pipeline: avatar generation is capped by an underspecified prompt and character/evidence assets do not share a common visual direction.
- Navigation: the shell has a game HUD layer, but secondary screens still inherit generic card and table primitives.
- Mobile/tablet/desktop: the same desktop composition is currently compressed rather than intentionally re-composed for each breakpoint.
- Accessibility: shared controls need stable focus treatment, explicit labels, and reduced-motion handling.
- RTL: layout direction is respected in the app, but display tracking, clipping, and inline icon order need RTL-safe rules.
- Animation: ambient motion exists, but interaction feedback and page transitions need consistent AVDS timing tokens.
- Performance: use lazy media, `decoding="async"`, bounded image dimensions, and avoid loading large board art as a required dependency.

## Implementation Order

1. Lock AVDS tokens, font roles, surface primitives, and CSS-only background.
2. Make the shell full-width and establish the desktop HUD, tablet composition, and mobile bottom navigation.
3. Keep Home, Mission Select, Active Mission, Map, Evidence, Characters, Report, Profile, Wallet, and Settings on shared HUD primitives.
4. Make time, stage, objective, risk, resources, and timeline visible in active mission contexts.
5. Upgrade avatar and evidence generation prompts to 1024px, preserve original assets, and apply responsive rendering without thumbnail upscaling.
6. Validate build, tests, keyboard focus, reduced motion, RTL/LTR, and desktop/tablet/phone screenshots.

## Acceptance Checks

- No main screen looks like an admin dashboard.
- Removing all images leaves a complete game HUD.
- No horizontal overflow at 390px, 768px, or 1440px.
- Every active mission surface exposes mission, stage, progress, time, risk, wallet, energy, objective, and next action without crowding.
- Generated portraits request 1024x1024 and use a shared high-quality art direction.
- Existing APIs, routes, gameplay actions, and feature behavior remain intact.

