package connection

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/Broderick-Westrope/charmutils"
	"github.com/OmegaRelay/mqtt-tui/subscription"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	connectionStateConnecting int32 = iota
	connectionStateReconnecting
	connectionStateConnected
	connectionStateDisconnected
)

var qosChoices = []string{
	"At most once",
	"At least once",
	"Exactly once",
}

type newSubModel struct {
	model    subscription.Model
	topic    textinput.Model
	focusQos bool
	qos      int
	cursor   int
}

type Model struct {
	broker   string
	port     int
	clientId string

	client mqtt.Client

	hasConnected    *atomic.Bool
	connectionState *atomic.Int32
	addNewSub       bool
	newSub          newSubModel
	subscriptions   list.Model
	messageIdx      int
	spinner         spinner.Model
	subscritionIdx  int
	windowSize      struct {
		height int
		width  int
	}
}

// Styles
var (
	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("38"))

	blurredBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))

	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
)

func NewModel(broker string, port int, clientID string, initSubs []subscription.Model) Model {
	delegate := list.NewDefaultDelegate()
	items := make([]list.Item, 0)
	if initSubs != nil {
		for _, sub := range initSubs {
			items = append(items, sub)
		}
	}

	delegate.ShowDescription = false
	subs := list.New(items, delegate, 10, 10)
	subs.SetShowTitle(false)

	m := Model{
		broker:          broker,
		port:            port,
		clientId:        clientID,
		subscriptions:   subs,
		hasConnected:    &atomic.Bool{},
		connectionState: &atomic.Int32{},
		spinner:         spinner.New(spinner.WithSpinner(spinner.Ellipsis), spinner.WithStyle(spinnerStyle)),
	}
	m.newSub.topic = textinput.New()
	m.newSub.topic.Cursor.Blink = true
	m.newSub.topic.CharLimit = 32
	m.newSub.topic.Focus()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", m.broker, m.port))
	opts.SetClientID(m.clientId)
	opts.SetDefaultPublishHandler(m.onPubHandler)
	opts.OnConnect = m.onConnectHandler
	opts.OnConnectionLost = m.onConnectionLostHandler
	opts.OnReconnecting = m.onReconnectingHandler
	opts.OnConnectAttempt = m.onConnectAttemptHandler
	opts.ConnectRetry = true
	opts.AutoReconnect = true
	m.client = mqtt.NewClient(opts)

	return m
}

func (m Model) onPubHandler(client mqtt.Client, msg mqtt.Message) {
}

func (m Model) onConnectHandler(client mqtt.Client) {
	m.hasConnected.Store(true)
	m.connectionState.Store(connectionStateConnected)
}

func (m Model) onConnectionLostHandler(client mqtt.Client, err error) {
	m.connectionState.Store(connectionStateDisconnected)
}

func (m Model) onReconnectingHandler(client mqtt.Client, opts *mqtt.ClientOptions) {
	m.connectionState.Store(connectionStateReconnecting)
}

func (m Model) onConnectAttemptHandler(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
	if m.hasConnected.Load() {
		m.connectionState.Store(connectionStateReconnecting)
	} else {
		m.connectionState.Store(connectionStateConnecting)
	}
	return tlsCfg
}

func (m Model) Init() tea.Cmd {
	m.client.Connect()
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.subscriptions.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		_, cmd := m.handleBaseKey(msg)
		if cmd != nil {
			return m, cmd
		}
		if m.addNewSub {
			return m.handleAddSubKey(msg)
		}
		return m.handleDefaultKey(msg)
	case tea.WindowSizeMsg:
		m.windowSize.height = msg.Height
		m.windowSize.width = msg.Width
	}

	if m.addNewSub {
		var cmd tea.Cmd
		if !m.newSub.focusQos {
			m.newSub.topic, cmd = m.newSub.topic.Update(msg)
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) handleBaseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m Model) handleDefaultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd = nil
	switch msg.String() {
	case "a":
		m.newSub.model = subscription.NewModel()
		m.addNewSub = true
		cmd = textinput.Blink
	case "r":
		items := m.subscriptions.Items()
		if len(items) == 0 {
			break
		}
		sub := items[m.subscriptions.GlobalIndex()].(subscription.Model)
		m.client.Unsubscribe(sub.Topic)
		m.subscriptions.RemoveItem(m.subscriptions.GlobalIndex())
	case "j":
		m.messageIdx = 0
		m.subscriptions.CursorDown()
	case "k":
		m.messageIdx = 0
		m.subscriptions.CursorUp()
	case "h":
		items := m.subscriptions.Items()
		if len(items) == 0 {
			break
		}
		sub := items[m.subscriptions.GlobalIndex()].(subscription.Model)
		messages := sub.Messages()
		if len(messages) == 0 {
			break
		}
		m.messageIdx = max(0, m.messageIdx-1)
		m.subscriptions.SetItem(m.subscriptions.GlobalIndex(), sub)
	case "l":
		items := m.subscriptions.Items()
		if len(items) == 0 {
			break
		}
		sub := items[m.subscriptions.GlobalIndex()].(subscription.Model)
		messages := sub.Messages()
		if len(messages) == 0 {
			break
		}
		m.messageIdx = min(len(messages)-1, m.messageIdx+1)
		m.subscriptions.SetItem(m.subscriptions.GlobalIndex(), sub)
	}
	return m, cmd
}

func (m Model) handleAddSubKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.newSub.focusQos {
			m.newSub.focusQos = false
		} else {
			m.newSub.focusQos = true
		}

	case "j":
		if m.newSub.focusQos {
			m.newSub.cursor++
			m.newSub.cursor %= len(qosChoices)
		}
	case "k":
		if m.newSub.focusQos {
			m.newSub.cursor--
			if m.newSub.cursor < 0 {
				m.newSub.cursor = 0
			}
		}
	case " ":
		if m.newSub.focusQos {
			m.newSub.qos = m.newSub.cursor
		}

	case "enter":
		m.newSub.model.Topic = m.newSub.topic.Value()
		m.newSub.model.Qos = byte(m.newSub.qos)
		items := m.subscriptions.Items()
		items = append(items, m.newSub.model)
		m.client.Subscribe(m.newSub.model.Topic, m.newSub.model.Qos, m.newSub.model.OnPubHandler)
		m.subscriptions.SetItems(items)
		m.addNewSub = false
	case "esc":
		m.newSub.model = subscription.NewModel()
		m.newSub.topic.SetValue("")
		m.addNewSub = false
	}
	var cmd tea.Cmd
	if !m.newSub.focusQos {
		m.newSub.topic, cmd = m.newSub.topic.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	state := m.connectionState.Load()

	if state == connectionStateConnecting {
		return m.connectingView()
	} else {
		return m.defaultView(m.addNewSub)
	}
}

func (m Model) connectingView() string {
	return fmt.Sprintf("Connecting to broker on %s:%d%s", m.broker, m.port, m.spinner.View())
}

func (m Model) defaultView(isBg bool) string {
	var borderStyle lipgloss.Style
	if isBg {
		borderStyle = blurredBorderStyle
	} else {
		borderStyle = focusedBorderStyle
	}

	m.subscriptions.SetSize(27, m.windowSize.height-12)
	subsListView := borderStyle.Render(m.subscriptions.View())

	l := lipgloss.Place(27, 6, lipgloss.Left, lipgloss.Top, "")
	l = borderStyle.Render(l)

	broker := viewport.New(27, 1)
	broker.SetContent(fmt.Sprintf("tcp://%s:%d", m.broker, m.port))
	brokerView := borderStyle.Render(broker.View())
	clientId := viewport.New(27, 1)
	clientId.SetContent(m.clientId)
	clientIdView := borderStyle.Render(clientId.View())
	leftView := lipgloss.JoinVertical(lipgloss.Top, brokerView, clientIdView, subsListView)

	recvTopic := viewport.New(m.windowSize.width-44, 1)
	messageNr := viewport.New(7, 1)
	recvAt := viewport.New(m.windowSize.width-35, 1)
	data := viewport.New(m.windowSize.width-35, m.windowSize.height-(6+6))
	subItems := m.subscriptions.Items()

	if len(subItems) > 0 {
		sub, _ := subItems[m.subscriptions.GlobalIndex()].(subscription.Model)
		messages := sub.Messages()
		messageNr.SetContent(fmt.Sprintf("%d/%d", min(m.messageIdx+1, len(messages)), len(messages)))
		if len(messages) > 0 {
			message := messages[m.messageIdx]
			recvTopic.SetContent(string(message.RecvTopic()))
			recvAt.SetContent(string(message.RecvAt().String()))
			data.SetContent(string(message.Data()))
		}
	}
	recvTopicView := borderStyle.Render(recvTopic.View())
	messageNrView := borderStyle.Render(messageNr.View())
	messagesHeaderView := lipgloss.JoinHorizontal(lipgloss.Left, recvTopicView, messageNrView)
	recvAtView := borderStyle.Render(recvAt.View())
	dataView := borderStyle.Render(data.View())
	messagesView := lipgloss.JoinVertical(lipgloss.Top, messagesHeaderView, recvAtView, dataView)
	messagesView = borderStyle.Render(messagesView)

	s := lipgloss.JoinHorizontal(lipgloss.Left, leftView, messagesView)

	if isBg {
		// add foreground widget
		if m.addNewSub {
			s, _ = charmutils.OverlayCenter(s, m.newSub.View(), false)
		}
	}

	return focusedBorderStyle.Render(s)
}

func (m newSubModel) Init() tea.Cmd {
	return nil
}

func (m newSubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m newSubModel) View() string {
	var content strings.Builder

	content.WriteString("New Subscription\n\n")
	content.WriteString("Topic\n")
	if m.focusQos {
		content.WriteString(m.topic.Value())
	} else {
		content.WriteString(m.topic.View())
	}
	content.WriteString("\n\n")
	content.WriteString("QoS\n")

	for i, choice := range qosChoices {
		cursor := " "
		if m.focusQos && m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if i == m.qos {
			checked = "x"
		}

		content.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice))
	}

	width, _, _ := term.GetSize(0)
	widget := viewport.New(width-4, 9)
	widget.SetContent(content.String())
	return focusedBorderStyle.Render(widget.View())

}
