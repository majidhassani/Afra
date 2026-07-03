# AgentVerse React Frontend - Architecture and API Integration Prompt

## Role

You are the Frontend Platform Architect for AgentVerse.

Build the frontend architecture so the app is maintainable, testable, and resilient when the backend is slow, offline, or returning game-generation states.

## Required Architecture

Use feature-based structure:

```text
src/
  app/
    App.tsx
    router.tsx
    providers.tsx
  shared/
    api/
    config/
    i18n/
    ui/
    hooks/
    utils/
    types/
  features/
    auth/
    profile/
    wallet/
    missions/
    map/
    characters/
    clues/
    guidance/
    time/
    journal/
    diagnostics/
  styles/
```

Keep business logic out of components where reasonable.

## API Client

Implement a typed API client:

- base URL from env
- JSON envelope parsing
- auth token injection
- refresh token handling
- uniform error type
- request cancellation support
- retry only for safe queries
- no hidden global fetch calls inside components

Backend response envelope:

```json
{
  "data": {}
}
```

Error envelope:

```json
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

## Environment

Create `.env.example`:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_ENABLE_MOCKS=false
VITE_DEFAULT_LANGUAGE=en
```

## Auth

Implement:

- register
- login
- logout
- refresh token
- persisted auth session
- route guards
- automatic redirect to login
- clear error states

Token fields:

```json
{
  "access_token": "string",
  "access_expires_at": "date",
  "refresh_token": "string",
  "refresh_expires_at": "date"
}
```

## TanStack Query

Use query keys like:

```ts
["me"]
["profile"]
["profile", "stats"]
["wallet"]
["wallet", "transactions", limit]
["wallet", "pricing"]
["missions"]
["mission", missionId]
["mission", missionId, "map"]
["mission", missionId, "events"]
["mission", missionId, "characters"]
["mission", missionId, "character", characterId]
["mission", missionId, "clues"]
["mission", missionId, "clue", clueId]
["mission", missionId, "journal"]
["mission", missionId, "time"]
["health"]
["ready"]
```

Invalidate relevant queries after mutations.

## Mission Generation UX

When creating a mission:

- send `POST /api/v1/missions`
- backend returns `202`
- mission status starts as `generating`
- poll mission detail/events until `ready`, `active`, or `failed`
- also connect SSE stream if possible
- show progress using events
- never freeze the UI

## SSE Mission Stream

Implement `EventSource` or fetch-based SSE wrapper for:

```http
GET /api/v1/missions/{missionID}/stream
```

Because EventSource cannot set Authorization headers natively, choose one:

1. Use fetch streaming with Authorization header.
2. If backend later supports token query parameter, switch to EventSource.
3. Fallback to polling `/events` every few seconds.

Implement automatic reconnect/backoff.

## Mock Mode

If backend is unavailable or `VITE_ENABLE_MOCKS=true`:

- mock all major endpoints
- preserve backend response envelope shape
- include realistic mission data
- support both English and Persian sample content
- mock mission generation progress
- mock wallet cost deductions

Mock mode is for frontend development only and must be clearly isolated.

## Type Safety

Create TypeScript types for:

- User
- TokenPair
- Profile
- Wallet
- Transaction
- Pricing
- Mission
- Objective
- MapView
- Marker
- LocationDetail
- Character
- CharacterDetail
- ChatResult
- Clue
- ClueInspectResult
- ClueExplainResult
- GuidanceResult
- TimeInfo
- TimeAdvanceResult
- JournalNote
- MissionEvent
- APIError

## Error Handling

Implement user-friendly handling for:

- unauthenticated
- token expired
- insufficient balance
- mission still generating
- hidden location/clue not found
- validation errors
- network offline
- backend unavailable
- SSE disconnected

Use translated messages.

## Privacy Rules

The UI must never expose:

- WorldBible
- hidden_state
- character private_state
- clue internal_truth
- system prompts
- hidden mission solution

If any field like this appears in a response unexpectedly, do not render it.

## Build Verification

Before completion:

- TypeScript build passes
- lint passes if configured
- tests pass
- no console errors in main flows
- app can run with backend and mock mode
