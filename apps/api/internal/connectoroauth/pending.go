package connectoroauth

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PendingEntry is a one-time handoff from OAuth callback to an authenticated API consumer.
type PendingEntry struct {
	Principal uuid.UUID
	Provider  string
	Patch     json.RawMessage
	Expires   time.Time
}

// PendingStore stores short-lived OAuth results keyed by opaque sid.
type PendingStore struct {
	mu sync.Mutex
	m  map[string]*PendingEntry
}

func NewPendingStore() *PendingStore {
	return &PendingStore{m: make(map[string]*PendingEntry)}
}

func (s *PendingStore) Put(sid string, e *PendingEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[sid] = e
}

func (s *PendingStore) Take(sid string) (*PendingEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[sid]
	if !ok {
		return nil, false
	}
	delete(s.m, sid)
	if time.Now().After(e.Expires) {
		return nil, false
	}
	return e, true
}
