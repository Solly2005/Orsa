package httpapi

import (
	"sync"
	"time"
)

// rateLimiter is a per-key fixed-window limiter guarding the expensive chat
// endpoint, so a single authenticated account cannot spam the paid LLM backend.
// In-memory and per-process, matching dailyQuota's scope.
type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	windows map[string]*rlWindow
}

type rlWindow struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, windows: map[string]*rlWindow{}}
}

// allow records a request for key and reports whether it is within the limit.
func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	w, ok := r.windows[key]
	if !ok || now.After(w.reset) {
		r.windows[key] = &rlWindow{count: 1, reset: now.Add(r.window)}
		return true
	}
	if w.count >= r.limit {
		return false
	}
	w.count++
	return true
}
