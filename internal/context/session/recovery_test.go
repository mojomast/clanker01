package session

import (
	"testing"
	"time"

	swarmapi "github.com/swarm-ai/swarm/pkg/api"
)

func TestNewRecoveryManager(t *testing.T) {
	persistence, _ := NewSessionPersistence(PersistenceBackendMemory, nil)
	compressor := NewCompressionManager()

	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	if rm == nil {
		t.Fatal("NewRecoveryManager returned nil")
	}

	if rm.persistence != persistence {
		t.Error("Persistence should be set")
	}

	if rm.compressor != compressor {
		t.Error("Compressor should be set")
	}

	if rm.checkpointDir != "/tmp/checkpoints" {
		t.Errorf("Expected checkpoint dir '/tmp/checkpoints', got '%s'", rm.checkpointDir)
	}
}

func TestCreateCheckpoint(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

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

	checkpoint, err := rm.CreateCheckpoint(session)
	if err != nil {
		t.Fatalf("CreateCheckpoint failed: %v", err)
	}

	if checkpoint.SessionID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, checkpoint.SessionID)
	}

	if checkpoint.Checksum == "" {
		t.Error("Checksum should not be empty")
	}
}

func TestValidateSession(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	validSession := &Session{
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

	err := rm.validateSession(validSession)
	if err != nil {
		t.Errorf("Valid session should not fail validation: %v", err)
	}
}

func TestValidateSession_Invalid(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	invalidSession := &Session{
		ID: "",
	}

	err := rm.validateSession(invalidSession)
	if err == nil {
		t.Error("Invalid session should fail validation")
	}
}

func TestValidateSession_NilStates(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	session := &Session{
		ID:           "session-123",
		ProjectID:    "project-123",
		Status:       SessionActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: nil,
		TaskState:    nil,
		Context:      nil,
		AgentCoord:   nil,
	}

	err := rm.validateSession(session)
	if err == nil {
		t.Error("Session with nil states should fail validation")
	}
}

func TestRecoverSession(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

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

	persistence.sessions = map[string]*Session{"session-123": session}

	recovered, err := rm.RecoverSession("session-123")
	if err != nil {
		t.Fatalf("RecoverSession failed: %v", err)
	}

	if recovered.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, recovered.ID)
	}
}

func TestRecoverSession_NotFound(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	_, err := rm.RecoverSession("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestAttemptRecovery(t *testing.T) {
	persistence := &SessionPersistence{}
	persistence.sessions = make(map[string]*Session)
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: nil,
		TaskState:    nil,
		Context:      nil,
		AgentCoord:   nil,
	}

	_ = persistence.Create(session)

	state := &RecoveryState{
		SessionID:     session.ID,
		Status:        RecoveryStatusCorrupted,
		RecoveryCount: 0,
	}

	err := rm.attemptRecovery(session, state)
	if err != nil {
		t.Fatalf("attemptRecovery failed: %v", err)
	}

	if session.Conversation == nil {
		t.Error("Conversation should be initialized after recovery")
	}

	if session.TaskState == nil {
		t.Error("TaskState should be initialized after recovery")
	}

	if session.Context == nil {
		t.Error("Context should be initialized after recovery")
	}

	if session.AgentCoord == nil {
		t.Error("AgentCoord should be initialized after recovery")
	}
}

func TestRepairConversation(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	session := &Session{
		ID:        "session-123",
		Status:    SessionActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Conversation: &ConversationState{
			Messages:            make([]swarmapi.Message, 1),
			CompressedSummaries: make([]*CompressedSummary, 1),
			TotalMessages:       5,
		},
		TaskState:  &TaskState{},
		Context:    &ContextState{},
		AgentCoord: &AgentCoordState{},
	}

	session.Conversation.Messages[0] = swarmapi.Message{Role: "user", Content: "Hello"}
	session.Conversation.CompressedSummaries[0] = &CompressedSummary{
		MessageCount: 2,
	}

	err := rm.RepairConversation(session)
	if err != nil {
		t.Fatalf("RepairConversation failed: %v", err)
	}

	if session.Conversation.TotalMessages != 3 {
		t.Errorf("Expected total messages 3, got %d", session.Conversation.TotalMessages)
	}
}

func TestHealthCheck(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

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

	persistence.sessions = map[string]*Session{"session-123": session}

	state, err := rm.HealthCheck("session-123")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if state.Status != RecoveryStatusHealthy {
		t.Errorf("Expected status %s, got %s", RecoveryStatusHealthy, state.Status)
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	session := &Session{
		ID:           "session-123",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: nil,
		TaskState:    nil,
		Context:      nil,
		AgentCoord:   nil,
	}

	persistence.sessions = map[string]*Session{"session-123": session}

	state, err := rm.HealthCheck("session-123")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if state.Status != RecoveryStatusCorrupted {
		t.Errorf("Expected status %s, got %s", RecoveryStatusCorrupted, state.Status)
	}

	if state.LastError == nil {
		t.Error("LastError should be set for corrupted session")
	}
}

func TestSetRecoveryPolicy(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	policy := &RecoveryPolicy{
		MaxCheckpointAge:    48 * time.Hour,
		MaxRecoveryAttempts: 5,
		AutoRecover:         false,
		RecoveryTimeout:     10 * time.Minute,
	}

	rm.SetRecoveryPolicy(policy)

	retrieved := rm.GetRecoveryPolicy()
	if retrieved.MaxCheckpointAge != policy.MaxCheckpointAge {
		t.Errorf("Expected max checkpoint age %v, got %v", policy.MaxCheckpointAge, retrieved.MaxCheckpointAge)
	}
}

func TestCreateRecoveryReport(t *testing.T) {
	persistence := &SessionPersistence{}
	compressor := NewCompressionManager()
	rm := NewRecoveryManager(persistence, compressor, "/tmp/checkpoints")

	session1 := &Session{
		ID:        "session-1",
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

	session2 := &Session{
		ID:           "session-2",
		Status:       SessionActive,
		ProjectID:    "project-123",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Conversation: nil,
		TaskState:    nil,
		Context:      nil,
		AgentCoord:   nil,
	}

	persistence.sessions = map[string]*Session{
		"session-1": session1,
		"session-2": session2,
	}

	report := rm.CreateRecoveryReport()

	if report.TotalSessions != 2 {
		t.Errorf("Expected 2 total sessions, got %d", report.TotalSessions)
	}

	if report.HealthySessions != 1 {
		t.Errorf("Expected 1 healthy session, got %d", report.HealthySessions)
	}

	if report.CorruptedSessions != 1 {
		t.Errorf("Expected 1 corrupted session, got %d", report.CorruptedSessions)
	}
}
