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
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"

	"orsa.ai/go-ai-mongo/internal/config"
)

const maxVisionPayloadBytes = 12 << 20
const maxExtractedPDFTextChars = 6000

// Client calls the GitHub Models vision endpoint.
type Client struct {
	baseURL string
	model   string
	token   string
	http    *http.Client
}

// New creates a vision client. If no GitHub token is configured, Available()
// returns false and Analyze returns a safe no-op summary.
func New(cfg config.Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.GitHubModelsBase, "/"),
		model:   cfg.VisionModelID,
		token:   cfg.GitHubToken,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Available reports whether a GitHub token is configured for vision calls.
func (c *Client) Available() bool { return c != nil && strings.TrimSpace(c.token) != "" }

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

	payload := map[string]any{
		"model": c.model,
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": visionPrompt,
					},
					map[string]any{
						"type":      "image_url",
						"image_url": map[string]any{"url": dataURI},
					},
				},
			},
		},
		"max_tokens":  900,
		"temperature": 0.1,
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
	req.Header.Set("User-Agent", "ORSA-Vision/1.0")

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
		return "", fmt.Errorf("vision API status %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

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
	summary := strings.TrimSpace(parsed.Choices[0].Message.Content)
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

const visionPrompt = `You are a clinical extraction assistant reviewing a patient-uploaded medical image or PDF report.

Extract the clinically relevant content for downstream GPT-OSS reasoning. Focus on:
- Report/document values, units, reference ranges, dates, and abnormal flags when visible
- Visible injuries, wounds, rashes, swelling, scans, or abnormal findings
- Any urgent findings that would require immediate attention

If the upload is blurry, cropped, unreadable, non-medical, or lacks extractable findings, say that plainly. Do not invent values or diagnoses. Respond in plain English. Do not mention AI, models, or internal system details.`
