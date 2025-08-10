package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"time"

	"github.com/Broderick-Westrope/charmutils"
	"github.com/OmegaRelay/mqtt-tui/connection"
	"github.com/OmegaRelay/mqtt-tui/form"
	"github.com/OmegaRelay/mqtt-tui/program"
	"github.com/OmegaRelay/mqtt-tui/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/google/uuid"
)

const kTitle = `███╗   ███╗ ██████╗ ████████╗████████╗    ████████╗██╗   ██╗██╗
████╗ ████║██╔═══██╗╚══██╔══╝╚══██╔══╝    ╚══██╔══╝██║   ██║██║
██╔████╔██║██║   ██║   ██║      ██║          ██║   ██║   ██║██║
██║╚██╔╝██║██║▄▄ ██║   ██║      ██║          ██║   ██║   ██║██║
██║ ╚═╝ ██║╚██████╔╝   ██║      ██║          ██║   ╚██████╔╝██║
╚═╝     ╚═╝ ╚══▀▀═╝    ╚═╝      ╚═╝          ╚═╝    ╚═════╝ ╚═╝
`

const kErrorPopupDuration = 10 * time.Second

type model struct {
	connections    list.Model
	connection     tea.Model
	newConnection  tea.Model
	editConnection bool

	keys keyMap
	help help.Model

	err      error
	errTimer *time.Timer
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
	KeyFile      textinput.Model
	CertFile     textinput.Model
	CaFile       textinput.Model
}

type newConnectionModel struct {
	form form.Model
}

const kConnectionSaveFileName = "connections.json"

var (
	gCacheDir string
	gProgram  tea.Program
)

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
		items = append(items, connection.NewModel(v))
	}
	conns := list.New(items, delegate, 10, 10)
	conns.Title = "Connections"
	conns.SetShowHelp(false)

	model := model{
		connections: conns,
		keys:        keys,
		help:        help.New(),
	}
	gProgram := tea.NewProgram(model,
		tea.WithAltScreen(), tea.WithReportFocus(), tea.WithoutCatchPanics())
	program.SetProgram(gProgram)

	_, err = gProgram.Run()
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

	switch msg := msg.(type) {
	case program.ErrorMsg:
		m.err = msg.Err
		if m.err != nil {
			if m.errTimer != nil {
				m.errTimer.Stop()
			}
			m.errTimer = time.NewTimer(kErrorPopupDuration)
			go func() {
				<-m.errTimer.C
				program.SendErrorMsg(nil)
			}()
		}
	}

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
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Add):
			m.newConnection = NewConnectionModel(nil)
			return m, m.newConnection.Init()
		case key.Matches(msg, m.keys.Remove):
			items := m.connections.Items()
			if len(items) == 0 {
				break
			}
			m.connections.RemoveItem(m.connections.GlobalIndex())
			m.saveConnections()
		case key.Matches(msg, m.keys.Edit):
			items := m.connections.Items()
			conn := items[m.connections.GlobalIndex()].(connection.Model)
			inputs := newConnectionInputs{}.Copy(conn)
			m.newConnection = NewConnectionModel(&inputs)
			m.editConnection = true
			return m, m.newConnection.Init()
		case key.Matches(msg, m.keys.Down):
			m.connections.CursorDown()
		case key.Matches(msg, m.keys.Up):
			m.connections.CursorUp()
		case key.Matches(msg, m.keys.Select):
			m.connection = m.connections.Items()[m.connections.GlobalIndex()].(connection.Model)
			cmd := m.connection.Init()
			return m, cmd
		}

	case newConnectionMsg:
		if m.editConnection {
			m.connections.SetItem(m.connections.GlobalIndex(), connection.Model(msg))
		} else {
			items := m.connections.Items()
			items = append(items, connection.Model(msg))
			m.connections.SetItems(items)
		}
		m.saveConnections()
	}

	return m, nil
}

func (m model) View() string {
	s := ""
	if m.connection != nil {
		s = m.connection.View()
	} else {
		borderStyle := styles.FocusedBorderStyle
		if m.newConnection != nil {
			borderStyle = styles.BlurredBorderStyle
		}

		width, height, _ := term.GetSize(0)
		m.connections.SetSize(styles.MenuWidth, height-12)
		connectionsWidget := viewport.New(styles.MenuWidth, height-12)
		connectionsWidget.SetContent(m.connections.View())

		s = lipgloss.JoinVertical(lipgloss.Top, kTitle, borderStyle.Render(connectionsWidget.View()), m.help.View(m.keys))
		widget := viewport.New(width-2, height-2)
		widget.SetContent(s)
		s = styles.FocusedBorderStyle.Render(widget.View())

		if m.newConnection != nil {
			s, _ = charmutils.OverlayCenter(s, m.newConnection.View(), false)
		}
	}

	if m.err != nil {
		msg := "ERROR: " + m.err.Error()
		errorWidget := viewport.New(len(msg), 1)
		errorWidget.SetContent(msg)
		errorView := styles.ErrorBorderStyle.Render(errorWidget.View())
		row := (lipgloss.Height(s) - 3) - (lipgloss.Height(errorView) / 2)
		row = max(0, row)
		col := (lipgloss.Width(s) - lipgloss.Width(errorView)) / 2
		col = max(0, col)
		s, _ = charmutils.Overlay(s, errorView, row, col, false)
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

func NewConnectionModel(inputs *newConnectionInputs) newConnectionModel {
	m := newConnectionModel{}
	if inputs == nil {
		inputs = &newConnectionInputs{}
		m.form = form.New("New Connection", &newConnectionInputs{})
	} else {
		m.form = form.New("New Connection", nil)
		m.form.SetInputs(inputs)
	}
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
			KeyFilePath:  inputs.KeyFile.Value(),
			CertFilePath: inputs.CertFile.Value(),
			CaFilePath:   inputs.CaFile.Value(),
			Id:           uuid.NewString(),
		},
	)

	return newConnectionMsg(newModel)
}

func (m newConnectionInputs) Copy(conn connection.Model) newConnectionInputs {
	r := reflect.ValueOf(&m).Elem()
	for i := range r.NumField() {
		switch r.Field(i).Interface().(type) {
		case textinput.Model:
			r.Field(i).Set(reflect.ValueOf(textinput.New()))
		}
	}

	data := conn.Data()

	m.Name.SetValue(data.Name)
	m.ClientId.SetValue(data.ClientId)
	m.Broker.SetValue(data.Broker)
	m.Port.SetValue(strconv.FormatInt(int64(data.Port), 10))
	m.Username.SetValue(data.Username)
	m.Password.SetValue(data.Password)
	m.UseTls = data.UseTls
	m.Authenticate = data.Authenticate
	m.KeyFile.SetValue(data.KeyFilePath)
	m.CertFile.SetValue(data.CertFilePath)
	m.CaFile.SetValue(data.CaFilePath)

	return m
}
