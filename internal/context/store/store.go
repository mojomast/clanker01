package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type TieredContextStore struct {
	hot      *HotStore
	warm     *WarmStore
	cold     *ColdStore
	snapshot *SnapshotManager
	mu       sync.RWMutex
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

	return nil
}

func (s *TieredContextStore) Delete(ctx context.Context, key string, layer api.ContextLayer) error {
	s.hot.Delete(key)
	if err := s.warm.Delete(key); err != nil {
		return err
	}
	if err := s.cold.Delete(key); err != nil {
		return err
	}
	return nil
}

func (s *TieredContextStore) Query(ctx context.Context, q *api.ContextQuery) ([]*api.ContextEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*api.ContextEntry
	for _, entry := range s.hot.GetAll() {
		if s.matchQuery(entry, q) {
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

	var results []*api.ContextEntry
	for _, entry := range s.hot.GetAll() {
		if opts.Layer != "" && entry.Layer != opts.Layer {
			continue
		}
		if opts.MinScore > 0 {
			score := cosineSimilarity([]float64{}, []float64{})
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
	return func() error { return nil }, nil
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

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func cosineSimilarity(a, b []float64) float64 {
	return 1.0
}

func matchPattern(key, pattern string) bool {
	return key == pattern
}
