package tui

import (
	"fmt"
	"strings"

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
	return lipgloss.NewStyle().
		Width(m.width - m.sidebar.width).
		Height(m.height - 2).
		Render("Dashboard View\n\nAgents and tasks will be displayed here.")
}

func (m Model) renderAgents() string {
	return lipgloss.NewStyle().
		Width(m.width - m.sidebar.width).
		Height(m.height - 2).
		Render("Agents View\n\nManage your agents here.")
}

func (m Model) renderAgentDetail() string {
	return lipgloss.NewStyle().
		Width(m.width - m.sidebar.width).
		Height(m.height - 2).
		Render("Agent Detail View\n\nAgent details will be displayed here.")
}

func (m Model) renderTasks() string {
	return lipgloss.NewStyle().
		Width(m.width - m.sidebar.width).
		Height(m.height - 2).
		Render("Tasks View\n\nManage your tasks here.")
}

func (m Model) renderTaskDetail() string {
	return lipgloss.NewStyle().
		Width(m.width - m.sidebar.width).
		Height(m.height - 2).
		Render("Task Detail View\n\nTask details will be displayed here.")
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
	return lipgloss.NewStyle().
		Width(m.width - m.sidebar.width).
		Height(m.height - 2).
		Render("Config View\n\nConfigure your SWARM settings here.")
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

	return m.theme.FooterStyle.Render(
		strings.Join(binds, "  "),
	)
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
