package connection

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"

	"github.com/OmegaRelay/mqtt-tui/subscription"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	connectionStateConnecting int32 = iota
	connectionStateReconnecting
	connectionStateConnected
	connectionStateDisconnected
)

type Model struct {
	broker   string
	port     int
	clientId string

	client mqtt.Client

	connectionState *atomic.Int32
	subscriptions   list.Model
	messageIdx      int
	spinner         spinner.Model
	subscritionIdx  int
	windowSize      struct {
		height int
		width  int
	}
}

var defaultSub = subscription.NewModel("topic/#", 0, "json")

// Styles
var (
	borderStyle = lipgloss.NewStyle().
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
		connectionState: &atomic.Int32{},
		spinner:         spinner.New(spinner.WithSpinner(spinner.Ellipsis), spinner.WithStyle(spinnerStyle)),
	}

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
	s := fmt.Sprintf("Pub received on %s; %s\n", msg.Topic(), string(msg.Payload()))
	os.WriteFile("mqttui.log", []byte(s), 0666)
}

func (m Model) onConnectHandler(client mqtt.Client) {
	os.WriteFile("mqttui.log", []byte("connected\n"), 0666)
	m.connectionState.Store(connectionStateConnected)
}

func (m Model) onConnectionLostHandler(client mqtt.Client, err error) {
	os.WriteFile("mqttui.log", []byte("connection lost: "+err.Error()+"\n"), 0666)
	m.connectionState.Store(connectionStateDisconnected)
}

func (m Model) onReconnectingHandler(client mqtt.Client, opts *mqtt.ClientOptions) {
	os.WriteFile("mqttui.log", []byte("reconnecting\n"), 0666)
	m.connectionState.Store(connectionStateReconnecting)
}

func (m Model) onConnectAttemptHandler(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
	os.WriteFile("mqttui.log", []byte("connecting to "+broker.String()+"\n"), 0666)
	m.connectionState.Store(connectionStateConnecting)
	return tlsCfg
}

func (m Model) Init() tea.Cmd {

	os.WriteFile("mqttui.log", []byte("initialized\n"), 0666)

	m.client.Connect()
	return m.spinner.Tick
}

var itemCounter = 0

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.subscriptions.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit
		case "a":
			items := m.subscriptions.Items()
			items = append(items, defaultSub)
			itemCounter++
			m.client.Subscribe(defaultSub.Topic, defaultSub.Qos, defaultSub.OnPubHandler)
			m.subscriptions.SetItems(items)
			// TODO: show subscription dialog
		case "r":
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
	case tea.WindowSizeMsg:
		m.windowSize.height = msg.Height
		m.windowSize.width = msg.Width
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	state := m.connectionState.Load()

	if state == connectionStateConnecting {
		return m.connectingView()
	} else {
		return m.defaultView()
	}
}

func (m Model) connectingView() string {
	return fmt.Sprintf("Connecting to broker on %s:%d%s", m.broker, m.port, m.spinner.View())
}

func (m Model) defaultView() string {
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
	return borderStyle.Render(s)
}
