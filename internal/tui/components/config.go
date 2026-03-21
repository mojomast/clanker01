package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/swarm-ai/swarm/internal/tui"
)

type ConfigModel struct {
	config        tui.Config
	width         int
	height        int
	theme         tui.Theme
	selectedTab   int
	selectedField int
	fields        []ConfigField
	isEditing     bool
	editBuffer    string
}

type ConfigField struct {
	Key         string
	Label       string
	Value       interface{}
	Type        FieldType
	Options     []string
	Min         float64
	Max         float64
	Required    bool
	Description string
}

type FieldType int

const (
	FieldString FieldType = iota
	FieldNumber
	FieldBoolean
	FieldSelect
	FieldSlider
	FieldMultiline
)

func NewConfigModel(theme tui.Theme) *ConfigModel {
	model := &ConfigModel{
		theme:       theme,
		config:      tui.Config{},
		selectedTab: 0,
		isEditing:   false,
	}

	model.initFields()
	return model
}

func (m *ConfigModel) initFields() {
	m.fields = []ConfigField{
		{
			Key:         "api_endpoint",
			Label:       "API Endpoint",
			Value:       "https://api.openai.com/v1",
			Type:        FieldString,
			Required:    true,
			Description: "OpenAI API endpoint URL",
		},
		{
			Key:         "default_model",
			Label:       "Default Model",
			Value:       "gpt-4-turbo",
			Type:        FieldSelect,
			Options:     []string{"gpt-4-turbo", "gpt-4", "gpt-3.5-turbo"},
			Required:    true,
			Description: "Default AI model to use",
		},
		{
			Key:         "max_agents",
			Label:       "Max Agents",
			Value:       10,
			Type:        FieldNumber,
			Min:         1,
			Max:         100,
			Required:    true,
			Description: "Maximum number of concurrent agents",
		},
		{
			Key:         "task_timeout",
			Label:       "Task Timeout",
			Value:       "30m",
			Type:        FieldString,
			Required:    true,
			Description: "Default timeout for task execution",
		},
		{
			Key:         "log_level",
			Label:       "Log Level",
			Value:       "Info",
			Type:        FieldSelect,
			Options:     []string{"Debug", "Info", "Warn", "Error"},
			Required:    true,
			Description: "Minimum log level to display",
		},
		{
			Key:         "temperature",
			Label:       "Temperature",
			Value:       0.7,
			Type:        FieldSlider,
			Min:         0.0,
			Max:         2.0,
			Required:    true,
			Description: "Default temperature for AI responses",
		},
		{
			Key:         "max_tokens",
			Label:       "Max Tokens",
			Value:       4096,
			Type:        FieldNumber,
			Min:         256,
			Max:         32768,
			Required:    true,
			Description: "Maximum tokens per response",
		},
		{
			Key:         "retry_attempts",
			Label:       "Retry Attempts",
			Value:       3,
			Type:        FieldNumber,
			Min:         0,
			Max:         10,
			Required:    true,
			Description: "Number of retry attempts on failure",
		},
		{
			Key:         "retry_delay",
			Label:       "Retry Delay",
			Value:       "5s",
			Type:        FieldString,
			Required:    true,
			Description: "Delay between retry attempts",
		},
		{
			Key:         "refresh_rate",
			Label:       "Refresh Rate",
			Value:       "100ms",
			Type:        FieldString,
			Required:    true,
			Description: "UI refresh rate",
		},
	}
}

func (m *ConfigModel) SetConfig(config tui.Config) {
	m.config = config
	m.updateFieldsFromConfig()
}

func (m *ConfigModel) updateFieldsFromConfig() {
	for i := range m.fields {
		field := &m.fields[i]
		switch field.Key {
		case "api_endpoint":
			field.Value = m.config.APIEndpoint
		case "default_model":
			field.Value = m.config.DefaultModel
		case "max_agents":
			field.Value = m.config.MaxAgents
		case "task_timeout":
			field.Value = m.config.TaskTimeout.String()
		case "log_level":
			field.Value = m.levelString(m.config.LogLevel)
		case "temperature":
			field.Value = m.config.Temperature
		case "max_tokens":
			field.Value = m.config.MaxTokens
		case "retry_attempts":
			field.Value = m.config.RetryAttempts
		case "retry_delay":
			field.Value = m.config.RetryDelay.String()
		case "refresh_rate":
			field.Value = m.config.RefreshRate.String()
		}
	}
}

func (m *ConfigModel) levelString(level tui.LogLevel) string {
	switch level {
	case tui.LevelDebug:
		return "Debug"
	case tui.LevelInfo:
		return "Info"
	case tui.LevelWarn:
		return "Warn"
	case tui.LevelError:
		return "Error"
	default:
		return "Info"
	}
}

func (m *ConfigModel) Update(msg tea.Msg) (*ConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.isEditing {
			return m.handleEditKey(msg)
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *ConfigModel) handleKey(msg tea.KeyMsg) (*ConfigModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedField > 0 {
			m.selectedField--
		}
	case "down", "j":
		if m.selectedField < len(m.fields)-1 {
			m.selectedField++
		}
	case "enter":
		field := m.fields[m.selectedField]
		if field.Type == FieldSelect {
			m.cycleSelectValue(field)
		} else if field.Type == FieldBoolean {
			m.toggleBoolean(field)
		} else {
			m.isEditing = true
			m.editBuffer = fmt.Sprintf("%v", field.Value)
		}
	case "tab":
		tabs := []string{"General", "Agent Defaults", "UI Settings"}
		m.selectedTab = (m.selectedTab + 1) % len(tabs)
	case "esc":
		m.isEditing = false
		m.editBuffer = ""
	}

	return m, nil
}

func (m *ConfigModel) handleEditKey(msg tea.KeyMsg) (*ConfigModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.isEditing = false
		m.saveFieldValue()
		m.editBuffer = ""
	case "esc":
		m.isEditing = false
		m.editBuffer = ""
	case "backspace":
		if len(m.editBuffer) > 0 {
			m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.editBuffer += msg.String()
		}
	}

	return m, nil
}

func (m *ConfigModel) cycleSelectValue(field ConfigField) {
	currentValue := fmt.Sprintf("%v", field.Value)
	currentIndex := -1
	for i, option := range field.Options {
		if option == currentValue {
			currentIndex = i
			break
		}
	}
	if currentIndex >= 0 {
		nextIndex := (currentIndex + 1) % len(field.Options)
		field.Value = field.Options[nextIndex]
		m.fields[m.selectedField] = field
	}
}

func (m *ConfigModel) toggleBoolean(field ConfigField) {
	currentValue, ok := field.Value.(bool)
	if ok {
		field.Value = !currentValue
		m.fields[m.selectedField] = field
	}
}

func (m *ConfigModel) saveFieldValue() {
	if m.selectedField >= 0 && m.selectedField < len(m.fields) {
		field := &m.fields[m.selectedField]
		switch field.Type {
		case FieldString, FieldMultiline:
			field.Value = m.editBuffer
		case FieldNumber:
			var num int
			fmt.Sscanf(m.editBuffer, "%d", &num)
			field.Value = num
		case FieldSlider:
			var num float64
			fmt.Sscanf(m.editBuffer, "%f", &num)
			if num < field.Min {
				num = field.Min
			}
			if num > field.Max {
				num = field.Max
			}
			field.Value = num
		}
	}
}

func (m *ConfigModel) GetConfig() tui.Config {
	config := tui.Config{}
	for _, field := range m.fields {
		switch field.Key {
		case "api_endpoint":
			config.APIEndpoint = field.Value.(string)
		case "default_model":
			config.DefaultModel = field.Value.(string)
		case "max_agents":
			config.MaxAgents = field.Value.(int)
		case "task_timeout":
			config.TaskTimeout = parseDuration(field.Value.(string))
		case "log_level":
			config.LogLevel = parseLogLevel(field.Value.(string))
		case "temperature":
			config.Temperature = field.Value.(float64)
		case "max_tokens":
			config.MaxTokens = field.Value.(int)
		case "retry_attempts":
			config.RetryAttempts = field.Value.(int)
		case "retry_delay":
			config.RetryDelay = parseDuration(field.Value.(string))
		case "refresh_rate":
			config.RefreshRate = parseDuration(field.Value.(string))
		}
	}
	return config
}

func parseDuration(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

func parseLogLevel(s string) tui.LogLevel {
	switch strings.ToLower(s) {
	case "debug":
		return tui.LevelDebug
	case "info":
		return tui.LevelInfo
	case "warn":
		return tui.LevelWarn
	case "error":
		return tui.LevelError
	default:
		return tui.LevelInfo
	}
}

func (m *ConfigModel) View() string {
	header := m.renderHeader()
	tabs := m.renderTabs()
	content := m.renderContent()

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabs,
		content,
	)

	return lipgloss.NewStyle().
		Width(m.width - 20).
		Height(m.height - 4).
		Render(layout)
}

func (m *ConfigModel) renderHeader() string {
	leftText := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Primary).
		Bold(true).
		Render("CONFIGURATION")

	rightText := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Muted).
		Render("[Save] [Reset] [Export]")

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

func (m *ConfigModel) renderTabs() string {
	tabs := []string{"General", "Agent Defaults", "UI Settings"}

	var tabItems []string
	for i, tab := range tabs {
		if i == m.selectedTab {
			tabItems = append(tabItems, lipgloss.NewStyle().
				Foreground(m.theme.Colors.Primary).
				Bold(true).
				Underline(true).
				Render(tab))
		} else {
			tabItems = append(tabItems, lipgloss.NewStyle().
				Foreground(m.theme.Colors.Muted).
				Render(tab))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabItems...)
}

func (m *ConfigModel) renderContent() string {
	return m.renderFieldList()
}

func (m *ConfigModel) renderFieldList() string {
	var rows []string

	for i, field := range m.fields {
		row := m.renderField(field, i == m.selectedField, i == m.selectedField && m.isEditing)
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Colors.Border).
		Padding(0, 1).
		Width(m.width - 22).
		Height(m.height - 10)

	return boxStyle.Render(content)
}

func (m *ConfigModel) renderField(field ConfigField, selected, editing bool) string {
	labelStyle := lipgloss.NewStyle().
		Width(20).
		Render(field.Label + ":")

	var valueStyle lipgloss.Style
	if selected {
		valueStyle = lipgloss.NewStyle().
			Foreground(m.theme.Colors.Primary).
			Background(m.theme.Colors.Border)
	} else {
		valueStyle = lipgloss.NewStyle().
			Foreground(m.theme.Colors.Foreground)
	}

	var valueText string
	if editing {
		valueText = m.editBuffer
	} else {
		valueText = fmt.Sprintf("%v", field.Value)
	}

	if field.Type == FieldString || field.Type == FieldMultiline {
		maxLen := 40
		if len(valueText) > maxLen {
			valueText = valueText[:maxLen-3] + "..."
		}
		valueText = fmt.Sprintf("[%s%s]", valueText, strings.Repeat("_", maxLen-len(valueText)))
	} else if field.Type == FieldNumber {
		valueText = fmt.Sprintf("[%s___]", valueText)
	} else if field.Type == FieldSlider {
		valueText = fmt.Sprintf("[%.1f] %s", field.Value.(float64), m.renderSlider(field.Value.(float64), field.Min, field.Max))
	} else if field.Type == FieldSelect {
		valueText = fmt.Sprintf("[%s ▾]", valueText)
	} else if field.Type == FieldBoolean {
		if field.Value.(bool) {
			valueText = "[x]"
		} else {
			valueText = "[ ]"
		}
	}

	valueRender := valueStyle.Render(valueText)

	return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle, " ", valueRender)
}

func (m *ConfigModel) renderSlider(value, min, max float64) string {
	barWidth := 30
	normalized := (value - min) / (max - min)
	filled := int(normalized * float64(barWidth))
	empty := barWidth - filled

	bar := strings.Repeat("●", filled) + strings.Repeat("○", empty)
	return fmt.Sprintf("%s [%s]", bar, fmt.Sprintf("%.1f", value))
}
