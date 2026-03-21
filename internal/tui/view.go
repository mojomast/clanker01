package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.loading {
		return fmt.Sprintf("\n   %s Loading...\n\n", m.spinner.View())
	}

	header := m.renderHeader()
	sidebar := m.renderSidebar()
	main := m.renderMain()
	footer := m.renderFooter()

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		main,
	)

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		footer,
	)

	if m.modal.active {
		layout = m.renderModal(layout)
	}

	return layout
}

func (m Model) renderHeader() string {
	left := m.theme.HeaderStyle.Left.Render(
		fmt.Sprintf("SWARM %s", m.header.version),
	)

	center := m.theme.HeaderStyle.Center.Render(
		fmt.Sprintf("Agents: %d │ Tasks: %d │ CPU: %.0f%% │ MEM: %s",
			m.header.agentCount,
			m.header.taskCount,
			m.header.cpuUsage,
			formatMemory(m.header.memUsage),
		),
	)

	right := m.theme.HeaderStyle.Right.Render(
		m.header.time.Format("15:04"),
	)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		lipgloss.PlaceHorizontal(
			m.width-lipgloss.Width(left)-lipgloss.Width(right),
			lipgloss.Center,
			center,
		),
		right,
	)
}

func (m Model) renderSidebar() string {
	var items []string

	for i, item := range m.sidebar.items {
		style := m.theme.SidebarStyle.Item
		if i == m.sidebar.selected {
			style = m.theme.SidebarStyle.Selected
		}

		label := fmt.Sprintf("%s %s", item.icon, item.label)
		if item.badge > 0 {
			label += fmt.Sprintf(" (%d)", item.badge)
		}

		items = append(items, style.Render(label))
	}

	return m.theme.BaseStyle.
		Width(m.sidebar.width).
		Height(m.height - 2).
		Render(strings.Join(items, "\n"))
}

func (m Model) renderMain() string {
	switch m.view {
	case ViewDashboard:
		return m.renderDashboard()
	case ViewAgents:
		return m.renderAgents()
	case ViewAgentDetail:
		return m.renderAgentDetail()
	case ViewTasks:
		return m.renderTasks()
	case ViewTaskDetail:
		return m.renderTaskDetail()
	case ViewLogs:
		return m.renderLogs()
	case ViewConfig:
		return m.renderConfig()
	default:
		return "Unknown view"
	}
}

func (m Model) renderDashboard() string {
	mainWidth := m.width - m.sidebar.width

	// Summary section
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7aa2f7")).
		Render("DASHBOARD")

	agentSummary := fmt.Sprintf("Agents: %d total", len(m.agents))
	runningAgents := 0
	idleAgents := 0
	errorAgents := 0
	for _, a := range m.agents {
		switch a.Status {
		case StatusRunning:
			runningAgents++
		case StatusIdle:
			idleAgents++
		case StatusError:
			errorAgents++
		}
	}
	agentSummary += fmt.Sprintf(" (%d running, %d idle, %d error)", runningAgents, idleAgents, errorAgents)

	taskSummary := fmt.Sprintf("Tasks: %d total", len(m.tasks))
	pendingTasks := 0
	runningTasks := 0
	completedTasks := 0
	failedTasks := 0
	for _, t := range m.tasks {
		switch t.Status {
		case TaskStatusPending, TaskStatusQueued:
			pendingTasks++
		case TaskStatusRunning:
			runningTasks++
		case TaskStatusCompleted:
			completedTasks++
		case TaskStatusFailed:
			failedTasks++
		}
	}
	taskSummary += fmt.Sprintf(" (%d running, %d pending, %d done, %d failed)", runningTasks, pendingTasks, completedTasks, failedTasks)

	// Recent agents list
	agentHeader := lipgloss.NewStyle().Bold(true).Render("Recent Agents")
	var agentLines []string
	displayAgents := m.agents
	if len(displayAgents) > 5 {
		displayAgents = displayAgents[:5]
	}
	for _, a := range displayAgents {
		status := "?"
		switch a.Status {
		case StatusIdle:
			status = "idle"
		case StatusRunning:
			status = "running"
		case StatusPaused:
			status = "paused"
		case StatusError:
			status = "error"
		case StatusStopped:
			status = "stopped"
		}
		agentLines = append(agentLines, fmt.Sprintf("  %-16s %-12s %-10s %s", a.Name, string(a.Role), status, a.Model))
	}
	if len(agentLines) == 0 {
		agentLines = append(agentLines, "  No agents available.")
	}

	// Recent tasks list
	taskHeader := lipgloss.NewStyle().Bold(true).Render("Recent Tasks")
	var taskLines []string
	displayTasks := m.tasks
	if len(displayTasks) > 5 {
		displayTasks = displayTasks[:5]
	}
	for _, t := range displayTasks {
		status := "?"
		switch t.Status {
		case TaskStatusPending:
			status = "pending"
		case TaskStatusQueued:
			status = "queued"
		case TaskStatusRunning:
			status = "running"
		case TaskStatusCompleted:
			status = "done"
		case TaskStatusFailed:
			status = "failed"
		case TaskStatusCancelled:
			status = "cancelled"
		}
		taskLines = append(taskLines, fmt.Sprintf("  %-12s %-20s %-10s %.0f%%", t.ID, t.Name, status, t.Progress))
	}
	if len(taskLines) == 0 {
		taskLines = append(taskLines, "  No tasks available.")
	}

	content := strings.Join([]string{
		title,
		"",
		agentSummary,
		taskSummary,
		"",
		agentHeader,
		strings.Join(agentLines, "\n"),
		"",
		taskHeader,
		strings.Join(taskLines, "\n"),
	}, "\n")

	return lipgloss.NewStyle().
		Width(mainWidth).
		Height(m.height - 2).
		Render(content)
}

func (m Model) renderAgents() string {
	mainWidth := m.width - m.sidebar.width

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7aa2f7")).
		Render("AGENTS")

	header := fmt.Sprintf("  %-16s %-14s %-10s %-16s %-8s %-10s",
		"Name", "Role", "Status", "Model", "Tasks", "Uptime")
	separator := "  " + strings.Repeat("─", mainWidth-4)

	var rows []string
	for _, a := range m.agents {
		status := "?"
		switch a.Status {
		case StatusIdle:
			status = "idle"
		case StatusRunning:
			status = "running"
		case StatusPaused:
			status = "paused"
		case StatusError:
			status = "error"
		case StatusStopped:
			status = "stopped"
		}
		taskInfo := fmt.Sprintf("%d/%d", a.TasksCompleted, a.TasksCompleted+a.TasksFailed)
		uptime := formatDuration(a.Uptime)
		rows = append(rows, fmt.Sprintf("  %-16s %-14s %-10s %-16s %-8s %-10s",
			a.Name, string(a.Role), status, a.Model, taskInfo, uptime))
	}

	if len(rows) == 0 {
		rows = append(rows, "  No agents configured. Press [a] to add one.")
	}

	content := strings.Join([]string{
		title,
		"",
		header,
		separator,
		strings.Join(rows, "\n"),
	}, "\n")

	return lipgloss.NewStyle().
		Width(mainWidth).
		Height(m.height - 2).
		Render(content)
}

func (m Model) renderAgentDetail() string {
	mainWidth := m.width - m.sidebar.width

	content := "Agent Detail View\n\nSelect an agent from the Agents view to see details."

	return lipgloss.NewStyle().
		Width(mainWidth).
		Height(m.height - 2).
		Render(content)
}

func (m Model) renderTasks() string {
	mainWidth := m.width - m.sidebar.width

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7aa2f7")).
		Render("TASK QUEUE")

	header := fmt.Sprintf("  %-12s %-20s %-10s %-10s %-10s %-8s",
		"ID", "Name", "Type", "Priority", "Status", "Progress")
	separator := "  " + strings.Repeat("─", mainWidth-4)

	var rows []string
	for _, t := range m.tasks {
		status := "?"
		switch t.Status {
		case TaskStatusPending:
			status = "pending"
		case TaskStatusQueued:
			status = "queued"
		case TaskStatusRunning:
			status = "running"
		case TaskStatusCompleted:
			status = "done"
		case TaskStatusFailed:
			status = "failed"
		case TaskStatusCancelled:
			status = "cancelled"
		}
		priority := "?"
		switch t.Priority {
		case PriorityLow:
			priority = "low"
		case PriorityMedium:
			priority = "medium"
		case PriorityHigh:
			priority = "high"
		case PriorityCritical:
			priority = "critical"
		}
		rows = append(rows, fmt.Sprintf("  %-12s %-20s %-10s %-10s %-10s %5.0f%%",
			t.ID, truncate(t.Name, 20), string(t.Type), priority, status, t.Progress))
	}

	if len(rows) == 0 {
		rows = append(rows, "  No tasks in queue. Press [t] to create one.")
	}

	content := strings.Join([]string{
		title,
		"",
		header,
		separator,
		strings.Join(rows, "\n"),
	}, "\n")

	return lipgloss.NewStyle().
		Width(mainWidth).
		Height(m.height - 2).
		Render(content)
}

func (m Model) renderTaskDetail() string {
	mainWidth := m.width - m.sidebar.width

	content := "Task Detail View\n\nSelect a task from the Tasks view to see details."

	return lipgloss.NewStyle().
		Width(mainWidth).
		Height(m.height - 2).
		Render(content)
}

func (m Model) renderLogs() string {
	var entries []string
	for _, entry := range m.logEntries {
		timestamp := entry.Timestamp.Format("15:04:05")
		level := m.levelString(entry.Level)
		line := fmt.Sprintf("%s [%-5s] %s: %s", timestamp, level, entry.AgentID, entry.Message)
		entries = append(entries, line)
	}

	if len(entries) == 0 {
		entries = append(entries, "No log entries available.")
	}

	content := strings.Join(entries, "\n")

	box := lipgloss.NewStyle().
		Width(m.width - m.sidebar.width - 2).
		Height(m.height - 4).
		Border(lipgloss.NormalBorder()).
		Padding(1).
		Render(content)

	return box
}

func (m Model) renderConfig() string {
	mainWidth := m.width - m.sidebar.width

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7aa2f7")).
		Render("CONFIGURATION")

	content := title + "\n\nConfigure your SWARM settings here.\n\nPress [Enter] to edit a field, [Esc] to cancel."

	return lipgloss.NewStyle().
		Width(mainWidth).
		Height(m.height - 2).
		Render(content)
}

func (m Model) renderFooter() string {
	var binds []string

	switch m.view {
	case ViewDashboard:
		binds = []string{
			"[a] Add Agent",
			"[t] New Task",
			"[l] Logs",
			"[r] Refresh",
			"[?] Help",
			"[q] Quit",
		}
	case ViewAgents:
		binds = []string{
			"[Enter] Details",
			"[a] Add",
			"[d] Delete",
			"[r] Refresh",
			"[?] Help",
			"[q] Quit",
		}
	case ViewTasks:
		binds = []string{
			"[Enter] View",
			"[d] Delete",
			"[↑↓] Navigate",
			"[r] Retry",
			"[c] Cancel",
		}
	case ViewLogs:
		binds = []string{
			"[f] Follow",
			"[↑↓] Scroll",
			"[/] Search",
			"[c] Clear",
			"[q] Quit",
		}
	case ViewConfig:
		binds = []string{
			"[Tab] Next Field",
			"[Enter] Edit",
			"[Esc] Cancel",
			"[Ctrl+S] Save",
			"[q] Quit",
		}
	default:
		binds = []string{
			"[?] Help",
			"[Esc] Back",
			"[q] Quit",
		}
	}

	footer := strings.Join(binds, "  ")

	// Show error in the status bar if present and not expired
	if m.lastError != "" && time.Now().Before(m.errorExpiry) {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f7768e")).
			Bold(true)
		footer = errStyle.Render("ERROR: "+m.lastError) + "  " + footer
	}

	return m.theme.FooterStyle.Render(footer)
}

func (m Model) renderModal(background string) string {
	var modal string

	switch m.modal.modalType {
	case ModalAddAgent:
		modal = m.renderAddAgentModal()
	case ModalNewTask:
		modal = m.renderNewTaskModal()
	case ModalConfirm:
		modal = m.renderConfirmModal()
	case ModalTextInput:
		modal = m.renderTextInputModal()
	default:
		modal = "Unknown modal"
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")),
		lipgloss.WithWhitespaceChars("░"),
	)
}

func (m Model) renderAddAgentModal() string {
	title := m.theme.ModalTitleStyle.Render("ADD NEW AGENT")

	fields := []string{
		"Name:       [new-agent__________________]",
		"Role:       [coder ▾]",
		"Model:      [gpt-4-turbo ▾]",
		"",
		"Tools:",
		"  [x] file-read    [x] file-write",
		"  [x] code-exec    [ ] web-search",
	}

	content := strings.Join(fields, "\n")

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.theme.ButtonStyle.Render("[Cancel]"),
		"  ",
		m.theme.ButtonPrimaryStyle.Render("[Add Agent]"),
	)

	return m.theme.ModalBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			content,
			"",
			buttons,
		),
	)
}

func (m Model) renderNewTaskModal() string {
	title := m.theme.ModalTitleStyle.Render("NEW TASK")

	fields := []string{
		"Task Type:     [research ▾]",
		"Priority:      [medium ▾]",
		"Assign To:     [auto ▾]",
		"",
		"Description:",
		"  ┌─────────────────────────────────────────────┐",
		"  │ Analyze the authentication flow and suggest │",
		"  │ security improvements for the JWT implement │",
		"  │ ation_______________________________________│",
		"  └─────────────────────────────────────────────┘",
	}

	content := strings.Join(fields, "\n")

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.theme.ButtonStyle.Render("[Cancel]"),
		"  ",
		m.theme.ButtonPrimaryStyle.Render("[Create Task]"),
	)

	return m.theme.ModalBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			content,
			"",
			buttons,
		),
	)
}

func (m Model) renderConfirmModal() string {
	title := m.theme.ModalTitleStyle.Render("CONFIRM")

	message := m.theme.ModalMessageStyle.Render(
		"Are you sure you want to proceed?",
	)

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.theme.ButtonStyle.Render("[Cancel]"),
		"  ",
		m.theme.ButtonDangerStyle.Render("[Confirm]"),
	)

	return m.theme.ModalBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			message,
			"",
			buttons,
		),
	)
}

func (m Model) renderTextInputModal() string {
	title := m.theme.ModalTitleStyle.Render("INPUT")

	content := "Enter your input below:"

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.theme.ButtonStyle.Render("[Cancel]"),
		"  ",
		m.theme.ButtonPrimaryStyle.Render("[OK]"),
	)

	return m.theme.ModalBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			content,
			"",
			buttons,
		),
	)
}

func (m Model) levelString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func formatMemory(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
