package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTickMsg(t *testing.T) {
	now := time.Now()
	msg := TickMsg(now)

	assert.Equal(t, now, time.Time(msg))
}

func TestViewChangeMsg(t *testing.T) {
	msg := ViewChangeMsg{View: ViewAgents}

	assert.Equal(t, ViewAgents, msg.View)
}

func TestFocusChangeMsg(t *testing.T) {
	msg := FocusChangeMsg{Focus: FocusModal}

	assert.Equal(t, FocusModal, msg.Focus)
}

func TestAgentListMsg(t *testing.T) {
	agents := []Agent{
		{ID: "agent-1", Name: "Agent 1"},
		{ID: "agent-2", Name: "Agent 2"},
	}

	msg := AgentListMsg{
		Agents: agents,
		Err:    nil,
	}

	assert.Len(t, msg.Agents, 2)
	assert.Equal(t, "agent-1", msg.Agents[0].ID)
	assert.Nil(t, msg.Err)
}

func TestAgentListMsgWithError(t *testing.T) {
	err := assert.AnError
	msg := AgentListMsg{
		Agents: []Agent{},
		Err:    err,
	}

	assert.Empty(t, msg.Agents)
	assert.Equal(t, err, msg.Err)
}

func TestAgentUpdateMsg(t *testing.T) {
	agent := Agent{
		ID:     "agent-1",
		Name:   "Updated Agent",
		Status: StatusRunning,
	}

	msg := AgentUpdateMsg{Agent: agent}

	assert.Equal(t, "agent-1", msg.Agent.ID)
	assert.Equal(t, "Updated Agent", msg.Agent.Name)
	assert.Equal(t, StatusRunning, msg.Agent.Status)
}

func TestAgentAddedMsg(t *testing.T) {
	agent := Agent{
		ID:     "agent-new",
		Name:   "New Agent",
		Status: StatusIdle,
	}

	msg := AgentAddedMsg{
		Agent: agent,
		Err:   nil,
	}

	assert.Equal(t, "agent-new", msg.Agent.ID)
	assert.Nil(t, msg.Err)
}

func TestAgentAddedMsgWithError(t *testing.T) {
	err := assert.AnError
	msg := AgentAddedMsg{
		Agent: Agent{},
		Err:   err,
	}

	assert.Equal(t, err, msg.Err)
}

func TestAgentRemovedMsg(t *testing.T) {
	msg := AgentRemovedMsg{
		ID:  "agent-1",
		Err: nil,
	}

	assert.Equal(t, "agent-1", msg.ID)
	assert.Nil(t, msg.Err)
}

func TestTaskListMsg(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Name: "Task 1"},
		{ID: "task-2", Name: "Task 2"},
	}

	msg := TaskListMsg{
		Tasks: tasks,
		Err:   nil,
	}

	assert.Len(t, msg.Tasks, 2)
	assert.Equal(t, "task-1", msg.Tasks[0].ID)
	assert.Nil(t, msg.Err)
}

func TestTaskListMsgWithError(t *testing.T) {
	err := assert.AnError
	msg := TaskListMsg{
		Tasks: []Task{},
		Err:   err,
	}

	assert.Empty(t, msg.Tasks)
	assert.Equal(t, err, msg.Err)
}

func TestTaskUpdateMsg(t *testing.T) {
	task := Task{
		ID:     "task-1",
		Name:   "Updated Task",
		Status: TaskStatusRunning,
	}

	msg := TaskUpdateMsg{Task: task}

	assert.Equal(t, "task-1", msg.Task.ID)
	assert.Equal(t, "Updated Task", msg.Task.Name)
	assert.Equal(t, TaskStatusRunning, msg.Task.Status)
}

func TestTaskCreatedMsg(t *testing.T) {
	task := Task{
		ID:     "task-new",
		Name:   "New Task",
		Status: TaskStatusPending,
	}

	msg := TaskCreatedMsg{
		Task: task,
		Err:  nil,
	}

	assert.Equal(t, "task-new", msg.Task.ID)
	assert.Nil(t, msg.Err)
}

func TestTaskCreatedMsgWithError(t *testing.T) {
	err := assert.AnError
	msg := TaskCreatedMsg{
		Task: Task{},
		Err:  err,
	}

	assert.Equal(t, err, msg.Err)
}

func TestTaskStatusMsg(t *testing.T) {
	msg := TaskStatusMsg{
		ID:     "task-1",
		Status: TaskStatusCompleted,
		Err:    nil,
	}

	assert.Equal(t, "task-1", msg.ID)
	assert.Equal(t, TaskStatusCompleted, msg.Status)
	assert.Nil(t, msg.Err)
}

func TestLogStreamMsg(t *testing.T) {
	entries := []LogEntry{
		{
			Timestamp: time.Now(),
			Level:     LevelInfo,
			AgentID:   "agent-1",
			Message:   "Log entry 1",
		},
		{
			Timestamp: time.Now(),
			Level:     LevelWarn,
			AgentID:   "agent-2",
			Message:   "Log entry 2",
		},
	}

	msg := LogStreamMsg{Entries: entries}

	assert.Len(t, msg.Entries, 2)
	assert.Equal(t, "Log entry 1", msg.Entries[0].Message)
	assert.Equal(t, "Log entry 2", msg.Entries[1].Message)
}

func TestLogClearedMsg(t *testing.T) {
	msg := LogClearedMsg{}

	assert.NotNil(t, msg)
}

func TestConfigLoadedMsg(t *testing.T) {
	config := Config{
		APIEndpoint: "https://api.example.com",
		MaxAgents:   10,
	}

	msg := ConfigLoadedMsg{
		Config: config,
		Err:    nil,
	}

	assert.Equal(t, "https://api.example.com", msg.Config.APIEndpoint)
	assert.Equal(t, 10, msg.Config.MaxAgents)
	assert.Nil(t, msg.Err)
}

func TestConfigLoadedMsgWithError(t *testing.T) {
	err := assert.AnError
	msg := ConfigLoadedMsg{
		Config: Config{},
		Err:    err,
	}

	assert.Equal(t, err, msg.Err)
}

func TestConfigSavedMsg(t *testing.T) {
	msg := ConfigSavedMsg{
		Err: nil,
	}

	assert.Nil(t, msg.Err)
}

func TestConfigSavedMsgWithError(t *testing.T) {
	err := assert.AnError
	msg := ConfigSavedMsg{
		Err: err,
	}

	assert.Equal(t, err, msg.Err)
}

func TestModalOpenMsg(t *testing.T) {
	initialData := map[string]interface{}{
		"name":  "test",
		"value": 123,
	}

	msg := ModalOpenMsg{
		Type:    ModalAddAgent,
		Initial: initialData,
	}

	assert.Equal(t, ModalAddAgent, msg.Type)
	assert.Equal(t, "test", msg.Initial["name"])
	assert.Equal(t, 123, msg.Initial["value"])
}

func TestModalCloseMsg(t *testing.T) {
	data := map[string]interface{}{
		"result": "success",
	}

	msg := ModalCloseMsg{
		Result: ModalResult{
			Confirmed: true,
			Data:      data,
		},
	}

	assert.True(t, msg.Result.Confirmed)
	assert.Equal(t, "success", msg.Result.Data["result"])
}

func TestErrorMsg(t *testing.T) {
	err := assert.AnError
	timestamp := time.Now()

	msg := ErrorMsg{
		Err:       err,
		Timestamp: timestamp,
		Context:   "test context",
	}

	assert.Equal(t, err, msg.Err)
	assert.Equal(t, timestamp, msg.Timestamp)
	assert.Equal(t, "test context", msg.Context)
}

func TestAllMessageTypes(t *testing.T) {
	tests := []struct {
		name   string
		create func() interface{}
	}{
		{"TickMsg", func() interface{} { return TickMsg(time.Now()) }},
		{"ViewChangeMsg", func() interface{} { return ViewChangeMsg{View: ViewDashboard} }},
		{"FocusChangeMsg", func() interface{} { return FocusChangeMsg{Focus: FocusMain} }},
		{"AgentListMsg", func() interface{} { return AgentListMsg{Agents: []Agent{}, Err: nil} }},
		{"AgentUpdateMsg", func() interface{} { return AgentUpdateMsg{Agent: Agent{}} }},
		{"AgentAddedMsg", func() interface{} { return AgentAddedMsg{Agent: Agent{}, Err: nil} }},
		{"AgentRemovedMsg", func() interface{} { return AgentRemovedMsg{ID: "", Err: nil} }},
		{"TaskListMsg", func() interface{} { return TaskListMsg{Tasks: []Task{}, Err: nil} }},
		{"TaskUpdateMsg", func() interface{} { return TaskUpdateMsg{Task: Task{}} }},
		{"TaskCreatedMsg", func() interface{} { return TaskCreatedMsg{Task: Task{}, Err: nil} }},
		{"TaskStatusMsg", func() interface{} { return TaskStatusMsg{ID: "", Status: TaskStatusPending, Err: nil} }},
		{"LogStreamMsg", func() interface{} { return LogStreamMsg{Entries: []LogEntry{}} }},
		{"LogClearedMsg", func() interface{} { return LogClearedMsg{} }},
		{"ConfigLoadedMsg", func() interface{} { return ConfigLoadedMsg{Config: Config{}, Err: nil} }},
		{"ConfigSavedMsg", func() interface{} { return ConfigSavedMsg{Err: nil} }},
		{"ModalOpenMsg", func() interface{} { return ModalOpenMsg{Type: ModalNone, Initial: nil} }},
		{"ModalCloseMsg", func() interface{} { return ModalCloseMsg{Result: ModalResult{}} }},
		{"ErrorMsg", func() interface{} { return ErrorMsg{Err: nil, Timestamp: time.Now(), Context: ""} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.create()
			assert.NotNil(t, msg)
		})
	}
}
