# Repository Upgrade Report — AgentVerse Mission Engine

This report defines how the existing CaseMind backend is upgraded — not rewritten — into the AgentVerse Mission Engine.

## 1. Current Architecture Summary

- **Language/runtime:** Go, stdlib `net/http` mux (Go 1.22 route patterns), pgx v5 pool, Redis (rate limiting), embedded SQL migrations, slog logging.
- **Layering:** `handler → service → repository`, one module per domain under `internal/`. Handlers never contain business logic. Cross-module access goes through narrow gateway interfaces (e.g. `CaseGateway.EnsureOwned`).
- **Agent architecture:** `orchestrator → runtime → llm.Client`. Agents are declarative (Manifest + Prompt + Parse); the runtime owns timeouts, JSON extraction, retries with repair prompts, and audit logging into `agent_runs`. Agents return *intentions*; services validate and persist.
- **LLM:** `llm.Client` interface with a GLM (OpenAI-compatible) provider and a deterministic mock provider keyed on a `TASK_TYPE:` prompt marker.
- **Security:** JWT auth middleware, per-user ownership checks, prompt-injection detector, security preamble in every agent prompt, and a privacy-guard middleware that scans outgoing JSON for Truth Layer keys.
- **Infra:** Docker Compose with `api`, `postgres`, `redis` (+ optional `qdrant`, `minio`, `adminer`), health checks, auto-migrate on boot, Swagger UI at `/swagger`.

## 2. Existing Inventory

- **Domains:** auth, detective, case, casebible, suspect, evidence, location, timeline, conversation, notes, memory, history, solve.
- **Handlers:** auth, detective, cases, suspects, evidence, timeline, location(map), notes, solve (all under `/api/v1`).
- **Services:** one per domain plus case Generator (multi-agent pipeline) and memory service.
- **Repositories:** PG implementations per domain, `agent_runs` audit repo.
- **DB tables:** users, refresh_tokens, detective_profiles, cases, case_bibles, suspects, evidence, locations, real_timeline_events, player_timeline_events, conversations, conversation_messages, player_notes, discovered_facts, solve_attempts, agent_runs, case_events, evidence_inspections.
- **Docker services:** api, postgres, redis (+ profiles: qdrant, minio, adminer).

## 3. Decisions

### Kept (unchanged)
Auth, platform (db/redis/logger/http middleware), pkg (errors/response/validator), llm + GLM provider, agent runtime core, orchestrator, memory, notification bus, Docker topology, migration runner, legacy detective/case gameplay (still functional during transition).

### Renamed (new modules + data conversion; legacy tables/routes retained temporarily)
| Old | New |
|---|---|
| Case | Mission (`internal/mission`, `missions`) |
| CaseBible | WorldBible (`internal/worldbible`, `world_bibles`) |
| Suspect | Character (`internal/character`, `characters`) |
| Evidence | Clue (`internal/clue`, `clues`) |
| Location | MapLocation (`internal/gamemap`, `map_locations`) |
| Conversation | Interaction (`internal/interaction`, `interactions`) |
| Player notes | Journal (`internal/journal`, `journal_notes`) |
| DetectiveProfile | PlayerProfile (`internal/playerprofile`, `player_profiles`) |
| SolveAttempt | MissionOutcome (mission `complete` + JudgeAgent) |
| Case events | Mission events (`mission_events`) |

Detective remains a mission type: `mission_type = detective`.

### Refactored / Extended
- `agent/runtime`: Task/Run gain `MissionID`; `ExecuteMeta` returns model + token usage so the wallet can settle actual cost; `agent_runs` gains `mission_id`.
- `llm/mock`: canned schema-valid outputs for every new mission task type (mission_plan, world_generation, mission_map_generation, character_generation, clue_generation, character_dialogue, ai_guidance, clue_explanation, clue_inspection, location_action, time_advance, mission_director, mission_judgment).
- `privacy`: forbidden keys extended with `hidden_state`, `character_secrets`, `clue_truth`, `map_truth`, `timeline_truth`, `failure_rules`, `private_state`, `system_prompt`, `hidden_antagonist`.
- `auth`: profile creation now creates both detective (legacy) and player profiles.

### Added
- `internal/wallet` — wallets, transactions, reservations, pricing rules, LLM usage logs, rewarded ads, purchase receipts; **WalletGuard** (estimate → reserve → run → settle → refund) that every paid LLM action must pass through.
- `internal/mission` — mission lifecycle (draft/generating/ready/active/paused/completed/failed/archived), objectives, dashboard (Swift contract), multi-agent generation pipeline, completion via MissionJudgeAgent.
- `internal/worldbible` — private truth store (never serialized to clients).
- `internal/character` — NPCs with avatar/thumbnail prompts, trust/stress/mood, chat via DialogueAgent.
- `internal/clue` — clues with visual descriptions + thumbnail prompts, public/internal split, AI inspect + explain.
- `internal/gamemap` — Google-Maps-compatible markers (center/zoom/lat/lng/status/risk/badges), location detail, location actions, ask-AI.
- `internal/interaction` — persisted dialogue threads (NPC chat, guidance).
- `internal/guidance` — GuidanceAgent: hints at every stage, never reveals truth.
- `internal/timeengine` — mission clock ("Day N - HH:MM"), advance-time flow (TimeAgent + MissionDirectorAgent), world change events.
- `internal/journal` — mission-scoped notes CRUD.
- `internal/playerprofile` — level/XP/rank, mission stats, clue/AI/coin counters, history.
- New agents under `internal/agent/`: `missionagent`, `worldagent`, `missionmap`, `characteragent`, `clueagent`, `dialogueagent`, `guidanceagent`, `timeagent`, `missiondirector`, `missionjudge`.

## 4. LLM Call Path (enforced)

```
Handler -> Service -> WalletGuard(reserve) -> Orchestrator -> Agent -> Runtime -> GLM/Mock
        <- Service <- WalletGuard(settle/refund + llm_usage_log) <- intention (validated)
```

No handler touches the LLM. No agent touches the database. Every paid action produces a wallet transaction and an LLM usage log traceable to mission, agent, and action.

## 5. Data Migration (0002)

1. Create all new tables (missions, world_bibles, characters, map_locations, clues, interactions, interaction_messages, mission_events, journal_notes, player_profiles, wallets, wallet_transactions, wallet_reservations, pricing_rules, llm_usage_logs, rewarded_ads, purchase_receipts).
2. Add `mission_id` to `agent_runs`.
3. Convert existing data preserving ownership and IDs:
   - cases → missions (`type='detective'`), case_bibles → world_bibles, suspects → characters, evidence → clues, locations → map_locations, conversations(+messages) → interactions(+messages), player_notes → journal_notes, detective_profiles → player_profiles, case_events → mission_events.
4. Seed `pricing_rules` (server-side prices only) and create wallets with a starting balance for existing users.

## 6. Incremental Implementation Plan

1. **Migration 0002** — schema + data conversion + pricing seed.
2. **Wallet + WalletGuard** — reserve/settle/refund, pricing, rewarded ad, purchase verify, usage logs.
3. **PlayerProfile** — profile/stats/history endpoints.
4. **Mission + WorldBible + events** — CRUD, generation pipeline, dashboard, SSE stream.
5. **Character/Clue/Map/Interaction/Journal** — public DTOs + AI actions.
6. **Guidance + Time engine** — hints everywhere, advance time.
7. **Agents + mock outputs** — all mission task types.
8. **Router/main wiring, privacy keys, Swagger, README.**
9. **Tests** — wallet, ownership, privacy scans, agent JSON retry, endpoint flows with mock LLM.
10. **Docker verification** — `docker compose up --build`.

## 7. API Surface (new, under `/api/v1`)

Profile: `GET/PUT /profile`, `GET /profile/stats`, `GET /profile/history`.
Wallet: `GET /wallet`, `GET /wallet/transactions`, `GET /wallet/pricing`, `POST /wallet/rewarded-ad/claim`, `POST /wallet/purchase/verify`.
Missions: `POST/GET /missions`, `GET /missions/{id}`, `POST /missions/{id}/start|archive|complete`, `GET /missions/{id}/events`, `GET /missions/{id}/stream`.
Guidance: `POST /missions/{id}/guidance`.
Map: `GET /missions/{id}/map`, `GET /missions/{id}/locations/{locID}`, `POST /missions/{id}/locations/{locID}/actions`, `POST /missions/{id}/locations/{locID}/ask-ai`.
Characters: `GET /missions/{id}/characters`, `GET /missions/{id}/characters/{charID}`, `POST /missions/{id}/characters/{charID}/chat`.
Clues: `GET /missions/{id}/clues`, `GET /missions/{id}/clues/{clueID}`, `POST /missions/{id}/clues/{clueID}/inspect|explain`.
Journal: `GET/POST /missions/{id}/journal`, `PUT/DELETE /missions/{id}/journal/{noteID}`.
Time: `GET /missions/{id}/time`, `POST /missions/{id}/time/advance`.

Legacy `/api/v1/cases/**` routes remain during the transition.
