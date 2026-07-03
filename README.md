# AgentVerse Mission Engine

Run the API locally with Docker:

```bash
docker compose up --build
```

The default compose stack starts:

- API on `http://localhost:8080`
- PostgreSQL
- Redis
- the deterministic mock LLM provider by default

Useful checks:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

Swagger UI:

```text
http://localhost:8080/swagger
```

Postman collection with auth, profile, wallet, mission, map, NPC chat,
clue, guidance, time, journal, and negative scenarios:

```text
docs/casemind.postman_collection.json
```

Core AgentVerse endpoints are under `/api/v1`:

```text
POST /api/v1/auth/register
POST /api/v1/missions
GET  /api/v1/missions/{missionID}/map
POST /api/v1/missions/{missionID}/guidance
POST /api/v1/missions/{missionID}/characters/{characterID}/chat
POST /api/v1/missions/{missionID}/clues/{clueID}/explain
POST /api/v1/missions/{missionID}/time/advance
GET  /api/v1/wallet
GET  /api/v1/profile
```

Optional services such as MinIO and Qdrant are kept behind the `extras`
profile:

```bash
MINIO_ENDPOINT=minio:9000 QDRANT_URL=http://qdrant:6333 docker compose --profile extras up --build
```

For a real GLM/OpenAI-compatible LLM, set these values in `.env`:

```env
LLM_PROVIDER=glm
LLM_BASE_URL=https://your-openai-compatible-endpoint/v1
LLM_API_KEY=your-key
```

Legacy CaseMind `/api/v1/cases` routes are still mounted for compatibility,
but new clients should use `/api/v1/missions`.
