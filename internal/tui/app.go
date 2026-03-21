package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type App struct {
	model   Model
	program *tea.Program
}

func NewApp() *App {
	return &App{
		model: InitialModel(),
	}
}

func (a *App) SetModel(model Model) {
	a.model = model
}

func (a *App) GetModel() Model {
	return a.model
}

func (a *App) Run() error {
	a.program = tea.NewProgram(
		a.model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := a.program.Run()
	return err
}

func (a *App) RunWithOutput() (Model, error) {
	a.program = tea.NewProgram(
		a.model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	result, err := a.program.Run()
	if err != nil {
		return a.model, err
	}
	finalModel, ok := result.(Model)
	if !ok {
		return a.model, fmt.Errorf("unexpected model type: %T", result)
	}
	return finalModel, nil
}

func (a *App) Quit() {
	if a.program != nil {
		a.program.Quit()
	}
}

func (a *App) Send(msg tea.Msg) {
	if a.program != nil {
		a.program.Send(msg)
	}
}
