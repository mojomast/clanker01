package auth

import (
	"context"
	"testing"
	"time"
)

func TestDefaultSessionConfig(t *testing.T) {
	config := DefaultSessionConfig()

	if config.SessionTimeout != time.Hour*24 {
		t.Errorf("Expected session timeout 24h, got %v", config.SessionTimeout)
	}

	if config.CleanupInterval != time.Hour {
		t.Errorf("Expected cleanup interval 1h, got %v", config.CleanupInterval)
	}

	if config.MaxSessionsPerUser != 5 {
		t.Errorf("Expected max sessions 5, got %d", config.MaxSessionsPerUser)
	}
}

func TestNewMemorySessionManager(t *testing.T) {
	config := DefaultSessionConfig()
	mgr := NewMemorySessionManager(config)

	if mgr == nil {
		t.Fatal("Expected session manager to be created")
	}

	if mgr.config != config {
		t.Error("Expected config to be set")
	}

	mgr.Shutdown()
}

func TestMemorySessionManager_CreateSession(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()
	sessionID, err := mgr.CreateSession(ctx, user)

	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if sessionID == "" {
		t.Error("Expected session ID to be generated")
	}

	if mgr.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", mgr.GetSessionCount())
	}
}

func TestMemorySessionManager_CreateSessionNilUser(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	ctx := context.Background()
	_, err := mgr.CreateSession(ctx, nil)

	if err == nil {
		t.Error("Expected error when creating session with nil user")
	}
}

func TestMemorySessionManager_GetSession(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()
	sessionID, err := mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	retrievedUser, err := mgr.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrievedUser.ID)
	}

	if retrievedUser.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrievedUser.Username)
	}
}

func TestMemorySessionManager_GetSessionNotFound(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	ctx := context.Background()
	_, err := mgr.GetSession(ctx, "nonexistent-session")

	if err != ErrSessionNotFound {
		t.Errorf("Expected error %v, got %v", ErrSessionNotFound, err)
	}
}

func TestMemorySessionManager_DeleteSession(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()
	sessionID, err := mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err = mgr.DeleteSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	if mgr.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after deletion, got %d", mgr.GetSessionCount())
	}

	// Verify session is deleted
	_, err = mgr.GetSession(ctx, sessionID)
	if err != ErrSessionNotFound {
		t.Errorf("Expected error %v, got %v", ErrSessionNotFound, err)
	}
}

func TestMemorySessionManager_ValidateSession(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()
	sessionID, err := mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	valid, err := mgr.ValidateSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to validate session: %v", err)
	}

	if !valid {
		t.Error("Expected session to be valid")
	}

	// Test invalid session
	valid, err = mgr.ValidateSession(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Failed to validate invalid session: %v", err)
	}

	if valid {
		t.Error("Expected invalid session to be invalid")
	}
}

func TestMemorySessionManager_GetUserSessions(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()

	// Create multiple sessions for the same user
	_, err := mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}

	_, err = mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	sessions, err := mgr.GetUserSessions(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestMemorySessionManager_DeleteUserSessions(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()

	// Create multiple sessions for the same user
	_, err := mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}

	_, err = mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	err = mgr.DeleteUserSessions(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user sessions: %v", err)
	}

	if mgr.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after deletion, got %d", mgr.GetSessionCount())
	}
}

func TestMemorySessionManager_UpdateSessionMetadata(t *testing.T) {
	mgr := NewMemorySessionManager(DefaultSessionConfig())
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()
	sessionID, err := mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	metadata := map[string]interface{}{
		"ip_address": "192.168.1.1",
		"user_agent": "test-agent",
	}

	err = mgr.UpdateSessionMetadata(ctx, sessionID, metadata)
	if err != nil {
		t.Fatalf("Failed to update session metadata: %v", err)
	}

	// Verify metadata was updated
	sessions, err := mgr.GetUserSessions(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user sessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Fatal("Expected to find session")
	}

	if sessions[0].Metadata["ip_address"] != "192.168.1.1" {
		t.Errorf("Expected ip_address 192.168.1.1, got %v", sessions[0].Metadata["ip_address"])
	}
}

func TestMemorySessionManager_MaxSessionsPerUser(t *testing.T) {
	config := &SessionConfig{
		SessionTimeout:     time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 3,
	}

	mgr := NewMemorySessionManager(config)
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()

	// Create max sessions
	for i := 0; i < 3; i++ {
		_, err := mgr.CreateSession(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
	}

	// Create one more - should remove oldest
	_, err := mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session beyond max: %v", err)
	}

	// Should still have max sessions
	sessions, err := mgr.GetUserSessions(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions (max), got %d", len(sessions))
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == id2 {
		t.Error("Expected unique session IDs")
	}

	if id1 == "" {
		t.Error("Expected non-empty session ID")
	}
}

func TestMemorySessionManager_Cleanup(t *testing.T) {
	config := &SessionConfig{
		SessionTimeout:  time.Millisecond * 100,
		CleanupInterval: time.Millisecond * 50,
	}

	mgr := NewMemorySessionManager(config)
	defer mgr.Shutdown()

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	ctx := context.Background()

	// Create session
	_, err := mgr.CreateSession(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if mgr.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", mgr.GetSessionCount())
	}

	// Wait for session to expire
	time.Sleep(time.Millisecond * 150)

	// Trigger cleanup
	mgr.cleanup()

	// Session should be removed
	if mgr.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after cleanup, got %d", mgr.GetSessionCount())
	}
}
