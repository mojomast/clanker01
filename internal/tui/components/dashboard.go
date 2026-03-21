package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/swarm-ai/swarm/internal/tui"
)

type DashboardModel struct {
	agents        []tui.Agent
	tasks         []tui.Task
	activity      []ActivityEntry
	width         int
	height        int
	theme         tui.Theme
	agentTable    table.Model
	activityIndex int
}

type ActivityEntry struct {
	Timestamp time.Time
	AgentID   string
	Message   string
	Type      ActivityType
}

type ActivityType int

const (
	ActivityTaskCreated ActivityType = iota
	ActivityTaskCompleted
	ActivityTaskFailed
	ActivityAgentStarted
	ActivityAgentStopped
	ActivityError
)

func NewDashboardModel(theme tui.Theme) *DashboardModel {
	columns := []table.Column{
		{Title: "Name", Width: 15},
		{Title: "Status", Width: 10},
		{Title: "Tasks", Width: 8},
		{Title: "Uptime", Width: 10},
		{Title: "CPU", Width: 8},
	}

	agentTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
		table.WithHeight(8),
	)

	return &DashboardModel{
		theme:      theme,
		agentTable: agentTable,
		activity:   []ActivityEntry{},
	}
}

func (m *DashboardModel) SetAgents(agents []tui.Agent) {
	m.agents = agents
	m.updateAgentTable()
}

func (m *DashboardModel) SetTasks(tasks []tui.Task) {
	m.tasks = tasks
}

func (m *DashboardModel) SetActivity(activity []ActivityEntry) {
	m.activity = activity
}

func (m *DashboardModel) AddActivity(entry ActivityEntry) {
	m.activity = append([]ActivityEntry{entry}, m.activity...)
	if len(m.activity) > 50 {
		m.activity = m.activity[:50]
	}
}

func (m *DashboardModel) Update(msg tea.Msg) (*DashboardModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tui.AgentListMsg:
		m.agents = msg.Agents
		m.updateAgentTable()

	case tui.TaskListMsg:
		m.tasks = msg.Tasks
	}

	m.agentTable, cmd = m.agentTable.Update(msg)
	return m, cmd
}

func (m *DashboardModel) updateAgentTable() {
	rows := make([]table.Row, 0, len(m.agents))
	for _, agent := range m.agents {
		statusIcon := "○"
		statusColor := m.theme.Colors.Muted
		if agent.Status == tui.StatusRunning {
			statusIcon = "●"
			statusColor = m.theme.Colors.Success
		} else if agent.Status == tui.StatusError {
			statusIcon = "✗"
			statusColor = m.theme.Colors.Error
		} else if agent.Status == tui.StatusPaused {
			statusIcon = "◌"
			statusColor = m.theme.Colors.Warning
		}

		rows = append(rows, table.Row{
			agent.Name,
			lipgloss.NewStyle().Foreground(statusColor).Render(
				fmt.Sprintf("%s %s", statusIcon, m.statusString(agent.Status)),
			),
			fmt.Sprintf("%d", agent.TasksCompleted),
			formatDuration(agent.Uptime),
			fmt.Sprintf("%.0f%%", agent.CPUUsage),
		})
	}

	columns := []table.Column{
		{Title: "Name", Width: 15},
		{Title: "Status", Width: 10},
		{Title: "Tasks", Width: 8},
		{Title: "Uptime", Width: 10},
		{Title: "CPU", Width: 8},
	}

	m.agentTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(8),
	)
	m.agentTable.SetStyles(m.tableStyles())
}

func (m *DashboardModel) View() string {
	agentOverview := m.renderAgentOverview()
	taskQueue := m.renderTaskQueue()
	activity := m.renderActivity()

	topSection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		taskQueue,
		activity,
	)

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		agentOverview,
		topSection,
	)

	return lipgloss.NewStyle().
		Width(m.width - 20).
		Height(m.height - 4).
		Render(layout)
}

func (m *DashboardModel) renderAgentOverview() string {
	header := m.theme.BoxHeaderStyle.Render("AGENT OVERVIEW")
	table := m.agentTable.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		table,
	)
}

func (m *DashboardModel) renderTaskQueue() string {
	var items []string

	maxTasks := 8
	for i, task := range m.tasks {
		if i >= maxTasks {
			break
		}

		statusIcon := "○"
		statusColor := m.theme.Colors.Muted
		if task.Status == tui.TaskStatusRunning {
			statusIcon = "●"
			statusColor = m.theme.Colors.Info
		} else if task.Status == tui.TaskStatusCompleted {
			statusIcon = "✓"
			statusColor = m.theme.Colors.Success
		} else if task.Status == tui.TaskStatusFailed {
			statusIcon = "✗"
			statusColor = m.theme.Colors.Error
		}

		priorityColor := m.theme.Colors.Foreground
		if task.Priority == tui.PriorityCritical {
			priorityColor = m.theme.Colors.Error
		} else if task.Priority == tui.PriorityHigh {
			priorityColor = m.theme.Colors.Warning
		}

		line := lipgloss.NewStyle().
			Foreground(statusColor).
			Render(fmt.Sprintf("%s", statusIcon))

		line += " " + lipgloss.NewStyle().
			Foreground(priorityColor).
			Width(20).
			Render(task.Name)

		items = append(items, line)
	}

	if len(items) == 0 {
		items = append(items, lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Render("No tasks in queue"))
	}

	content := strings.Join(items, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(30).
		Height(12)

	box := boxStyle.Render(content)

	header := m.theme.BoxHeaderStyle.Render("TASK QUEUE")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		box,
	)
}

func (m *DashboardModel) renderActivity() string {
	var items []string

	maxEntries := 8
	for i, entry := range m.activity {
		if i >= maxEntries {
			break
		}

		timestamp := entry.Timestamp.Format("15:04:05")
		line := fmt.Sprintf("%s %s %s",
			lipgloss.NewStyle().Foreground(m.theme.Colors.Muted).Render(timestamp),
			lipgloss.NewStyle().Foreground(m.theme.Colors.Info).Render(entry.AgentID),
			truncate(entry.Message, 25),
		)
		items = append(items, line)
	}

	if len(items) == 0 {
		items = append(items, lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Render("No recent activity"))
	}

	content := strings.Join(items, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(40).
		Height(12)

	box := boxStyle.Render(content)

	header := m.theme.BoxHeaderStyle.Render("RECENT ACTIVITY")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		box,
	)
}

func (m *DashboardModel) tableStyles() table.Styles {
	baseStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Foreground)

	headerStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Primary).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Background).
		Background(m.theme.Colors.Primary).
		Bold(true)

	return table.Styles{
		Header:   headerStyle,
		Cell:     baseStyle,
		Selected: selectedStyle,
	}
}

func (m *DashboardModel) statusString(status tui.AgentStatus) string {
	switch status {
	case tui.StatusIdle:
		return "idle"
	case tui.StatusRunning:
		return "running"
	case tui.StatusPaused:
		return "paused"
	case tui.StatusError:
		return "error"
	case tui.StatusStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
