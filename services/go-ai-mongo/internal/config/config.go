package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"orsa.ai/go-ai-mongo/internal/auth"
)

type Config struct {
	// HTTP REST server (external API consumed by the browser via the Angular proxy).
	HTTPPort string

	// Session-token verification (shared HS256 secret with the C# auth service)
	// and the browser origins allowed to call the REST API.
	SessionSecret      string
	CORSAllowedOrigins []string

	// MongoDB chat persistence.
	MongoAtlasURI string
	MongoDatabase string

	// Clinical knowledge + specialist model.
	UMLSAPIKey      string
	BertBaseModelID string
	BertModelDir    string
	BertOnnxDir     string
	OnnxRuntimeLib  string // path to onnxruntime.dll/.so; empty = look in PATH

	// Vision attachment ingestion (GitHub Models / Azure inference).
	GitHubModelsBase string
	VisionModelID    string
	GitHubToken      string

	// GPT-OSS conversational triage (Hugging Face Router, OpenAI-compatible).
	GptOssBaseURL string
	GptOssModelID string
	GptOssToken   string

	// C# Supabase engine (gRPC) for user settings/profile/consent.
	CsharpGrpcURL string
}

func Load() Config {
	loadDotEnv()
	return Config{
		HTTPPort:           getenv("HTTP_PORT", getenv("PORT", "3000")),
		SessionSecret:      resolveSessionSecret(),
		CORSAllowedOrigins: splitAndTrim(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:4200,http://127.0.0.1:4200")),
		MongoAtlasURI:      os.Getenv("MONGODB_ATLAS_URI"),
		MongoDatabase:      getenv("MONGODB_DATABASE", "orsa"),
		UMLSAPIKey:         os.Getenv("UMLS_API_KEY"),
		BertBaseModelID:    getenv("BERT_BASE_MODEL_ID", "emilyalsentzer/Bio_ClinicalBERT"),
		BertModelDir:       getenv("BERT_ESI_MODEL_DIR", "./models/bert_esi"),
		BertOnnxDir:        getenv("BERT_ESI_ONNX_DIR", "./models/bert_esi_onnx"),
		OnnxRuntimeLib:     firstNonEmpty(os.Getenv("ONNX_RUNTIME_LIB"), os.Getenv("ONNXRUNTIME_SHARED_LIBRARY_PATH")),
		GitHubModelsBase:   getenv("GITHUB_MODELS_BASE", "https://models.inference.ai.azure.com"),
		VisionModelID:      getenv("VISION_MODEL_ID", "meta/Llama-3.2-90B-Vision-Instruct"),
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		GptOssBaseURL:      firstNonEmpty(os.Getenv("GPT_OSS_ENDPOINT"), os.Getenv("HF_BASE"), "https://router.huggingface.co/v1"),
		GptOssModelID:      getenv("GPT_OSS_MODEL_ID", "openai/gpt-oss-120b"),
		GptOssToken:        firstNonEmpty(os.Getenv("GPT_OSS_AUTH_TOKEN"), os.Getenv("HF_TOKEN"), os.Getenv("HUGGING_FACE_API_KEY")),
		CsharpGrpcURL:      getenv("CSHARP_SUPABASE_GRPC_URL", "localhost:50053"),
	}
}

// resolveSessionSecret reads the shared session-token secret, falling back to a
// well-known insecure dev value (matching the C# service) so local development
// works without configuration. main.go warns when the fallback is in use.
func resolveSessionSecret() string {
	if v := firstNonEmpty(os.Getenv("ORSA_SESSION_SECRET"), os.Getenv("SESSION_SECRET")); v != "" {
		return v
	}
	return auth.DevSecret
}

// splitAndTrim splits a comma-separated list and drops empty/blank entries.
func splitAndTrim(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// loadDotEnv loads key=value pairs from the nearest .env file found by walking up
// from the working directory. Existing environment variables are never overwritten,
// so explicit process env always wins. Best-effort: missing files are ignored.
func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if file, err := os.Open(candidate); err == nil {
			applyDotEnv(file)
			file.Close()
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func applyDotEnv(file *os.File) {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		value = strings.Trim(value, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}
