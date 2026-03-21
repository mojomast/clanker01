package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/swarm-ai/swarm/internal/tui"
)

type Modal struct {
	active     bool
	modalType  tui.ModalType
	formData   map[string]interface{}
	fields     []ModalField
	currentIdx int
	result     chan tui.ModalResult
	width      int
	height     int
	theme      tui.Theme
}

type ModalField struct {
	Key         string
	Label       string
	Type        FieldType
	Value       string
	Placeholder string
	Required    bool
	Options     []string
	Multiline   bool
	MinLength   int
	MaxLength   int
}

func NewModal(theme tui.Theme) *Modal {
	return &Modal{
		theme:    theme,
		formData: make(map[string]interface{}),
		fields:   []ModalField{},
		result:   make(chan tui.ModalResult, 1),
	}
}

func (m *Modal) Show(modalType tui.ModalType, initial map[string]interface{}) {
	m.active = true
	m.modalType = modalType
	m.formData = make(map[string]interface{})

	if initial != nil {
		for k, v := range initial {
			m.formData[k] = v
		}
	}

	m.initFields()
}

func (m *Modal) Hide() {
	m.active = false
	m.modalType = tui.ModalNone
	m.fields = []ModalField{}
	m.currentIdx = 0
}

func (m *Modal) initFields() {
	switch m.modalType {
	case tui.ModalAddAgent:
		m.initAddAgentFields()
	case tui.ModalNewTask:
		m.initNewTaskFields()
	case tui.ModalConfirm:
		m.initConfirmFields()
	case tui.ModalTextInput:
		m.initTextInputFields()
	}
}

func (m *Modal) initAddAgentFields() {
	m.fields = []ModalField{
		{
			Key:         "name",
			Label:       "Name",
			Type:        FieldString,
			Value:       "new-agent",
			Placeholder: "Enter agent name",
			Required:    true,
		},
		{
			Key:      "role",
			Label:    "Role",
			Type:     FieldSelect,
			Value:    "coder",
			Options:  []string{"coder", "researcher", "reviewer", "tester", "orchestrator"},
			Required: true,
		},
		{
			Key:      "model",
			Label:    "Model",
			Type:     FieldSelect,
			Value:    "gpt-4-turbo",
			Options:  []string{"gpt-4-turbo", "gpt-4", "gpt-3.5-turbo"},
			Required: true,
		},
	}
}

func (m *Modal) initNewTaskFields() {
	m.fields = []ModalField{
		{
			Key:      "task_type",
			Label:    "Task Type",
			Type:     FieldSelect,
			Value:    "research",
			Options:  []string{"research", "code", "test", "review", "document"},
			Required: true,
		},
		{
			Key:      "priority",
			Label:    "Priority",
			Type:     FieldSelect,
			Value:    "medium",
			Options:  []string{"low", "medium", "high", "critical"},
			Required: true,
		},
		{
			Key:      "assign_to",
			Label:    "Assign To",
			Type:     FieldSelect,
			Value:    "auto",
			Options:  []string{"auto", "coder", "researcher", "reviewer", "tester"},
			Required: true,
		},
		{
			Key:         "description",
			Label:       "Description",
			Type:        FieldMultiline,
			Value:       "",
			Placeholder: "Enter task description",
			Required:    true,
			Multiline:   true,
			MinLength:   10,
			MaxLength:   1000,
		},
	}
}

func (m *Modal) initConfirmFields() {
	m.fields = []ModalField{}
}

func (m *Modal) initTextInputFields() {
	m.fields = []ModalField{
		{
			Key:         "input",
			Label:       "Input",
			Type:        FieldString,
			Value:       "",
			Placeholder: "Enter your input",
			Required:    true,
		},
	}
}

func (m *Modal) Update(msg tea.Msg) (*Modal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Modal) handleKey(msg tea.KeyMsg) (*Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.result <- tui.ModalResult{Confirmed: false, Data: m.formData}
		m.Hide()
		return m, nil

	case "enter":
		if m.currentIdx >= 0 && m.currentIdx < len(m.fields) {
			field := &m.fields[m.currentIdx]
			m.formData[field.Key] = field.Value

			if m.currentIdx < len(m.fields)-1 {
				m.currentIdx++
				return m, nil
			}
		}

		if m.validate() {
			m.result <- tui.ModalResult{Confirmed: true, Data: m.formData}
			m.Hide()
			return m, nil
		}

	case "tab":
		if m.currentIdx < len(m.fields)-1 {
			m.currentIdx++
		} else {
			m.currentIdx = 0
		}
		return m, nil

	case "shift+tab":
		if m.currentIdx > 0 {
			m.currentIdx--
		} else {
			m.currentIdx = len(m.fields) - 1
		}
		return m, nil

	case "up", "k":
		if m.currentIdx > 0 {
			m.currentIdx--
		}
		return m, nil

	case "down", "j":
		if m.currentIdx < len(m.fields)-1 {
			m.currentIdx++
		}
		return m, nil

	default:
		if m.currentIdx >= 0 && m.currentIdx < len(m.fields) {
			field := &m.fields[m.currentIdx]
			if field.Type == FieldString || field.Type == FieldMultiline {
				if len(msg.String()) == 1 {
					if field.MaxLength == 0 || len(field.Value) < field.MaxLength {
						field.Value += msg.String()
					}
				} else if msg.String() == "backspace" && len(field.Value) > 0 {
					field.Value = field.Value[:len(field.Value)-1]
				}
			}
		}
	}

	return m, nil
}

func (m *Modal) validate() bool {
	for _, field := range m.fields {
		if field.Required && (field.Value == "" || len(field.Value) < field.MinLength) {
			return false
		}
	}
	return true
}

func (m *Modal) View() string {
	if !m.active {
		return ""
	}

	switch m.modalType {
	case tui.ModalAddAgent:
		return m.renderAddAgentModal()
	case tui.ModalNewTask:
		return m.renderNewTaskModal()
	case tui.ModalConfirm:
		return m.renderConfirmModal()
	case tui.ModalTextInput:
		return m.renderTextInputModal()
	default:
		return m.renderUnknownModal()
	}
}

func (m *Modal) renderModalBackground(content string) string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")),
		lipgloss.WithWhitespaceChars("░"),
	)
}

func (m *Modal) renderAddAgentModal() string {
	title := m.theme.ModalTitleStyle.Render("ADD NEW AGENT")

	fields := m.renderFields()

	buttons := m.renderButtons("[Cancel]", "[Add Agent]", true)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		fields,
		"",
		buttons,
	)

	modalBox := m.theme.ModalBoxStyle.Width(60).Render(content)

	return m.renderModalBackground(modalBox)
}

func (m *Modal) renderNewTaskModal() string {
	title := m.theme.ModalTitleStyle.Render("NEW TASK")

	fields := m.renderFields()

	buttons := m.renderButtons("[Cancel]", "[Create Task]", true)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		fields,
		"",
		buttons,
	)

	modalBox := m.theme.ModalBoxStyle.Width(70).Render(content)

	return m.renderModalBackground(modalBox)
}

func (m *Modal) renderConfirmModal() string {
	title := m.theme.ModalTitleStyle.Render("CONFIRM")

	message := m.theme.ModalMessageStyle.Render(
		"Are you sure you want to proceed?",
	)

	buttons := m.renderButtons("[Cancel]", "[Confirm]", false)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		message,
		"",
		buttons,
	)

	modalBox := m.theme.ModalBoxStyle.Width(50).Render(content)

	return m.renderModalBackground(modalBox)
}

func (m *Modal) renderTextInputModal() string {
	title := m.theme.ModalTitleStyle.Render("INPUT")

	fields := m.renderFields()

	buttons := m.renderButtons("[Cancel]", "[OK]", true)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		fields,
		"",
		buttons,
	)

	modalBox := m.theme.ModalBoxStyle.Width(50).Render(content)

	return m.renderModalBackground(modalBox)
}

func (m *Modal) renderUnknownModal() string {
	title := m.theme.ModalTitleStyle.Render("UNKNOWN MODAL")

	message := m.theme.ModalMessageStyle.Render("An unknown modal type was requested.")

	buttons := m.renderButtons("[Close]", "", false)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		message,
		"",
		buttons,
	)

	modalBox := m.theme.ModalBoxStyle.Width(50).Render(content)

	return m.renderModalBackground(modalBox)
}

func (m *Modal) renderFields() string {
	var fieldRows []string

	for i, field := range m.fields {
		fieldRow := m.renderField(field, i == m.currentIdx)
		fieldRows = append(fieldRows, fieldRow)

		if field.Type == FieldMultiline {
			fieldRows = append(fieldRows, m.renderMultilineField(field, i == m.currentIdx))
		}

		if field.Type == FieldSelect && field.Key == "tools" {
			fieldRows = append(fieldRows, m.renderToolsCheckboxes(field))
		}
	}

	return strings.Join(fieldRows, "\n")
}

func (m *Modal) renderField(field ModalField, focused bool) string {
	labelStyle := lipgloss.NewStyle().
		Width(15).
		Render(field.Label + ":")

	var valueStyle lipgloss.Style
	if focused {
		valueStyle = lipgloss.NewStyle().
			Foreground(m.theme.Colors.Foreground).
			Background(m.theme.Colors.Border)
	} else {
		valueStyle = lipgloss.NewStyle().
			Foreground(m.theme.Colors.Foreground)
	}

	var valueText string
	switch field.Type {
	case FieldSelect:
		valueText = fmt.Sprintf("%s ▾", field.Value)
	default:
		valueText = field.Value
	}

	if field.Placeholder != "" && valueText == "" {
		valueText = field.Placeholder
	}

	padding := 25 - len(valueText)
	if padding < 0 {
		padding = 0
	}
	valueRender := valueStyle.Render(fmt.Sprintf("[%s%s]", valueText, strings.Repeat("_", padding)))

	return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle, " ", valueRender)
}

func (m *Modal) renderMultilineField(field ModalField, focused bool) string {
	boxWidth := 54
	boxHeight := 4

	content := field.Value
	if content == "" {
		content = field.Placeholder
	}

	lines := strings.Split(content, "\n")
	for len(lines) < boxHeight {
		lines = append(lines, "")
	}

	var renderedLines []string
	for _, line := range lines {
		if len(line) > boxWidth {
			line = line[:boxWidth-3] + "..."
		}
		renderedLines = append(renderedLines, fmt.Sprintf("│ %s │", padRight(line, boxWidth)))
	}

	topBorder := fmt.Sprintf("┌─%s─┐", strings.Repeat("─", boxWidth))
	bottomBorder := fmt.Sprintf("└─%s─┘", strings.Repeat("─", boxWidth))

	boxStyle := lipgloss.NewStyle().
		Foreground(m.theme.Colors.Foreground)

	if focused {
		boxStyle = boxStyle.Foreground(m.theme.Colors.Primary)
	}

	box := boxStyle.Render(
		strings.Join([]string{topBorder, strings.Join(renderedLines, "\n"), bottomBorder}, "\n"),
	)

	return "  " + box
}

func (m *Modal) renderToolsCheckboxes(field ModalField) string {
	tools := []string{"file-read", "file-write", "code-exec", "web-search", "shell-exec", "api-call"}
	selectedTools, _ := m.formData["tools"].([]string)
	if selectedTools == nil {
		selectedTools = []string{}
	}

	var checkboxes []string
	for i, tool := range tools {
		checked := false
		for _, selected := range selectedTools {
			if selected == tool {
				checked = true
				break
			}
		}

		checkbox := "[ ]"
		if checked {
			checkbox = "[x]"
		}

		checkboxes = append(checkboxes, fmt.Sprintf("  %s %s", checkbox, tool))

		if i%2 == 1 {
			checkboxes[len(checkboxes)-1] += "\n"
		}
	}

	return strings.Join(checkboxes, "")
}

func (m *Modal) renderButtons(cancel, primary string, showPrimary bool) string {
	cancelBtn := m.theme.ButtonStyle.Render(cancel)

	buttons := cancelBtn

	if showPrimary && primary != "" {
		primaryBtn := m.theme.ButtonPrimaryStyle.Render(primary)
		buttons = lipgloss.JoinHorizontal(
			lipgloss.Top,
			cancelBtn,
			"  ",
			primaryBtn,
		)
	}

	return buttons
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
