package modelpool

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newStub returns a server that records hits and replies with the given content.
func newStub(t *testing.T, content string, status int) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(status)
		_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, content)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func textBuild(ep Endpoint) ([]byte, error) { return []byte(`{}`), nil }
func passthroughParse(raw []byte) (string, error) {
	return string(raw), nil
}

func TestRoundRobinRotatesAcrossProviders(t *testing.T) {
	a, aHits := newStub(t, "a", http.StatusOK)
	b, bHits := newStub(t, "b", http.StatusOK)

	pool := NewPool(http.DefaultClient,
		Provider{Name: "a", Text: Endpoint{BaseURL: a.URL, Model: "m", Token: "t"}},
		Provider{Name: "b", Text: Endpoint{BaseURL: b.URL, Model: "m", Token: "t"}},
	)

	// Four successful calls should split 2/2 across the two providers.
	for i := 0; i < 4; i++ {
		if _, err := pool.Do(context.Background(), Text, textBuild, passthroughParse); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if *aHits != 2 || *bHits != 2 {
		t.Fatalf("expected 2/2 distribution, got a=%d b=%d", *aHits, *bHits)
	}
}

func TestFailoverToNextProvider(t *testing.T) {
	bad, badHits := newStub(t, "bad", http.StatusInternalServerError)
	good, goodHits := newStub(t, "good", http.StatusOK)

	pool := NewPool(http.DefaultClient,
		Provider{Name: "bad", Text: Endpoint{BaseURL: bad.URL, Model: "m", Token: "t"}},
		Provider{Name: "good", Text: Endpoint{BaseURL: good.URL, Model: "m", Token: "t"}},
	)

	// First pick lands on "bad" (idx 0), which 500s, then fails over to "good".
	out, err := pool.Do(context.Background(), Text, textBuild, passthroughParse)
	if err != nil {
		t.Fatalf("expected failover success, got %v", err)
	}
	if *badHits != 1 || *goodHits != 1 {
		t.Fatalf("expected both tried once, got bad=%d good=%d", *badHits, *goodHits)
	}
	if want := `"content":"good"`; !contains(out, want) {
		t.Fatalf("expected good response, got %q", out)
	}
}

func TestVisionSkipsTextOnlyProvider(t *testing.T) {
	vis, visHits := newStub(t, "vision", http.StatusOK)

	// One text-only provider (no Vision endpoint) and one vision provider.
	pool := NewPool(http.DefaultClient,
		Provider{Name: "text-only", Text: Endpoint{BaseURL: "http://127.0.0.1:1", Model: "m", Token: "t"}},
		Provider{Name: "vision", Vision: Endpoint{BaseURL: vis.URL, Model: "m", Token: "t"}},
	)

	if !pool.Available(Vision) {
		t.Fatal("expected a vision-capable provider")
	}
	for i := 0; i < 3; i++ {
		if _, err := pool.Do(context.Background(), Vision, textBuild, passthroughParse); err != nil {
			t.Fatalf("vision call %d: %v", i, err)
		}
	}
	if *visHits != 3 {
		t.Fatalf("expected all 3 vision calls on the vision provider, got %d", *visHits)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
