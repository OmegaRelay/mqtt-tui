package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/Broderick-Westrope/charmutils"
	"github.com/OmegaRelay/mqtt-tui/connection"
	"github.com/OmegaRelay/mqtt-tui/form"
	"github.com/OmegaRelay/mqtt-tui/styles"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	connections   list.Model
	connection    tea.Model
	newConnection tea.Model
}

type newConnectionMsg connection.Model

type newConnectionInputs struct {
	Name         textinput.Model
	ClientId     textinput.Model
	Broker       textinput.Model
	Port         textinput.Model
	Username     textinput.Model
	Password     textinput.Model
	UseTls       bool
	Authenticate bool
	Keyfile      textinput.Model
	Certfile     textinput.Model
	CaFile       textinput.Model
}

type newConnectionModel struct {
	form form.Model
}

const kConnectionSaveFileName = "connections.json"

var gCacheDir string

func main() {
	err := initStorage()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	connections := make([]connection.Data, 0)
	data, err := os.ReadFile(path.Join(gCacheDir, kConnectionSaveFileName))
	if err != nil {
		if os.IsNotExist(err) {
			os.WriteFile(path.Join(gCacheDir, kConnectionSaveFileName), []byte("[]"), 0660)
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		err = json.Unmarshal(data, &connections)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	delegate := list.NewDefaultDelegate()
	items := make([]list.Item, 0)
	for _, v := range connections {
		items = append(items, connection.NewModel(v, nil))
	}
	conns := list.New(items, delegate, 10, 10)
	conns.Title = "Connections"

	model := model{connections: conns}
	p := tea.NewProgram(model,
		tea.WithAltScreen(), tea.WithReportFocus())
	_, err = p.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initStorage() error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("could not get user cache directory: %w", err)
	}
	gCacheDir = path.Join(cacheDir, "mqtt-tui")
	err = os.MkdirAll(gCacheDir, 0777)
	if err != nil {
		return fmt.Errorf("could not create cache directory: %w", err)
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil

	if m.connection != nil {
		m.connection, cmd = m.connection.Update(msg)
		return m, cmd
	} else if m.newConnection != nil {
		m.newConnection, cmd = m.newConnection.Update(msg)
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
			m.newConnection = NewConnectionModel()
			return m, m.newConnection.Init()
		case "r":
			items := m.connections.Items()
			if len(items) == 0 {
				break
			}
			m.connections.RemoveItem(m.connections.GlobalIndex())
			m.saveConnections()
		case "j":
			m.connections.CursorDown()
		case "k":
			m.connections.CursorUp()
		case "enter":
			m.connection = m.connections.Items()[m.connections.GlobalIndex()].(connection.Model)
			cmd := m.connection.Init()
			return m, cmd
		}
	case newConnectionMsg:
		items := m.connections.Items()
		items = append(items, connection.Model(msg))
		m.connections.SetItems(items)
		m.saveConnections()
	}

	return m, nil
}

func (m model) View() string {
	if m.connection != nil {
		return m.connection.View()
	}

	borderStyle := styles.FocusedBorderStyle
	if m.newConnection != nil {
		borderStyle = styles.BlurredBorderStyle
	}

	width, height, _ := term.GetSize(0)
	connectionsWidget := viewport.New(27, height-11)
	connectionsWidget.SetContent(m.connections.View())

	s := lipgloss.JoinVertical(lipgloss.Top, title, borderStyle.Render(connectionsWidget.View()))
	widget := viewport.New(width-2, height-2)
	widget.SetContent(s)
	s = styles.FocusedBorderStyle.Render(widget.View())

	if m.newConnection != nil {
		s, _ = charmutils.OverlayCenter(s, m.newConnection.View(), false)
	}
	return s
}

func (m model) saveConnections() {
	connectionsData := make([]connection.Data, 0)
	items := m.connections.Items()
	for _, v := range items {
		v, ok := v.(connection.Model)
		if !ok {
			continue
		}
		connectionsData = append(connectionsData, v.Data())
	}

	data, _ := json.MarshalIndent(connectionsData, "", "  ")
	os.WriteFile(path.Join(gCacheDir, kConnectionSaveFileName), data, 0660)
}

func NewConnectionModel() newConnectionModel {
	m := newConnectionModel{}
	m.form = form.New("New Connection", &newConnectionInputs{})
	return m
}

func (m newConnectionModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m newConnectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case form.SubmitMsg:
		return nil, m.complete
	case form.CancelMsg:
		return nil, nil
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m newConnectionModel) View() string {
	content := m.form.View()
	width, _, _ := term.GetSize(0)
	widget := viewport.New(width-4, 20)
	widget.SetContent(content)
	return styles.FocusedBorderStyle.Render(widget.View())
}

func (m newConnectionModel) complete() tea.Msg {
	inputs := m.form.Inputs().(*newConnectionInputs)
	port, _ := strconv.ParseInt(inputs.Port.Value(), 10, 32)
	newModel := connection.NewModel(
		connection.Data{
			Name:         inputs.Name.Value(),
			Broker:       inputs.Broker.Value(),
			Port:         int(port),
			ClientId:     inputs.ClientId.Value(),
			Username:     inputs.Username.Value(),
			Password:     inputs.Password.Value(),
			UseTls:       inputs.UseTls,
			Authenticate: inputs.Authenticate,
			KeyFilePath:  inputs.Keyfile.Value(),
			CertFilePath: inputs.Certfile.Value(),
			CaFilePath:   inputs.CaFile.Value(),
		},
		nil,
	)

	return newConnectionMsg(newModel)
}
