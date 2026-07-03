# Repository Analysis — CaseMind → AgentVerse Mission Engine

Analysis date: 2026-07-03

## Capability Inventory

| Capability | Present? | Where |
|---|---|---|
| Auth (register/login/refresh/logout/me) | Yes | `internal/auth` |
| Users | Yes | `internal/auth` (users + refresh_tokens tables) |
| Cases | Yes | `internal/case` |
| Case Bible (private truth) | Yes | `internal/casebible` |
| Suspects | Yes | `internal/suspect` |
| Evidence | Yes | `internal/evidence` |
| Timeline | Yes | `internal/timeline` |
| Map locations | Yes | `internal/location` |
| Conversations (interrogation) | Yes | `internal/conversation` |
| Notes | Yes | `internal/notes` |
| Detective profile | Yes | `internal/detective` |
| Solve/judgment | Yes | `internal/solve` |
| Event history + SSE | Yes | `internal/history`, `internal/notification` |
| LLM provider abstraction | Yes | `internal/llm` (GLM + mock) |
| Agent runtime (manifest/prompt/parse, retries, run logging) | Yes | `internal/agent/runtime` |
| Agent orchestrator | Yes | `internal/agent/orchestrator` |
| Agents (8) | Yes | `internal/agent/{casegenerator,suspect,evidence,timeline,map,psychology,director,judge}` |
| Privacy guard (Truth Layer scan) | Yes | `internal/privacy` + HTTP middleware |
| Prompt-injection detection | Yes | `internal/agent/runtime/guard.go` |
| Docker / Compose | Yes | `Dockerfile`, `docker-compose.yml` (api, postgres, redis; extras: qdrant, minio, adminer) |
| Migrations (embedded, lexical order) | Yes | `migrations/` + `internal/platform/database` |
| Swagger | Yes | `docs/openapi.yaml`, `/swagger` route |
| Tests | Yes | auth, case, solve, evidence, suspect, runtime, mock, privacy, middleware |
| Wallet | **No** | — |
| Player-facing AI guidance | **No** | — |
| Mission time engine | **No** | — |
| LLM usage/cost logs | **No** | (agent_runs has token usage but no pricing/coins) |

## Module Classification

| Module | Current Path | Classification | Reason | Action |
|---|---|---|---|---|
| Auth | `internal/auth` | KEEP | Clean JWT + refresh rotation + ownership context | Wire profile creation to PlayerProfile too |
| Platform HTTP | `internal/platform/http` | KEEP | Middleware chain, rate limit, privacy guard | Add new routes |
| Platform DB/Redis/Logger | `internal/platform/*` | KEEP | Sound | None |
| Storage/Vector | `internal/platform/{storage,vector}` | KEEP | Optional extras degrade gracefully | None |
| Config | `internal/config` | EXTEND | Missing wallet settings | Add starting balance / reward envs |
| pkg errors/response/validator | `pkg/*` | KEEP | Uniform envelope + error kinds | None |
| httpx helpers | `internal/httpx` | KEEP | — | None |
| LLM runtime | `internal/llm` (+`glm`) | KEEP | OpenAI-compatible GLM 5.2 client + JSON mode | None |
| Mock LLM | `internal/llm/mock` | EXTEND | Only knows case task types | Add canned outputs for all mission task types |
| Agent runtime | `internal/agent/runtime` | EXTEND | Solid (retries, repair prompts, audit) | Add MissionID to Task/Run; expose usage metadata for billing |
| Orchestrator | `internal/agent/orchestrator` | EXTEND | — | Add RunWithMeta (usage/model for wallet settle) |
| CaseGenerator agent | `internal/agent/casegenerator` | KEEP (legacy) | Detective flow still works | New MissionAgent/WorldAgent generalize it |
| Suspect agent | `internal/agent/suspect` | KEEP (legacy) | Pattern reused | New DialogueAgent generalizes to any NPC |
| Evidence agent | `internal/agent/evidence` | KEEP (legacy) | — | New ClueAgent handles generation/inspection/explanation |
| Timeline agent | `internal/agent/timeline` | KEEP (legacy) | — | Superseded by WorldAgent event schedule + TimeAgent |
| Map agent | `internal/agent/map` | KEEP (legacy) | — | New MissionMapAgent adds risk/status/visual prompts/actions |
| Psychology agent | `internal/agent/psychology` | KEEP | Detective-type flavor | None |
| Director agent | `internal/agent/director` | KEEP (legacy) | — | New MissionDirectorAgent for time/world reactions |
| Judge agent | `internal/agent/judge` | KEEP (legacy) | — | New MissionJudgeAgent for objective-based evaluation |
| Case | `internal/case` | RENAME → Mission | Case is one mission type | New `internal/mission`; legacy routes stay temporarily; data converted |
| CaseBible | `internal/casebible` | RENAME → WorldBible | Same private-truth invariant | New `internal/worldbible`; data converted |
| Suspect | `internal/suspect` | RENAME → Character | Generalized NPC with avatar prompts | New `internal/character`; data converted |
| Evidence | `internal/evidence` | RENAME → Clue | + visual/thumbnail prompt fields | New `internal/clue`; data converted |
| Location | `internal/location` | RENAME → Map | Google-Maps markers, risk, actions | New `internal/gamemap`; data converted |
| Conversation | `internal/conversation` | RENAME → Interaction | Generalized (NPC chat + guidance) | New `internal/interaction`; data converted |
| Timeline | `internal/timeline` | RENAME → MissionEvent/TimeEngine | — | New `internal/timeengine`; history generalized |
| Notes | `internal/notes` | RENAME → Journal | Mission-scoped | New `internal/journal`; data converted |
| Detective | `internal/detective` | RENAME → PlayerProfile | + XP/level/wallet stats | New `internal/playerprofile`; data converted |
| Solve | `internal/solve` | RENAME → MissionOutcome | Objective/judgment based | Mission `complete` endpoint via MissionJudgeAgent |
| History/Notification | `internal/history`, `internal/notification` | EXTEND | Case-keyed events | Reused as mission event recorder (same tables pattern, mission_events) |
| Memory (facts + qdrant) | `internal/memory` | KEEP | Useful for discovered facts | Reused by mission services |
| Privacy | `internal/privacy` | EXTEND | Missing new truth keys | Add hidden_state, character_secrets, clue_truth, map_truth, timeline_truth, failure_rules, private_state, system_prompt, hidden_antagonist |
| Wallet | — | **ADD** | Mandatory | `internal/wallet` (+ guard, pricing, usage logs) |
| Guidance | — | **ADD** | Mandatory | `internal/guidance` |
| Time engine | — | **ADD** | Mandatory | `internal/timeengine` |
| REMOVE | — | none | No dead modules found | — |

## Notable Strengths To Preserve

- Handlers contain zero business logic; services own rules; repositories own SQL.
- Agents are declarative (Manifest/Prompt/Parse); the runtime owns retries with repair prompts and audit logging (`agent_runs`).
- The mock provider selects canned schema-valid JSON via a `TASK_TYPE:` marker in the prompt — the full loop works offline.
- Truth Layer discipline: internal entities carry no JSON tags for secrets; `Public()` DTO conversion; response-scanning privacy middleware.
- Ownership enforced via `GetForUser`/`EnsureOwned` gateway interfaces.
