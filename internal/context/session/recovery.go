package session

import (
	"context"
	"fmt"
	"time"

	swarmapi "github.com/swarm-ai/swarm/pkg/api"
)

type RecoveryManager struct {
	persistence    *SessionPersistence
	compressor     *CompressionManager
	checkpointDir  string
	recoveryPolicy *RecoveryPolicy
}

type RecoveryPolicy struct {
	MaxCheckpointAge    time.Duration
	MaxRecoveryAttempts int
	AutoRecover         bool
	RecoveryTimeout     time.Duration
}

type RecoveryState struct {
	SessionID      string
	LastCheckpoint time.Time
	CheckpointData []byte
	RecoveryCount  int
	Status         RecoveryStatus
	LastError      error
}

type RecoveryStatus string

const (
	RecoveryStatusHealthy    RecoveryStatus = "healthy"
	RecoveryStatusCorrupted  RecoveryStatus = "corrupted"
	RecoveryStatusRecovering RecoveryStatus = "recovering"
	RecoveryStatusFailed     RecoveryStatus = "failed"
)

type Checkpoint struct {
	SessionID string
	Timestamp time.Time
	Version   int64
	Checksum  string
	Data      []byte
	Metadata  map[string]interface{}
}

func NewRecoveryManager(persistence *SessionPersistence, compressor *CompressionManager, checkpointDir string) *RecoveryManager {
	policy := &RecoveryPolicy{
		MaxCheckpointAge:    24 * time.Hour,
		MaxRecoveryAttempts: 3,
		AutoRecover:         true,
		RecoveryTimeout:     5 * time.Minute,
	}

	rm := &RecoveryManager{
		persistence:    persistence,
		compressor:     compressor,
		checkpointDir:  checkpointDir,
		recoveryPolicy: policy,
	}

	return rm
}

func (rm *RecoveryManager) CreateCheckpoint(session *Session) (*Checkpoint, error) {
	checkpoint := &Checkpoint{
		SessionID: session.ID,
		Timestamp: time.Now(),
		Version:   session.UpdatedAt.Unix(),
		Checksum:  computeChecksum(session),
		Data:      []byte{},
		Metadata: map[string]interface{}{
			"status":       session.Status,
			"messageCount": session.Conversation.TotalMessages,
			"projectID":    session.ProjectID,
		},
	}

	return checkpoint, nil
}

func (rm *RecoveryManager) SaveCheckpoint(checkpoint *Checkpoint) error {
	return fmt.Errorf("checkpoint saving not implemented")
}

func (rm *RecoveryManager) LoadCheckpoint(sessionID string) (*Checkpoint, error) {
	return nil, fmt.Errorf("checkpoint loading not implemented")
}

func (rm *RecoveryManager) RecoverSession(sessionID string) (*Session, error) {
	session, err := rm.persistence.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	state := &RecoveryState{
		SessionID: sessionID,
		Status:    RecoveryStatusRecovering,
	}

	if err := rm.validateSession(session); err != nil {
		state.Status = RecoveryStatusCorrupted
		state.LastError = err
		state.RecoveryCount++

		if !rm.recoveryPolicy.AutoRecover || state.RecoveryCount > rm.recoveryPolicy.MaxRecoveryAttempts {
			state.Status = RecoveryStatusFailed
			return nil, fmt.Errorf("session corrupted, recovery failed: %w", err)
		}

		if err := rm.attemptRecovery(session, state); err != nil {
			state.LastError = err
			state.Status = RecoveryStatusFailed
			return nil, fmt.Errorf("recovery attempt failed: %w", err)
		}
	}

	state.Status = RecoveryStatusHealthy
	return session, nil
}

func (rm *RecoveryManager) validateSession(session *Session) error {
	if session.ID == "" {
		return fmt.Errorf("session ID is empty")
	}

	if session.ProjectID == "" {
		return fmt.Errorf("project ID is empty")
	}

	if session.Conversation == nil {
		return fmt.Errorf("conversation state is nil")
	}

	if session.TaskState == nil {
		return fmt.Errorf("task state is nil")
	}

	if session.Context == nil {
		return fmt.Errorf("context state is nil")
	}

	if session.AgentCoord == nil {
		return fmt.Errorf("agent coordination state is nil")
	}

	// Sum the actual message count: current messages + messages represented
	// by each compressed summary (using summary.MessageCount, not just counting summaries).
	totalMessages := len(session.Conversation.Messages)
	for _, summary := range session.Conversation.CompressedSummaries {
		totalMessages += summary.MessageCount
	}
	if totalMessages != session.Conversation.TotalMessages {
		return fmt.Errorf("message count mismatch: expected %d, got %d",
			session.Conversation.TotalMessages, totalMessages)
	}

	return nil
}

func (rm *RecoveryManager) attemptRecovery(session *Session, state *RecoveryState) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.Conversation == nil {
		session.Conversation = &ConversationState{
			Messages:            make([]swarmapi.Message, 0),
			CompressedSummaries: make([]*CompressedSummary, 0),
			TotalMessages:       0,
		}
	}

	if session.TaskState == nil {
		session.TaskState = &TaskState{
			History:  make([]*swarmapi.Task, 0),
			Blockers: make([]string, 0),
		}
	}

	if session.Context == nil {
		session.Context = &ContextState{
			ActiveFiles:   make([]FileInfo, 0),
			OpenSymbols:   make([]SymbolInfo, 0),
			RecentEdits:   make([]Edit, 0),
			SearchHistory: make([]string, 0),
		}
	}

	if session.AgentCoord == nil {
		session.AgentCoord = &AgentCoordState{
			ActiveAgents:     make(map[string]string),
			SharedBlackboard: make(map[string]any),
			PendingHandoffs:  make([]string, 0),
		}
	}

	session.Status = SessionResumed
	session.UpdatedAt = time.Now()

	if err := rm.persistence.Update(session); err != nil {
		return fmt.Errorf("failed to update recovered session: %w", err)
	}

	return nil
}

func (rm *RecoveryManager) RecoverFromCheckpoint(sessionID string) (*Session, error) {
	checkpoint, err := rm.LoadCheckpoint(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	if time.Since(checkpoint.Timestamp) > rm.recoveryPolicy.MaxCheckpointAge {
		return nil, fmt.Errorf("checkpoint too old: %v", checkpoint.Timestamp)
	}

	session := &Session{
		ID:        checkpoint.SessionID,
		Status:    SessionResumed,
		CreatedAt: checkpoint.Timestamp,
		UpdatedAt: checkpoint.Timestamp,
	}

	return session, nil
}

func (rm *RecoveryManager) RestoreCompressedSummary(session *Session, summaryID string) error {
	session.Conversation.mu.Lock()
	defer session.Conversation.mu.Unlock()

	for _, summary := range session.Conversation.CompressedSummaries {
		if summary.ID == summaryID {
			expanded, err := rm.compressor.Compress(context.Background(), summary.Content, 10000)
			if err != nil {
				return fmt.Errorf("failed to expand summary: %w", err)
			}

			session.Conversation.Messages = append(session.Conversation.Messages, swarmapi.Message{
				Role:    "system",
				Content: expanded,
			})
			return nil
		}
	}

	return fmt.Errorf("compressed summary not found: %s", summaryID)
}

func (rm *RecoveryManager) RepairConversation(session *Session) error {
	session.Conversation.mu.Lock()
	defer session.Conversation.mu.Unlock()

	if session.Conversation.Messages == nil {
		session.Conversation.Messages = make([]swarmapi.Message, 0)
	}

	if session.Conversation.CompressedSummaries == nil {
		session.Conversation.CompressedSummaries = make([]*CompressedSummary, 0)
	}

	expectedMessages := session.Conversation.TotalMessages
	actualMessages := len(session.Conversation.Messages)

	for _, summary := range session.Conversation.CompressedSummaries {
		actualMessages += summary.MessageCount
	}

	if actualMessages != expectedMessages {
		session.Conversation.TotalMessages = actualMessages
	}

	session.UpdatedAt = time.Now()

	return nil
}

func (rm *RecoveryManager) HealthCheck(sessionID string) (*RecoveryState, error) {
	session, err := rm.persistence.Get(sessionID)
	if err != nil {
		return &RecoveryState{
			SessionID: sessionID,
			Status:    RecoveryStatusFailed,
			LastError: err,
		}, nil
	}

	state := &RecoveryState{
		SessionID:      sessionID,
		LastCheckpoint: session.UpdatedAt,
	}

	if err := rm.validateSession(session); err != nil {
		state.Status = RecoveryStatusCorrupted
		state.LastError = err
		return state, nil
	}

	state.Status = RecoveryStatusHealthy
	return state, nil
}

func (rm *RecoveryManager) SetRecoveryPolicy(policy *RecoveryPolicy) {
	rm.recoveryPolicy = policy
}

func (rm *RecoveryManager) GetRecoveryPolicy() *RecoveryPolicy {
	return rm.recoveryPolicy
}

func (rm *RecoveryManager) ListCorruptedSessions() ([]string, error) {
	sessions, err := rm.persistence.List(nil)
	if err != nil {
		return nil, err
	}

	var corrupted []string
	for _, session := range sessions {
		if err := rm.validateSession(session); err != nil {
			corrupted = append(corrupted, session.ID)
		}
	}

	return corrupted, nil
}

func (rm *RecoveryManager) AutoRecoverAll() ([]string, error) {
	corrupted, err := rm.ListCorruptedSessions()
	if err != nil {
		return nil, err
	}

	var recovered []string
	for _, sessionID := range corrupted {
		_, err := rm.RecoverSession(sessionID)
		if err == nil {
			recovered = append(recovered, sessionID)
		}
	}

	return recovered, nil
}

func (rm *RecoveryManager) CreateRecoveryReport() *RecoveryReport {
	sessions, _ := rm.persistence.List(nil)

	report := &RecoveryReport{
		TotalSessions:     len(sessions),
		HealthySessions:   0,
		CorruptedSessions: 0,
		RecoveryAttempts:  0,
		GeneratedAt:       time.Now(),
	}

	for _, session := range sessions {
		if err := rm.validateSession(session); err != nil {
			report.CorruptedSessions++
		} else {
			report.HealthySessions++
		}
	}

	return report
}

type RecoveryReport struct {
	TotalSessions     int
	HealthySessions   int
	CorruptedSessions int
	RecoveryAttempts  int
	GeneratedAt       time.Time
}
