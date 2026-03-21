package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/swarm-ai/swarm/internal/tui"
)

type TaskQueueModel struct {
	tasks         []tui.Task
	width         int
	height        int
	theme         tui.Theme
	selectedIndex int
	cursor        int
	filter        TaskFilter
	showDetails   bool
	taskDetail    *tui.Task
}

type TaskFilter struct {
	Status   tui.TaskStatus
	Priority tui.Priority
	Type     tui.TaskType
	Search   string
	AgentID  string
}

func NewTaskQueueModel(theme tui.Theme) *TaskQueueModel {
	return &TaskQueueModel{
		theme:  theme,
		cursor: 0,
		filter: TaskFilter{Status: -1, Priority: -1},
	}
}

func (m *TaskQueueModel) SetTasks(tasks []tui.Task) {
	m.tasks = tasks
}

func (m *TaskQueueModel) SetFilter(filter TaskFilter) {
	m.filter = filter
}

func (m *TaskQueueModel) Update(msg tea.Msg) (*TaskQueueModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.showDetails {
			return m.handleDetailKey(msg)
		}
		return m.handleKey(msg)

	case tui.TaskUpdateMsg:
		for i, task := range m.tasks {
			if task.ID == msg.Task.ID {
				m.tasks[i] = msg.Task
				if m.showDetails && m.taskDetail != nil && m.taskDetail.ID == msg.Task.ID {
					m.taskDetail = &msg.Task
				}
				break
			}
		}
	}

	return m, nil
}

func (m *TaskQueueModel) handleKey(msg tea.KeyMsg) (*TaskQueueModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filteredTasks())-1 {
			m.cursor++
		}
	case "enter":
		tasks := m.filteredTasks()
		if len(tasks) > 0 && m.cursor < len(tasks) {
			m.taskDetail = &tasks[m.cursor]
			m.showDetails = true
		}
	case "esc":
		m.showDetails = false
		m.taskDetail = nil
	}
	return m, nil
}

func (m *TaskQueueModel) handleDetailKey(msg tea.KeyMsg) (*TaskQueueModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showDetails = false
		m.taskDetail = nil
	}
	return m, nil
}

func (m *TaskQueueModel) filteredTasks() []tui.Task {
	filtered := []tui.Task{}

	for _, task := range m.tasks {
		if m.filter.Status != -1 && task.Status != m.filter.Status {
			continue
		}
		if m.filter.Priority != -1 && task.Priority != m.filter.Priority {
			continue
		}
		if m.filter.Type != "" && task.Type != m.filter.Type {
			continue
		}
		if m.filter.AgentID != "" && task.AgentID != m.filter.AgentID {
			continue
		}
		if m.filter.Search != "" {
			searchLower := strings.ToLower(m.filter.Search)
			if !strings.Contains(strings.ToLower(task.Name), searchLower) &&
				!strings.Contains(strings.ToLower(task.Description), searchLower) {
				continue
			}
		}
		filtered = append(filtered, task)
	}

	return filtered
}

func (m *TaskQueueModel) View() string {
	if m.showDetails && m.taskDetail != nil {
		return m.renderDetailView()
	}
	return m.renderQueueView()
}

func (m *TaskQueueModel) renderQueueView() string {
	filterBar := m.renderFilterBar()
	taskTable := m.renderTaskTable()
	footer := m.renderFooter()

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		filterBar,
		taskTable,
		footer,
	)

	return lipgloss.NewStyle().
		Width(m.width - 20).
		Height(m.height - 4).
		Render(layout)
}

func (m *TaskQueueModel) renderFilterBar() string {
	filterLabel := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render("Filter:")

	statusFilter := m.renderFilterDropdown("Status", m.statusFilterString(m.filter.Status))
	priorityFilter := m.renderFilterDropdown("Priority", m.priorityFilterString(m.filter.Priority))
	searchBox := m.renderSearchBox()

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		filterLabel,
		" ",
		statusFilter,
		" ",
		priorityFilter,
		" ",
		searchBox,
	)

	return lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Width(m.width - 22).
		Render(content)
}

func (m *TaskQueueModel) renderFilterDropdown(label, value string) string {
	if value == "" {
		value = "any"
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render(label + ":")

	valueStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Foreground).
		Background(m.theme.Colors.Border).
		Padding(0, 1).
		Render(fmt.Sprintf("[%s ▾]", value))

	return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle, " ", valueStyle)
}

func (m *TaskQueueModel) renderSearchBox() string {
	searchLabel := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render("Search:")

	searchBox := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Foreground).
		Background(m.theme.Colors.Border).
		Padding(0, 1).
		Render(fmt.Sprintf("[%s________]", m.filter.Search))

	return lipgloss.JoinHorizontal(lipgloss.Top, searchLabel, " ", searchBox)
}

func (m *TaskQueueModel) renderTaskTable() string {
	tasks := m.filteredTasks()

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.renderHeaderCell("ID", 12),
		m.renderHeaderCell("Name", 20),
		m.renderHeaderCell("Priority", 10),
		m.renderHeaderCell("Status", 10),
		m.renderHeaderCell("Agent", 10),
		m.renderHeaderCell("Age", 8),
	)

	separator := strings.Repeat("─", m.width-24)

	var rows []string
	for i, task := range tasks {
		if i >= m.height-15 {
			break
		}

		row := lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.renderCell(task.ID, 12),
			m.renderCell(task.Name, 20),
			m.renderPriorityCell(task.Priority),
			m.renderStatusCell(task.Status),
			m.renderCell(task.AgentID, 10),
			m.renderAgeCell(task.CreatedAt),
		)

		if i == m.cursor {
			row = lipgloss.NewStyle().
				Background(m.theme.Colors.Highlight).
				Foreground(m.theme.Colors.Background).
				Render(row)
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Render("No tasks match the current filter"))
	}

	content := strings.Join([]string{header, separator}, "\n")
	content += "\n" + strings.Join(rows, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(m.width - 22).
		Height(m.height - 10)

	return boxStyle.Render(content)
}

func (m *TaskQueueModel) renderDetailView() string {
	task := m.taskDetail

	header := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Primary).
		Bold(true).
		Render(fmt.Sprintf("TASK: %s", task.ID))

	infoSection := m.renderTaskInfo(task)
	progressSection := m.renderProgressSection(task)
	outputSection := m.renderOutputSection(task)

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		infoSection,
		progressSection,
		outputSection,
	)

	return lipgloss.NewStyle().
		Width(m.width - 20).
		Height(m.height - 4).
		Render(layout)
}

func (m *TaskQueueModel) renderTaskInfo(task *tui.Task) string {
	items := []struct {
		label string
		value string
	}{
		{"Name", task.Name},
		{"Type", string(task.Type)},
		{"Priority", m.priorityString(task.Priority)},
		{"Status", m.statusString(task.Status)},
		{"Agent", task.AgentID},
		{"Created", task.CreatedAt.Format("2006-01-02 15:04:05")},
	}

	if task.StartedAt != nil {
		items = append(items, struct {
			label string
			value string
		}{"Started", task.StartedAt.Format("2006-01-02 15:04:05")})
	}

	if task.CompletedAt != nil {
		items = append(items, struct {
			label string
			value string
		}{"Completed", task.CompletedAt.Format("2006-01-02 15:04:05")})
	}

	var rows []string
	for _, item := range items {
		labelStyle := lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Width(12).
			Render(item.label)

		valueStyle := lipgloss.NewStyle().
			Foreground(m.theme.Colors.Foreground).
			Render(item.value)

		rows = append(rows, labelStyle+": "+valueStyle)
	}

	content := strings.Join(rows, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(m.width - 22).
		Height(10)

	box := boxStyle.Render(content)
	header := m.theme.BoxHeaderStyle.Render("TASK DETAILS")

	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (m *TaskQueueModel) renderProgressSection(task *tui.Task) string {
	progressBar := m.renderProgressBar(task.Progress)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(m.theme.Colors.Muted).Render("Progress:"),
		progressBar,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(m.width - 22).
		Height(4)

	box := boxStyle.Render(content)
	header := m.theme.BoxHeaderStyle.Render("PROGRESS")

	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (m *TaskQueueModel) renderOutputSection(task *tui.Task) string {
	descriptionBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(m.width - 22).
		Render(task.Description)

	header := m.theme.BoxHeaderStyle.Render("DESCRIPTION")

	outputText := ""
	if task.Output != "" {
		outputLines := strings.Split(task.Output, "\n")
		if len(outputLines) > 5 {
			outputLines = outputLines[:5]
		}
		outputText = strings.Join(outputLines, "\n")

		outputBox := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(m.theme.Colors.Border).
			Padding(0, 1).
			Width(m.width - 22).
			Render(outputText)

		outputHeader := m.theme.BoxHeaderStyle.Render("OUTPUT PREVIEW")

		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			descriptionBox,
			"",
			outputHeader,
			outputBox,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, descriptionBox)
}

func (m *TaskQueueModel) renderProgressBar(progress float64) string {
	barWidth := m.width - 40
	filled := int(progress / 100 * float64(barWidth))
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	bar = lipgloss.NewStyle().
		Foreground(m.theme.Colors.Primary).
		Render(bar)

	return fmt.Sprintf("%s %.0f%%", bar, progress)
}

func (m *TaskQueueModel) renderHeaderCell(text string, width int) string {
	return lipgloss.NewStyle().
		Foreground(m.theme.Colors.Primary).
		Bold(true).
		Width(width).
		Render(text)
}

func (m *TaskQueueModel) renderCell(text string, width int) string {
	if len(text) > width {
		text = text[:width-3] + "..."
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

func (m *TaskQueueModel) renderPriorityCell(priority tui.Priority) string {
	color := m.theme.Colors.Foreground
	switch priority {
	case tui.PriorityCritical:
		color = m.theme.Colors.Error
	case tui.PriorityHigh:
		color = m.theme.Colors.Warning
	case tui.PriorityMedium:
		color = m.theme.Colors.Info
	}

	return lipgloss.NewStyle().
		Foreground(color).
		Width(10).
		Render(m.priorityString(priority))
}

func (m *TaskQueueModel) renderStatusCell(status tui.TaskStatus) string {
	icon := "○"
	color := m.theme.Colors.Muted
	switch status {
	case tui.TaskStatusRunning:
		icon = "●"
		color = m.theme.Colors.Info
	case tui.TaskStatusCompleted:
		icon = "✓"
		color = m.theme.Colors.Success
	case tui.TaskStatusFailed:
		icon = "✗"
		color = m.theme.Colors.Error
	case tui.TaskStatusPending, tui.TaskStatusQueued:
		icon = "○"
		color = m.theme.Colors.Muted
	}

	return lipgloss.NewStyle().
		Foreground(color).
		Width(10).
		Render(fmt.Sprintf("%s %s", icon, m.statusString(status)))
}

func (m *TaskQueueModel) renderAgeCell(createdAt time.Time) string {
	age := time.Since(createdAt)
	return lipgloss.NewStyle().Width(8).Render(formatDurationShort(age))
}

func (m *TaskQueueModel) renderFooter() string {
	tasks := m.filteredTasks()
	var selectedText string
	if len(tasks) > 0 && m.cursor < len(tasks) {
		selectedText = fmt.Sprintf("Selected: %s (%s)", tasks[m.cursor].ID, tasks[m.cursor].Name)
	} else {
		selectedText = "No task selected"
	}

	return lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render(selectedText)
}

func (m *TaskQueueModel) statusString(status tui.TaskStatus) string {
	switch status {
	case tui.TaskStatusPending:
		return "pending"
	case tui.TaskStatusQueued:
		return "queued"
	case tui.TaskStatusRunning:
		return "running"
	case tui.TaskStatusCompleted:
		return "done"
	case tui.TaskStatusFailed:
		return "failed"
	case tui.TaskStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func (m *TaskQueueModel) priorityString(priority tui.Priority) string {
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

func (m *TaskQueueModel) statusFilterString(status tui.TaskStatus) string {
	if status == -1 {
		return ""
	}
	return m.statusString(status)
}

func (m *TaskQueueModel) priorityFilterString(priority tui.Priority) string {
	if priority == -1 {
		return ""
	}
	return m.priorityString(priority)
}

func formatDurationShort(d time.Duration) string {
	minutes := int(d.Minutes())
	hours := minutes / 60
	minutes = minutes % 60

	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
