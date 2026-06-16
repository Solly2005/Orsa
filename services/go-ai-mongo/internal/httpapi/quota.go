package httpapi

import (
	"context"
	"sync"
	"time"
)

// quotaUsed reports today's attachment count, preferring the durable per-user
// count from the C# engine (survives restarts, shared across instances) and
// falling back to the in-memory counter when the user service is unreachable.
func (s *Server) quotaUsed(ctx context.Context, userID string) int {
	if s.users != nil {
		if u, err := s.users.GetAttachmentUsage(ctx, userID); err == nil {
			return u.UsedToday
		}
	}
	return s.quota.used(userID)
}

// quotaConsume reserves n attachments within the limit, preferring the durable
// store and falling back to the in-memory counter. Returns whether the
// reservation was allowed and the resulting used count.
func (s *Server) quotaConsume(ctx context.Context, userID string, n, limit int) (bool, int) {
	if s.users != nil {
		if u, err := s.users.ConsumeAttachment(ctx, userID, n, limit); err == nil {
			return u.Allowed, u.UsedToday
		}
	}
	return s.quota.tryConsume(userID, n, limit)
}

// dailyQuota enforces a per-user, per-day attachment limit in the gateway.
//
// The previous design only tracked uploads client-side in localStorage, which a
// user could reset or bypass trivially. This keeps the authoritative count
// server-side. It is in-memory (per process): the count resets on restart and
// is not shared across replicas — acceptable for the current single-instance
// deployment, and the natural place to swap in a Redis/DB counter later.
type dailyQuota struct {
	mu     sync.Mutex
	day    string
	counts map[string]int
}

func newDailyQuota() *dailyQuota {
	return &dailyQuota{day: today(), counts: map[string]int{}}
}

func today() string { return time.Now().UTC().Format("2006-01-02") }

// rollover resets the counters when the UTC day changes. Caller holds the lock.
func (q *dailyQuota) rollover() {
	if d := today(); d != q.day {
		q.day = d
		q.counts = map[string]int{}
	}
}

// used reports how many attachments the user has consumed today.
func (q *dailyQuota) used(userID string) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.rollover()
	return q.counts[userID]
}

// tryConsume reserves n attachments for the user if doing so stays within limit.
// It returns whether the reservation succeeded and the resulting used count.
func (q *dailyQuota) tryConsume(userID string, n, limit int) (bool, int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.rollover()
	current := q.counts[userID]
	if n <= 0 {
		return true, current
	}
	if current+n > limit {
		return false, current
	}
	q.counts[userID] = current + n
	return true, current + n
}
