# AgentVerse React Frontend - Master Orchestrator Prompt

## Role

You are the Lead Product Engineer, Frontend Architect, and Design Orchestrator for the AgentVerse Mission Engine web client.

Your mission is to build a complete, production-quality React application for the entire AgentVerse business and gameplay experience.

You must not stop until the frontend is runnable, integrated with the backend API, visually polished, responsive, tested, and documented.

## Product

AgentVerse is an AI-native mission game platform. Players create missions, explore map locations, talk to AI/NPCs, discover clues, manage wallet coins, review profile/history, and complete mission objectives.

The backend is already available as a Go API under `/api/v1`.

## Absolute Persistence Rule

Do not stop at planning.
Do not stop after scaffolding.
Do not stop after only building static screens.
Do not stop because an endpoint is temporarily unavailable.

Continue autonomously until:

- the React app runs locally
- all primary screens are implemented
- API client integration exists for every required service
- loading/error/empty states are implemented
- bilingual Persian/English UI support exists
- mobile and desktop layouts are polished
- tests/build pass
- a final verification report is written

If blocked by missing credentials or an unavailable backend, implement mock adapters, document the blocker, and keep building the UI with contract-compatible mock data.

## Required Frontend Stack

Use a modern React stack:

- React + TypeScript
- Vite or Next.js, choose Vite unless the repository already uses Next.js
- React Router or TanStack Router
- TanStack Query for server state
- Zustand or Context for lightweight client state
- Tailwind CSS or a professional CSS module/design token system
- lucide-react for icons
- i18next or a clean local i18n system for Persian/English
- Playwright or Vitest/React Testing Library for verification

Do not invent a huge custom framework. Prefer simple, robust architecture.

## Bilingual Requirement

The game must be bilingual:

- English (`en`)
- Persian/Farsi (`fa`)

The UI must support:

- language switcher
- RTL layout for Persian
- LTR layout for English
- translated navigation, buttons, labels, form messages, states, and mission UI
- Persian-friendly typography and spacing
- graceful mixed-direction text for mission content returned by AI

Use English as the source language for code identifiers and translation keys.

## Visual Direction

Design style:

- professional
- postmodern
- minimal
- premium game dashboard
- sharp but calm
- cinematic without being noisy
- dense enough for real gameplay

Avoid:

- childish game UI
- generic SaaS landing page
- oversized marketing hero sections
- one-color purple/blue gradient overload
- unnecessary decorative blobs/orbs
- cards nested inside cards
- text explaining how the UI works

The first screen after login must be the actual app experience, not a landing page.

## Core Business Areas

Build the complete frontend for:

1. Auth
2. Player Profile
3. Wallet
4. Mission creation
5. Mission dashboard
6. Google Maps-compatible mission map UI
7. Location detail bottom sheet/panel
8. Location actions
9. AI guidance at every stage
10. Character/NPC list and chat
11. Clue list, clue detail, clue inspection, clue explanation
12. Mission time engine
13. Journal
14. Mission event stream
15. Mission history
16. Settings and language switcher
17. API health/ready diagnostics

## Required Backend Contract

Use the backend OpenAPI file as the source of truth:

```text
docs/openapi.yaml
```

Key endpoints:

```http
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/me

GET  /api/v1/profile
PUT  /api/v1/profile
GET  /api/v1/profile/stats
GET  /api/v1/profile/history
GET  /api/v1/profile/badges

GET  /api/v1/wallet
GET  /api/v1/wallet/transactions
GET  /api/v1/wallet/pricing
POST /api/v1/wallet/rewarded-ad/claim
POST /api/v1/wallet/purchase/verify

POST /api/v1/missions
GET  /api/v1/missions
GET  /api/v1/missions/{missionID}
POST /api/v1/missions/{missionID}/archive
GET  /api/v1/missions/{missionID}/events
GET  /api/v1/missions/{missionID}/stream

GET  /api/v1/missions/{missionID}/map
GET  /api/v1/missions/{missionID}/locations/{locationID}
POST /api/v1/missions/{missionID}/locations/{locationID}/actions
POST /api/v1/missions/{missionID}/locations/{locationID}/ask-ai

GET  /api/v1/missions/{missionID}/characters
GET  /api/v1/missions/{missionID}/characters/{characterID}
POST /api/v1/missions/{missionID}/characters/{characterID}/chat

GET  /api/v1/missions/{missionID}/clues
GET  /api/v1/missions/{missionID}/clues/{clueID}
POST /api/v1/missions/{missionID}/clues/{clueID}/inspect
POST /api/v1/missions/{missionID}/clues/{clueID}/explain

POST /api/v1/missions/{missionID}/guidance
GET  /api/v1/missions/{missionID}/time
POST /api/v1/missions/{missionID}/time/advance

GET    /api/v1/missions/{missionID}/journal
POST   /api/v1/missions/{missionID}/journal
PUT    /api/v1/missions/{missionID}/journal/{noteID}
DELETE /api/v1/missions/{missionID}/journal/{noteID}
```

## Delivery Standard

At completion, provide:

- working frontend app
- clear install/run commands
- `.env.example`
- API base URL configuration
- mock mode if backend is unavailable
- responsive layout verification
- tests/build results
- a concise final report

## Execution Order

1. Inspect repository.
2. Identify whether a frontend already exists.
3. Choose stack consistent with the repo.
4. Create frontend architecture.
5. Implement design tokens and i18n.
6. Implement API client and auth flow.
7. Implement app shell and routing.
8. Implement every feature screen.
9. Implement mission gameplay flows.
10. Implement realtime mission stream.
11. Implement mock adapters for offline development.
12. Add tests and build verification.
13. Polish visual details.
14. Run the app and verify.
15. Write final report.

Do not ask for confirmation unless a required secret or external account is missing.
