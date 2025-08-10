package form

import "github.com/charmbracelet/bubbles/key"

type keyMapNormal struct {
	Insert key.Binding
	Next   key.Binding
	Prev   key.Binding
	Cancel key.Binding
	Help   key.Binding
	Quit   key.Binding
}

type keyMapInsert struct {
	Exit key.Binding
	Quit key.Binding
}

func (k keyMapNormal) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.Insert, k.Cancel, k.Help, k.Quit}
}

func (k keyMapNormal) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Prev}, // first column
		{k.Help, k.Quit}, // second column
	}
}

var keysNormal = keyMapNormal{
	Next: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/h", "previous"),
	),
	Insert: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "insert text/cycle options"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/^c", "quit"),
	),
}

func (k keyMapInsert) ShortHelp() []key.Binding {
	return []key.Binding{k.Exit, k.Quit}
}

func (k keyMapInsert) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Exit, k.Quit},
	}
}

var keysInsert = keyMapInsert{
	Exit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "exit insert mode"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/^c", "quit"),
	),
}
