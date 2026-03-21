package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	app := NewApp()

	assert.NotNil(t, app)
	assert.NotNil(t, app.model)
	assert.Nil(t, app.program)
}

func TestAppSetModel(t *testing.T) {
	app := NewApp()
	model := InitialModel()
	model.view = ViewAgents

	app.SetModel(model)

	assert.Equal(t, ViewAgents, app.model.view)
}

func TestAppGetModel(t *testing.T) {
	app := NewApp()
	model := InitialModel()
	model.view = ViewTasks
	app.model = model

	retrievedModel := app.GetModel()

	assert.Equal(t, ViewTasks, retrievedModel.view)
}

func TestAppQuitWithoutProgram(t *testing.T) {
	app := NewApp()

	assert.NotPanics(t, func() {
		app.Quit()
	})

	assert.Nil(t, app.program)
}

func TestAppSendWithoutProgram(t *testing.T) {
	app := NewApp()

	assert.NotPanics(t, func() {
		app.Send(tickMsg())
	})
}

func TestAppModelInitReturnsCommand(t *testing.T) {
	app := NewApp()
	cmd := app.model.Init()

	assert.NotNil(t, cmd)
}

func tickMsg() tea.Msg {
	return TickMsg{}
}
