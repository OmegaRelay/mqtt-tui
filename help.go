package main

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Add    key.Binding
	Remove key.Binding
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Quit   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Add, k.Remove, k.Up, k.Down, k.Select, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Add, k.Remove, k.Up, k.Down, k.Select, k.Quit},
	}
}

var keys = keyMap{
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add connection"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "remove connection"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/cycle options"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q/esc/^c", "quit"),
	),
}
