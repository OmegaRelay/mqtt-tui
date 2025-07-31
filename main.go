package main

import (
	"github.com/OmegaRelay/mqtt-tui/connection"
	"github.com/OmegaRelay/mqtt-tui/styles"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

const title = `███╗   ███╗ ██████╗ ████████╗████████╗    ████████╗██╗   ██╗██╗
████╗ ████║██╔═══██╗╚══██╔══╝╚══██╔══╝    ╚══██╔══╝██║   ██║██║
██╔████╔██║██║   ██║   ██║      ██║          ██║   ██║   ██║██║
██║╚██╔╝██║██║▄▄ ██║   ██║      ██║          ██║   ██║   ██║██║
██║ ╚═╝ ██║╚██████╔╝   ██║      ██║          ██║   ╚██████╔╝██║
╚═╝     ╚═╝ ╚══▀▀═╝    ╚═╝      ╚═╝          ╚═╝    ╚═════╝ ╚═╝
`

type model struct {
	connections list.Model
	connection  tea.Model
}

func main() {
	delegate := list.NewDefaultDelegate()
	items := make([]list.Item, 0) // TODO: load conns from save file
	conns := list.New(items, delegate, 10, 10)
	conns.Title = "Connections"

	model := model{connections: conns}
	p := tea.NewProgram(model,
		tea.WithAltScreen(), tea.WithReportFocus())
	_, err := p.Run()
	if err != nil {
		panic(err)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil

	if m.connection != nil {
		m.connection, cmd = m.connection.Update(msg)
		return m, cmd
	} else {
		m.connections.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "a":
			cmd = textinput.Blink
			items := m.connections.Items()
			items = append(items, connection.NewModel("name", "localhost", 1883, "client_1", nil))
			m.connections.SetItems(items)
		case "r":
			items := m.connections.Items()
			if len(items) == 0 {
				break
			}
			m.connections.RemoveItem(m.connections.GlobalIndex())
		case "j":
			m.connections.CursorDown()
		case "k":
			m.connections.CursorUp()
		case "enter":
			m.connection = m.connections.Items()[m.connections.GlobalIndex()].(connection.Model)
			cmd := m.connection.Init()
			return m, cmd
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.connection != nil {
		return m.connection.View()
	}

	width, height, _ := term.GetSize(0)
	widget := viewport.New(width-2, height-2)
	s := title + "\n"
	connectionsWidget := viewport.New(27, height-11)
	m.connections.SetHeight(height - 11)
	m.connections.SetWidth(width - 4)
	connectionsWidget.SetContent(m.connections.View())
	s += styles.FocusedBorderStyle.Render(connectionsWidget.View())
	widget.SetContent(s)
	return styles.FocusedBorderStyle.Render(widget.View())
}
