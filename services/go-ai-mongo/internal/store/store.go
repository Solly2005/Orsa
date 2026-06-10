// Package store persists chat threads (conversation metadata + dialogue state)
// for the triage service. A MongoDB-backed implementation is used in production;
// an in-memory implementation is the automatic fallback when Mongo is not
// configured or unreachable, so the website still works locally.
package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"orsa.ai/go-ai-mongo/internal/triage"
)

// Thread is a single conversation: metadata plus the notebook dialogue state.
type Thread struct {
	UserID    string       `bson:"user_id" json:"userId"`
	ThreadID  string       `bson:"thread_id" json:"threadId"`
	Title     string       `bson:"title" json:"title"`
	State     triage.State `bson:"state" json:"state"`
	Deleted   bool         `bson:"deleted" json:"deleted"`
	CreatedAt time.Time    `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time    `bson:"updated_at" json:"updatedAt"`
}

// ConversationSummary is the lightweight list item shown in the sidebar.
type ConversationSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updatedAt"`
	Deleted   bool      `json:"deleted"`
}

// Store is the persistence contract used by the HTTP handlers.
type Store interface {
	ListConversations(ctx context.Context, userID string) ([]ConversationSummary, error)
	GetThread(ctx context.Context, userID, threadID string) (*Thread, error)
	SaveThread(ctx context.Context, thread *Thread) error
	SetDeleted(ctx context.Context, userID, threadID string, deleted bool) error
	ChangedThreads(ctx context.Context, userID string, since time.Time) ([]Thread, error)
	Close(ctx context.Context) error
}

// ---- in-memory fallback ----

type MemoryStore struct {
	mu      sync.RWMutex
	threads map[string]*Thread // key: userID + "\x00" + threadID
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{threads: map[string]*Thread{}}
}

func key(userID, threadID string) string { return userID + "\x00" + threadID }

func (m *MemoryStore) ListConversations(_ context.Context, userID string) ([]ConversationSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []ConversationSummary
	for _, t := range m.threads {
		if t.UserID != userID || t.Deleted {
			continue
		}
		out = append(out, ConversationSummary{ID: t.ThreadID, Title: t.Title, UpdatedAt: t.UpdatedAt, Deleted: t.Deleted})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (m *MemoryStore) GetThread(_ context.Context, userID, threadID string) (*Thread, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t, ok := m.threads[key(userID, threadID)]; ok {
		clone := *t
		return &clone, nil
	}
	return nil, nil
}

func (m *MemoryStore) SaveThread(_ context.Context, thread *Thread) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := *thread
	m.threads[key(thread.UserID, thread.ThreadID)] = &clone
	return nil
}

func (m *MemoryStore) SetDeleted(_ context.Context, userID, threadID string, deleted bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.threads[key(userID, threadID)]; ok {
		t.Deleted = deleted
		t.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (m *MemoryStore) ChangedThreads(_ context.Context, userID string, since time.Time) ([]Thread, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Thread
	for _, t := range m.threads {
		if t.UserID == userID && t.UpdatedAt.After(since) {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (m *MemoryStore) Close(context.Context) error { return nil }
