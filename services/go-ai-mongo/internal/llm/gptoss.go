// Package llm wraps the GPT-OSS-120b model served over the Hugging Face Router
// (OpenAI-compatible /chat/completions). It mirrors the notebook's gpt client:
// robust JSON extraction for the M1-M5 structured steps and free text for M6.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"orsa.ai/go-ai-mongo/internal/modelpool"
)

type Client struct {
	pool *modelpool.Pool
}

// New wraps the shared model pool for text (M1-M6) calls. The pool is built once
// in main and shared with the vision client so both rotate over the same
// provider slots.
func New(pool *modelpool.Pool) *Client {
	return &Client{pool: pool}
}

// Available reports whether at least one text-capable provider is configured.
// When false the pipeline uses the same safe fallbacks the notebook used when
// the client was nil.
func (c *Client) Available() bool { return c != nil && c.pool.Available(modelpool.Text) }

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
	build := func(ep modelpool.Endpoint) ([]byte, error) {
		return json.Marshal(chatRequest{
			Model:       ep.Model,
			Temperature: temperature,
			MaxTokens:   maxTokens,
			Messages: []chatMessage{
				{Role: "system", Content: system},
				{Role: "user", Content: user},
			},
		})
	}
	parse := func(raw []byte) (string, error) {
		var parsed chatResponse
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return "", err
		}
		if len(parsed.Choices) == 0 {
			return "", fmt.Errorf("model returned no choices")
		}
		return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
	}
	return c.pool.Do(ctx, modelpool.Text, build, parse)
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
