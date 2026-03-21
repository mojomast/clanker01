package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	swarmapi "github.com/swarm-ai/swarm/pkg/api"
)

func TestNewSessionPersistence_Memory(t *testing.T) {
	sp, err := NewSessionPersistence(PersistenceBackendMemory, nil)

	if err != nil {
		t.Fatalf("NewSessionPersistence failed: %v", err)
	}

	if sp.backend != PersistenceBackendMemory {
		t.Errorf("Expected backend %s, got %s", PersistenceBackendMemory, sp.backend)
	}

	if sp.sessions == nil {
		t.Error("Sessions map should be initialized")
	}
}

func TestNewSessionPersistence_File(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sessions.json")

	config := map[string]interface{}{
		"path": filePath,
	}

	sp, err := NewSessionPersistence(PersistenceBackendFile, config)

	if err != nil {
		t.Fatalf("NewSessionPersistence failed: %v", err)
	}

	if sp.backend != PersistenceBackendFile {
		t.Errorf("Expected backend %s, got %s", PersistenceBackendFile, sp.backend)
	}

	if sp.filePath != filePath {
		t.Errorf("Expected file path %s, got %s", filePath, sp.filePath)
	}

	session := &Session{
		ID:        "test-session",
		Status:    SessionActive,
		ProjectID: "test-project",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Conversation: &ConversationState{
			Messages:            make([]swarmapi.Message, 0),
			CompressedSummaries: make([]*CompressedSummary, 0),
			TotalMessages:       0,
		},
		TaskState: &TaskState{
			History:  make([]*swarmapi.Task, 0),
			Blockers: make([]string, 0),
		},
		Context: &ContextState{
			ActiveFiles:   make([]FileInfo, 0),
			OpenSymbols:   make([]SymbolInfo, 0),
			RecentEdits:   make([]Edit, 0),
			SearchHistory: make([]string, 0),
		},
		AgentCoord: &AgentCoordState{
			ActiveAgents:     make(map[string]string),
			SharedBlackboard: make(map[string]any),
			PendingHandoffs:  make([]string, 0),
		},
	}

	_ = sp.Create(session)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Session file should be created after adding a session")
	}
}

func TestSessionPersistence_Create(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	session := &Session{
		ID:        "session-123",
		Status:    SessionActive,
		ProjectID: "project-123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Conversation: &ConversationState{
			Messages:            make([]swarmapi.Message, 0),
			CompressedSummaries: make([]*CompressedSummary, 0),
			TotalMessages:       0,
		},
		TaskState: &TaskState{
			History:  make([]*swarmapi.Task, 0),
			Blockers: make([]string, 0),
		},
		Context: &ContextState{
			ActiveFiles:   make([]FileInfo, 0),
			OpenSymbols:   make([]SymbolInfo, 0),
			RecentEdits:   make([]Edit, 0),
			SearchHistory: make([]string, 0),
		},
		AgentCoord: &AgentCoordState{
			ActiveAgents:     make(map[string]string),
			SharedBlackboard: make(map[string]any),
			PendingHandoffs:  make([]string, 0),
		},
	}

	err := sp.Create(session)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if len(sp.sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sp.sessions))
	}
}

func TestSessionPersistence_Get(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: &ConversationState{},
		TaskState:    &TaskState{},
		Context:      &ContextState{},
		AgentCoord:   &AgentCoordState{},
	}

	_ = sp.Create(session)

	retrieved, err := sp.Get("session-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
	}
}

func TestSessionPersistence_Get_NotFound(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	_, err := sp.Get("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestSessionPersistence_Update(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: &ConversationState{},
		TaskState:    &TaskState{},
		Context:      &ContextState{},
		AgentCoord:   &AgentCoordState{},
	}

	_ = sp.Create(session)

	session.Status = SessionSuspended
	session.UpdatedAt = time.Now()

	err := sp.Update(session)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := sp.Get("session-123")
	if retrieved.Status != SessionSuspended {
		t.Errorf("Expected status %s, got %s", SessionSuspended, retrieved.Status)
	}
}

func TestSessionPersistence_Delete(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: &ConversationState{},
		TaskState:    &TaskState{},
		Context:      &ContextState{},
		AgentCoord:   &AgentCoordState{},
	}

	_ = sp.Create(session)

	err := sp.Delete("session-123")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if len(sp.sessions) != 0 {
		t.Errorf("Expected 0 sessions after delete, got %d", len(sp.sessions))
	}
}

func TestSessionPersistence_List_NoFilter(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	_ = sp.Create(&Session{ID: "1", Status: SessionActive, ProjectID: "p1", CreatedAt: time.Now(), UpdatedAt: time.Now(), Conversation: &ConversationState{}, TaskState: &TaskState{}, Context: &ContextState{}, AgentCoord: &AgentCoordState{}})
	_ = sp.Create(&Session{ID: "2", Status: SessionSuspended, ProjectID: "p2", CreatedAt: time.Now(), UpdatedAt: time.Now(), Conversation: &ConversationState{}, TaskState: &TaskState{}, Context: &ContextState{}, AgentCoord: &AgentCoordState{}})

	sessions, err := sp.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestSessionPersistence_List_WithFilter(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	now := time.Now()
	_ = sp.Create(&Session{ID: "1", Status: SessionActive, ProjectID: "p1", CreatedAt: now, UpdatedAt: now, Conversation: &ConversationState{}, TaskState: &TaskState{}, Context: &ContextState{}, AgentCoord: &AgentCoordState{}})
	_ = sp.Create(&Session{ID: "2", Status: SessionSuspended, ProjectID: "p2", CreatedAt: now, UpdatedAt: now, Conversation: &ConversationState{}, TaskState: &TaskState{}, Context: &ContextState{}, AgentCoord: &AgentCoordState{}})

	filter := &SessionFilter{
		ProjectID: "p1",
		Status:    SessionActive,
	}

	sessions, err := sp.List(filter)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	if sessions[0].ID != "1" {
		t.Errorf("Expected session ID '1', got '%s'", sessions[0].ID)
	}
}

func TestSessionPersistence_List_WithLimit(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	now := time.Now()
	for i := 0; i < 10; i++ {
		_ = sp.Create(&Session{
			ID:           string(rune('0' + i)),
			Status:       SessionActive,
			ProjectID:    "p1",
			CreatedAt:    now,
			UpdatedAt:    now,
			Conversation: &ConversationState{},
			TaskState:    &TaskState{},
			Context:      &ContextState{},
			AgentCoord:   &AgentCoordState{},
		})
	}

	filter := &SessionFilter{
		Limit: 5,
	}

	sessions, _ := sp.List(filter)
	if len(sessions) != 5 {
		t.Errorf("Expected 5 sessions with limit, got %d", len(sessions))
	}
}

func TestSessionPersistence_Export(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: &ConversationState{},
		TaskState:    &TaskState{},
		Context:      &ContextState{},
		AgentCoord:   &AgentCoordState{},
	}

	_ = sp.Create(session)

	data, err := sp.Export("session-123")
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Exported data should not be empty")
	}
}

func TestSessionPersistence_Import(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: &ConversationState{},
		TaskState:    &TaskState{},
		Context:      &ContextState{},
		AgentCoord:   &AgentCoordState{},
	}

	_ = sp.Create(session)

	data, _ := sp.Export("session-123")

	sp2, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	imported, err := sp2.Import(data)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if imported.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, imported.ID)
	}
}

func TestSessionPersistence_Backup(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: &ConversationState{},
		TaskState:    &TaskState{},
		Context:      &ContextState{},
		AgentCoord:   &AgentCoordState{},
	}

	_ = sp.Create(session)

	tempDir := t.TempDir()
	backupPath := filepath.Join(tempDir, "backup.json")

	err := sp.Backup(backupPath)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}
}

func TestSessionPersistence_Restore(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: &ConversationState{},
		TaskState:    &TaskState{},
		Context:      &ContextState{},
		AgentCoord:   &AgentCoordState{},
	}

	_ = sp.Create(session)

	tempDir := t.TempDir()
	backupPath := filepath.Join(tempDir, "backup.json")

	_ = sp.Backup(backupPath)

	sp2, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	err := sp2.Restore(backupPath)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	retrieved, _ := sp2.Get("session-123")
	if retrieved == nil {
		t.Error("Session should be restored")
	}
}

func TestSessionPersistence_Cleanup(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	now := time.Now()
	closedTime := now.Add(-48 * time.Hour)

	session := &Session{
		ID:           "session-123",
		Status:       SessionClosed,
		ProjectID:    "project-123",
		CreatedAt:    now,
		UpdatedAt:    now,
		ClosedAt:     &closedTime,
		Conversation: &ConversationState{},
		TaskState:    &TaskState{},
		Context:      &ContextState{},
		AgentCoord:   &AgentCoordState{},
	}

	_ = sp.Create(session)

	count := sp.Cleanup(now.Add(-24 * time.Hour))
	if count != 1 {
		t.Errorf("Expected to cleanup 1 session, got %d", count)
	}

	if len(sp.sessions) != 0 {
		t.Errorf("Expected 0 sessions after cleanup, got %d", len(sp.sessions))
	}
}

func TestSessionPersistence_GetStats(t *testing.T) {
	sp, _ := NewSessionPersistence(PersistenceBackendMemory, nil)

	now := time.Now()
	closedTime := now

	_ = sp.Create(&Session{ID: "1", Status: SessionActive, ProjectID: "p1", CreatedAt: now, UpdatedAt: now, Conversation: &ConversationState{}, TaskState: &TaskState{}, Context: &ContextState{}, AgentCoord: &AgentCoordState{}})
	_ = sp.Create(&Session{ID: "2", Status: SessionSuspended, ProjectID: "p1", CreatedAt: now, UpdatedAt: now, Conversation: &ConversationState{}, TaskState: &TaskState{}, Context: &ContextState{}, AgentCoord: &AgentCoordState{}})
	_ = sp.Create(&Session{ID: "3", Status: SessionClosed, ProjectID: "p1", CreatedAt: now, UpdatedAt: now, ClosedAt: &closedTime, Conversation: &ConversationState{}, TaskState: &TaskState{}, Context: &ContextState{}, AgentCoord: &AgentCoordState{}})

	stats := sp.GetStats()

	if stats.TotalSessions != 3 {
		t.Errorf("Expected 3 total sessions, got %d", stats.TotalSessions)
	}

	if stats.ActiveSessions != 1 {
		t.Errorf("Expected 1 active session, got %d", stats.ActiveSessions)
	}

	if stats.SuspendedSessions != 1 {
		t.Errorf("Expected 1 suspended session, got %d", stats.SuspendedSessions)
	}

	if stats.ClosedSessions != 1 {
		t.Errorf("Expected 1 closed session, got %d", stats.ClosedSessions)
	}
}

func TestComputeChecksum(t *testing.T) {
	session := &Session{
		ID:        "session-123",
		Status:    SessionActive,
		UpdatedAt: time.Now(),
	}

	checksum := computeChecksum(session)
	if checksum == "" {
		t.Error("Checksum should not be empty")
	}

	checksum2 := computeChecksum(session)
	if checksum != checksum2 {
		t.Error("Checksums should be consistent")
	}
}
