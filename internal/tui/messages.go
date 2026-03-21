package tui

import (
	"time"
)

type TickMsg time.Time

type ViewChangeMsg struct {
	View ViewID
}

type FocusChangeMsg struct {
	Focus FocusArea
}

type AgentListMsg struct {
	Agents []Agent
	Err    error
}

type AgentUpdateMsg struct {
	Agent Agent
}

type AgentAddedMsg struct {
	Agent Agent
	Err   error
}

type AgentRemovedMsg struct {
	ID  string
	Err error
}

type TaskListMsg struct {
	Tasks []Task
	Err   error
}

type TaskUpdateMsg struct {
	Task Task
}

type TaskCreatedMsg struct {
	Task Task
	Err  error
}

type TaskStatusMsg struct {
	ID     string
	Status TaskStatus
	Err    error
}

type LogStreamMsg struct {
	Entries []LogEntry
}

type LogClearedMsg struct{}

type ConfigLoadedMsg struct {
	Config Config
	Err    error
}

type ConfigSavedMsg struct {
	Err error
}

type ModalOpenMsg struct {
	Type    ModalType
	Initial map[string]interface{}
}

type ModalCloseMsg struct {
	Result ModalResult
}

type ErrorMsg struct {
	Err       error
	Timestamp time.Time
	Context   string
}
