package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PersistenceBackend string

const (
	PersistenceBackendMemory PersistenceBackend = "memory"
	PersistenceBackendFile   PersistenceBackend = "file"
	PersistenceBackendSQLite PersistenceBackend = "sqlite"
)

type SessionPersistence struct {
	backend      PersistenceBackend
	filePath     string
	mu           sync.RWMutex
	sessions     map[string]*Session
	syncInterval time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
}

// PersistedSession is the on-disk representation of a Session. It stores a
// pointer to the Session rather than a copy to avoid copying the sync.RWMutex
// embedded in Session (copying a mutex is forbidden by Go).
type PersistedSession struct {
	*Session
	Version    int64
	Checksum   string
	LastSyncAt time.Time
}

func NewSessionPersistence(backend PersistenceBackend, config map[string]interface{}) (*SessionPersistence, error) {
	ctx, cancel := context.WithCancel(context.Background())

	sp := &SessionPersistence{
		backend:      backend,
		sessions:     make(map[string]*Session),
		syncInterval: 60 * time.Second,
		ctx:          ctx,
		cancel:       cancel,
	}

	switch backend {
	case PersistenceBackendFile:
		if path, ok := config["path"].(string); ok {
			sp.filePath = path
			if err := os.MkdirAll(filepath.Dir(sp.filePath), 0755); err != nil {
				cancel()
				return nil, fmt.Errorf("failed to create persistence directory: %w", err)
			}
			if err := sp.loadFromFile(); err != nil {
				cancel()
				return nil, fmt.Errorf("failed to load sessions from file: %w", err)
			}
		} else {
			cancel()
			return nil, fmt.Errorf("file backend requires 'path' configuration")
		}
	case PersistenceBackendMemory:
	case PersistenceBackendSQLite:
		cancel()
		return nil, fmt.Errorf("SQLite backend not yet implemented")
	default:
		cancel()
		return nil, fmt.Errorf("unknown persistence backend: %s", backend)
	}

	go sp.syncLoop()

	return sp, nil
}

func (p *SessionPersistence) Create(session *Session) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.sessions[session.ID] = session

	if p.backend == PersistenceBackendFile {
		return p.saveToFile()
	}

	return nil
}

func (p *SessionPersistence) Get(id string) (*Session, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	session, ok := p.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	return session, nil
}

func (p *SessionPersistence) Update(session *Session) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.sessions[session.ID]; !ok {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	p.sessions[session.ID] = session

	if p.backend == PersistenceBackendFile {
		return p.saveToFile()
	}

	return nil
}

func (p *SessionPersistence) Delete(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.sessions[id]; !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	delete(p.sessions, id)

	if p.backend == PersistenceBackendFile {
		return p.saveToFile()
	}

	return nil
}

func (p *SessionPersistence) List(filter *SessionFilter) ([]*Session, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var sessions []*Session
	for _, session := range p.sessions {
		if p.matchesFilter(session, filter) {
			sessions = append(sessions, session)
		}
	}

	if filter != nil && filter.Limit > 0 && len(sessions) > filter.Limit {
		sessions = sessions[:filter.Limit]
	}

	return sessions, nil
}

func (p *SessionPersistence) matchesFilter(session *Session, filter *SessionFilter) bool {
	if filter == nil {
		return true
	}

	if filter.ProjectID != "" && session.ProjectID != filter.ProjectID {
		return false
	}

	if filter.Status != "" && session.Status != filter.Status {
		return false
	}

	if !filter.CreatedAfter.IsZero() && session.CreatedAt.Before(filter.CreatedAfter) {
		return false
	}

	if !filter.CreatedBefore.IsZero() && session.CreatedAt.After(filter.CreatedBefore) {
		return false
	}

	return true
}

func (p *SessionPersistence) loadFromFile() error {
	if p.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(p.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var persisted map[string]*PersistedSession
	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}

	for id, ps := range persisted {
		p.sessions[id] = ps.Session
	}

	return nil
}

func (p *SessionPersistence) saveToFile() error {
	if p.filePath == "" {
		return nil
	}

	persisted := make(map[string]*PersistedSession)
	for id, session := range p.sessions {
		persisted[id] = &PersistedSession{
			Session:    session,
			Version:    time.Now().Unix(),
			Checksum:   computeChecksum(session),
			LastSyncAt: time.Now(),
		}
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}

	tempPath := p.filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tempPath, p.filePath)
}

func (p *SessionPersistence) Export(id string) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	session, ok := p.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	return json.MarshalIndent(session, "", "  ")
}

func (p *SessionPersistence) Import(data []byte) (*Session, error) {
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.sessions[session.ID] = &session

	if p.backend == PersistenceBackendFile {
		if err := p.saveToFile(); err != nil {
			return nil, err
		}
	}

	return &session, nil
}

func (p *SessionPersistence) Backup(backupPath string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	persisted := make(map[string]*PersistedSession)
	for id, session := range p.sessions {
		persisted[id] = &PersistedSession{
			Session:    session,
			Version:    time.Now().Unix(),
			Checksum:   computeChecksum(session),
			LastSyncAt: time.Now(),
		}
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

func (p *SessionPersistence) Restore(backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	var persisted map[string]*PersistedSession
	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for id, ps := range persisted {
		p.sessions[id] = ps.Session
	}

	if p.backend == PersistenceBackendFile {
		return p.saveToFile()
	}

	return nil
}

func (p *SessionPersistence) Cleanup(olderThan time.Time) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	var toDelete []string
	for id, session := range p.sessions {
		if session.ClosedAt != nil && session.ClosedAt.Before(olderThan) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(p.sessions, id)
	}

	if len(toDelete) > 0 && p.backend == PersistenceBackendFile {
		_ = p.saveToFile()
	}

	return len(toDelete)
}

func (p *SessionPersistence) GetStats() *PersistenceStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := &PersistenceStats{
		TotalSessions:     len(p.sessions),
		ActiveSessions:    0,
		SuspendedSessions: 0,
		ClosedSessions:    0,
	}

	for _, session := range p.sessions {
		switch session.Status {
		case SessionActive, SessionResumed:
			stats.ActiveSessions++
		case SessionSuspended:
			stats.SuspendedSessions++
		case SessionClosed:
			stats.ClosedSessions++
		}
	}

	if p.filePath != "" {
		if info, err := os.Stat(p.filePath); err == nil {
			stats.DiskSizeBytes = info.Size()
		}
	}

	return stats
}

type PersistenceStats struct {
	TotalSessions     int
	ActiveSessions    int
	SuspendedSessions int
	ClosedSessions    int
	DiskSizeBytes     int64
	LastSyncAt        time.Time
}

func (p *SessionPersistence) Shutdown() {
	p.cancel()
	if p.backend == PersistenceBackendFile {
		p.mu.Lock()
		_ = p.saveToFile()
		p.mu.Unlock()
	}
}

func (p *SessionPersistence) syncLoop() {
	ticker := time.NewTicker(p.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mu.RLock()
			_ = p.saveToFile()
			p.mu.RUnlock()
		}
	}
}

func computeChecksum(session *Session) string {
	return fmt.Sprintf("%s:%d:%s", session.ID, session.UpdatedAt.Unix(), session.Status)
}

type SQLiteStore struct {
	dbPath string
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	return &SQLiteStore{dbPath: dbPath}, nil
}

func (s *SQLiteStore) Create(session *Session) error {
	return fmt.Errorf("sqlite backend not implemented")
}

func (s *SQLiteStore) Get(id string) (*Session, error) {
	return nil, fmt.Errorf("sqlite backend not implemented")
}

func (s *SQLiteStore) Update(session *Session) error {
	return fmt.Errorf("sqlite backend not implemented")
}

func (s *SQLiteStore) Delete(id string) error {
	return fmt.Errorf("sqlite backend not implemented")
}

func (s *SQLiteStore) List(filter *SessionFilter) ([]*Session, error) {
	return nil, fmt.Errorf("sqlite backend not implemented")
}
