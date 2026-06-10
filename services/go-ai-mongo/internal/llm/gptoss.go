// Package llm wraps the GPT-OSS-120b model served over the Hugging Face Router
// (OpenAI-compatible /chat/completions). It mirrors the notebook's gpt client:
// robust JSON extraction for the M1-M5 structured steps and free text for M6.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"orsa.ai/go-ai-mongo/internal/config"
)

type Client struct {
	baseURL string
	model   string
	token   string
	http    *http.Client
}

func New(cfg config.Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.GptOssBaseURL, "/"),
		model:   cfg.GptOssModelID,
		token:   cfg.GptOssToken,
		http:    &http.Client{Timeout: 90 * time.Second},
	}
}

// Available reports whether a token is configured. When false the pipeline uses
// the same safe fallbacks the notebook used when the client was nil.
func (c *Client) Available() bool { return c != nil && strings.TrimSpace(c.token) != "" }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Messages    []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error any `json:"error"`
}

func (c *Client) call(ctx context.Context, system, user string, temperature float64, maxTokens int) (string, error) {
	payload := chatRequest{
		Model:       c.model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	// The HF router proxies gpt-oss to upstream providers (e.g. Cerebras) fronted
	// by Cloudflare, which 403s default SDK user-agents. Present a browser-like UA.
	req.Header.Set("User-Agent", "ORSA-Triage/1.0 (+https://router.huggingface.co)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gpt-oss call failed with status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("gpt-oss returned no choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// Text returns free-form model output (M6 patient synthesis).
func (c *Client) Text(ctx context.Context, system, user string, temperature float64, maxTokens int) string {
	if !c.Available() {
		return "[GPT-OSS unavailable]"
	}
	out, err := c.call(ctx, system, user, temperature, maxTokens)
	if err != nil {
		return fmt.Sprintf("[GPT-OSS error: %v]", err)
	}
	return out
}

// JSON returns parsed JSON output (M1-M5). On any failure it returns the fallback.
func (c *Client) JSON(ctx context.Context, system, user string, temperature float64, maxTokens int, fallback map[string]any) map[string]any {
	if !c.Available() {
		return fallback
	}
	out, err := c.call(ctx, system, user, temperature, maxTokens)
	if err != nil {
		return fallback
	}
	if parsed := extractJSON(out); parsed != nil {
		return parsed
	}
	return fallback
}

var fenceRE = regexp.MustCompile("(?m)^```(?:json)?\\s*|\\s*```$")

// extractJSON mirrors the notebook _extract_json: strip code fences, try a direct
// parse, else scan for the first balanced {...} object.
func extractJSON(text string) map[string]any {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	clean := strings.TrimSpace(fenceRE.ReplaceAllString(strings.TrimSpace(text), ""))

	var direct map[string]any
	if err := json.Unmarshal([]byte(clean), &direct); err == nil {
		return direct
	}

	start := strings.Index(clean, "{")
	if start == -1 {
		return nil
	}
	depth := 0
	for i := start; i < len(clean); i++ {
		switch clean[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				var obj map[string]any
				if err := json.Unmarshal([]byte(clean[start:i+1]), &obj); err == nil {
					return obj
				}
				return nil
			}
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
