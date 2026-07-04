# AgentVerse Web

React + TypeScript client for the AgentVerse Mission Engine (Go backend in this
repository, contract in `../docs/openapi.yaml`).

## Run

```bash
cd web
npm install
npm run dev        # http://localhost:5173
```

Configuration (`.env` / `.env.local`, see `.env.example`):

```env
VITE_API_BASE_URL=http://localhost:8080   # Go backend
VITE_ENABLE_MOCKS=false                   # true = fully offline mock backend
VITE_DEFAULT_LANGUAGE=en                  # en | fa
```

With `VITE_ENABLE_MOCKS=true` the app runs without any backend: an in-memory
adapter (`src/shared/api/mock/`) mirrors the API envelope, seeds a playable
mission, simulates mission generation and charges mock coins.

## Verify

```bash
npm run build      # tsc -b && vite build
npm test           # vitest
```

## Architecture

```
src/
  app/        App, router, providers, AppShell
  shared/
    api/      typed client (envelope, refresh, errors), endpoints, SSE, mocks
    config/   env
    i18n/     local i18n (en/fa dictionaries, RTL switching)
    types/    backend DTO mirrors
    ui/       badges, states, toasts, language switcher
  features/   auth, missions, map, characters, clues, guidance,
              time, journal, events, wallet, profile, settings, diagnostics
  styles/     design tokens + app css
```

Server state is TanStack Query; auth session is a persisted Zustand store; the
mission event stream uses fetch-based SSE (Authorization header) with
automatic reconnect and polling fallback.

The UI never renders truth-layer fields (`internal_truth`, `private_state`,
`hidden_state`, WorldBible) — they are intentionally absent from the
TypeScript types.
