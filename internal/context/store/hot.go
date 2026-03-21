package store

import (
	"container/list"
	"sync"

	"github.com/swarm-ai/swarm/pkg/api"
)

type HotStore struct {
	mu       sync.RWMutex
	entries  map[string]*list.Element
	lru      *list.List
	maxBytes int64
	current  int64
}

type lruEntry struct {
	key   string
	entry *api.ContextEntry
	size  int64
}

func NewHotStore(maxBytes int64) *HotStore {
	return &HotStore{
		entries:  make(map[string]*list.Element),
		lru:      list.New(),
		maxBytes: maxBytes,
	}
}

func (s *HotStore) Get(key string) (*api.ContextEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	elem, ok := s.entries[key]
	if !ok {
		return nil, false
	}

	s.lru.MoveToFront(elem)
	le := elem.Value.(*lruEntry)
	le.entry.AccessedAt = le.entry.AccessedAt.Add(0)
	return le.entry, true
}

func (s *HotStore) Set(entry *api.ContextEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	size := estimateSize(entry)

	if elem, ok := s.entries[entry.Key]; ok {
		le := elem.Value.(*lruEntry)
		s.current -= le.size
		s.lru.Remove(elem)
		delete(s.entries, entry.Key)
	}

	if s.current+size > s.maxBytes {
		s.evict(size)
	}

	le := &lruEntry{
		key:   entry.Key,
		entry: entry,
		size:  size,
	}
	elem := s.lru.PushFront(le)
	s.entries[entry.Key] = elem
	s.current += size

	return nil
}

func (s *HotStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elem, ok := s.entries[key]; ok {
		le := elem.Value.(*lruEntry)
		s.current -= le.size
		s.lru.Remove(elem)
		delete(s.entries, key)
	}
}

func (s *HotStore) GetAll() []*api.ContextEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*api.ContextEntry, 0, len(s.entries))
	for _, elem := range s.entries {
		le := elem.Value.(*lruEntry)
		result = append(result, le.entry)
	}
	return result
}

func (s *HotStore) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

func (s *HotStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

func (s *HotStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[string]*list.Element)
	s.lru = list.New()
	s.current = 0
}

func (s *HotStore) evict(needed int64) {
	for s.current+needed > s.maxBytes && s.lru.Len() > 0 {
		elem := s.lru.Back()
		if elem == nil {
			break
		}

		le := elem.Value.(*lruEntry)
		delete(s.entries, le.key)
		s.lru.Remove(elem)
		s.current -= le.size
	}
}

func estimateSize(entry *api.ContextEntry) int64 {
	size := int64(len(entry.ID) + len(entry.Key) + len(string(entry.Layer)))
	if entry.Content != nil {
		switch v := entry.Content.(type) {
		case string:
			size += int64(len(v))
		case []byte:
			size += int64(len(v))
		default:
			size += 100
		}
	}
	if entry.Embedding != nil {
		size += int64(len(entry.Embedding) * 8)
	}
	for k, v := range entry.Metadata {
		size += int64(len(k) + 50)
		_ = v
	}
	return size + 100
}
