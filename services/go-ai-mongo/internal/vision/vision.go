// Package vision wraps the GitHub Models Llama-3.2-90B-Vision-Instruct endpoint
// (Azure inference, OpenAI-compatible) to generate clinical summaries from
// uploaded medical media for the M0 step.
package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"

	"orsa.ai/go-ai-mongo/internal/modelpool"
)

const maxVisionPayloadBytes = 12 << 20
const maxExtractedPDFTextChars = 6000

// Client round-robins attachment analysis across the vision-capable providers.
type Client struct {
	pool *modelpool.Pool
}

// New wraps the shared model pool for vision (M0) calls. The pool is built once
// in main and shared with the text client. For the GPT-OSS provider the pool
// routes vision to its paired Llama-3.2-Vision endpoint; the other providers
// (GPT-4.1, Gemini, Llama-4-Maverick) are multimodal. If no vision-capable
// provider is configured Available() is false and Analyze returns a safe no-op.
func New(pool *modelpool.Pool) *Client {
	return &Client{pool: pool}
}

// Available reports whether at least one vision-capable provider is configured.
func (c *Client) Available() bool { return c != nil && c.pool.Available(modelpool.Vision) }

// Analyze sends supported medical media bytes to the vision model and returns a
// brief clinical extraction summary. Browser uploads currently allow images and
// PDFs; unsupported media returns an explicit unreadable summary.
func (c *Client) Analyze(ctx context.Context, data []byte, mimeType, fileName string) (string, error) {
	if isPDFMedia(mimeType, fileName, data) {
		if text, err := extractPDFText(data); err == nil && strings.TrimSpace(text) != "" {
			return fmt.Sprintf("Readable text extracted from PDF %s:\n%s", fileName, text), nil
		}
	}
	if !c.Available() {
		return "[Vision model unavailable: no GITHUB_TOKEN configured]", nil
	}
	if !isVisionSupportedMIME(mimeType) {
		return fmt.Sprintf("Attached file: %s (%s) - this file type is not supported for vision extraction.", fileName, mimeType), nil
	}
	if len(data) > maxVisionPayloadBytes {
		return fmt.Sprintf("Attached file: %s (%s) - file is too large for vision extraction in this request.", fileName, mimeType), nil
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	dataURI := "data:" + mimeType + ";base64," + b64

	build := func(ep modelpool.Endpoint) ([]byte, error) {
		return json.Marshal(map[string]any{
			"model": ep.Model,
			"messages": []any{
				map[string]any{
					"role": "user",
					"content": []any{
						map[string]any{"type": "text", "text": visionPrompt},
						map[string]any{"type": "image_url", "image_url": map[string]any{"url": dataURI}},
					},
				},
			},
			"max_tokens":  900,
			"temperature": 0.1,
		})
	}
	parse := func(raw []byte) (string, error) {
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Error any `json:"error"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return "", err
		}
		if len(parsed.Choices) == 0 {
			return "", fmt.Errorf("vision model returned no choices")
		}
		return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
	}

	summary, err := c.pool.Do(ctx, modelpool.Vision, build, parse)
	if err != nil {
		return "", err
	}
	if summary == "" {
		return "[Vision model returned empty summary]", nil
	}
	return summary, nil
}

func isVisionSupportedMIME(mime string) bool {
	normalized := strings.ToLower(strings.TrimSpace(mime))
	return strings.HasPrefix(normalized, "image/") || normalized == "application/pdf"
}

func isPDFMedia(mimeType, fileName string, data []byte) bool {
	normalized := strings.ToLower(strings.TrimSpace(mimeType))
	if semi := strings.Index(normalized, ";"); semi >= 0 {
		normalized = strings.TrimSpace(normalized[:semi])
	}
	return normalized == "application/pdf" ||
		strings.EqualFold(filepath.Ext(fileName), ".pdf") ||
		(len(data) >= 4 && string(data[:4]) == "%PDF")
}

func extractPDFText(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	plain, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}
	raw, err := io.ReadAll(io.LimitReader(plain, maxExtractedPDFTextChars*2))
	if err != nil {
		return "", err
	}
	return compactExtractedText(string(raw), maxExtractedPDFTextChars), nil
}

func compactExtractedText(value string, limit int) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			kept = append(kept, line)
		}
	}
	text := strings.TrimSpace(strings.Join(kept, "\n"))
	if limit > 0 && len(text) > limit {
		return strings.TrimSpace(text[:limit]) + "\n[PDF text truncated]"
	}
	return text
}

const visionPrompt = `You are a clinical extraction assistant reviewing a patient-uploaded medical image or PDF report.

Extract the clinically relevant content for downstream GPT-OSS reasoning. Focus on:
- Report/document values, units, reference ranges, dates, and abnormal flags when visible
- Visible injuries, wounds, rashes, swelling, scans, or abnormal findings
- Any urgent findings that would require immediate attention

If the upload is blurry, cropped, unreadable, non-medical, or lacks extractable findings, say that plainly. Do not invent values or diagnoses. Respond in plain English. Do not mention AI, models, or internal system details.`
