package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type MessageType string

const (
	MessageTypeAgentUpdate MessageType = "agent_update"
	MessageTypeTaskEvent   MessageType = "task_event"
	MessageTypeLog         MessageType = "log"
	MessageTypeError       MessageType = "error"
	MessageTypePing        MessageType = "ping"
	MessageTypePong        MessageType = "pong"
)

type Message struct {
	Type      MessageType    `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      interface{}    `json:"data"`
	AgentID   string         `json:"agent_id,omitempty"`
	TaskID    string         `json:"task_id,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type AgentUpdate struct {
	ID        string            `json:"id"`
	Type      api.AgentType     `json:"type"`
	Name      string            `json:"name"`
	Status    api.AgentStatus   `json:"status"`
	Metrics   *api.AgentMetrics `json:"metrics,omitempty"`
	Health    *api.AgentHealth  `json:"health,omitempty"`
	Config    *api.AgentConfig  `json:"config,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

type TaskEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Status    api.TaskStatus  `json:"status"`
	Priority  int             `json:"priority"`
	AgentType api.AgentType   `json:"agent_type"`
	Progress  float64         `json:"progress"`
	Result    *api.TaskResult `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	AgentID   string    `json:"agent_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type PingMessage struct {
	Timestamp time.Time `json:"timestamp"`
}

type PongMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Latency   int64     `json:"latency_ms"`
}

type BroadcastTarget struct {
	All      bool
	UserIDs  []string
	AgentIDs []string
	TaskIDs  []string
}

func NewAgentUpdate(agent api.Agent, metrics *api.AgentMetrics, health *api.AgentHealth) *Message {
	return &Message{
		Type:      MessageTypeAgentUpdate,
		Timestamp: time.Now(),
		AgentID:   agent.ID(),
		Data: &AgentUpdate{
			ID:        agent.ID(),
			Type:      agent.Type(),
			Name:      agent.Name(),
			Status:    agent.Status(),
			Metrics:   metrics,
			Health:    health,
			Timestamp: time.Now(),
		},
	}
}

func NewTaskEvent(task *api.Task, eventType string, result *api.TaskResult, err error) *Message {
	if task == nil {
		return nil
	}

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	return &Message{
		Type:      MessageTypeTaskEvent,
		Timestamp: time.Now(),
		TaskID:    task.ID,
		Data: &TaskEvent{
			ID:        task.ID,
			Type:      eventType,
			Status:    task.Status,
			Priority:  task.Priority,
			AgentType: task.AgentType,
			Progress:  task.Progress,
			Result:    result,
			Error:     errorMsg,
			Timestamp: time.Now(),
		},
	}
}

func NewLogEntry(level, message string, agentID, taskID string) *Message {
	return &Message{
		Type:      MessageTypeLog,
		Timestamp: time.Now(),
		AgentID:   agentID,
		TaskID:    taskID,
		Data: &LogEntry{
			Level:     level,
			Message:   message,
			AgentID:   agentID,
			TaskID:    taskID,
			Timestamp: time.Now(),
		},
	}
}

func NewError(code, message, details string) *Message {
	return &Message{
		Type:      MessageTypeError,
		Timestamp: time.Now(),
		Data: &ErrorMessage{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

func NewPing() *Message {
	return &Message{
		Type:      MessageTypePing,
		Timestamp: time.Now(),
		Data:      &PingMessage{Timestamp: time.Now()},
	}
}

func NewPong(pingTime time.Time) *Message {
	return &Message{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
		Data: &PongMessage{
			Timestamp: time.Now(),
			Latency:   time.Since(pingTime).Milliseconds(),
		},
	}
}

func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type Subscription struct {
	mu           sync.RWMutex
	AgentIDs     map[string]bool
	TaskIDs      map[string]bool
	MessageTypes map[MessageType]bool
}

func NewSubscription() *Subscription {
	return &Subscription{
		AgentIDs:     make(map[string]bool),
		TaskIDs:      make(map[string]bool),
		MessageTypes: make(map[MessageType]bool),
	}
}

func (s *Subscription) AddAgentID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AgentIDs[id] = true
}

func (s *Subscription) RemoveAgentID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.AgentIDs, id)
}

func (s *Subscription) AddTaskID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TaskIDs[id] = true
}

func (s *Subscription) RemoveTaskID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.TaskIDs, id)
}

func (s *Subscription) AddMessageType(msgType MessageType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MessageTypes[msgType] = true
}

func (s *Subscription) RemoveMessageType(msgType MessageType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.MessageTypes, msgType)
}

func (s *Subscription) SubscribeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AgentIDs = nil
	s.TaskIDs = nil
	s.MessageTypes = nil
}

func (s *Subscription) ShouldReceive(msg *Message) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.MessageTypes != nil && len(s.MessageTypes) > 0 && !s.MessageTypes[msg.Type] {
		return false
	}

	if s.AgentIDs != nil && len(s.AgentIDs) > 0 && msg.AgentID != "" && !s.AgentIDs[msg.AgentID] {
		return false
	}

	if s.TaskIDs != nil && len(s.TaskIDs) > 0 && msg.TaskID != "" && !s.TaskIDs[msg.TaskID] {
		return false
	}

	return true
}
