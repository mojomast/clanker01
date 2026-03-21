package session

import (
	"fmt"
	"testing"

	swarmapi "github.com/swarm-ai/swarm/pkg/api"
)

func TestNewSessionManager(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}

	sm := NewSessionManager(persistence, compressor)

	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}

	if sm.compressionThreshold != 100 {
		t.Errorf("Expected compression threshold 100, got %d", sm.compressionThreshold)
	}
}

func TestCreateSession(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")

	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.ProjectID != "project-123" {
		t.Errorf("Expected project ID project-123, got %s", session.ProjectID)
	}

	if session.Status != SessionActive {
		t.Errorf("Expected status %s, got %s", SessionActive, session.Status)
	}

	if session.Conversation == nil {
		t.Error("Conversation state should not be nil")
	}

	if session.TaskState == nil {
		t.Error("Task state should not be nil")
	}

	if session.Context == nil {
		t.Error("Context state should not be nil")
	}

	if session.AgentCoord == nil {
		t.Error("Agent coordination state should not be nil")
	}
}

func TestGetSession(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	created, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	retrieved, err := sm.GetSession(created.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected session ID %s, got %s", created.ID, retrieved.ID)
	}

	if retrieved.ProjectID != created.ProjectID {
		t.Errorf("Expected project ID %s, got %s", created.ProjectID, retrieved.ProjectID)
	}
}

func TestSuspendSession(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = sm.Suspend(session.ID)
	if err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}

	suspended, err := sm.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if suspended.Status != SessionSuspended {
		t.Errorf("Expected status %s, got %s", SessionSuspended, suspended.Status)
	}
}

func TestResumeSession(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	_ = sm.Suspend(session.ID)

	resumed, err := sm.Resume(session.ID)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	if resumed.Status != SessionResumed {
		t.Errorf("Expected status %s, got %s", SessionResumed, resumed.Status)
	}
}

func TestCloseSession(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = sm.Close(session.ID)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	closed, err := sm.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if closed.Status != SessionClosed {
		t.Errorf("Expected status %s, got %s", SessionClosed, closed.Status)
	}

	if closed.ClosedAt == nil {
		t.Error("ClosedAt should not be nil")
	}
}

func TestAddMessage(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	msg := swarmapi.Message{
		Role:    "user",
		Content: "Hello, world!",
	}

	err = sm.AddMessage(session.ID, msg)
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	messages, _, err := sm.GetMessages(session.ID)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("Expected role user, got %s", messages[0].Role)
	}
}

func TestSetCurrentTask(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	task := &swarmapi.Task{
		ID:     "task-123",
		Status: swarmapi.TaskStatusPending,
	}

	err = sm.SetCurrentTask(session.ID, task)
	if err != nil {
		t.Fatalf("SetCurrentTask failed: %v", err)
	}

	currentTask, err := sm.GetCurrentTask(session.ID)
	if err != nil {
		t.Fatalf("GetCurrentTask failed: %v", err)
	}

	if currentTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, currentTask.ID)
	}
}

func TestAddBlocker(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = sm.AddBlocker(session.ID, "Missing dependency")
	if err != nil {
		t.Fatalf("AddBlocker failed: %v", err)
	}

	session.mu.Lock()
	blockers := session.TaskState.Blockers
	session.mu.Unlock()

	if len(blockers) != 1 {
		t.Errorf("Expected 1 blocker, got %d", len(blockers))
	}

	if blockers[0] != "Missing dependency" {
		t.Errorf("Expected blocker 'Missing dependency', got '%s'", blockers[0])
	}
}

func TestClearBlockers(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	_ = sm.AddBlocker(session.ID, "Blocker 1")
	_ = sm.AddBlocker(session.ID, "Blocker 2")

	err = sm.ClearBlockers(session.ID)
	if err != nil {
		t.Fatalf("ClearBlockers failed: %v", err)
	}

	session.mu.Lock()
	blockers := session.TaskState.Blockers
	session.mu.Unlock()

	if len(blockers) != 0 {
		t.Errorf("Expected 0 blockers, got %d", len(blockers))
	}
}

func TestUpdateContext(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = sm.UpdateContext(session.ID, func(ctx *ContextState) {
		ctx.ActiveFiles = append(ctx.ActiveFiles, FileInfo{
			Path:     "/path/to/file.go",
			Language: "go",
		})
	})

	if err != nil {
		t.Fatalf("UpdateContext failed: %v", err)
	}

	session.mu.Lock()
	files := session.Context.ActiveFiles
	session.mu.Unlock()

	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

func TestRegisterAgent(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = sm.RegisterAgent(session.ID, "agent-123", "coder")
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	session.mu.Lock()
	agents := session.AgentCoord.ActiveAgents
	session.mu.Unlock()

	if len(agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(agents))
	}

	if agents["agent-123"] != "coder" {
		t.Errorf("Expected role 'coder', got '%s'", agents["agent-123"])
	}
}

func TestListSessions(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	_, _ = sm.CreateSession("project-1")
	_, _ = sm.CreateSession("project-2")
	_, _ = sm.CreateSession("project-1")

	filter := &SessionFilter{
		ProjectID: "project-1",
	}

	sessions, err := sm.ListSessions(filter)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestDeleteSession(t *testing.T) {
	compressor := NewCompressionManager()
	persistence := &MockSessionStore{}
	sm := NewSessionManager(persistence, compressor)

	session, err := sm.CreateSession("project-123")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = sm.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	_, err = sm.GetSession(session.ID)
	if err == nil {
		t.Error("Expected error when getting deleted session")
	}
}

type MockSessionStore struct {
	sessions map[string]*Session
}

func (m *MockSessionStore) Create(session *Session) error {
	if m.sessions == nil {
		m.sessions = make(map[string]*Session)
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *MockSessionStore) Get(id string) (*Session, error) {
	if m.sessions == nil {
		return nil, fmt.Errorf("session not found")
	}
	session, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (m *MockSessionStore) Update(session *Session) error {
	if m.sessions != nil {
		m.sessions[session.ID] = session
	}
	return nil
}

func (m *MockSessionStore) Delete(id string) error {
	if m.sessions != nil {
		delete(m.sessions, id)
	}
	return nil
}

func (m *MockSessionStore) List(filter *SessionFilter) ([]*Session, error) {
	var sessions []*Session
	for _, session := range m.sessions {
		if filter == nil {
			sessions = append(sessions, session)
			continue
		}

		if filter.ProjectID != "" && session.ProjectID != filter.ProjectID {
			continue
		}

		if filter.Status != "" && session.Status != filter.Status {
			continue
		}

		sessions = append(sessions, session)
	}
	return sessions, nil
}
