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
	"orsa.ai/go-ai-mongo/internal/modelpool"
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

	// Shared round-robin model pool. Each provider is one rotation slot:
	//  - "openai" pairs GPT-OSS (text) with Llama-3.2-Vision (vision compensator)
	//  - GPT-4.1, Llama-4-Maverick, Gemini are multimodal (same endpoint both ways)
	// Providers without a configured token are dropped, so a minimal config (only
	// GPT-OSS + GitHub vision) behaves exactly as before.
	modelPool := modelpool.NewPool(
		&http.Client{Timeout: 90 * time.Second},
		modelpool.Provider{
			Name:   "openai-gptoss+llama-vision",
			Text:   modelpool.Endpoint{BaseURL: cfg.GptOssBaseURL, Model: cfg.GptOssModelID, Token: cfg.GptOssToken},
			Vision: modelpool.Endpoint{BaseURL: cfg.GitHubModelsBase, Model: cfg.VisionModelID, Token: cfg.GitHubToken},
		},
		modelpool.Provider{
			Name:   "github-gpt-4.1",
			Text:   modelpool.Endpoint{BaseURL: cfg.GitHubModelsBase, Model: cfg.GitHubGptModelID, Token: cfg.GitHubToken},
			Vision: modelpool.Endpoint{BaseURL: cfg.GitHubModelsBase, Model: cfg.GitHubGptModelID, Token: cfg.GitHubToken},
		},
		modelpool.Provider{
			Name:   "github-llama-4-maverick",
			Text:   modelpool.Endpoint{BaseURL: cfg.GitHubModelsBase, Model: cfg.GitHubLlamaModelID, Token: cfg.GitHubToken},
			Vision: modelpool.Endpoint{BaseURL: cfg.GitHubModelsBase, Model: cfg.GitHubLlamaModelID, Token: cfg.GitHubToken},
		},
		modelpool.Provider{
			Name:   "gemini-2.5-flash",
			Text:   modelpool.Endpoint{BaseURL: cfg.GeminiBaseURL, Model: cfg.GeminiModelID, Token: cfg.GeminiToken},
			Vision: modelpool.Endpoint{BaseURL: cfg.GeminiBaseURL, Model: cfg.GeminiModelID, Token: cfg.GeminiToken},
		},
	)

	// GPT-OSS conversational engine + BERT specialist signal (nil => safe default).
	gpt := llm.New(modelPool)
	if !gpt.Available() {
		logger.Warn("no text model configured; triage replies will use safe fallbacks")
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

	vis := vision.New(modelPool)
	if vis.Available() {
		logger.Info("vision client ready")
	} else {
		logger.Warn("vision client unavailable; no vision-capable model configured")
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
