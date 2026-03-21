package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/internal/tui"
)

func TestNewModal(t *testing.T) {
	modal := NewModal(tui.DarkTheme)

	assert.NotNil(t, modal)
	assert.NotNil(t, modal.theme)
	assert.False(t, modal.active)
	assert.Empty(t, modal.fields)
	assert.NotNil(t, modal.result)
}

func TestModal_Show(t *testing.T) {
	modal := NewModal(tui.DarkTheme)
	modal.Show(tui.ModalAddAgent, nil)

	assert.True(t, modal.active)
	assert.Equal(t, tui.ModalAddAgent, modal.modalType)
	assert.NotEmpty(t, modal.fields)
}

func TestModal_Hide(t *testing.T) {
	modal := NewModal(tui.DarkTheme)
	modal.Show(tui.ModalAddAgent, nil)
	modal.Hide()

	assert.False(t, modal.active)
	assert.Equal(t, tui.ModalNone, modal.modalType)
	assert.Empty(t, modal.fields)
	assert.Equal(t, 0, modal.currentIdx)
}

func TestModal_initFields(t *testing.T) {
	modal := NewModal(tui.DarkTheme)

	t.Run("Add Agent fields", func(t *testing.T) {
		modal.modalType = tui.ModalAddAgent
		modal.initFields()

		assert.NotEmpty(t, modal.fields)
		hasName := false
		hasRole := false
		hasModel := false
		for _, field := range modal.fields {
			if field.Key == "name" {
				hasName = true
			}
			if field.Key == "role" {
				hasRole = true
			}
			if field.Key == "model" {
				hasModel = true
			}
		}
		assert.True(t, hasName)
		assert.True(t, hasRole)
		assert.True(t, hasModel)
	})

	t.Run("New Task fields", func(t *testing.T) {
		modal.modalType = tui.ModalNewTask
		modal.initFields()

		assert.NotEmpty(t, modal.fields)
		hasTaskType := false
		hasPriority := false
		hasDescription := false
		for _, field := range modal.fields {
			if field.Key == "task_type" {
				hasTaskType = true
			}
			if field.Key == "priority" {
				hasPriority = true
			}
			if field.Key == "description" {
				hasDescription = true
			}
		}
		assert.True(t, hasTaskType)
		assert.True(t, hasPriority)
		assert.True(t, hasDescription)
	})

	t.Run("Confirm fields", func(t *testing.T) {
		modal.modalType = tui.ModalConfirm
		modal.initFields()

		assert.Empty(t, modal.fields)
	})

	t.Run("Text Input fields", func(t *testing.T) {
		modal.modalType = tui.ModalTextInput
		modal.initFields()

		assert.NotEmpty(t, modal.fields)
		hasInput := false
		for _, field := range modal.fields {
			if field.Key == "input" {
				hasInput = true
			}
		}
		assert.True(t, hasInput)
	})
}

func TestModal_handleKey_Esc(t *testing.T) {
	modal := NewModal(tui.DarkTheme)
	modal.Show(tui.ModalConfirm, nil)

	resultChan := make(chan tui.ModalResult, 1)
	modal.result = resultChan

	updatedModal, _ := modal.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	select {
	case result := <-resultChan:
		assert.False(t, result.Confirmed)
	default:
		assert.Fail(t, "No result sent")
	}

	assert.False(t, updatedModal.active)
}

func TestModal_handleKey_Navigate(t *testing.T) {
	modal := NewModal(tui.DarkTheme)
	modal.Show(tui.ModalAddAgent, nil)

	t.Run("Down", func(t *testing.T) {
		updatedModal, _ := modal.handleKey(tea.KeyMsg{Type: tea.KeyDown})
		assert.Equal(t, 1, updatedModal.currentIdx)
	})

	t.Run("Up", func(t *testing.T) {
		modal.currentIdx = 1
		updatedModal, _ := modal.handleKey(tea.KeyMsg{Type: tea.KeyUp})
		assert.Equal(t, 0, updatedModal.currentIdx)
	})

	t.Run("Tab", func(t *testing.T) {
		updatedModal, _ := modal.handleKey(tea.KeyMsg{Type: tea.KeyTab})
		assert.Equal(t, 1, updatedModal.currentIdx)
	})

	t.Run("Shift+Tab", func(t *testing.T) {
		modal.currentIdx = 2
		updatedModal, _ := modal.handleKey(tea.KeyMsg{Type: tea.KeyShiftTab})
		assert.Equal(t, 1, updatedModal.currentIdx)
	})
}

func TestModal_validate(t *testing.T) {
	modal := NewModal(tui.DarkTheme)
	modal.fields = []ModalField{
		{Key: "name", Type: FieldString, Value: "test", Required: true},
		{Key: "optional", Type: FieldString, Value: "", Required: false},
	}

	t.Run("Valid fields", func(t *testing.T) {
		assert.True(t, modal.validate())
	})

	t.Run("Missing required field", func(t *testing.T) {
		modal.fields[0].Value = ""
		assert.False(t, modal.validate())
	})

	t.Run("Below minimum length", func(t *testing.T) {
		modal.fields[0].Value = "ab"
		modal.fields[0].MinLength = 5
		assert.False(t, modal.validate())
	})
}

func TestModal_View(t *testing.T) {
	modal := NewModal(tui.DarkTheme)
	modal.width = 100
	modal.height = 50

	t.Run("Not active", func(t *testing.T) {
		view := modal.View()
		assert.Empty(t, view)
	})

	t.Run("Add Agent modal", func(t *testing.T) {
		modal.Show(tui.ModalAddAgent, nil)
		view := modal.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "ADD NEW AGENT")
	})

	t.Run("New Task modal", func(t *testing.T) {
		modal.Show(tui.ModalNewTask, nil)
		view := modal.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "NEW TASK")
	})

	t.Run("Confirm modal", func(t *testing.T) {
		modal.Show(tui.ModalConfirm, nil)
		view := modal.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "CONFIRM")
	})

	t.Run("Text Input modal", func(t *testing.T) {
		modal.Show(tui.ModalTextInput, nil)
		view := modal.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "INPUT")
	})
}

func TestModal_renderButtons(t *testing.T) {
	modal := NewModal(tui.DarkTheme)

	t.Run("Both buttons", func(t *testing.T) {
		buttons := modal.renderButtons("[Cancel]", "[OK]", true)
		assert.Contains(t, buttons, "[Cancel]")
		assert.Contains(t, buttons, "[OK]")
	})

	t.Run("Cancel only", func(t *testing.T) {
		buttons := modal.renderButtons("[Close]", "", false)
		assert.Contains(t, buttons, "[Close]")
		assert.NotContains(t, buttons, "[OK]")
	})
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"test", 10, "test      "},
		{"longer", 5, "longer"},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := padRight(tt.input, tt.width)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModal_handleKey_Enter(t *testing.T) {
	modal := NewModal(tui.DarkTheme)
	modal.Show(tui.ModalConfirm, nil)
	modal.fields = []ModalField{
		{Key: "name", Type: FieldString, Value: "test", Required: true},
	}

	resultChan := make(chan tui.ModalResult, 1)
	modal.result = resultChan

	updatedModal, _ := modal.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	select {
	case result := <-resultChan:
		assert.True(t, result.Confirmed)
		assert.Equal(t, "test", result.Data["name"])
	default:
		assert.Fail(t, "No result sent")
	}

	assert.False(t, updatedModal.active)
}
