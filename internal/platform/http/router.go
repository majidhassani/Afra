package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"casemind/internal/auth"
	"casemind/internal/avatar"
	cases "casemind/internal/case"
	"casemind/internal/character"
	"casemind/internal/clue"
	"casemind/internal/config"
	"casemind/internal/detective"
	"casemind/internal/evidence"
	"casemind/internal/gamemap"
	"casemind/internal/guidance"
	"casemind/internal/journal"
	"casemind/internal/location"
	"casemind/internal/mission"
	"casemind/internal/missioncomplete"
	"casemind/internal/missiontimeline"
	"casemind/internal/notes"
	"casemind/internal/playerprofile"
	"casemind/internal/solve"
	"casemind/internal/suspect"
	"casemind/internal/timeengine"
	"casemind/internal/timeline"
	"casemind/internal/wallet"
	"casemind/pkg/response"
)

// Handlers aggregates every module handler for routing.
type Handlers struct {
	Auth            *auth.Handler
	Avatar          *avatar.Handler
	Detective       *detective.Handler
	Profile         *playerprofile.Handler
	Wallet          *wallet.Handler
	Missions        *mission.Handler
	MissionComplete *missioncomplete.Handler
	MissionTimeline *missiontimeline.Handler
	Characters      *character.Handler
	Clues           *clue.Handler
	GameMap         *gamemap.Handler
	Guidance        *guidance.Handler
	Time            *timeengine.Handler
	Journal         *journal.Handler
	Cases           *cases.Handler
	Suspects        *suspect.Handler
	Evidence        *evidence.Handler
	Timeline        *timeline.Handler
	Location        *location.Handler
	Notes           *notes.Handler
	Solve           *solve.Handler
}

// NewRouter builds the full API router with middleware.
func NewRouter(
	cfg *config.Config,
	log *slog.Logger,
	pool *pgxpool.Pool,
	rdb *goredis.Client,
	tokens *auth.TokenManager,
	h Handlers,
	openAPISpec []byte,
) http.Handler {
	mux := http.NewServeMux()

	requireAuth := auth.RequireAuth(tokens)
	authLimit := RateLimit(rdb, log, cfg.RateLimit.AuthPerMinute, time.Minute, "auth")
	agentLimit := RateLimit(rdb, log, cfg.RateLimit.AgentPerMinute, time.Minute, "agent")

	public := func(handler http.HandlerFunc, mw ...Middleware) http.Handler {
		return Chain(handler, mw...)
	}
	protected := func(handler http.HandlerFunc, mw ...Middleware) http.Handler {
		return Chain(handler, append([]Middleware{requireAuth}, mw...)...)
	}

	// Health.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		checks := map[string]string{"postgres": "ok", "redis": "ok"}
		status := http.StatusOK
		if err := pool.Ping(ctx); err != nil {
			checks["postgres"] = "down"
			status = http.StatusServiceUnavailable
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			checks["redis"] = "down"
			status = http.StatusServiceUnavailable
		}
		response.JSON(w, status, checks)
	})

	// Swagger / OpenAPI.
	mux.HandleFunc("GET /swagger/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(openAPISpec)
	})
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerHTML))
	})

	// Auth.
	mux.Handle("POST /api/v1/auth/register", public(h.Auth.Register, authLimit))
	mux.Handle("POST /api/v1/auth/login", public(h.Auth.Login, authLimit))
	mux.Handle("POST /api/v1/auth/refresh", public(h.Auth.Refresh, authLimit))
	mux.Handle("POST /api/v1/auth/logout", public(h.Auth.Logout, authLimit))
	mux.Handle("GET /api/v1/me", protected(h.Auth.Me))

	// Avatars.
	mux.Handle("GET /api/v1/avatars/options", protected(h.Avatar.Options))
	mux.Handle("POST /api/v1/avatars/generate", protected(h.Avatar.Generate, agentLimit))

	// Detective.
	mux.Handle("GET /api/v1/detective/profile", protected(h.Detective.Profile))
	mux.Handle("GET /api/v1/detective/history", protected(h.Detective.History))

	// Player profile + wallet.
	mux.Handle("GET /api/v1/profile", protected(h.Profile.Profile))
	mux.Handle("PUT /api/v1/profile", protected(h.Profile.Update))
	mux.Handle("GET /api/v1/profile/stats", protected(h.Profile.Stats))
	mux.Handle("GET /api/v1/profile/history", protected(h.Profile.History))
	mux.Handle("GET /api/v1/profile/badges", protected(h.Profile.Badges))

	mux.Handle("GET /api/v1/wallet", protected(h.Wallet.Get))
	mux.Handle("GET /api/v1/wallet/transactions", protected(h.Wallet.Transactions))
	mux.Handle("GET /api/v1/wallet/pricing", protected(h.Wallet.Pricing))
	mux.Handle("GET /api/v1/wallet/config", protected(h.Wallet.Config))
	mux.Handle("POST /api/v1/wallet/rewarded-ad/claim", protected(h.Wallet.ClaimRewardedAd))
	mux.Handle("POST /api/v1/wallet/purchase/verify", protected(h.Wallet.VerifyPurchase))

	// AgentVerse missions.
	mux.Handle("POST /api/v1/missions", protected(h.Missions.Create, agentLimit))
	mux.Handle("GET /api/v1/missions", protected(h.Missions.List))
	mux.Handle("GET /api/v1/missions/{missionID}", protected(h.Missions.Get))
	mux.Handle("GET /api/v1/missions/{missionID}/dashboard", protected(h.Missions.Dashboard))
	mux.Handle("GET /api/v1/missions/{missionID}/completion-check", protected(h.MissionComplete.Check))
	mux.Handle("POST /api/v1/missions/{missionID}/complete", protected(h.MissionComplete.Complete, agentLimit))
	mux.Handle("POST /api/v1/missions/{missionID}/archive", protected(h.Missions.Archive))
	// /events is the raw chronological feed (dev/detail); /timeline is the
	// official curated player-facing story timeline.
	mux.Handle("GET /api/v1/missions/{missionID}/events", protected(h.Missions.Events))
	mux.Handle("GET /api/v1/missions/{missionID}/timeline", protected(h.MissionTimeline.Get))
	mux.Handle("GET /api/v1/missions/{missionID}/stream", protected(h.Missions.Stream))

	mux.Handle("GET /api/v1/missions/{missionID}/map", protected(h.GameMap.Map))
	mux.Handle("GET /api/v1/missions/{missionID}/locations/{locationID}", protected(h.GameMap.Detail))
	mux.Handle("POST /api/v1/missions/{missionID}/locations/{locationID}/actions", protected(h.GameMap.Action, agentLimit))
	mux.Handle("POST /api/v1/missions/{missionID}/locations/{locationID}/ask-ai", protected(h.Guidance.AskAtLocation, agentLimit))

	mux.Handle("GET /api/v1/missions/{missionID}/characters", protected(h.Characters.List))
	mux.Handle("GET /api/v1/missions/{missionID}/characters/{characterID}", protected(h.Characters.Get))
	mux.Handle("POST /api/v1/missions/{missionID}/characters/{characterID}/chat", protected(h.Characters.Chat, agentLimit))

	mux.Handle("GET /api/v1/missions/{missionID}/clues", protected(h.Clues.List))
	mux.Handle("GET /api/v1/missions/{missionID}/clues/{clueID}", protected(h.Clues.Get))
	mux.Handle("POST /api/v1/missions/{missionID}/clues/{clueID}/inspect", protected(h.Clues.Inspect, agentLimit))
	mux.Handle("POST /api/v1/missions/{missionID}/clues/{clueID}/explain", protected(h.Clues.Explain, agentLimit))

	mux.Handle("POST /api/v1/missions/{missionID}/guidance", protected(h.Guidance.Guide, agentLimit))
	mux.Handle("GET /api/v1/missions/{missionID}/time", protected(h.Time.Get))
	mux.Handle("POST /api/v1/missions/{missionID}/time/advance", protected(h.Time.Advance, agentLimit))

	mux.Handle("GET /api/v1/missions/{missionID}/journal", protected(h.Journal.List))
	mux.Handle("POST /api/v1/missions/{missionID}/journal", protected(h.Journal.Create))
	mux.Handle("PUT /api/v1/missions/{missionID}/journal/{noteID}", protected(h.Journal.Update))
	mux.Handle("DELETE /api/v1/missions/{missionID}/journal/{noteID}", protected(h.Journal.Delete))

	// Cases.
	mux.Handle("POST /api/v1/cases", protected(h.Cases.Create, agentLimit))
	mux.Handle("GET /api/v1/cases", protected(h.Cases.List))
	mux.Handle("GET /api/v1/cases/{caseID}", protected(h.Cases.Get))
	mux.Handle("POST /api/v1/cases/{caseID}/archive", protected(h.Cases.Archive))
	mux.Handle("GET /api/v1/cases/{caseID}/events", protected(h.Cases.Events))
	mux.Handle("GET /api/v1/cases/{caseID}/stream", protected(h.Cases.Stream))

	// Suspects.
	mux.Handle("GET /api/v1/cases/{caseID}/suspects", protected(h.Suspects.List))
	mux.Handle("GET /api/v1/cases/{caseID}/suspects/{suspectID}", protected(h.Suspects.Get))
	mux.Handle("POST /api/v1/cases/{caseID}/suspects/{suspectID}/interrogate", protected(h.Suspects.Interrogate, agentLimit))

	// Evidence.
	mux.Handle("GET /api/v1/cases/{caseID}/evidence", protected(h.Evidence.List))
	mux.Handle("GET /api/v1/cases/{caseID}/evidence/{evidenceID}", protected(h.Evidence.Get))
	mux.Handle("POST /api/v1/cases/{caseID}/evidence/{evidenceID}/inspect", protected(h.Evidence.Inspect, agentLimit))

	// Timeline.
	mux.Handle("GET /api/v1/cases/{caseID}/timeline", protected(h.Timeline.Get))
	mux.Handle("POST /api/v1/cases/{caseID}/timeline/player", protected(h.Timeline.CreatePlayerEvent))
	mux.Handle("PUT /api/v1/cases/{caseID}/timeline/player/{eventID}", protected(h.Timeline.UpdatePlayerEvent))
	mux.Handle("DELETE /api/v1/cases/{caseID}/timeline/player/{eventID}", protected(h.Timeline.DeletePlayerEvent))

	// Map.
	mux.Handle("GET /api/v1/cases/{caseID}/map", protected(h.Location.Map))

	// Notes.
	mux.Handle("GET /api/v1/cases/{caseID}/notes", protected(h.Notes.List))
	mux.Handle("POST /api/v1/cases/{caseID}/notes", protected(h.Notes.Create))
	mux.Handle("PUT /api/v1/cases/{caseID}/notes/{noteID}", protected(h.Notes.Update))
	mux.Handle("DELETE /api/v1/cases/{caseID}/notes/{noteID}", protected(h.Notes.Delete))

	// Solve.
	mux.Handle("POST /api/v1/cases/{caseID}/solve", protected(h.Solve.Solve, agentLimit))
	mux.Handle("GET /api/v1/cases/{caseID}/solve-attempts", protected(h.Solve.Attempts))

	return Chain(mux,
		Recover(log),
		RequestLogger(log),
		CORS(),
		PrivacyGuard(log),
	)
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <title>CaseMind API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
  window.onload = () => {
    SwaggerUIBundle({ url: "/swagger/openapi.yaml", dom_id: "#swagger-ui" });
  };
</script>
</body>
</html>`
