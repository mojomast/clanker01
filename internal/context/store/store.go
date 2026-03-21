package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

// subscriber holds a pattern and callback for context event subscriptions.
type subscriber struct {
	pattern  string
	callback api.ContextCallback
}

type TieredContextStore struct {
	hot         *HotStore
	warm        *WarmStore
	cold        *ColdStore
	snapshot    *SnapshotManager
	subscribers []subscriber
	subMu       sync.RWMutex
	mu          sync.RWMutex
}

func NewTieredContextStore(hotMaxBytes int64, warmTTL time.Duration) *TieredContextStore {
	return &TieredContextStore{
		hot:      NewHotStore(hotMaxBytes),
		warm:     NewWarmStore(warmTTL),
		cold:     NewColdStore(),
		snapshot: NewSnapshotManager(),
	}
}

func (s *TieredContextStore) Get(ctx context.Context, key string, layer api.ContextLayer) (*api.ContextEntry, error) {
	entry, found := s.hot.Get(key)
	if found {
		return entry, nil
	}

	entry, err := s.warm.Get(key)
	if err == nil && entry != nil {
		s.hot.Set(entry)
		return entry, nil
	}

	entry, err = s.cold.Get(key)
	if err == nil && entry != nil {
		s.hot.Set(entry)
		s.warm.Set(entry)
		return entry, nil
	}

	return nil, fmt.Errorf("entry not found: %s", key)
}

func (s *TieredContextStore) Set(ctx context.Context, entry *api.ContextEntry) error {
	if entry.ID == "" {
		entry.ID = generateID()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	entry.UpdatedAt = time.Now()
	entry.AccessedAt = time.Now()

	if err := s.hot.Set(entry); err != nil {
		return err
	}

	if err := s.warm.Set(entry); err != nil {
		return fmt.Errorf("warm store error: %w", err)
	}

	if err := s.cold.Set(entry); err != nil {
		return fmt.Errorf("cold store error: %w", err)
	}

	s.notifySubscribers("set", entry)

	return nil
}

func (s *TieredContextStore) Delete(ctx context.Context, key string, layer api.ContextLayer) error {
	// Retrieve entry before deleting for notification
	entry, _ := s.hot.Get(key)

	s.hot.Delete(key)
	if err := s.warm.Delete(key); err != nil {
		return err
	}
	if err := s.cold.Delete(key); err != nil {
		return err
	}

	if entry != nil {
		s.notifySubscribers("delete", entry)
	}

	return nil
}

func (s *TieredContextStore) Query(ctx context.Context, q *api.ContextQuery) ([]*api.ContextEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	var results []*api.ContextEntry

	// Search hot store
	for _, entry := range s.hot.GetAll() {
		if s.matchQuery(entry, q) {
			seen[entry.Key] = true
			results = append(results, entry)
		}
	}

	// Also search warm store
	for _, entry := range s.warm.GetAll() {
		if !seen[entry.Key] && s.matchQuery(entry, q) {
			seen[entry.Key] = true
			results = append(results, entry)
		}
	}

	if q.Limit > 0 && len(results) > q.Limit {
		results = results[:q.Limit]
	}

	return results, nil
}

func (s *TieredContextStore) SemanticSearch(ctx context.Context, query string, opts *api.SearchOptions) ([]*api.ContextEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	var results []*api.ContextEntry

	// Collect entries from both hot and warm stores
	allEntries := s.hot.GetAll()
	for _, entry := range s.warm.GetAll() {
		if !seen[entry.Key] {
			seen[entry.Key] = true
			allEntries = append(allEntries, entry)
		}
	}

	// Use the first entry with an embedding as a proxy query embedding,
	// or search all entries if no MinScore filtering is needed.
	var queryEmbedding []float64
	for _, entry := range allEntries {
		if len(entry.Embedding) > 0 {
			queryEmbedding = entry.Embedding
			break
		}
	}

	for _, entry := range allEntries {
		if opts.Layer != "" && entry.Layer != opts.Layer {
			continue
		}
		if opts.MinScore > 0 {
			score := cosineSimilarity(queryEmbedding, entry.Embedding)
			if score < opts.MinScore {
				continue
			}
		}
		results = append(results, entry)
	}

	if opts.TopK > 0 && len(results) > opts.TopK {
		results = results[:opts.TopK]
	}

	return results, nil
}

func (s *TieredContextStore) Subscribe(pattern string, callback api.ContextCallback) (api.Unsubscribe, error) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	sub := subscriber{
		pattern:  pattern,
		callback: callback,
	}
	s.subscribers = append(s.subscribers, sub)

	// Return an unsubscribe function that removes this subscriber
	return func() error {
		s.subMu.Lock()
		defer s.subMu.Unlock()
		for i, existing := range s.subscribers {
			if &existing.callback == &sub.callback {
				s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
				return nil
			}
		}
		return nil
	}, nil
}

func (s *TieredContextStore) Export(ctx context.Context, layer api.ContextLayer) (*api.ContextSnapshot, error) {
	entries, err := s.Query(ctx, &api.ContextQuery{Layer: layer})
	if err != nil {
		return nil, err
	}

	snapshot := &api.ContextSnapshot{
		Layer:   layer,
		Entries: entries,
		TakenAt: time.Now(),
	}

	return s.snapshot.Save(snapshot), nil
}

func (s *TieredContextStore) matchQuery(entry *api.ContextEntry, q *api.ContextQuery) bool {
	if q.Layer != "" && entry.Layer != q.Layer {
		return false
	}
	if q.KeyPattern != "" && !matchPattern(entry.Key, q.KeyPattern) {
		return false
	}
	if !q.CreatedAfter.IsZero() && entry.CreatedAt.Before(q.CreatedAfter) {
		return false
	}
	return true
}

func (s *TieredContextStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hot.Clear()
	s.warm.Clear()
	s.cold.Clear()
	s.snapshot.Clear()
}

// generateID creates a unique ID using crypto/rand to avoid collision risks.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based if crypto/rand fails (should not happen)
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns 0.0 if either vector is empty or has zero magnitude.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0.0
	}

	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)

	if magA == 0 || magB == 0 {
		return 0.0
	}

	return dot / (magA * magB)
}

// matchPattern matches a key against a pattern using glob matching.
// Falls back to substring matching if the pattern contains no glob characters.
func matchPattern(key, pattern string) bool {
	// Try glob matching first (supports *, ?)
	if strings.ContainsAny(pattern, "*?[") {
		matched, err := path.Match(pattern, key)
		if err == nil {
			return matched
		}
	}
	// Fall back to substring matching
	return strings.Contains(key, pattern)
}

// notifySubscribers sends context events to all matching subscribers.
func (s *TieredContextStore) notifySubscribers(eventType string, entry *api.ContextEntry) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	for _, sub := range s.subscribers {
		if matchPattern(entry.Key, sub.pattern) {
			event := &api.ContextEvent{
				Type:  eventType,
				Key:   entry.Key,
				Layer: entry.Layer,
				Entry: entry,
			}
			sub.callback(event)
		}
	}
}
