package store

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type RedisClient interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
}

type WarmStore struct {
	client RedisClient
	ttl    time.Duration
	mu     sync.RWMutex
}

type mockRedisClient struct {
	data map[string][]byte
	ttl  map[string]time.Time
	mu   sync.RWMutex
}

func NewMockRedisClient() RedisClient {
	return &mockRedisClient{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Time),
	}
}

func (m *mockRedisClient) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if exp, ok := m.ttl[key]; ok && time.Now().After(exp) {
		return nil, nil
	}
	return m.data[key], nil
}

func (m *mockRedisClient) Set(key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	if ttl > 0 {
		m.ttl[key] = time.Now().Add(ttl)
	}
	return nil
}

func (m *mockRedisClient) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	delete(m.ttl, key)
	return nil
}

func (m *mockRedisClient) Exists(key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if exp, ok := m.ttl[key]; ok && time.Now().After(exp) {
		return false, nil
	}
	_, ok := m.data[key]
	return ok, nil
}

func NewWarmStore(ttl time.Duration) *WarmStore {
	return &WarmStore{
		client: NewMockRedisClient(),
		ttl:    ttl,
	}
}

func (s *WarmStore) SetClient(client RedisClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = client
}

func (s *WarmStore) Get(key string) (*api.ContextEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.client.Get(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var entry api.ContextEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *WarmStore) Set(entry *api.ContextEntry) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return s.client.Set(entry.Key, data, s.ttl)
}

func (s *WarmStore) Delete(key string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client.Delete(key)
}

func (s *WarmStore) Exists(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client.Exists(key)
}

func (s *WarmStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, ok := s.client.(*mockRedisClient); ok {
		client.mu.Lock()
		client.data = make(map[string][]byte)
		client.ttl = make(map[string]time.Time)
		client.mu.Unlock()
	}
}
