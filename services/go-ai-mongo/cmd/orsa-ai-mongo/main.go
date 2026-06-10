package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orsa.ai/go-ai-mongo/internal/auth"
	"orsa.ai/go-ai-mongo/internal/bert"
	"orsa.ai/go-ai-mongo/internal/config"
	"orsa.ai/go-ai-mongo/internal/httpapi"
	"orsa.ai/go-ai-mongo/internal/llm"
	"orsa.ai/go-ai-mongo/internal/store"
	"orsa.ai/go-ai-mongo/internal/triage"
	"orsa.ai/go-ai-mongo/internal/umls"
	"orsa.ai/go-ai-mongo/internal/userclient"
	"orsa.ai/go-ai-mongo/internal/vision"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Chat persistence: MongoDB Atlas, with an in-memory fallback so the website
	// keeps working when Mongo is unreachable.
	var chatStore store.Store = store.NewMemoryStore()
	if cfg.MongoAtlasURI != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		if mongoStore, err := store.NewMongoStore(ctx, cfg.MongoAtlasURI, cfg.MongoDatabase); err != nil {
			logger.Warn("mongo unavailable; using in-memory chat store", "error", err)
		} else {
			chatStore = mongoStore
			logger.Info("connected to MongoDB Atlas", "database", cfg.MongoDatabase)
		}
		cancel()
	}

	// GPT-OSS conversational engine + BERT specialist signal (nil => safe default).
	gpt := llm.New(cfg)
	if !gpt.Available() {
		logger.Warn("GPT-OSS token not configured; triage replies will use safe fallbacks")
	}
	var bertPredictor triage.BertPredictor // nil = escalate-only safe default
	if bp, err := bert.New(cfg); err != nil {
		logger.Warn("BERT-ESI ONNX unavailable; falling back to GPT-only reconciliation", "error", err)
	} else {
		bertPredictor = bp
		defer bp.Close()
		logger.Info("BERT-ESI ONNX loaded", "onnx_dir", cfg.BertOnnxDir)
	}

	umlsClient := umls.New(cfg.UMLSAPIKey)
	if umlsClient != nil {
		logger.Info("UMLS client ready")
	} else {
		logger.Warn("UMLS_API_KEY not configured; SNOMED concept coding disabled")
	}

	engine := triage.NewEngine(gpt, bertPredictor, umlsClient)

	// C# Supabase engine (gRPC) for user settings/profile/consent.
	var users httpapi.UserService
	if client, err := userclient.New(cfg.CsharpGrpcURL); err != nil {
		logger.Warn("could not init C# user client; settings/profile use fallbacks", "error", err)
	} else {
		users = client
		defer client.Close()
	}

	vis := vision.New(cfg)
	if vis.Available() {
		logger.Info("vision client ready", "model", cfg.VisionModelID)
	} else {
		logger.Warn("vision client unavailable; GITHUB_TOKEN not configured")
	}

	if cfg.SessionSecret == auth.DevSecret {
		logger.Warn("ORSA_SESSION_SECRET not set; using the insecure dev session secret. Set it before deploying.")
	}

	server := httpapi.NewServer(engine, chatStore, users, vis, cfg.SessionSecret, cfg.CORSAllowedOrigins)
	httpServer := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: server.Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("orsa go ai+mongo REST gateway listening",
			"port", cfg.HTTPPort, "gpt_model", cfg.GptOssModelID, "bert_onnx_dir", cfg.BertOnnxDir)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	_ = chatStore.Close(context.Background())
}
