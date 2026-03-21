package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestDarkTheme(t *testing.T) {
	theme := DarkTheme

	assert.Equal(t, "dark", theme.Name)
	assert.NotNil(t, theme.Colors)
	assert.NotNil(t, theme.BaseStyle)
	assert.NotNil(t, theme.HeaderStyle.Base)
	assert.NotNil(t, theme.HeaderStyle.Left)
	assert.NotNil(t, theme.HeaderStyle.Center)
	assert.NotNil(t, theme.HeaderStyle.Right)
	assert.NotNil(t, theme.FooterStyle)
	assert.NotNil(t, theme.ModalBoxStyle)
	assert.NotNil(t, theme.BoxHeaderStyle)
	assert.NotNil(t, theme.ButtonStyle)
	assert.NotNil(t, theme.ButtonPrimaryStyle)
	assert.NotNil(t, theme.ButtonDangerStyle)
	assert.NotNil(t, theme.ModalTitleStyle)
	assert.NotNil(t, theme.ModalMessageStyle)
}

func TestDarkThemeColors(t *testing.T) {
	theme := DarkTheme

	assert.Equal(t, lipgloss.Color("#1a1b26"), theme.Colors.Background)
	assert.Equal(t, lipgloss.Color("#c0caf5"), theme.Colors.Foreground)
	assert.Equal(t, lipgloss.Color("#7aa2f7"), theme.Colors.Primary)
	assert.Equal(t, lipgloss.Color("#bb9af7"), theme.Colors.Secondary)
	assert.Equal(t, lipgloss.Color("#7dcfff"), theme.Colors.Accent)
	assert.Equal(t, lipgloss.Color("#9ece6a"), theme.Colors.Success)
	assert.Equal(t, lipgloss.Color("#e0af68"), theme.Colors.Warning)
	assert.Equal(t, lipgloss.Color("#f7768e"), theme.Colors.Error)
	assert.Equal(t, lipgloss.Color("#7dcfff"), theme.Colors.Info)
	assert.Equal(t, lipgloss.Color("#3b4261"), theme.Colors.Border)
	assert.Equal(t, lipgloss.Color("#7aa2f7"), theme.Colors.BorderFocus)
	assert.Equal(t, lipgloss.Color("#565f89"), theme.Colors.Muted)
	assert.Equal(t, lipgloss.Color("#ff9e64"), theme.Colors.Highlight)
}

func TestDarkThemeBaseStyle(t *testing.T) {
	theme := DarkTheme

	assert.NotEmpty(t, theme.BaseStyle.Render("test"))
}

func TestDarkThemeHeaderStyle(t *testing.T) {
	theme := DarkTheme

	assert.NotEmpty(t, theme.HeaderStyle.Base.Render("Test"))
	assert.NotEmpty(t, theme.HeaderStyle.Left.Render("Test"))
	assert.NotEmpty(t, theme.HeaderStyle.Center.Render("Test"))
	assert.NotEmpty(t, theme.HeaderStyle.Right.Render("Test"))
}

func TestDarkThemeFooterStyle(t *testing.T) {
	theme := DarkTheme

	assert.NotEmpty(t, theme.FooterStyle.String())
}

func TestDarkThemeModalBoxStyle(t *testing.T) {
	theme := DarkTheme

	assert.NotEmpty(t, theme.ModalBoxStyle.String())
}

func TestDarkThemeBoxHeaderStyle(t *testing.T) {
	theme := DarkTheme

	assert.NotEmpty(t, theme.BoxHeaderStyle.String())
}

func TestDarkThemeButtonStyle(t *testing.T) {
	theme := DarkTheme

	assert.NotEmpty(t, theme.ButtonStyle.String())
	assert.NotEmpty(t, theme.ButtonPrimaryStyle.String())
	assert.NotEmpty(t, theme.ButtonDangerStyle.String())
}

func TestDarkThemeModalTitleStyle(t *testing.T) {
	theme := DarkTheme

	assert.NotEmpty(t, theme.ModalTitleStyle.String())
}

func TestDarkThemeModalMessageStyle(t *testing.T) {
	theme := DarkTheme

	assert.NotEmpty(t, theme.ModalMessageStyle.Render("Test"))
}

func TestThemeName(t *testing.T) {
	theme := DarkTheme

	assert.Equal(t, "dark", theme.Name)
}

func TestThemeColorScheme(t *testing.T) {
	theme := DarkTheme

	tests := []struct {
		name  string
		color lipgloss.Color
	}{
		{"Background", theme.Colors.Background},
		{"Foreground", theme.Colors.Foreground},
		{"Primary", theme.Colors.Primary},
		{"Secondary", theme.Colors.Secondary},
		{"Accent", theme.Colors.Accent},
		{"Success", theme.Colors.Success},
		{"Warning", theme.Colors.Warning},
		{"Error", theme.Colors.Error},
		{"Info", theme.Colors.Info},
		{"Border", theme.Colors.Border},
		{"BorderFocus", theme.Colors.BorderFocus},
		{"Muted", theme.Colors.Muted},
		{"Highlight", theme.Colors.Highlight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.color)
			assert.NotEqual(t, lipgloss.Color(""), tt.color)
		})
	}
}

func TestThemeStyles(t *testing.T) {
	theme := DarkTheme

	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"BaseStyle", theme.BaseStyle},
		{"FooterStyle", theme.FooterStyle},
		{"ModalBoxStyle", theme.ModalBoxStyle},
		{"BoxHeaderStyle", theme.BoxHeaderStyle},
		{"ButtonStyle", theme.ButtonStyle},
		{"ButtonPrimaryStyle", theme.ButtonPrimaryStyle},
		{"ButtonDangerStyle", theme.ButtonDangerStyle},
		{"ModalTitleStyle", theme.ModalTitleStyle},
		{"ModalMessageStyle", theme.ModalMessageStyle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.style)
		})
	}
}

func TestThemeHeaderStyles(t *testing.T) {
	theme := DarkTheme

	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"Header.Base", theme.HeaderStyle.Base},
		{"Header.Left", theme.HeaderStyle.Left},
		{"Header.Center", theme.HeaderStyle.Center},
		{"Header.Right", theme.HeaderStyle.Right},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.style)
		})
	}
}

func TestThemeColorValues(t *testing.T) {
	theme := DarkTheme

	expectedColors := map[string]string{
		"Background":  "#1a1b26",
		"Foreground":  "#c0caf5",
		"Primary":     "#7aa2f7",
		"Secondary":   "#bb9af7",
		"Accent":      "#7dcfff",
		"Success":     "#9ece6a",
		"Warning":     "#e0af68",
		"Error":       "#f7768e",
		"Info":        "#7dcfff",
		"Border":      "#3b4261",
		"BorderFocus": "#7aa2f7",
		"Muted":       "#565f89",
		"Highlight":   "#ff9e64",
	}

	colorMap := map[string]lipgloss.Color{
		"Background":  theme.Colors.Background,
		"Foreground":  theme.Colors.Foreground,
		"Primary":     theme.Colors.Primary,
		"Secondary":   theme.Colors.Secondary,
		"Accent":      theme.Colors.Accent,
		"Success":     theme.Colors.Success,
		"Warning":     theme.Colors.Warning,
		"Error":       theme.Colors.Error,
		"Info":        theme.Colors.Info,
		"Border":      theme.Colors.Border,
		"BorderFocus": theme.Colors.BorderFocus,
		"Muted":       theme.Colors.Muted,
		"Highlight":   theme.Colors.Highlight,
	}

	for name, color := range colorMap {
		t.Run(name, func(t *testing.T) {
			expected, ok := expectedColors[name]
			assert.True(t, ok, "Expected color not found")
			assert.Equal(t, lipgloss.Color(expected), color)
		})
	}
}

func TestThemeButtonStyles(t *testing.T) {
	theme := DarkTheme

	normal := theme.ButtonStyle.Render("Cancel")
	primary := theme.ButtonPrimaryStyle.Render("Submit")
	danger := theme.ButtonDangerStyle.Render("Delete")

	assert.NotEmpty(t, normal)
	assert.NotEmpty(t, primary)
	assert.NotEmpty(t, danger)

	assert.NotEqual(t, normal, primary)
	assert.NotEqual(t, normal, danger)
	assert.NotEqual(t, primary, danger)
}

func TestThemeModalStyles(t *testing.T) {
	theme := DarkTheme

	title := theme.ModalTitleStyle.Render("Title")
	message := theme.ModalMessageStyle.Render("Message")
	box := theme.ModalBoxStyle.Render("Content")

	assert.NotEmpty(t, title)
	assert.NotEmpty(t, message)
	assert.NotEmpty(t, box)
}

func TestThemeHeader(t *testing.T) {
	theme := DarkTheme

	left := theme.HeaderStyle.Left.Render("SWARM")
	center := theme.HeaderStyle.Center.Render("Info")
	right := theme.HeaderStyle.Right.Render("Time")

	assert.NotEmpty(t, left)
	assert.NotEmpty(t, center)
	assert.NotEmpty(t, right)

	assert.NotEqual(t, left, center)
	assert.NotEqual(t, left, right)
}

func TestThemeSidebar(t *testing.T) {
	theme := DarkTheme

	item := theme.SidebarStyle.Item.Render("Dashboard")
	selected := theme.SidebarStyle.Selected.Render("Dashboard")

	assert.NotEmpty(t, item)
	assert.NotEmpty(t, selected)
}
