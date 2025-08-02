package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/Broderick-Westrope/charmutils"
	"github.com/OmegaRelay/mqtt-tui/connection"
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

type newConnectionModel struct {
	cursor int
	Inputs struct {
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
	mi := reflect.ValueOf(&m.Inputs).Elem()
	for i := range mi.NumField() {
		v := mi.Field(i)
		_, ok := v.Interface().(textinput.Model)
		if ok {
			v.Set(reflect.ValueOf(textinput.New()))
		}
	}

	return m
}

func (m newConnectionModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m newConnectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			nrInputs := reflect.ValueOf(m.Inputs).NumField()
			if m.cursor < nrInputs {
				mi := reflect.ValueOf(&m.Inputs).Elem()
				for i := range mi.NumField() {
					v := mi.Field(i)
					switch v := v.Interface().(type) {
					case textinput.Model:
						if m.cursor == i {
							if v.Focused() {
								v.Blur()
								reflect.ValueOf(&m.Inputs).Elem().Field(m.cursor).Set(reflect.ValueOf(v))
							} else {
								v.Focus()
								mi.Field(i).Set(reflect.ValueOf(v))
							}
						} else {
							v.Blur()
							mi.Field(i).Set(reflect.ValueOf(v))
						}
					case bool:
						if m.cursor == i {
							if v {
								tmp := false
								mi.Field(i).SetBool(tmp)
							} else {
								tmp := true
								mi.Field(i).SetBool(tmp)
							}
						}
					}
				}
			} else if m.cursor == nrInputs { // cancel
				return nil, nil
			} else { // submit
				return nil, m.complete
			}

		case "j":
			if m.cursor < reflect.ValueOf(m.Inputs).NumField() {
				f, ok := reflect.ValueOf(m.Inputs).Field(m.cursor).Interface().(textinput.Model)
				if ok && f.Focused() {
					break
				}
			}

			m.cursor++
			nrInputs := (reflect.ValueOf(m.Inputs).NumField() + 2)
			if m.cursor >= nrInputs {
				m.cursor = nrInputs - 1
			}

		case "k":
			if m.cursor < reflect.ValueOf(m.Inputs).NumField() {
				f, ok := reflect.ValueOf(m.Inputs).Field(m.cursor).Interface().(textinput.Model)
				if ok && f.Focused() {
					break
				}
			}

			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}

		case " ":

		case "esc":
			if m.cursor < reflect.ValueOf(m.Inputs).NumField() {
				f, ok := reflect.ValueOf(m.Inputs).Field(m.cursor).Interface().(textinput.Model)
				if ok && f.Focused() {
					f.SetValue("")
					f.Blur()
					reflect.ValueOf(&m.Inputs).Elem().Field(m.cursor).Set(reflect.ValueOf(f))
					break
				}
			}
			return nil, nil

		}
	}
	cmds := make([]tea.Cmd, 0)
	mi := reflect.ValueOf(&m.Inputs).Elem()
	for i := range mi.NumField() {
		v, ok := mi.Field(i).Interface().(textinput.Model)
		if ok {
			var cmd tea.Cmd
			v, cmd = v.Update(msg)
			cmds = append(cmds, cmd)
			mi.Field(i).Set(reflect.ValueOf(v))

		}
	}

	return m, tea.Batch(cmds...)
}

func (m newConnectionModel) View() string {
	var content strings.Builder

	content.WriteString("New Connection\n")
	mi := reflect.ValueOf(m.Inputs)
	for i := range mi.NumField() {
		t := mi.Type().Field(i)
		v := mi.Field(i)

		cursor := "   "
		if m.cursor == i {
			cursor = " > "
		}
		name := t.Name
		input := v.Kind().String()

		switch v.Kind() {
		case reflect.Bool:
			if v.Bool() {
				input = "[x]"
			} else {
				input = "[ ]"
			}
		case reflect.Struct:
			switch v := v.Interface().(type) {
			case textinput.Model:
				input = v.View()
			}
		default:
			continue
		}

		content.WriteString(fmt.Sprintf("%s%s %s\n", cursor, name, input))
	}
	content.WriteString("\n")

	if m.cursor == mi.NumField() {
		content.WriteString(" [cancel] ")
	} else {
		content.WriteString("  cancel  ")
	}

	if m.cursor == (mi.NumField() + 1) {
		content.WriteString(" [submit] ")
	} else {
		content.WriteString("  submit  ")
	}

	width, _, _ := term.GetSize(0)
	widget := viewport.New(width-4, 20)
	widget.SetContent(content.String())
	return styles.FocusedBorderStyle.Render(widget.View())

}

func (m newConnectionModel) complete() tea.Msg {
	port, _ := strconv.ParseInt(m.Inputs.Port.Value(), 10, 32)
	newModel := connection.NewModel(
		connection.Data{
			Name:         m.Inputs.Name.Value(),
			Broker:       m.Inputs.Broker.Value(),
			Port:         int(port),
			ClientId:     m.Inputs.ClientId.Value(),
			Username:     m.Inputs.Username.Value(),
			Password:     m.Inputs.Password.Value(),
			UseTls:       m.Inputs.UseTls,
			Authenticate: m.Inputs.Authenticate,
			KeyFilePath:  m.Inputs.Keyfile.Value(),
			CertFilePath: m.Inputs.Certfile.Value(),
			CaFilePath:   m.Inputs.CaFile.Value(),
		},
		nil,
	)

	return newConnectionMsg(newModel)
}
