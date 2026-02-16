package store

import (
	"fmt"
	"sync"
	"time"
)

type Entry struct {
	Value     []byte
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Store struct {
	mu      sync.RWMutex
	data    map[string]*Entry
	version int64

	// subscribers for watch
	watchers   map[string][]chan WatchEvent
	watchersMu sync.RWMutex
}

type WatchEvent struct {
	Key       string
	Value     []byte
	EventType string // "PUT" or "DELETE"
	Version   int64
}

func New() *Store {
	return &Store{
		data:     make(map[string]*Entry),
		watchers: make(map[string][]chan WatchEvent),
	}
}

func (s *Store) Get(key string) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok {
		return nil, false
	}
	// return a copy
	cp := *e
	return &cp, true
}

func (s *Store) Put(key string, value []byte) int64 {
	s.mu.Lock()
	s.version++
	v := s.version
	now := time.Now()

	existing, ok := s.data[key]
	if ok {
		existing.Value = value
		existing.Version = v
		existing.UpdatedAt = now
	} else {
		s.data[key] = &Entry{
			Value:     value,
			Version:   v,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	s.mu.Unlock()

	s.notify(WatchEvent{Key: key, Value: value, EventType: "PUT", Version: v})
	return v
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	_, ok := s.data[key]
	if ok {
		delete(s.data, key)
		s.version++
	}
	v := s.version
	s.mu.Unlock()

	if ok {
		s.notify(WatchEvent{Key: key, EventType: "DELETE", Version: v})
	}
	return ok
}

func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

func (s *Store) Subscribe(prefix string) chan WatchEvent {
	ch := make(chan WatchEvent, 64)
	s.watchersMu.Lock()
	s.watchers[prefix] = append(s.watchers[prefix], ch)
	s.watchersMu.Unlock()
	return ch
}

func (s *Store) notify(event WatchEvent) {
	s.watchersMu.RLock()
	defer s.watchersMu.RUnlock()
	for prefix, chs := range s.watchers {
		if len(prefix) == 0 || len(event.Key) >= len(prefix) && event.Key[:len(prefix)] == prefix {
			for _, ch := range chs {
				select {
				case ch <- event:
				default:
					// drop if subscriber is slow
				}
			}
		}
	}
}

func (s *Store) Stats() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fmt.Sprintf("keys=%d version=%d", len(s.data), s.version)
}
// unexport internal helper
