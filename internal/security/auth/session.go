package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired")
	ErrInvalidSessionID = errors.New("invalid session ID")
)

// SessionConfig holds session configuration
type SessionConfig struct {
	SessionTimeout     time.Duration `json:"session_timeout"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
	MaxSessionsPerUser int           `json:"max_sessions_per_user"`
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		SessionTimeout:     time.Hour * 24,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 5,
	}
}

// Session represents a user session
type Session struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	User       *User                  `json:"user"`
	CreatedAt  time.Time              `json:"created_at"`
	ExpiresAt  time.Time              `json:"expires_at"`
	LastAccess time.Time              `json:"last_access"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// MemorySessionManager implements in-memory session management
type MemorySessionManager struct {
	config       *SessionConfig
	sessions     map[string]*Session
	userSessions map[string]map[string]bool // userID -> sessionID set
	mu           sync.RWMutex
	stopCleanup  chan struct{}
}

// NewMemorySessionManager creates a new in-memory session manager
func NewMemorySessionManager(config *SessionConfig) *MemorySessionManager {
	if config == nil {
		config = DefaultSessionConfig()
	}

	mgr := &MemorySessionManager{
		config:       config,
		sessions:     make(map[string]*Session),
		userSessions: make(map[string]map[string]bool),
		stopCleanup:  make(chan struct{}),
	}

	// Start background cleanup goroutine
	go mgr.cleanupExpiredSessions()

	return mgr
}

// CreateSession creates a new session for the given user
func (m *MemorySessionManager) CreateSession(ctx context.Context, user *User) (string, error) {
	if user == nil {
		return "", errors.New("user cannot be nil")
	}

	sessionID := generateSessionID()
	now := time.Now()

	session := &Session{
		ID:         sessionID,
		UserID:     user.ID,
		User:       user,
		CreatedAt:  now,
		LastAccess: now,
		ExpiresAt:  now.Add(m.config.SessionTimeout),
		Metadata:   make(map[string]interface{}),
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check max sessions per user
	userSessions := m.userSessions[user.ID]
	if userSessions == nil {
		userSessions = make(map[string]bool)
		m.userSessions[user.ID] = userSessions
	}

	if len(userSessions) >= m.config.MaxSessionsPerUser {
		// Remove oldest session
		var oldestSessionID string
		var oldestTime time.Time
		for sid := range userSessions {
			if s, exists := m.sessions[sid]; exists {
				if oldestTime.IsZero() || s.LastAccess.Before(oldestTime) {
					oldestTime = s.LastAccess
					oldestSessionID = sid
				}
			}
		}
		if oldestSessionID != "" {
			m.deleteSessionLocked(oldestSessionID)
		}
	}

	m.sessions[sessionID] = session
	userSessions[sessionID] = true

	return sessionID, nil
}

// GetSession retrieves a session by ID
func (m *MemorySessionManager) GetSession(ctx context.Context, sessionID string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	// Update last access time
	session.LastAccess = time.Now()

	return session.User, nil
}

// DeleteSession removes a session
func (m *MemorySessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.deleteSessionLocked(sessionID)
}

// deleteSessionLocked removes a session (must hold lock)
func (m *MemorySessionManager) deleteSessionLocked(sessionID string) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	// Remove from user sessions
	if userSessions, ok := m.userSessions[session.UserID]; ok {
		delete(userSessions, sessionID)
		if len(userSessions) == 0 {
			delete(m.userSessions, session.UserID)
		}
	}

	delete(m.sessions, sessionID)
	return nil
}

// ValidateSession checks if a session is valid
func (m *MemorySessionManager) ValidateSession(ctx context.Context, sessionID string) (bool, error) {
	_, err := m.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrSessionExpired) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetUserSessions returns all active sessions for a user
func (m *MemorySessionManager) GetUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*Session
	for sessionID := range m.userSessions[userID] {
		if session, exists := m.sessions[sessionID]; exists {
			if time.Now().Before(session.ExpiresAt) {
				sessions = append(sessions, session)
			}
		}
	}

	return sessions, nil
}

// DeleteUserSessions removes all sessions for a user
func (m *MemorySessionManager) DeleteUserSessions(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	userSessions := m.userSessions[userID]
	if userSessions == nil {
		return nil
	}

	for sessionID := range userSessions {
		delete(m.sessions, sessionID)
	}

	delete(m.userSessions, userID)
	return nil
}

// UpdateSessionMetadata updates session metadata
func (m *MemorySessionManager) UpdateSessionMetadata(ctx context.Context, sessionID string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}

	for k, v := range metadata {
		session.Metadata[k] = v
	}

	return nil
}

// cleanupExpiredSessions removes expired sessions periodically
func (m *MemorySessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCleanup:
			return
		}
	}
}

// cleanup removes expired sessions
func (m *MemorySessionManager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for sessionID, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			m.deleteSessionLocked(sessionID)
		}
	}
}

// Shutdown stops the session manager cleanup routine
func (m *MemorySessionManager) Shutdown() {
	close(m.stopCleanup)
}

// GetSessionCount returns the number of active sessions
func (m *MemorySessionManager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	count := 0
	for _, session := range m.sessions {
		if now.Before(session.ExpiresAt) {
			count++
		}
	}
	return count
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("sess_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// SessionToJSON converts a session to JSON
func SessionToJSON(session *Session) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SessionFromJSON creates a session from JSON
func SessionFromJSON(jsonStr string) (*Session, error) {
	var session Session
	err := json.Unmarshal([]byte(jsonStr), &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
