// CaseMind API server — composition root.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"casemind/docs"
	"casemind/internal/agent/casegenerator"
	"casemind/internal/agent/characteragent"
	"casemind/internal/agent/clueagent"
	"casemind/internal/agent/dialogueagent"
	directoragent "casemind/internal/agent/director"
	evidenceagent "casemind/internal/agent/evidence"
	"casemind/internal/agent/guidanceagent"
	judgeagent "casemind/internal/agent/judge"
	"casemind/internal/agent/locationagent"
	mapagent "casemind/internal/agent/map"
	"casemind/internal/agent/missionagent"
	"casemind/internal/agent/missiondirector"
	"casemind/internal/agent/missionjudge"
	"casemind/internal/agent/missionmap"
	"casemind/internal/agent/orchestrator"
	psychologyagent "casemind/internal/agent/psychology"
	"casemind/internal/agent/runtime"
	suspectagent "casemind/internal/agent/suspect"
	"casemind/internal/agent/timeagent"
	timelineagent "casemind/internal/agent/timeline"
	"casemind/internal/agent/worldagent"
	"casemind/internal/auth"
	"casemind/internal/avatar"
	cases "casemind/internal/case"
	"casemind/internal/casebible"
	"casemind/internal/character"
	"casemind/internal/clue"
	"casemind/internal/config"
	"casemind/internal/conversation"
	"casemind/internal/detective"
	"casemind/internal/evidence"
	"casemind/internal/gamemap"
	"casemind/internal/guidance"
	"casemind/internal/history"
	"casemind/internal/interaction"
	"casemind/internal/journal"
	"casemind/internal/llm"
	llmmock "casemind/internal/llm/mock"
	"casemind/internal/llm/openaicompat"
	"casemind/internal/location"
	"casemind/internal/memory"
	"casemind/internal/mission"
	"casemind/internal/missioncomplete"
	"casemind/internal/missionevent"
	"casemind/internal/missiontimeline"
	"casemind/internal/notes"
	"casemind/internal/notification"
	"casemind/internal/platform/database"
	httpserver "casemind/internal/platform/http"
	"casemind/internal/platform/logger"
	platformredis "casemind/internal/platform/redis"
	"casemind/internal/platform/storage"
	"casemind/internal/platform/vector"
	"casemind/internal/playerprofile"
	"casemind/internal/solve"
	"casemind/internal/suspect"
	"casemind/internal/timeengine"
	timelinepkg "casemind/internal/timeline"
	"casemind/internal/wallet"
	"casemind/internal/worldbible"
	"casemind/migrations"
)

type dualProfileCreator struct {
	detective *detective.Service
	player    *playerprofile.Service
}

func (c dualProfileCreator) CreateProfile(ctx context.Context, userID uuid.UUID) error {
	if c.detective != nil {
		if err := c.detective.CreateProfile(ctx, userID); err != nil {
			return err
		}
	}
	if c.player != nil {
		return c.player.CreateProfile(ctx, userID)
	}
	return nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}
	log := logger.New(cfg.Env)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Platform.
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		n, err := database.Migrate(ctx, pool, migrations.FS, log)
		if err != nil {
			log.Error("migrations failed", "error", err)
			os.Exit(1)
		}
		log.Info("migrations up to date", "applied", n)
	}

	rdb, err := platformredis.Connect(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Error("redis connection failed", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	store, err := storage.New(ctx, cfg.MinIO)
	if err != nil {
		log.Warn("minio unavailable, attachments disabled", "error", err)
	} else if store != nil {
		log.Info("minio connected", "bucket", cfg.MinIO.Bucket)
	}

	vc := vector.New(cfg.QdrantURL)
	if vc != nil {
		if err := vc.Ping(ctx); err != nil {
			log.Warn("qdrant unavailable, semantic memory disabled", "error", err)
			vc = nil
		} else {
			log.Info("qdrant connected")
		}
	}

	// LLM runtime. Any non-mock provider uses the OpenAI-compatible client;
	// config.Load has already validated the provider settings.
	var llmClient llm.Client
	if cfg.LLM.IsMock() {
		llmClient = llmmock.New()
	} else {
		llmClient = openaicompat.New(cfg.LLM, log)
	}

	// Fail-fast startup check: verify the model actually answers before the
	// server accepts traffic. A silent fallback to canned data is worse than
	// a refused deploy.
	probeCtx, probeCancel := context.WithTimeout(ctx, 60*time.Second)
	if err := llmClient.Ping(probeCtx); err != nil {
		probeCancel()
		log.Error("llm startup check failed — refusing to start",
			"provider", llmClient.Name(), "error", err)
		os.Exit(1)
	}
	probeCancel()
	log.Info("llm provider ready", "provider", llmClient.Name())

	// Avatar generation: uses the image API when configured, otherwise the
	// built-in procedural generator (always available).
	var imageAPI llm.ImageGenerator
	if cfg.Image.Enabled() {
		imageAPI = openaicompat.NewImageGen(openaicompat.ImageGenConfig{
			Provider: cfg.Image.Provider,
			BaseURL:  cfg.Image.BaseURL,
			APIKey:   cfg.Image.APIKey,
			Model:    cfg.Image.Model,
			Timeout:  cfg.Image.Timeout,
		}, log)
		log.Info("image API configured", "provider", imageAPI.Name())
	}
	avatarService := avatar.NewService(imageAPI, log)

	// Agent runtime + orchestrator with all eight agents.
	rt := runtime.New(llmClient, runtime.NewPGRunRepository(pool), log)
	orch := orchestrator.New(rt)
	orch.Register(casegenerator.New())
	orch.Register(suspectagent.New())
	orch.Register(evidenceagent.New())
	orch.Register(timelineagent.New())
	orch.Register(mapagent.New())
	orch.Register(psychologyagent.New())
	orch.Register(directoragent.New())
	orch.Register(judgeagent.New())
	orch.Register(missionagent.New())
	orch.Register(worldagent.New())
	orch.Register(missionmap.New())
	orch.Register(characteragent.New())
	orch.Register(clueagent.New())
	orch.Register(dialogueagent.New())
	orch.Register(guidanceagent.New())
	orch.Register(locationagent.New())
	orch.Register(timeagent.New())
	orch.Register(missiondirector.New())
	orch.Register(missionjudge.New())

	// Repositories.
	userRepo := auth.NewPGUserRepository(pool)
	tokenRepo := auth.NewPGRefreshTokenRepository(pool)
	detectiveRepo := detective.NewPGRepository(pool)
	caseRepo := cases.NewPGRepository(pool)
	bibleRepo := casebible.NewPGRepository(pool)
	suspectRepo := suspect.NewPGRepository(pool)
	evidenceRepo := evidence.NewPGRepository(pool)
	locationRepo := location.NewPGRepository(pool)
	timelineRepo := timelinepkg.NewPGRepository(pool)
	convRepo := conversation.NewPGRepository(pool)
	notesRepo := notes.NewPGRepository(pool)
	factRepo := memory.NewPGFactRepository(pool)
	eventRepo := history.NewPGRepository(pool)
	solveRepo := solve.NewPGRepository(pool)
	missionRepo := mission.NewPGRepository(pool)
	worldBibleRepo := worldbible.NewPGRepository(pool)
	mapRepo := gamemap.NewPGRepository(pool)
	characterRepo := character.NewPGRepository(pool)
	clueRepo := clue.NewPGRepository(pool)
	interactionRepo := interaction.NewPGRepository(pool)
	missionEventRepo := missionevent.NewPGRepository(pool)
	journalRepo := journal.NewPGRepository(pool)
	walletRepo := wallet.NewPGRepository(pool)
	playerProfileRepo := playerprofile.NewPGRepository(pool)

	// Cross-cutting services.
	bus := notification.NewBus()
	recorder := history.NewRecorder(eventRepo, bus, log)
	missionRecorder := missionevent.NewRecorder(missionEventRepo, bus, log)
	memoryService := memory.NewService(factRepo, vc, log)
	walletService := wallet.NewService(walletRepo)
	walletGuard := wallet.NewGuard(walletService, log)
	playerProfileService := playerprofile.NewService(playerProfileRepo, walletService)

	// Application services.
	detectiveService := detective.NewService(detectiveRepo)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTTL)
	authService := auth.NewService(userRepo, tokenRepo, dualProfileCreator{
		detective: detectiveService,
		player:    playerProfileService,
	}, tokens, cfg.RefreshTTL)

	generator := cases.NewGenerator(orch, caseRepo, bibleRepo, suspectRepo, evidenceRepo,
		locationRepo, timelineRepo, recorder, log)
	caseService := cases.NewService(caseRepo, suspectRepo, evidenceRepo, eventRepo,
		detectiveService, generator, log)

	suspectService := suspect.NewService(suspectRepo, caseService, convRepo, evidenceRepo,
		memoryService, orch, recorder, log)
	evidenceService := evidence.NewService(evidenceRepo, caseService, locationRepo, timelineRepo,
		memoryService, orch, recorder, log)
	timelineService := timelinepkg.NewService(timelineRepo, caseService)
	locationService := location.NewService(locationRepo, caseService)
	notesService := notes.NewService(notesRepo, caseService)
	solveService := solve.NewService(solveRepo, caseService, detectiveService, suspectRepo,
		bibleRepo, memoryService, orch, recorder, log)

	missionGenerator := mission.NewGenerator(orch, missionRepo, worldBibleRepo, mapRepo,
		characterRepo, clueRepo, missionRecorder, walletGuard, log)
	missionService := mission.NewService(missionRepo, characterRepo, clueRepo, mapRepo,
		interactionRepo, missionEventRepo, missionGenerator, walletGuard, playerProfileService, log)
	characterService := character.NewService(characterRepo, missionService, walletGuard, orch,
		interactionRepo, clueRepo, missionEventRepo, missionRecorder, playerProfileService, log)
	clueService := clue.NewService(clueRepo, missionService, walletGuard, orch,
		missionEventRepo, missionRecorder, playerProfileService, log)
	mapService := gamemap.NewService(mapRepo, missionService, walletGuard, orch,
		clueRepo, characterRepo, missionEventRepo, missionRecorder, playerProfileService, log)
	missionCompleteService := missioncomplete.NewService(missionService, clueRepo, mapRepo,
		interactionRepo, missionEventRepo, worldBibleRepo, orch, walletGuard, walletService,
		playerProfileService, missionRecorder, log)
	guidanceService := guidance.NewService(missionService, walletGuard, orch,
		interactionRepo, clueRepo, mapRepo, characterRepo, missionEventRepo, missionRecorder, playerProfileService, log)
	timeService := timeengine.NewService(missionService, walletGuard, orch,
		worldBibleRepo, clueRepo, mapRepo, interactionRepo, missionEventRepo, missionRecorder, log)
	journalService := journal.NewService(journalRepo, missionService)
	missionTimelineService := missiontimeline.NewService(missionEventRepo, missionService)

	// HTTP layer.
	handlers := httpserver.Handlers{
		Auth:            auth.NewHandler(authService),
		Avatar:          avatar.NewHandler(avatarService),
		Detective:       detective.NewHandler(detectiveService),
		Profile:         playerprofile.NewHandler(playerProfileService),
		Wallet:          wallet.NewHandler(walletService),
		Missions:        mission.NewHandler(missionService, bus),
		MissionComplete: missioncomplete.NewHandler(missionCompleteService),
		MissionTimeline: missiontimeline.NewHandler(missionTimelineService),
		Characters:      character.NewHandler(characterService),
		Clues:           clue.NewHandler(clueService),
		GameMap:         gamemap.NewHandler(mapService),
		Guidance:        guidance.NewHandler(guidanceService),
		Time:            timeengine.NewHandler(timeService),
		Journal:         journal.NewHandler(journalService),
		Cases:           cases.NewHandler(caseService, bus),
		Suspects:        suspect.NewHandler(suspectService),
		Evidence:        evidence.NewHandler(evidenceService),
		Timeline:        timelinepkg.NewHandler(timelineService),
		Location:        location.NewHandler(locationService),
		Notes:           notes.NewHandler(notesService),
		Solve:           solve.NewHandler(solveService),
	}
	router := httpserver.NewRouter(cfg, log, pool, rdb, tokens, handlers, docs.OpenAPISpec)

	if err := httpserver.Serve(log, cfg.Port, router); err != nil {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
}
