package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	swarmapi "github.com/swarm-ai/swarm/pkg/api"
)

type SessionStatus string

const (
	SessionCreated   SessionStatus = "created"
	SessionActive    SessionStatus = "active"
	SessionSuspended SessionStatus = "suspended"
	SessionResumed   SessionStatus = "resumed"
	SessionClosed    SessionStatus = "closed"
)

type SessionFilter struct {
	ProjectID     string
	Status        SessionStatus
	CreatedAfter  time.Time
	CreatedBefore time.Time
	Limit         int
}

type Session struct {
	ID        string
	Status    SessionStatus
	ProjectID string
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  *time.Time

	Conversation *ConversationState
	TaskState    *TaskState
	Context      *ContextState
	AgentCoord   *AgentCoordState
	mu           sync.RWMutex
}

type ConversationState struct {
	Messages            []swarmapi.Message
	CurrentTurn         int
	CompressedSummaries []*CompressedSummary
	TotalMessages       int
	mu                  sync.RWMutex
}

type CompressedSummary struct {
	ID           string
	Content      string
	CreatedAt    time.Time
	MessageCount int
	FromIndex    int
	ToIndex      int
}

type TaskState struct {
	CurrentTask *swarmapi.Task
	History     []*swarmapi.Task
	Blockers    []string
	mu          sync.RWMutex
}

type ContextState struct {
	ActiveFiles   []FileInfo
	OpenSymbols   []SymbolInfo
	RecentEdits   []Edit
	SearchHistory []string
	mu            sync.RWMutex
}

type FileInfo struct {
	Path       string
	ModifiedAt time.Time
	Size       int64
	Language   string
}

type SymbolInfo struct {
	Name     string
	Type     string
	File     string
	Line     int
	OpenedAt time.Time
}

type Edit struct {
	File      string
	Timestamp time.Time
	Operation string
	Preview   string
}

type AgentCoordState struct {
	ActiveAgents     map[string]string
	SharedBlackboard map[string]any
	PendingHandoffs  []string
	mu               sync.RWMutex
}

type SessionManager struct {
	store                SessionStore
	compressor           *CompressionManager
	autosaveInterval     time.Duration
	compressionThreshold int
	activeSessions       map[string]*Session
	mu                   sync.RWMutex
	ctx                  context.Context
	cancel               context.CancelFunc
}

type SessionStore interface {
	Create(session *Session) error
	Get(id string) (*Session, error)
	Update(session *Session) error
	Delete(id string) error
	List(filter *SessionFilter) ([]*Session, error)
}

func NewSessionManager(store SessionStore, compressor *CompressionManager) *SessionManager {
	ctx, cancel := context.WithCancel(context.Background())

	sm := &SessionManager{
		store:                store,
		compressor:           compressor,
		autosaveInterval:     30 * time.Second,
		compressionThreshold: 100,
		activeSessions:       make(map[string]*Session),
		ctx:                  ctx,
		cancel:               cancel,
	}

	go sm.autosaveLoop()
	return sm
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *SessionManager) CreateSession(projectID string) (*Session, error) {
	now := time.Now()

	session := &Session{
		ID:        generateID(),
		Status:    SessionActive,
		ProjectID: projectID,
		CreatedAt: now,
		UpdatedAt: now,
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

	if err := m.store.Create(session); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.activeSessions[session.ID] = session
	m.mu.Unlock()

	return session, nil
}

func (m *SessionManager) GetSession(id string) (*Session, error) {
	m.mu.RLock()
	session, ok := m.activeSessions[id]
	m.mu.RUnlock()

	if ok {
		return session, nil
	}

	return m.store.Get(id)
}

func (m *SessionManager) Suspend(id string) error {
	session, err := m.GetSession(id)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Conversation.mu.Lock()
	if len(session.Conversation.Messages) > m.compressionThreshold {
		toCompress := session.Conversation.Messages[:len(session.Conversation.Messages)-20]
		recent := session.Conversation.Messages[len(session.Conversation.Messages)-20:]

		text := messagesToText(toCompress)
		compressed, err := m.compressor.Compress(context.Background(), text, 5000)
		if err == nil {
			summary := &CompressedSummary{
				ID:           generateID(),
				Content:      compressed,
				CreatedAt:    time.Now(),
				MessageCount: len(toCompress),
				FromIndex:    0,
				ToIndex:      len(toCompress) - 1,
			}
			session.Conversation.CompressedSummaries = append(
				session.Conversation.CompressedSummaries,
				summary,
			)
			session.Conversation.Messages = recent
		}
	}
	session.Conversation.mu.Unlock()

	session.Status = SessionSuspended
	session.UpdatedAt = time.Now()

	if err := m.store.Update(session); err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.activeSessions, id)
	m.mu.Unlock()

	return nil
}

func (m *SessionManager) Resume(id string) (*Session, error) {
	session, err := m.store.Get(id)
	if err != nil {
		return nil, err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Status = SessionResumed
	session.UpdatedAt = time.Now()

	if err := m.store.Update(session); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.activeSessions[id] = session
	m.mu.Unlock()

	return session, nil
}

func (m *SessionManager) Close(id string) error {
	session, err := m.GetSession(id)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	now := time.Now()
	session.ClosedAt = &now
	session.Status = SessionClosed
	session.UpdatedAt = now

	if err := m.store.Update(session); err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.activeSessions, id)
	m.mu.Unlock()

	return nil
}

func (m *SessionManager) AddMessage(sessionID string, msg swarmapi.Message) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Conversation.mu.Lock()
	session.Conversation.Messages = append(session.Conversation.Messages, msg)
	session.Conversation.CurrentTurn++
	session.Conversation.TotalMessages++
	session.Conversation.mu.Unlock()

	session.UpdatedAt = time.Now()

	if len(session.Conversation.Messages) >= m.compressionThreshold {
		go m.autoCompress(sessionID)
	}

	return m.store.Update(session)
}

func (m *SessionManager) GetMessages(sessionID string) ([]swarmapi.Message, []*CompressedSummary, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, nil, err
	}

	session.Conversation.mu.RLock()
	defer session.Conversation.mu.RUnlock()

	messagesCopy := make([]swarmapi.Message, len(session.Conversation.Messages))
	copy(messagesCopy, session.Conversation.Messages)

	summariesCopy := make([]*CompressedSummary, len(session.Conversation.CompressedSummaries))
	copy(summariesCopy, session.Conversation.CompressedSummaries)

	return messagesCopy, summariesCopy, nil
}

func (m *SessionManager) SetCurrentTask(sessionID string, task *swarmapi.Task) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.TaskState.mu.Lock()
	if session.TaskState.CurrentTask != nil {
		session.TaskState.History = append(session.TaskState.History, session.TaskState.CurrentTask)
	}
	session.TaskState.CurrentTask = task
	session.TaskState.mu.Unlock()

	session.UpdatedAt = time.Now()

	return m.store.Update(session)
}

func (m *SessionManager) GetCurrentTask(sessionID string) (*swarmapi.Task, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	session.TaskState.mu.RLock()
	defer session.TaskState.mu.RUnlock()

	return session.TaskState.CurrentTask, nil
}

func (m *SessionManager) AddBlocker(sessionID, blocker string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.TaskState.mu.Lock()
	session.TaskState.Blockers = append(session.TaskState.Blockers, blocker)
	session.TaskState.mu.Unlock()

	session.UpdatedAt = time.Now()

	return m.store.Update(session)
}

func (m *SessionManager) ClearBlockers(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.TaskState.mu.Lock()
	session.TaskState.Blockers = make([]string, 0)
	session.TaskState.mu.Unlock()

	session.UpdatedAt = time.Now()

	return m.store.Update(session)
}

func (m *SessionManager) UpdateContext(sessionID string, updateFn func(*ContextState)) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Context.mu.Lock()
	updateFn(session.Context)
	session.Context.mu.Unlock()

	session.UpdatedAt = time.Now()

	return m.store.Update(session)
}

func (m *SessionManager) RegisterAgent(sessionID, agentID, role string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.AgentCoord.mu.Lock()
	session.AgentCoord.ActiveAgents[agentID] = role
	session.AgentCoord.mu.Unlock()

	session.UpdatedAt = time.Now()

	return m.store.Update(session)
}

func (m *SessionManager) ListSessions(filter *SessionFilter) ([]*Session, error) {
	return m.store.List(filter)
}

func (m *SessionManager) DeleteSession(id string) error {
	if err := m.store.Delete(id); err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.activeSessions, id)
	m.mu.Unlock()

	return nil
}

func (m *SessionManager) Shutdown() {
	m.cancel()
}

func (m *SessionManager) autosaveLoop() {
	ticker := time.NewTicker(m.autosaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.saveAllActiveSessions()
		}
	}
}

func (m *SessionManager) saveAllActiveSessions() {
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.activeSessions))
	for _, s := range m.activeSessions {
		sessions = append(sessions, s)
	}
	m.mu.RUnlock()

	for _, session := range sessions {
		_ = m.store.Update(session)
	}
}

func (m *SessionManager) autoCompress(sessionID string) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Conversation.mu.Lock()
	defer session.Conversation.mu.Unlock()

	if len(session.Conversation.Messages) < m.compressionThreshold {
		return
	}

	toCompress := session.Conversation.Messages[:len(session.Conversation.Messages)-20]
	recent := session.Conversation.Messages[len(session.Conversation.Messages)-20:]

	text := messagesToText(toCompress)
	compressed, err := m.compressor.Compress(context.Background(), text, 5000)
	if err != nil {
		return
	}

	summary := &CompressedSummary{
		ID:           generateID(),
		Content:      compressed,
		CreatedAt:    time.Now(),
		MessageCount: len(toCompress),
		FromIndex:    session.Conversation.TotalMessages - len(toCompress),
		ToIndex:      session.Conversation.TotalMessages - 1,
	}

	session.Conversation.CompressedSummaries = append(
		session.Conversation.CompressedSummaries,
		summary,
	)
	session.Conversation.Messages = recent
	session.UpdatedAt = time.Now()

	_ = m.store.Update(session)
}

func messagesToText(messages []swarmapi.Message) string {
	var text string
	for _, msg := range messages {
		if str, ok := msg.Content.(string); ok {
			text += msg.Role + ": " + str + "\n"
		}
	}
	return text
}
