# AgentVerse React Frontend - Product and Screen Map Prompt

## Role

You are the Product UX Architect for the AgentVerse frontend.

Define and implement the complete screen map for the entire business, not only a small demo.

## Product Scope

AgentVerse is a mission game dashboard where a player:

- registers/logs in
- manages a profile
- manages a wallet
- creates missions
- waits for mission generation
- enters a mission dashboard
- explores a map
- visits locations
- talks with characters
- discovers and analyzes clues
- asks AI for guidance
- advances mission time
- writes journal notes
- reviews events/history

## Required Routes

Implement routes like:

```text
/login
/register
/app
/app/dashboard
/app/profile
/app/wallet
/app/history
/app/settings
/app/missions
/app/missions/new
/app/missions/:missionId
/app/missions/:missionId/map
/app/missions/:missionId/locations/:locationId
/app/missions/:missionId/characters
/app/missions/:missionId/characters/:characterId
/app/missions/:missionId/clues
/app/missions/:missionId/clues/:clueId
/app/missions/:missionId/journal
/app/missions/:missionId/events
/app/missions/:missionId/time
/app/diagnostics
```

Use nested layouts where appropriate.

## App Shell

Create a serious game-product shell:

- left rail or bottom navigation depending on viewport
- top status bar with wallet balance, language switcher, current mission, health status
- mission-aware command surface
- responsive mobile navigation
- no marketing hero after login

## Auth Screens

Login/register screens must feel premium and direct:

- brand/title: AgentVerse
- form validation
- backend error display
- language switcher
- subtle visual identity
- no fake marketing carousel

## Dashboard

Dashboard must show:

- active mission card
- mission generation status
- wallet balance
- profile level/rank
- recent mission history
- quick actions
- health/ready diagnostic indicator

## Mission Creation

Mission creation must include:

- mission type selector
- difficulty selector
- region input
- language selector
- pricing/cost hint from server pricing
- wallet balance warning
- create mission button
- generation progress view after creation

Mission types:

```text
detective
wildlife_rescue
disaster_response
exploration
survival
diplomacy
medical_mystery
```

Difficulties:

```text
easy
medium
hard
expert
```

## Mission Dashboard

Mission dashboard must show:

- mission title, type, difficulty, region, status
- current mission time
- briefing
- objectives
- public state
- visible locations
- public characters
- discovered clues
- AI guidance composer
- event stream summary
- primary actions to map, characters, clues, journal, time

## Map Experience

Backend returns Google Maps-compatible markers. If Google Maps key is unavailable:

- implement a high-quality fallback map board using marker coordinates
- do not block the app
- show marker cards and spatial relative positions

Map UI must support:

- mission center
- markers
- marker status
- marker badges
- locked/visited/discovered states
- location panel/bottom sheet
- action buttons
- AI ask button
- clue/character indicators

## Character Chat

Character UI must include:

- character list
- avatar/thumbnail prompt display as placeholder visual when no image exists
- trust/stress/mood indicators
- message history
- chat composer
- cost display
- unlocked clues
- new facts
- loading and failure states

## Clue UI

Clue UI must include:

- discovered clue list only
- clue detail
- reliability/importance
- visual description
- thumbnail prompt placeholder
- inspect action
- explain action
- compare suggestions
- next steps

Never display internal truth fields.

## Guidance UI

AI guidance must exist across:

- mission dashboard
- map screen
- location panel
- clue detail
- journal review
- objective planning
- time review

It must show:

- message
- hint level
- referenced items
- cost charged

It must never imply that it can reveal hidden truth.

## Time Engine UI

Time UI must include:

- current time
- public state
- advance time controls
- amount/unit selector
- cost display
- time event results
- risk warnings

## Wallet UI

Wallet UI must include:

- balance
- reserved balance if present
- pricing rules
- transactions
- rewarded ad claim
- purchase verify mock/testing form
- insufficient balance messaging

## Profile UI

Profile UI must include:

- display name edit
- rank, level, XP
- mission stats
- clues found
- AI interactions
- coins spent/earned
- badges
- history

## Journal UI

Journal UI must include:

- note list
- create/edit/delete
- bilingual text support
- mission context sidebar
- AI guidance entry point

## Diagnostics UI

Diagnostics must include:

- `/health`
- `/ready`
- API base URL
- current auth state
- app version/build
- last API error

## Completion Criteria

Every route above must exist.
Every route must have useful loading, empty, error, and success states.
Every route must be responsive.
Every visible label must be translatable.
