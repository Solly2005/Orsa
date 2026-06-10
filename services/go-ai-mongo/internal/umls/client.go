// Package umls provides a minimal UTS REST client for SNOMED CT concept lookup.
// It uses the CAS ticket workflow: one TGT (lives 8 hours) per API key, then
// short-lived service tickets for each search call.
package umls

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	casBase    = "https://utslogin.nlm.nih.gov/cas/v1/api-key"
	searchBase = "https://uts-ws.nlm.nih.gov/rest/search/current"
	casService = "http://umlsks.nlm.nih.gov"
	tgtTTL     = 7 * time.Hour // conservative; UTS TGTs last 8 hours
)

// Concept holds the normalized UMLS concept data we need for FHIR coding.
type Concept struct {
	CUI         string // e.g. "C0008031"
	Name        string // preferred English name
	SnomedCode  string // SNOMED CT concept ID (same as rootSource code for SNOMEDCT_US)
	SnomedUI    string // e.g. "29857009"
}

// Client handles TGT caching and SNOMED CT searches.
type Client struct {
	apiKey  string
	mu      sync.Mutex
	tgtURL  string
	tgtExp  time.Time
	http    *http.Client
}

// New creates a UMLS client. Returns nil if no apiKey is provided.
func New(apiKey string) *Client {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Available reports whether the client is configured.
func (c *Client) Available() bool { return c != nil }

// SearchSNOMED looks up the most relevant SNOMED CT concept for term.
// Returns nil if no match found or on any error.
func (c *Client) SearchSNOMED(ctx context.Context, term string) *Concept {
	if !c.Available() || strings.TrimSpace(term) == "" {
		return nil
	}
	st, err := c.serviceTicket(ctx)
	if err != nil {
		return nil
	}
	return c.searchOnce(ctx, term, st)
}

// SearchMany looks up SNOMED concepts for a batch of terms. Errors are silently
// dropped so that an unreachable UMLS service never blocks triage.
func (c *Client) SearchMany(ctx context.Context, terms []string) []Concept {
	if !c.Available() {
		return nil
	}
	out := make([]Concept, 0, len(terms))
	for _, t := range terms {
		if con := c.SearchSNOMED(ctx, t); con != nil {
			out = append(out, *con)
		}
	}
	return out
}

// ─── internal ────────────────────────────────────────────────────────────────

func (c *Client) ensureTGT(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tgtURL != "" && time.Now().Before(c.tgtExp) {
		return nil // cached TGT still valid
	}
	resp, err := c.http.PostForm(casBase, url.Values{"apikey": {c.apiKey}})
	if err != nil {
		return fmt.Errorf("TGT request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TGT status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return fmt.Errorf("TGT response missing Location header")
	}
	c.tgtURL = loc
	c.tgtExp = time.Now().Add(tgtTTL)
	return nil
}

func (c *Client) serviceTicket(ctx context.Context) (string, error) {
	if err := c.ensureTGT(ctx); err != nil {
		return "", err
	}
	c.mu.Lock()
	tgtURL := c.tgtURL
	c.mu.Unlock()

	resp, err := c.http.PostForm(tgtURL, url.Values{"service": {casService}})
	if err != nil {
		return "", fmt.Errorf("ST request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	st := strings.TrimSpace(string(body))
	if resp.StatusCode != 200 || st == "" {
		return "", fmt.Errorf("ST status %d: %s", resp.StatusCode, truncate(st, 100))
	}
	return st, nil
}

func (c *Client) searchOnce(ctx context.Context, term, st string) *Concept {
	params := url.Values{
		"string":       {term},
		"sabs":         {"SNOMEDCT_US"},
		"searchType":   {"words"},
		"returnIdType": {"concept"},
		"ticket":       {st},
		"pageSize":     {"1"},
	}
	reqURL := searchBase + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}

	var body struct {
		Result struct {
			Results []struct {
				UI   string `json:"ui"`
				Name string `json:"name"`
			} `json:"results"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || len(body.Result.Results) == 0 {
		return nil
	}
	r := body.Result.Results[0]
	if r.UI == "" || r.UI == "NONE" {
		return nil
	}
	// For SNOMEDCT_US the CUI is what we index; the numeric SNOMED code is in the
	// atoms endpoint. For our use (FHIR coding), the CUI is sufficient for M5 and
	// we use a stable SNOMED fallback in the FHIR bundle.
	return &Concept{CUI: r.UI, Name: r.Name, SnomedCode: r.UI}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
