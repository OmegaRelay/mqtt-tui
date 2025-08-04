package connection

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Add          key.Binding
	Remove       key.Binding
	Up           key.Binding
	Down         key.Binding
	Next         key.Binding
	Prev         key.Binding
	JumpToNewest key.Binding
	Escape       key.Binding
	Help         key.Binding
	Quit         key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.Add, k.Remove, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Next, k.Prev, k.JumpToNewest},
		{k.Add, k.Remove},
		{k.Escape, k.Help, k.Quit},
	}
}

var keys = keyMap{
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add subscription"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "remove subscription"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Next: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "next message"),
	),
	Prev: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "previous message"),
	),
	JumpToNewest: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "jumps to newest message"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "return to connections overview"),
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
