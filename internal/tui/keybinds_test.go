package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyMap(t *testing.T) {
	keymap := DefaultKeyMap()

	assert.NotNil(t, keymap)
	assert.True(t, keymap.Navigation.Up.Enabled())
	assert.True(t, keymap.Navigation.Down.Enabled())
	assert.True(t, keymap.Navigation.Left.Enabled())
	assert.True(t, keymap.Navigation.Right.Enabled())
	assert.True(t, keymap.Navigation.Tab.Enabled())
	assert.True(t, keymap.Navigation.ShiftTab.Enabled())
	assert.True(t, keymap.Navigation.PageUp.Enabled())
	assert.True(t, keymap.Navigation.PageDown.Enabled())
	assert.True(t, keymap.Navigation.Home.Enabled())
	assert.True(t, keymap.Navigation.End.Enabled())

	assert.True(t, keymap.Actions.Enter.Enabled())
	assert.True(t, keymap.Actions.Escape.Enabled())
	assert.True(t, keymap.Actions.Quit.Enabled())
	assert.True(t, keymap.Actions.Help.Enabled())
	assert.True(t, keymap.Actions.Refresh.Enabled())
	assert.True(t, keymap.Actions.Search.Enabled())
	assert.True(t, keymap.Actions.Filter.Enabled())
	assert.True(t, keymap.Actions.Delete.Enabled())
	assert.True(t, keymap.Actions.Confirm.Enabled())
	assert.True(t, keymap.Actions.Cancel.Enabled())

	assert.True(t, keymap.Views.Dashboard.Enabled())
	assert.True(t, keymap.Views.Agents.Enabled())
	assert.True(t, keymap.Views.Tasks.Enabled())
	assert.True(t, keymap.Views.Logs.Enabled())
	assert.True(t, keymap.Views.Config.Enabled())

	assert.True(t, keymap.Modal.Submit.Enabled())
	assert.True(t, keymap.Modal.Close.Enabled())
	assert.True(t, keymap.Modal.NextField.Enabled())
	assert.True(t, keymap.Modal.PrevField.Enabled())
}

func TestNavigationKeyBindings(t *testing.T) {
	keymap := DefaultKeyMap()

	tests := []struct {
		binding  key.Binding
		expected []string
	}{
		{keymap.Navigation.Up, []string{"up", "k"}},
		{keymap.Navigation.Down, []string{"down", "j"}},
		{keymap.Navigation.Left, []string{"left", "h"}},
		{keymap.Navigation.Right, []string{"right", "l"}},
		{keymap.Navigation.Tab, []string{"tab"}},
		{keymap.Navigation.ShiftTab, []string{"shift+tab"}},
		{keymap.Navigation.PageUp, []string{"pgup"}},
		{keymap.Navigation.PageDown, []string{"pgdown"}},
		{keymap.Navigation.Home, []string{"home"}},
		{keymap.Navigation.End, []string{"end"}},
	}

	for _, tt := range tests {
		keys := tt.binding.Keys()
		assert.NotEmpty(t, keys)
		for i, key := range keys {
			assert.Contains(t, tt.expected, key)
			assert.Equal(t, tt.expected[i], key)
		}

		help := tt.binding.Help()
		assert.NotEmpty(t, help.Key)
		assert.NotEmpty(t, help.Desc)
	}
}

func TestActionKeyBindings(t *testing.T) {
	keymap := DefaultKeyMap()

	tests := []struct {
		binding  key.Binding
		expected []string
	}{
		{keymap.Actions.Enter, []string{"enter"}},
		{keymap.Actions.Escape, []string{"esc"}},
		{keymap.Actions.Quit, []string{"q", "ctrl+c"}},
		{keymap.Actions.Help, []string{"?"}},
		{keymap.Actions.Refresh, []string{"r", "f5"}},
		{keymap.Actions.Search, []string{"/"}},
		{keymap.Actions.Filter, []string{"f"}},
		{keymap.Actions.Delete, []string{"d", "delete"}},
		{keymap.Actions.Confirm, []string{"y"}},
		{keymap.Actions.Cancel, []string{"n"}},
	}

	for _, tt := range tests {
		keys := tt.binding.Keys()
		assert.NotEmpty(t, keys)
		for i, key := range keys {
			assert.Contains(t, tt.expected, key)
			if i < len(tt.expected) {
				assert.Equal(t, tt.expected[i], key)
			}
		}

		help := tt.binding.Help()
		assert.NotEmpty(t, help.Key)
		assert.NotEmpty(t, help.Desc)
	}
}

func TestViewKeyBindings(t *testing.T) {
	keymap := DefaultKeyMap()

	tests := []struct {
		binding  key.Binding
		expected []string
	}{
		{keymap.Views.Dashboard, []string{"1"}},
		{keymap.Views.Agents, []string{"2"}},
		{keymap.Views.Tasks, []string{"3"}},
		{keymap.Views.Logs, []string{"4"}},
		{keymap.Views.Config, []string{"5"}},
	}

	for _, tt := range tests {
		keys := tt.binding.Keys()
		assert.NotEmpty(t, keys)
		for i, key := range keys {
			assert.Contains(t, tt.expected, key)
			assert.Equal(t, tt.expected[i], key)
		}

		help := tt.binding.Help()
		assert.NotEmpty(t, help.Key)
		assert.NotEmpty(t, help.Desc)
	}
}

func TestModalKeyBindings(t *testing.T) {
	keymap := DefaultKeyMap()

	tests := []struct {
		binding  key.Binding
		expected []string
	}{
		{keymap.Modal.Submit, []string{"enter"}},
		{keymap.Modal.Close, []string{"esc"}},
		{keymap.Modal.NextField, []string{"tab"}},
		{keymap.Modal.PrevField, []string{"shift+tab"}},
	}

	for _, tt := range tests {
		keys := tt.binding.Keys()
		assert.NotEmpty(t, keys)
		for i, key := range keys {
			assert.Contains(t, tt.expected, key)
			assert.Equal(t, tt.expected[i], key)
		}

		help := tt.binding.Help()
		assert.NotEmpty(t, help.Key)
		assert.NotEmpty(t, help.Desc)
	}
}

func TestKeyMapHelpText(t *testing.T) {
	keymap := DefaultKeyMap()

	tests := []struct {
		name     string
		binding  key.Binding
		expected string
	}{
		{"Up", keymap.Navigation.Up, "up"},
		{"Down", keymap.Navigation.Down, "down"},
		{"Left", keymap.Navigation.Left, "left"},
		{"Right", keymap.Navigation.Right, "right"},
		{"Quit", keymap.Actions.Quit, "quit"},
		{"Help", keymap.Actions.Help, "help"},
		{"Refresh", keymap.Actions.Refresh, "refresh"},
		{"Dashboard", keymap.Views.Dashboard, "dashboard"},
		{"Agents", keymap.Views.Agents, "agents"},
		{"Tasks", keymap.Views.Tasks, "tasks"},
		{"Logs", keymap.Views.Logs, "logs"},
		{"Config", keymap.Views.Config, "config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			help := tt.binding.Help()
			assert.Equal(t, tt.expected, help.Desc)
		})
	}
}
