package vision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"orsa.ai/go-ai-mongo/internal/modelpool"
)

func TestAnalyzeSendsPDFPayloadToVisionEndpoint(t *testing.T) {
	var requestPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": "CBC report values extracted."},
			}},
		})
	}))
	defer server.Close()

	client := &Client{pool: modelpool.NewPool(server.Client(),
		modelpool.Provider{Name: "test", Vision: modelpool.Endpoint{BaseURL: server.URL, Model: "meta/Llama-3.2-90B-Vision-Instruct", Token: "test-token"}},
	)}

	summary, err := client.Analyze(context.Background(), []byte("%PDF-1.7"), "application/pdf", "labs.pdf")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if summary != "CBC report values extracted." {
		t.Fatalf("unexpected summary: %q", summary)
	}

	messages := requestPayload["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	media := content[1].(map[string]any)["image_url"].(map[string]any)["url"].(string)
	if !strings.HasPrefix(media, "data:application/pdf;base64,") {
		t.Fatalf("expected PDF data URI, got %q", media)
	}
}

func TestAnalyzeUnsupportedMIMEReturnsUnreadableSummary(t *testing.T) {
	client := &Client{pool: modelpool.NewPool(http.DefaultClient,
		modelpool.Provider{Name: "test", Vision: modelpool.Endpoint{BaseURL: "http://127.0.0.1:1", Model: "meta/Llama-3.2-90B-Vision-Instruct", Token: "test-token"}},
	)}

	summary, err := client.Analyze(context.Background(), []byte("id,value"), "text/csv", "labs.csv")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if !strings.Contains(summary, "not supported for vision extraction") {
		t.Fatalf("expected unsupported summary, got %q", summary)
	}
}
