package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/swarm-ai/swarm/internal/tui"
)

type AgentViewModel struct {
	agent        *tui.Agent
	recentOutput []OutputLine
	width        int
	height       int
	theme        tui.Theme
	outputIndex  int
	metricsIndex int
	progressBar  progress.Model
}

type OutputLine struct {
	Timestamp time.Time
	Message   string
	Level     OutputLevel
}

type OutputLevel int

const (
	OutputDebug OutputLevel = iota
	OutputInfo
	OutputWarn
	OutputError
)

func NewAgentViewModel(theme tui.Theme) *AgentViewModel {
	return &AgentViewModel{
		theme:        theme,
		recentOutput: []OutputLine{},
		progressBar:  progress.New(progress.WithDefaultGradient()),
	}
}

func (m *AgentViewModel) SetAgent(agent *tui.Agent) {
	m.agent = agent
}

func (m *AgentViewModel) SetOutput(output []OutputLine) {
	m.recentOutput = output
}

func (m *AgentViewModel) AddOutput(line OutputLine) {
	m.recentOutput = append(m.recentOutput, line)
	if len(m.recentOutput) > 100 {
		m.recentOutput = m.recentOutput[len(m.recentOutput)-100:]
	}
}

func (m *AgentViewModel) Update(msg tea.Msg) (*AgentViewModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tui.AgentUpdateMsg:
		m.agent = &msg.Agent
	}

	var model tea.Model = m.progressBar
	model, cmd = model.Update(msg)
	m.progressBar = model.(progress.Model)
	return m, cmd
}

func (m *AgentViewModel) View() string {
	if m.agent == nil {
		return lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Render("No agent selected")
	}

	header := m.renderHeader()
	statusSection := m.renderStatusSection()
	metricsAndTask := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderMetrics(),
		m.renderCurrentTask(),
	)
	outputSection := m.renderOutput()

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		statusSection,
		metricsAndTask,
		outputSection,
	)

	return lipgloss.NewStyle().
		Width(m.width - 20).
		Height(m.height - 4).
		Render(layout)
}

func (m *AgentViewModel) renderHeader() string {
	closeBtn := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render("[x] Close")

	title := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Primary).
		Bold(true).
		Render(fmt.Sprintf("AGENT: %s", m.agent.Name))

	rightSection := lipgloss.NewStyle().
		Render(fmt.Sprintf("◄ ► %s", closeBtn))

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		title,
		lipgloss.PlaceHorizontal(
			m.width-20-lipgloss.Width(title)-lipgloss.Width(rightSection),
			lipgloss.Right,
			rightSection,
		),
	)
}

func (m *AgentViewModel) renderStatusSection() string {
	statusColor := m.theme.Colors.Muted
	if m.agent.Status == tui.StatusRunning {
		statusColor = m.theme.Colors.Success
	} else if m.agent.Status == tui.StatusError {
		statusColor = m.theme.Colors.Error
	} else if m.agent.Status == tui.StatusPaused {
		statusColor = m.theme.Colors.Warning
	}

	statusText := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Render(fmt.Sprintf("STATUS: %s", m.statusString(m.agent.Status)))

	uptimeText := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Foreground).
		Render(fmt.Sprintf("UPTIME: %s", formatDuration(m.agent.Uptime)))

	content := lipgloss.JoinHorizontal(
		lipgloss.Left,
		statusText,
		lipgloss.NewStyle().Width(20).Render(""),
		uptimeText,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(m.width - 22).
		Height(3)

	return boxStyle.Render(content)
}

func (m *AgentViewModel) renderMetrics() string {
	totalTasks := m.agent.TasksCompleted + m.agent.TasksFailed
	successRate := float64(0)
	if totalTasks > 0 {
		successRate = float64(m.agent.TasksCompleted) / float64(totalTasks) * 100
	}

	items := []struct {
		label string
		value string
	}{
		{"Tasks Completed", fmt.Sprintf("%d", m.agent.TasksCompleted)},
		{"Tasks Failed", fmt.Sprintf("%d", m.agent.TasksFailed)},
		{"Avg Duration", fmt.Sprintf("%s", avgDurationString(m.agent.TasksCompleted, m.agent.Uptime))},
		{"Success Rate", fmt.Sprintf("%.1f%%", successRate)},
		{"Tokens Used", formatTokens(m.agent.TokensUsed)},
	}

	var rows []string
	for _, item := range items {
		labelStyle := lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Width(18).
			Render(item.label)

		valueStyle := lipgloss.NewStyle().
			Foreground(m.theme.Colors.Foreground).
			Render(item.value)

		rows = append(rows, labelStyle+": "+valueStyle)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(28).
		Height(7)

	box := boxStyle.Render(content)
	header := m.theme.BoxHeaderStyle.Render("METRICS")

	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (m *AgentViewModel) renderCurrentTask() string {
	var content string

	if m.agent.CurrentTask == nil {
		content = lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Render("No active task")
	} else {
		task := m.agent.CurrentTask

		items := []struct {
			label string
			value string
		}{
			{"ID", task.ID},
			{"Type", string(task.Type)},
			{"Priority", m.priorityString(task.Priority)},
			{"Started", task.StartedAt.Format("15:04:05")},
		}

		var rows []string
		for _, item := range items {
			labelStyle := lipgloss.NewStyle().
				Foreground(m.theme.Colors.Muted).
				Width(10).
				Render(item.label)

			valueStyle := lipgloss.NewStyle().
				Foreground(m.theme.Colors.Foreground).
				Render(item.value)

			rows = append(rows, labelStyle+": "+valueStyle)
		}

		progressSection := m.renderProgressBar(task.Progress)

		content = lipgloss.JoinVertical(lipgloss.Left, rows...)
		content = lipgloss.JoinVertical(lipgloss.Left, content, progressSection)
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(40).
		Height(7)

	box := boxStyle.Render(content)
	header := m.theme.BoxHeaderStyle.Render("CURRENT TASK")

	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (m *AgentViewModel) renderProgressBar(progress float64) string {
	if m.agent.CurrentTask == nil {
		return ""
	}

	bar := lipgloss.NewStyle().
		Width(36).
		Render(m.progressBar.ViewAs(progress / 100))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(m.theme.Colors.Muted).Render("Progress:"),
		bar,
	)
}

func (m *AgentViewModel) renderOutput() string {
	header := m.theme.BoxHeaderStyle.Render("RECENT OUTPUT")

	var lines []string
	maxLines := 10
	startIndex := len(m.recentOutput) - maxLines
	if startIndex < 0 {
		startIndex = 0
	}

	for i := startIndex; i < len(m.recentOutput); i++ {
		line := m.recentOutput[i]
		timestamp := line.Timestamp.Format("15:04:05")

		levelColor := m.theme.Colors.Foreground
		levelIcon := ""
		switch line.Level {
		case OutputInfo:
			levelColor = m.theme.Colors.Info
		case OutputWarn:
			levelColor = m.theme.Colors.Warning
			levelIcon = "⚠ "
		case OutputError:
			levelColor = m.theme.Colors.Error
			levelIcon = "✗ "
		}

		formattedLine := fmt.Sprintf("[%s] %s%s",
			lipgloss.NewStyle().Foreground(m.theme.Colors.Muted).Render(timestamp),
			lipgloss.NewStyle().Foreground(levelColor).Render(levelIcon),
			line.Message,
		)

		lines = append(lines, formattedLine)
	}

	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Render("No recent output"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(m.width - 22).
		Height(12)

	box := boxStyle.Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (m *AgentViewModel) statusString(status tui.AgentStatus) string {
	switch status {
	case tui.StatusIdle:
		return "IDLE"
	case tui.StatusRunning:
		return "RUNNING"
	case tui.StatusPaused:
		return "PAUSED"
	case tui.StatusError:
		return "ERROR"
	case tui.StatusStopped:
		return "STOPPED"
	default:
		return "UNKNOWN"
	}
}

func (m *AgentViewModel) priorityString(priority tui.Priority) string {
	switch priority {
	case tui.PriorityLow:
		return "low"
	case tui.PriorityMedium:
		return "medium"
	case tui.PriorityHigh:
		return "high"
	case tui.PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

func avgDurationString(tasks int, uptime time.Duration) string {
	if tasks == 0 {
		return "N/A"
	}
	avg := uptime / time.Duration(tasks)
	if avg.Seconds() < 60 {
		return fmt.Sprintf("%.0fs", avg.Seconds())
	}
	return fmt.Sprintf("%.1fm", avg.Minutes())
}

func formatTokens(tokens int64) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	} else if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}
