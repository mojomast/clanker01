package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	Navigation NavigationKeys
	Actions    ActionKeys
	Views      ViewKeys
	Modal      ModalKeys
}

type NavigationKeys struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
}

type ActionKeys struct {
	Enter   key.Binding
	Escape  key.Binding
	Quit    key.Binding
	Help    key.Binding
	Refresh key.Binding
	Search  key.Binding
	Filter  key.Binding
	Delete  key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

type ViewKeys struct {
	Dashboard key.Binding
	Agents    key.Binding
	Tasks     key.Binding
	Logs      key.Binding
	Config    key.Binding
}

type ModalKeys struct {
	Submit    key.Binding
	Close     key.Binding
	NextField key.Binding
	PrevField key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Navigation: NavigationKeys{
			Up: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("↑/k", "up"),
			),
			Down: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓/j", "down"),
			),
			Left: key.NewBinding(
				key.WithKeys("left", "h"),
				key.WithHelp("←/h", "left"),
			),
			Right: key.NewBinding(
				key.WithKeys("right", "l"),
				key.WithHelp("→/l", "right"),
			),
			Tab: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "next"),
			),
			ShiftTab: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "previous"),
			),
			PageUp: key.NewBinding(
				key.WithKeys("pgup"),
				key.WithHelp("pgup", "page up"),
			),
			PageDown: key.NewBinding(
				key.WithKeys("pgdown"),
				key.WithHelp("pgdown", "page down"),
			),
			Home: key.NewBinding(
				key.WithKeys("home"),
				key.WithHelp("home", "go to start"),
			),
			End: key.NewBinding(
				key.WithKeys("end"),
				key.WithHelp("end", "go to end"),
			),
		},
		Actions: ActionKeys{
			Enter: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "select"),
			),
			Escape: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back"),
			),
			Quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
			Help: key.NewBinding(
				key.WithKeys("?"),
				key.WithHelp("?", "help"),
			),
			Refresh: key.NewBinding(
				key.WithKeys("r", "f5"),
				key.WithHelp("r", "refresh"),
			),
			Search: key.NewBinding(
				key.WithKeys("/"),
				key.WithHelp("/", "search"),
			),
			Filter: key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "filter"),
			),
			Delete: key.NewBinding(
				key.WithKeys("d", "delete"),
				key.WithHelp("d", "delete"),
			),
			Confirm: key.NewBinding(
				key.WithKeys("y"),
				key.WithHelp("y", "confirm"),
			),
			Cancel: key.NewBinding(
				key.WithKeys("n"),
				key.WithHelp("n", "cancel"),
			),
		},
		Views: ViewKeys{
			Dashboard: key.NewBinding(
				key.WithKeys("1"),
				key.WithHelp("1", "dashboard"),
			),
			Agents: key.NewBinding(
				key.WithKeys("2"),
				key.WithHelp("2", "agents"),
			),
			Tasks: key.NewBinding(
				key.WithKeys("3"),
				key.WithHelp("3", "tasks"),
			),
			Logs: key.NewBinding(
				key.WithKeys("4"),
				key.WithHelp("4", "logs"),
			),
			Config: key.NewBinding(
				key.WithKeys("5"),
				key.WithHelp("5", "config"),
			),
		},
		Modal: ModalKeys{
			Submit: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "submit"),
			),
			Close: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "close"),
			),
			NextField: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "next field"),
			),
			PrevField: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "previous field"),
			),
		},
	}
}
