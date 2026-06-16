// Package modelpool provides a small round-robin load balancer over a set of
// model "providers", each exposing OpenAI-compatible /chat/completions endpoints
// (HF Router, GitHub Models, Google Gemini's OpenAI-compatible surface).
//
// A provider is one logical slot in the rotation. Crucially, a provider can use
// *different* endpoints for text vs. vision: GPT-OSS cannot read images, so its
// provider pairs GPT-OSS (text) with Llama-3.2-Vision (vision) as a single unit.
// The other providers (GPT-4.1, Gemini, Llama-4-Maverick) are multimodal and use
// the same endpoint for both. Text and vision calls share one rotation counter,
// so load is balanced evenly across the providers that support the requested
// capability, and each call fails over to the next capable provider on error.
package modelpool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
)

// Kind selects which capability a call needs.
type Kind int

const (
	Text Kind = iota
	Vision
)

func (k Kind) String() string {
	if k == Vision {
		return "vision"
	}
	return "text"
}

// Endpoint is one OpenAI-compatible chat model. The zero value is "unconfigured"
// and is skipped.
type Endpoint struct {
	BaseURL string // base URL; the pool appends "/chat/completions"
	Model   string // model id sent in the request body
	Token   string // bearer token for this provider
}

func (e Endpoint) usable() bool {
	return strings.TrimSpace(e.Token) != "" &&
		strings.TrimSpace(e.BaseURL) != "" &&
		strings.TrimSpace(e.Model) != ""
}

// Provider is one rotation slot. Text and Vision may point at the same endpoint
// (multimodal models) or different endpoints (the GPT-OSS + Llama-Vision pair).
type Provider struct {
	Name   string
	Text   Endpoint
	Vision Endpoint
}

func (p Provider) endpointFor(kind Kind) (Endpoint, bool) {
	ep := p.Text
	if kind == Vision {
		ep = p.Vision
	}
	if !ep.usable() {
		return Endpoint{}, false
	}
	ep.BaseURL = strings.TrimRight(ep.BaseURL, "/")
	return ep, true
}

// Pool round-robins across its providers. Build one with NewPool.
type Pool struct {
	providers []Provider
	idx       atomic.Uint64
	http      *http.Client
}

// NewPool keeps every provider that has at least one usable endpoint, preserving
// order so the rotation is deterministic.
func NewPool(httpClient *http.Client, providers ...Provider) *Pool {
	usable := make([]Provider, 0, len(providers))
	for _, pr := range providers {
		if pr.Text.usable() || pr.Vision.usable() {
			usable = append(usable, pr)
		}
	}
	return &Pool{providers: usable, http: httpClient}
}

// Available reports whether at least one provider supports the given capability.
func (p *Pool) Available(kind Kind) bool {
	if p == nil {
		return false
	}
	for _, pr := range p.providers {
		if _, ok := pr.endpointFor(kind); ok {
			return true
		}
	}
	return false
}

// Do executes one chat call with round-robin selection and failover across the
// providers that support kind.
//
//	build — produces the JSON request body for the chosen endpoint (model id
//	        comes from the endpoint, so the body must embed ep.Model).
//	parse — extracts the assistant text from a 2xx response body.
func (p *Pool) Do(
	ctx context.Context,
	kind Kind,
	build func(ep Endpoint) ([]byte, error),
	parse func(raw []byte) (string, error),
) (string, error) {
	if p == nil || len(p.providers) == 0 {
		return "", fmt.Errorf("modelpool: no providers configured")
	}

	n := len(p.providers)
	start := int(p.idx.Add(1)-1) % n

	var lastErr error
	tried := 0
	for i := 0; i < n; i++ {
		pr := p.providers[(start+i)%n]
		ep, ok := pr.endpointFor(kind)
		if !ok {
			continue // provider can't serve this capability; skip
		}
		tried++
		out, err := p.callOne(ctx, ep, build, parse)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", pr.Name, err)
			slog.Debug("modelpool endpoint failed; trying next", "provider", pr.Name, "kind", kind.String(), "error", err)
			continue
		}
		slog.Debug("modelpool served call", "provider", pr.Name, "kind", kind.String(), "model", ep.Model)
		return out, nil
	}
	if tried == 0 {
		return "", fmt.Errorf("modelpool: no provider supports %s calls", kind.String())
	}
	return "", fmt.Errorf("modelpool: all %d %s providers failed: %w", tried, kind.String(), lastErr)
}

func (p *Pool) callOne(
	ctx context.Context,
	ep Endpoint,
	build func(ep Endpoint) ([]byte, error),
	parse func(raw []byte) (string, error),
) (string, error) {
	body, err := build(ep)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+ep.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// The HF router proxies models behind Cloudflare, which 403s default SDK
	// user-agents; a browser-like UA is also harmless for the other providers.
	req.Header.Set("User-Agent", "ORSA-Triage/1.0 (+https://router.huggingface.co)")

	resp, err := p.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	return parse(raw)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
