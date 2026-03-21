package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name   string
	Colors ColorScheme

	BaseStyle          lipgloss.Style
	HeaderStyle        HeaderStyle
	SidebarStyle       SidebarStyle
	FooterStyle        lipgloss.Style
	ModalBoxStyle      lipgloss.Style
	BoxHeaderStyle     lipgloss.Style
	ButtonStyle        lipgloss.Style
	ButtonPrimaryStyle lipgloss.Style
	ButtonDangerStyle  lipgloss.Style
	ModalTitleStyle    lipgloss.Style
	ModalMessageStyle  lipgloss.Style
}

type ColorScheme struct {
	Background  lipgloss.Color
	Foreground  lipgloss.Color
	Primary     lipgloss.Color
	Secondary   lipgloss.Color
	Accent      lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
	Error       lipgloss.Color
	Info        lipgloss.Color
	Border      lipgloss.Color
	BorderFocus lipgloss.Color
	Muted       lipgloss.Color
	Highlight   lipgloss.Color
}

type HeaderStyle struct {
	Base   lipgloss.Style
	Left   lipgloss.Style
	Center lipgloss.Style
	Right  lipgloss.Style
}

type SidebarStyle struct {
	Base     lipgloss.Style
	Item     lipgloss.Style
	Selected lipgloss.Style
}

var DarkTheme = Theme{
	Name: "dark",
	Colors: ColorScheme{
		Background:  lipgloss.Color("#1a1b26"),
		Foreground:  lipgloss.Color("#c0caf5"),
		Primary:     lipgloss.Color("#7aa2f7"),
		Secondary:   lipgloss.Color("#bb9af7"),
		Accent:      lipgloss.Color("#7dcfff"),
		Success:     lipgloss.Color("#9ece6a"),
		Warning:     lipgloss.Color("#e0af68"),
		Error:       lipgloss.Color("#f7768e"),
		Info:        lipgloss.Color("#7dcfff"),
		Border:      lipgloss.Color("#3b4261"),
		BorderFocus: lipgloss.Color("#7aa2f7"),
		Muted:       lipgloss.Color("#565f89"),
		Highlight:   lipgloss.Color("#ff9e64"),
	},
	BaseStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#c0caf5")).
		Background(lipgloss.Color("#1a1b26")),
	HeaderStyle: HeaderStyle{
		Base: lipgloss.NewStyle().
			Background(lipgloss.Color("#24283b")).
			Foreground(lipgloss.Color("#c0caf5")).
			Padding(0, 1).
			Bold(true),
		Left: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aa2f7")),
		Center: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a9b1d6")),
		Right: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7dcfff")),
	},
	FooterStyle: lipgloss.NewStyle().
		Background(lipgloss.Color("#24283b")).
		Foreground(lipgloss.Color("#a9b1d6")).
		Padding(0, 1),
	ModalBoxStyle: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Background(lipgloss.Color("#1a1b26")).
		Foreground(lipgloss.Color("#c0caf5")).
		Padding(1, 2).
		Width(50),
	BoxHeaderStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7aa2f7")).
		Bold(true).
		MarginBottom(1),
	ButtonStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a9b1d6")).
		Background(lipgloss.Color("#3b4261")).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#565f89")),
	ButtonPrimaryStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1b26")).
		Background(lipgloss.Color("#7aa2f7")).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Bold(true),
	ButtonDangerStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1b26")).
		Background(lipgloss.Color("#f7768e")).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#f7768e")),
	ModalTitleStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#bb9af7")).
		Bold(true).
		MarginBottom(1),
	ModalMessageStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a9b1d6")),
	SidebarStyle: SidebarStyle{
		Base: lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1b26")).
			Foreground(lipgloss.Color("#c0caf5")),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a9b1d6")).
			Padding(0, 1),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aa2f7")).
			Background(lipgloss.Color("#24283b")).
			Bold(true).
			Padding(0, 1),
	},
}
