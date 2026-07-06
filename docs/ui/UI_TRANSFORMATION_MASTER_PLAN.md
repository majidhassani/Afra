# AgentVerse — UI Transformation Master Plan

Art direction brief: destroy every dashboard pattern and replace the product's
identity with a premium indie/AAA game feel. Within three seconds of opening,
AgentVerse must read as a **game**, not a web app. Philosophy: *information is
never data — it is gameplay.* Every screen answers "what should the player do?".

Reference philosophy (not assets): XCOM 2, Firewatch, Disco Elysium, Shadows of
Doubt, Her Story, Control, The Long Dark, The Division, Death Stranding, Marvel
Snap / Hearthstone polish, Monument Valley motion, Supercell clarity.

Hard constraint: **do not rewrite the backend.** Reuse the mission engine,
wallet guard, AI runtime, timeline, and existing API contracts. This is a
front-end art/UX/motion overhaul on top of a working system.

## 1. Current Screens & Problems

| Screen | Today | Problem |
|---|---|---|
| Mission Hub (`DashboardPage`) | card grid + list | reads as SaaS home |
| Mission Command (`MissionDashboardPage`) | HUD stats + panels | was stat-tiles; now cinematic hero (in progress) |
| Map (`MapPage`) | Google map + fallback | good base; needs fog/zones/animated markers |
| Characters | card grid | cards static, not "alive" |
| Character chat | transcript | fine; frame + portrait bg pending |
| Clues | list + detail | reads as records, not collectible artifacts |
| AI guidance | chat panel | reads as chat, not Mission Control briefings |
| Timeline | vertical log | needs investigation-board feel |
| Wallet | balance + receipt form | reads as billing, not game economy |
| Profile | stat blocks | reads as account page, not agent progression |
| History | list/cards | archive cards done in earlier phase |

Root cause: neutral flat surfaces, generic cards, no world identity, no motion,
no environment. Fixing this is a **design-system + theming + motion** problem,
not a per-component patch.

## 2. New Vision

- **The UI is part of the world.** A biome theme (forest/desert/city/snow/
  horror/space) tints the entire shell — backdrop, accents, glows — derived from
  the active mission. Nothing stays neutral in-mission.
- **Command Center, not dashboard.** The mission screen is a full-bleed HUD:
  resource strip, framed briefing panel, tactical rail, action bar over a
  biome-tinted cinematic backdrop.
- **Cards are objects, not records.** Clues are evidence artifacts; characters
  are dossiers with trust/mood/risk; missions are cinematic banners.
- **AI is Mission Control.** Structured briefings ("Commander, we have a
  problem…") with recommended action + risk + cost, not a chat bubble.
- **Everything moves** — within reduced-motion limits.

## 3. Wireframes (ASCII)

Mission Command Center (mobile-first, scales to desktop):

```
┌───────────────────────────────────────────────┐
│ ⛃ 25,430  ⏱ Day1-09:00  [HARD] [ACTIVE]   🟢 جنگل زمرد │  resource strip
│                                                │
│  WILDLIFE RESCUE                     ▢ map     │  eyebrow      rail
│  The Silent Crow                     ▢ chars   │  title        ▢
│  Find why collars went dark          ▢ clues   │  objective    ▢
│  PROGRESS ▓▓▓▓░ 40%  RISK ▓▓░ med    ▢ ai      │  meters       ▢
│  ⏱ 02:18:34                          ▢ log     │  timer        ▢
│                                                │
│  [ ▶ Open Map ] [ ✦ Mission Control ] [ ⏱ ]    │  action bar
└───────────────────────────────────────────────┘
```

Mission Hub (lobby):

```
┌──────── ACTIVE OPERATION (cinematic banner) ────────┐
│  rank crest • credits • [ RESUME ]  [ MAP ]          │
└──────────────────────────────────────────────────────┘
DEPLOY NEW OPERATION
[forest] [desert] [city] [snow] [horror] [space]  tiles
RECENT OPERATIONS  → archive cards
```

Clue as evidence artifact:

```
┌───────────────┐
│  [ image ]    │  ← generated or evidence placeholder
│  BROKEN COLLAR│  title
│  document·high│  category · importance chip
│  ▓ reliability│  meter
└───────────────┘
```

## 4. Component Migration

| Old | New | Notes |
|---|---|---|
| `.hud` stat tiles | `.cmd-hero` cinematic HUD | done |
| `page-header` (mission) | resource strip + rail | done |
| flat `.panel` | `.frame` corner-bracket glass | additive class |
| `.tac-card` corner | env-accent corner | via `[data-env]` |
| Wallet receipt form | Supply Packs + Mission Log | dev-gated form kept |
| Profile stat blocks | rank crest + XP + achievements | progression |
| Timeline log | investigation board | connect to entities |

New shared primitives (CSS, `cinematic.css`): `.env-backdrop`, `.frame` +
`.frame-brackets`, `.cmd-*` (hero/topbar/res/region/body/mission/meters/timer/
rail/actions). New module: `shared/theme/environment.ts`.

## 5. Mission Theme System (the core of "world theming")

Six biomes, each with accent, secondary, glow, scrim, and a layered radial
`--env-wash` backdrop:

| Biome | Accent | Mood |
|---|---|---|
| forest | green | fog, wood, soft sun |
| desert | gold/orange | dust, heat, stone |
| city | steel-blue / neon | rain, glass, cyber |
| snow | ice-blue | white, wind, frozen glass |
| horror | red on black | shadow, warning lights |
| space | violet/cyan | holograms, stars, radar |

Derivation: mission `region`/`summary` keywords first, then a `MissionType →
biome` fallback. Applied by setting `data-env` on `<html>` from the **AppShell**
whenever a mission is open, so map/characters/clues/AI/timeline all share it.
Cleared off-mission. CSS drives every visual from `--env-*` variables.

## 6. Animation Plan (respect `prefers-reduced-motion`)

- backdrop cross-fade on biome change (600ms)
- hero sheen sweep (idle, 9s loop)
- marker pulse/glow (recommended), bottom-sheet spring, card hover lift
- timeline reveal (staggered), star reveal + XP/coin count-up (result)
- AI typing dots; reward reveal on mission complete
- All gated: reduced-motion collapses durations to ~1ms and disables loops.

Motion tokens live in `tokens.css` (`--av-duration-*`, `--av-ease-out`).

## 7. Color System (semantic, never random)

green=nature, blue=AI, violet=mystery, gold=objective, orange=time,
red=danger, gray=completed. Biome accent overlays these in-mission but semantic
meaning (risk=red, objective=gold) is preserved.

## 8. Responsive Plan (mobile-first)

Bottom nav Mission | Map | AI | Clues | Profile; wallet chip in HUD. Safe-area
insets; ≥44px targets; full-screen map; bottom sheets on mobile, side panels on
desktop; `--kb-inset` keyboard-safe chat; landscape tolerated.

## 9. Implementation Order (screen by screen)

1. Theme system (6 biomes) + backdrop + framed primitives ← foundation
2. Mission Command Center (centerpiece)
3. Mission Hub (lobby)
4. Wallet → Resources (economy language + supply packs)
5. Profile → agent progression
6. Characters / Clues / AI / Timeline as game artifacts
7. Map fog-of-war / zones / animated markers polish

Each screen is a separate commit and independently revertable.

## 10. Acceptance Criteria

- Blur test: with all text hidden, every screen still reads as a premium game.
- Three-second test: a first-time viewer says "game", not "web app".
- Biome test: switching mission type visibly re-themes the whole shell.
- No dashboard patterns remain on player-facing screens (no metric-card grids,
  admin tables, or receipt forms in production).
- Build/lint/tests pass; reduced-motion honored; RTL (fa) intact.
