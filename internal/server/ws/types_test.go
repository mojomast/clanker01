package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/swarm-ai/swarm/pkg/api"
)

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		name MessageType
		want string
	}{
		{MessageTypeAgentUpdate, "agent_update"},
		{MessageTypeTaskEvent, "task_event"},
		{MessageTypeLog, "log"},
		{MessageTypeError, "error"},
		{MessageTypePing, "ping"},
		{MessageTypePong, "pong"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.name))
		})
	}
}

func TestNewAgentUpdate(t *testing.T) {
	agent := &mockAgent{id: "agent-1", agentType: api.AgentTypeCoder, name: "test-agent", status: api.AgentStatusReady}
	metrics := &api.AgentMetrics{TasksCompleted: 10}
	health := &api.AgentHealth{Status: "healthy"}

	msg := NewAgentUpdate(agent, metrics, health)

	assert.NotNil(t, msg)
	assert.Equal(t, MessageTypeAgentUpdate, msg.Type)
	assert.Equal(t, "agent-1", msg.AgentID)
	assert.False(t, msg.Timestamp.IsZero())

	data, ok := msg.Data.(*AgentUpdate)
	require.True(t, ok)
	assert.Equal(t, "agent-1", data.ID)
	assert.Equal(t, api.AgentTypeCoder, data.Type)
	assert.Equal(t, "test-agent", data.Name)
	assert.Equal(t, api.AgentStatusReady, data.Status)
	assert.NotNil(t, data.Metrics)
	assert.NotNil(t, data.Health)
}

func TestNewTaskEvent(t *testing.T) {
	task := &api.Task{
		ID:        "task-1",
		Status:    api.TaskStatusRunning,
		Priority:  5,
		AgentType: api.AgentTypeCoder,
		Progress:  0.5,
	}

	msg := NewTaskEvent(task, "started", nil, nil)

	assert.NotNil(t, msg)
	assert.Equal(t, MessageTypeTaskEvent, msg.Type)
	assert.Equal(t, "task-1", msg.TaskID)
	assert.False(t, msg.Timestamp.IsZero())

	data, ok := msg.Data.(*TaskEvent)
	require.True(t, ok)
	assert.Equal(t, "task-1", data.ID)
	assert.Equal(t, "started", data.Type)
	assert.Equal(t, api.TaskStatusRunning, data.Status)
	assert.Equal(t, 5, data.Priority)
	assert.Equal(t, api.AgentTypeCoder, data.AgentType)
	assert.Equal(t, 0.5, data.Progress)
}

func TestNewTaskEventWithError(t *testing.T) {
	task := &api.Task{
		ID:       "task-1",
		Status:   api.TaskStatusFailed,
		Priority: 5,
	}

	err := assert.AnError
	msg := NewTaskEvent(task, "failed", nil, err)

	assert.NotNil(t, msg)
	assert.Equal(t, MessageTypeTaskEvent, msg.Type)

	data, ok := msg.Data.(*TaskEvent)
	require.True(t, ok)
	assert.Equal(t, "assert.AnError general error for testing", data.Error)
}

func TestNewLogEntry(t *testing.T) {
	msg := NewLogEntry("info", "test message", "agent-1", "task-1")

	assert.NotNil(t, msg)
	assert.Equal(t, MessageTypeLog, msg.Type)
	assert.Equal(t, "agent-1", msg.AgentID)
	assert.Equal(t, "task-1", msg.TaskID)

	data, ok := msg.Data.(*LogEntry)
	require.True(t, ok)
	assert.Equal(t, "info", data.Level)
	assert.Equal(t, "test message", data.Message)
	assert.Equal(t, "agent-1", data.AgentID)
	assert.Equal(t, "task-1", data.TaskID)
	assert.False(t, data.Timestamp.IsZero())
}

func TestNewError(t *testing.T) {
	msg := NewError("ERR001", "test error", "additional details")

	assert.NotNil(t, msg)
	assert.Equal(t, MessageTypeError, msg.Type)

	data, ok := msg.Data.(*ErrorMessage)
	require.True(t, ok)
	assert.Equal(t, "ERR001", data.Code)
	assert.Equal(t, "test error", data.Message)
	assert.Equal(t, "additional details", data.Details)
}

func TestNewPing(t *testing.T) {
	msg := NewPing()

	assert.NotNil(t, msg)
	assert.Equal(t, MessageTypePing, msg.Type)
	assert.False(t, msg.Timestamp.IsZero())

	data, ok := msg.Data.(*PingMessage)
	require.True(t, ok)
	assert.False(t, data.Timestamp.IsZero())
}

func TestNewPong(t *testing.T) {
	pingTime := time.Now().Add(-100 * time.Millisecond)
	msg := NewPong(pingTime)

	assert.NotNil(t, msg)
	assert.Equal(t, MessageTypePong, msg.Type)
	assert.False(t, msg.Timestamp.IsZero())

	data, ok := msg.Data.(*PongMessage)
	require.True(t, ok)
	assert.False(t, data.Timestamp.IsZero())
	assert.Greater(t, data.Latency, int64(0))
	assert.Less(t, data.Latency, int64(time.Second))
}

func TestMessageToJSON(t *testing.T) {
	msg := NewLogEntry("info", "test message", "agent-1", "task-1")

	data, err := msg.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, string(MessageTypeLog), parsed["type"])
	assert.Equal(t, "agent-1", parsed["agent_id"])
	assert.Equal(t, "task-1", parsed["task_id"])
}

func TestParseMessage(t *testing.T) {
	originalMsg := NewLogEntry("info", "test message", "agent-1", "task-1")
	data, err := originalMsg.ToJSON()
	require.NoError(t, err)

	parsedMsg, err := ParseMessage(data)
	require.NoError(t, err)

	assert.Equal(t, originalMsg.Type, parsedMsg.Type)
	assert.Equal(t, originalMsg.AgentID, parsedMsg.AgentID)
	assert.Equal(t, originalMsg.TaskID, parsedMsg.TaskID)
}

func TestParseMessageInvalid(t *testing.T) {
	data := []byte("invalid json")

	msg, err := ParseMessage(data)
	assert.Error(t, err)
	assert.Nil(t, msg)
}

func TestSubscription(t *testing.T) {
	sub := NewSubscription()

	assert.NotNil(t, sub)
	assert.Empty(t, sub.AgentIDs)
	assert.Empty(t, sub.TaskIDs)
	assert.Empty(t, sub.MessageTypes)
}

func TestSubscriptionAgentIDs(t *testing.T) {
	sub := NewSubscription()

	sub.AddAgentID("agent-1")
	sub.AddAgentID("agent-2")

	assert.True(t, sub.ShouldReceive(&Message{AgentID: "agent-1"}))
	assert.True(t, sub.ShouldReceive(&Message{AgentID: "agent-2"}))
	assert.False(t, sub.ShouldReceive(&Message{AgentID: "agent-3"}))

	sub.RemoveAgentID("agent-1")
	assert.False(t, sub.ShouldReceive(&Message{AgentID: "agent-1"}))
	assert.True(t, sub.ShouldReceive(&Message{AgentID: "agent-2"}))
}

func TestSubscriptionTaskIDs(t *testing.T) {
	sub := NewSubscription()

	sub.AddTaskID("task-1")
	sub.AddTaskID("task-2")

	assert.True(t, sub.ShouldReceive(&Message{TaskID: "task-1"}))
	assert.True(t, sub.ShouldReceive(&Message{TaskID: "task-2"}))
	assert.False(t, sub.ShouldReceive(&Message{TaskID: "task-3"}))

	sub.RemoveTaskID("task-1")
	assert.False(t, sub.ShouldReceive(&Message{TaskID: "task-1"}))
	assert.True(t, sub.ShouldReceive(&Message{TaskID: "task-2"}))
}

func TestSubscriptionMessageTypes(t *testing.T) {
	sub := NewSubscription()

	sub.AddMessageType(MessageTypeLog)
	sub.AddMessageType(MessageTypeError)

	assert.True(t, sub.ShouldReceive(&Message{Type: MessageTypeLog}))
	assert.True(t, sub.ShouldReceive(&Message{Type: MessageTypeError}))
	assert.False(t, sub.ShouldReceive(&Message{Type: MessageTypePing}))

	sub.RemoveMessageType(MessageTypeLog)
	assert.False(t, sub.ShouldReceive(&Message{Type: MessageTypeLog}))
	assert.True(t, sub.ShouldReceive(&Message{Type: MessageTypeError}))
}

func TestSubscribeAll(t *testing.T) {
	sub := NewSubscription()

	sub.SubscribeAll()

	assert.True(t, sub.ShouldReceive(&Message{Type: MessageTypeLog}))
	assert.True(t, sub.ShouldReceive(&Message{Type: MessageTypePing}))
	assert.True(t, sub.ShouldReceive(&Message{AgentID: "agent-1"}))
	assert.True(t, sub.ShouldReceive(&Message{TaskID: "task-1"}))
}

func TestSubscriptionCombined(t *testing.T) {
	sub := NewSubscription()

	sub.AddMessageType(MessageTypeLog)
	sub.AddAgentID("agent-1")
	sub.AddTaskID("task-1")

	assert.True(t, sub.ShouldReceive(&Message{Type: MessageTypeLog, AgentID: "agent-1", TaskID: "task-1"}))
	assert.False(t, sub.ShouldReceive(&Message{Type: MessageTypePing, AgentID: "agent-1", TaskID: "task-1"}))
	assert.False(t, sub.ShouldReceive(&Message{Type: MessageTypeLog, AgentID: "agent-2", TaskID: "task-1"}))
	assert.False(t, sub.ShouldReceive(&Message{Type: MessageTypeLog, AgentID: "agent-1", TaskID: "task-2"}))
}

type mockAgent struct {
	id        string
	agentType api.AgentType
	name      string
	status    api.AgentStatus
}

func (m *mockAgent) ID() string {
	return m.id
}

func (m *mockAgent) Type() api.AgentType {
	return m.agentType
}

func (m *mockAgent) Name() string {
	return m.name
}

func (m *mockAgent) Status() api.AgentStatus {
	return m.status
}

func (m *mockAgent) Initialize(context.Context, *api.AgentConfig) error {
	return nil
}

func (m *mockAgent) Start(context.Context) error {
	return nil
}

func (m *mockAgent) Stop(context.Context) error {
	return nil
}

func (m *mockAgent) Pause(context.Context) error {
	return nil
}

func (m *mockAgent) Resume(context.Context) error {
	return nil
}

func (m *mockAgent) Execute(context.Context, *api.Task) (*api.AgentResult, error) {
	return nil, nil
}

func (m *mockAgent) SendMessage(context.Context, *api.AgentMessage) error {
	return nil
}

func (m *mockAgent) ReceiveMessage() <-chan *api.AgentMessage {
	return nil
}

func (m *mockAgent) Broadcast(context.Context, *api.AgentMessage) error {
	return nil
}

func (m *mockAgent) CurrentTask() *api.Task {
	return nil
}

func (m *mockAgent) Metrics() *api.AgentMetrics {
	return nil
}

func (m *mockAgent) Health() *api.AgentHealth {
	return nil
}
