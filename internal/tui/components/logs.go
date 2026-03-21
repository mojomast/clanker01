package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/swarm-ai/swarm/internal/tui"
)

type LogsModel struct {
	entries      []tui.LogEntry
	width        int
	height       int
	theme        tui.Theme
	scrollOffset int
	cursor       int
	filter       LogFilter
	follow       bool
}

type LogFilter struct {
	Level   tui.LogLevel
	AgentID string
	Search  string
	Since   time.Time
	Until   time.Time
}

func NewLogsModel(theme tui.Theme) *LogsModel {
	return &LogsModel{
		theme:  theme,
		filter: LogFilter{Level: -1},
		follow: true,
	}
}

func (m *LogsModel) SetEntries(entries []tui.LogEntry) {
	m.entries = entries
	if m.follow {
		m.scrollOffset = len(entries) - m.maxVisibleLines()
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	}
}

func (m *LogsModel) AddEntry(entry tui.LogEntry) {
	m.entries = append(m.entries, entry)
	if len(m.entries) > 10000 {
		m.entries = m.entries[len(m.entries)-10000:]
	}

	if m.follow {
		m.scrollOffset = len(m.entries) - m.maxVisibleLines()
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	}
}

func (m *LogsModel) Update(msg tea.Msg) (*LogsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tui.LogStreamMsg:
		for _, entry := range msg.Entries {
			m.AddEntry(entry)
		}
	}

	return m, nil
}

func (m *LogsModel) handleKey(msg tea.KeyMsg) (*LogsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.follow = false
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
	case "down", "j":
		if m.scrollOffset < len(m.filteredEntries())-m.maxVisibleLines() {
			m.scrollOffset++
		}
	case "pageup", "pgup":
		m.follow = false
		m.scrollOffset = max(0, m.scrollOffset-m.maxVisibleLines())
	case "pagedown", "pgdown":
		maxOffset := max(0, len(m.filteredEntries())-m.maxVisibleLines())
		m.scrollOffset = min(maxOffset, m.scrollOffset+m.maxVisibleLines())
	case "home":
		m.follow = false
		m.scrollOffset = 0
	case "end":
		m.follow = true
		m.scrollOffset = len(m.filteredEntries()) - m.maxVisibleLines()
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	case "f":
		m.follow = !m.follow
		if m.follow {
			m.scrollOffset = len(m.filteredEntries()) - m.maxVisibleLines()
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		}
	case "c":
		m.entries = []tui.LogEntry{}
		m.scrollOffset = 0
	}

	return m, nil
}

func (m *LogsModel) filteredEntries() []tui.LogEntry {
	filtered := []tui.LogEntry{}

	for _, entry := range m.entries {
		if m.filter.Level != -1 && entry.Level != m.filter.Level {
			continue
		}
		if m.filter.AgentID != "" && entry.AgentID != m.filter.AgentID {
			continue
		}
		if m.filter.Search != "" {
			searchLower := strings.ToLower(m.filter.Search)
			if !strings.Contains(strings.ToLower(entry.Message), searchLower) {
				continue
			}
		}
		if !m.filter.Since.IsZero() && entry.Timestamp.Before(m.filter.Since) {
			continue
		}
		if !m.filter.Until.IsZero() && entry.Timestamp.After(m.filter.Until) {
			continue
		}
		filtered = append(filtered, entry)
	}

	return filtered
}

func (m *LogsModel) maxVisibleLines() int {
	return m.height - 12
}

func (m *LogsModel) View() string {
	header := m.renderHeader()
	filterBar := m.renderFilterBar()
	logContent := m.renderLogContent()
	statistics := m.renderStatistics()

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		filterBar,
		logContent,
		statistics,
	)

	return lipgloss.NewStyle().
		Width(m.width - 20).
		Height(m.height - 4).
		Render(layout)
}

func (m *LogsModel) renderHeader() string {
	leftText := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Primary).
		Bold(true).
		Render("LOGS STREAM")

	rightText := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render(fmt.Sprintf("[%s] [%s] [%s]", m.filterLabel(), m.agentFilterLabel(), m.searchLabel()))

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftText,
		lipgloss.PlaceHorizontal(
			m.width-20-lipgloss.Width(leftText)-lipgloss.Width(rightText),
			lipgloss.Right,
			rightText,
		),
	)
}

func (m *LogsModel) renderFilterBar() string {
	levelFilter := m.renderLevelFilter()
	agentFilter := m.renderAgentFilter()
	searchBox := m.renderSearchBox()

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		"Level:",
		levelFilter,
		"  ",
		"Agent:",
		agentFilter,
		"  ",
		searchBox,
	)

	return lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Width(m.width - 22).
		Render(content)
}

func (m *LogsModel) renderLevelFilter() string {
	levels := []string{"All", "Debug", "Info", "Warn", "Error"}
	selectedLevel := "All"
	if m.filter.Level == tui.LevelDebug {
		selectedLevel = "Debug"
	} else if m.filter.Level == tui.LevelInfo {
		selectedLevel = "Info"
	} else if m.filter.Level == tui.LevelWarn {
		selectedLevel = "Warn"
	} else if m.filter.Level == tui.LevelError {
		selectedLevel = "Error"
	}

	var parts []string
	for _, level := range levels {
		if level == selectedLevel {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(m.theme.Colors.Background).
				Background(m.theme.Colors.Primary).
				Bold(true).
				Render(fmt.Sprintf("●%s", level)))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(m.theme.Colors.Muted).
				Render(fmt.Sprintf("○%s", level)))
		}
	}

	return strings.Join(parts, " ")
}

func (m *LogsModel) renderAgentFilter() string {
	if m.filter.AgentID == "" {
		return lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Render("[All ▾]")
	}
	return lipgloss.NewStyle().
		Foreground(m.theme.Colors.Foreground).
		Render(fmt.Sprintf("[%s ▾]", m.filter.AgentID))
}

func (m *LogsModel) renderSearchBox() string {
	searchLabel := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render("Search:")

	searchBox := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Foreground).
		Background(m.theme.Colors.Border).
		Padding(0, 1).
		Render(fmt.Sprintf("[%s__________________]", m.filter.Search))

	return lipgloss.JoinHorizontal(lipgloss.Top, searchLabel, " ", searchBox)
}

func (m *LogsModel) renderLogContent() string {
	entries := m.filteredEntries()
	maxLines := m.maxVisibleLines()

	startIndex := m.scrollOffset
	if startIndex < 0 {
		startIndex = 0
	}

	endIndex := startIndex + maxLines
	if endIndex > len(entries) {
		endIndex = len(entries)
	}

	var lines []string
	for i := startIndex; i < endIndex; i++ {
		entry := entries[i]
		lines = append(lines, m.renderLogEntry(entry))
	}

	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(m.theme.Colors.Muted).
			Render("No log entries"))
	}

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(m.width - 22).
		Height(maxLines)

	box := boxStyle.Render(content)

	if m.follow {
		followIndicator := lipgloss.NewStyle().
			Foreground(m.theme.Colors.Success).
			Render("▼ Auto-scrolling")
		box = lipgloss.JoinVertical(
			lipgloss.Left,
			box,
			followIndicator,
		)
	}

	return box
}

func (m *LogsModel) renderLogEntry(entry tui.LogEntry) string {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	level := m.levelString(entry.Level)
	agentID := entry.AgentID
	message := entry.Message

	timestampStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Width(20).
		Render(timestamp)

	levelStyle := lipgloss.NewStyle().
		Foreground(m.levelColor(entry.Level)).
		Width(7).
		Render(level)

	agentStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Info).
		Width(15).
		Render(agentID + ":")

	messageStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Foreground).
		Render(message)

	return fmt.Sprintf("%s [%s] %s %s", timestampStyle, levelStyle, agentStyle, messageStyle)
}

func (m *LogsModel) renderStatistics() string {
	entries := m.filteredEntries()

	counts := map[tui.LogLevel]int{tui.LevelDebug: 0, tui.LevelInfo: 0, tui.LevelWarn: 0, tui.LevelError: 0}
	for _, entry := range entries {
		counts[entry.Level]++
	}

	total := len(entries)
	statsText := fmt.Sprintf("Statistics: %d entries │ %d INFO │ %d DEBUG │ %d WARN │ %d ERROR",
		total, counts[tui.LevelInfo], counts[tui.LevelDebug], counts[tui.LevelWarn], counts[tui.LevelError])

	return lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render(statsText)
}

func (m *LogsModel) levelString(level tui.LogLevel) string {
	switch level {
	case tui.LevelDebug:
		return "DEBUG"
	case tui.LevelInfo:
		return "INFO"
	case tui.LevelWarn:
		return "WARN"
	case tui.LevelError:
		return "ERROR"
	default:
		return "UNKN"
	}
}

func (m *LogsModel) levelColor(level tui.LogLevel) lipgloss.Color {
	switch level {
	case tui.LevelDebug:
		return m.theme.Colors.Muted
	case tui.LevelInfo:
		return m.theme.Colors.Info
	case tui.LevelWarn:
		return m.theme.Colors.Warning
	case tui.LevelError:
		return m.theme.Colors.Error
	default:
		return m.theme.Colors.Foreground
	}
}

func (m *LogsModel) filterLabel() string {
	if m.filter.Level == -1 {
		return "All"
	}
	return m.levelString(m.filter.Level)
}

func (m *LogsModel) agentFilterLabel() string {
	if m.filter.AgentID == "" {
		return "All"
	}
	return m.filter.AgentID
}

func (m *LogsModel) searchLabel() string {
	if m.filter.Search == "" {
		return "-"
	}
	return fmt.Sprintf("\"%s\"", m.filter.Search)
}
