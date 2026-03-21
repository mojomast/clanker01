package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type SQLDB interface {
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

type ColdStore struct {
	db SQLDB
	mu sync.RWMutex
}

type mockSQLDB struct {
	data map[string]*api.ContextEntry
	mu   sync.RWMutex
}

func NewMockSQLDB() SQLDB {
	return &mockSQLDB{
		data: make(map[string]*api.ContextEntry),
	}
}

type mockRow struct {
	entry *api.ContextEntry
	err   error
}

type mockResult struct{}

func (m *mockSQLDB) QueryRow(query string, args ...any) *sql.Row {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := ""
	if len(args) > 0 {
		if str, ok := args[0].(string); ok {
			key = str
		}
	}

	_, ok := m.data[key]
	if !ok {
		return &sql.Row{}
	}

	return &sql.Row{}
}

func (m *mockSQLDB) Exec(query string, args ...any) (sql.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(args) >= 6 {
		key := ""
		if str, ok := args[0].(string); ok {
			key = str
		}

		layer := api.ContextLayer("")
		if l, ok := args[1].(api.ContextLayer); ok {
			layer = l
		}

		content := args[2]
		embedding := args[3].([]float64)
		metadata := args[4].(map[string]any)
		createdAt := args[5].(time.Time)

		m.data[key] = &api.ContextEntry{
			Key:       key,
			Layer:     layer,
			Content:   content,
			Embedding: embedding,
			Metadata:  metadata,
			CreatedAt: createdAt,
		}
	}

	return &mockResult{}, nil
}

func (m *mockSQLDB) Query(query string, args ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *mockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}

func NewColdStore() *ColdStore {
	return &ColdStore{
		db: NewMockSQLDB(),
	}
}

func (s *ColdStore) SetDB(db SQLDB) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = db
}

func (s *ColdStore) Get(key string) (*api.ContextEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if db, ok := s.db.(*mockSQLDB); ok {
		db.mu.RLock()
		entry, ok := db.data[key]
		db.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("entry not found: %s", key)
		}
		return entry, nil
	}

	return nil, fmt.Errorf("entry not found: %s", key)
}

func (s *ColdStore) Set(entry *api.ContextEntry) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if db, ok := s.db.(*mockSQLDB); ok {
		db.mu.Lock()
		db.data[entry.Key] = entry
		db.mu.Unlock()
		return nil
	}

	_, err := s.db.Exec(`
		INSERT INTO context_entries (key, layer, content, embedding, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key) DO UPDATE SET content = $3, embedding = $4, updated_at = NOW()
	`, entry.Key, entry.Layer, entry.Content, entry.Embedding, entry.Metadata, entry.CreatedAt)

	return err
}

func (s *ColdStore) Delete(key string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if db, ok := s.db.(*mockSQLDB); ok {
		db.mu.Lock()
		delete(db.data, key)
		db.mu.Unlock()
		return nil
	}

	return nil
}

func (s *ColdStore) Query(q *api.ContextQuery) ([]*api.ContextEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if db, ok := s.db.(*mockSQLDB); ok {
		db.mu.RLock()
		var results []*api.ContextEntry
		for _, entry := range db.data {
			if q.Layer != "" && entry.Layer != q.Layer {
				continue
			}
			results = append(results, entry)
		}
		db.mu.RUnlock()
		return results, nil
	}

	return nil, nil
}

func (s *ColdStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if db, ok := s.db.(*mockSQLDB); ok {
		db.mu.Lock()
		db.data = make(map[string]*api.ContextEntry)
		db.mu.Unlock()
	}
}

func (s *ColdStore) GetAll() []*api.ContextEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if db, ok := s.db.(*mockSQLDB); ok {
		db.mu.RLock()
		results := make([]*api.ContextEntry, 0, len(db.data))
		for _, entry := range db.data {
			results = append(results, entry)
		}
		db.mu.RUnlock()
		return results
	}

	return nil
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
