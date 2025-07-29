package main

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

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

type Message struct {
	recvTopic string
	recvAt time.Time
	data []byte
}

type Subscription struct {
	topic string
	messages *[]Message 
	messagesMu *sync.Mutex
	messageIdx int
	qos byte
}

type CnxModel struct {
	broker string
	port int
	clientId string

	client mqtt.Client

	connectionState *atomic.Int32
	subscriptions list.Model
	spinner spinner.Model
	subscritionIdx int
	windowSize struct {
		height int
		width int
	}
}

var defaultMessages = []Message {
	{
		recvTopic: "test/1",
		recvAt: time.Date(2025, time.July, 27, 0, 0, 0, 0, time.UTC),
		data: []byte("hello world"),
	},
	{
		recvTopic: "test/2",
		recvAt: time.Date(2025, time.July, 27, 0, 0, 0, 0, time.UTC),
		data: []byte("{\"hello\": \"world\"}"),
	},
}

var defaultSub = Subscription{
	topic: fmt.Sprintf("topic/#"),
	messages: &[]Message{},
	messagesMu: &sync.Mutex{},
}

// Styles
var (
	borderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
)

func NewCnxModel(broker string, port int, clientID string, initSubs []Subscription) tea.Model {	
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

	m := CnxModel{
		broker: broker,
		port: port,
		clientId: clientID,
		subscriptions: subs,
		connectionState: &atomic.Int32{},
		spinner: spinner.New(spinner.WithSpinner(spinner.Ellipsis), spinner.WithStyle(spinnerStyle)),
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

func (s Subscription) Title() string       { return s.topic }
func (s Subscription) Description() string { return "" }
func (s Subscription) FilterValue() string { return s.topic }

func (s Subscription) onPubHandler(client mqtt.Client, msg mqtt.Message) {
	str := fmt.Sprintf("subscription: Pub received on %s; %s\n", msg.Topic(), string(msg.Payload()))
	os.WriteFile("mqttui.log", []byte(str), 0666)

	s.messagesMu.Lock()
	defer s.messagesMu.Unlock()
	m := *s.messages
	m = append(m, Message{
		recvTopic: msg.Topic(), 
		recvAt: time.Now(), 
		data: msg.Payload(),})
	s.messages = &m
}

func (m CnxModel) onPubHandler(client mqtt.Client, msg mqtt.Message) {
	s := fmt.Sprintf("Pub received on %s; %s\n", msg.Topic(), string(msg.Payload()))
	os.WriteFile("mqttui.log", []byte(s), 0666)
}

func (m CnxModel) onConnectHandler(client mqtt.Client) {
	os.WriteFile("mqttui.log", []byte("connected\n"), 0666)
	m.connectionState.Store(connectionStateConnected)
}

func (m CnxModel) onConnectionLostHandler(client mqtt.Client, err error) {
	os.WriteFile("mqttui.log", []byte("connection lost: " + err.Error() + "\n"), 0666)
	m.connectionState.Store(connectionStateDisconnected)
}

func (m CnxModel) onReconnectingHandler(client mqtt.Client, opts *mqtt.ClientOptions) {
	os.WriteFile("mqttui.log", []byte("reconnecting\n"), 0666)
	m.connectionState.Store(connectionStateReconnecting)
}

func (m CnxModel) onConnectAttemptHandler(broker *url.URL, tlsCfg *tls.Config) *tls.Config{
	os.WriteFile("mqttui.log", []byte("connecting to " + broker.String() + "\n"), 0666)
	m.connectionState.Store(connectionStateConnecting)
	return tlsCfg
}

func (m CnxModel) Init() tea.Cmd {

	os.WriteFile("mqttui.log", []byte("initialized\n"), 0666)

	m.client.Connect()
    return m.spinner.Tick
}

var itemCounter = 0

func (m CnxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.client.Subscribe(defaultSub.topic, defaultSub.qos, defaultSub.onPubHandler)
			m.subscriptions.SetItems(items)
			// TODO: show subscription dialog
		case "r":
			m.subscriptions.RemoveItem(m.subscriptions.GlobalIndex())
		case "j":
			m.subscriptions.CursorDown()
		case "k":
			m.subscriptions.CursorUp()
		case "h":
			items := m.subscriptions.Items()
			if len(items) == 0 {
				break
			}
			sub := items[m.subscriptions.GlobalIndex()].(Subscription)
			sub.messagesMu.Lock()
			messages := *sub.messages
			if len(messages) == 0 {
				break
			}
			sub.messageIdx = max(0, sub.messageIdx-1)
			m.subscriptions.SetItem(m.subscriptions.GlobalIndex(), sub)
			sub.messagesMu.Unlock()
		case "l":
			items := m.subscriptions.Items()
			if len(items) == 0 {
				break
			}
			sub := items[m.subscriptions.GlobalIndex()].(Subscription)
			sub.messagesMu.Lock()
			messages := *sub.messages
			if len(messages) == 0 {
				break
			}
			sub.messageIdx = min(len(messages) - 1, sub.messageIdx+1)
			m.subscriptions.SetItem(m.subscriptions.GlobalIndex(), sub)
			sub.messagesMu.Unlock()
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

func (m CnxModel) View() string {
	state := m.connectionState.Load()

	if state == connectionStateConnecting {
		return m.connectingView()
	} else {
		return m.defaultView()
	}
}

func (m CnxModel) connectingView() string {
	return fmt.Sprintf("Connecting to broker on %s:%d%s", m.broker, m.port, m.spinner.View())
}

func (m CnxModel) defaultView() string {
	m.subscriptions.SetSize(27, m.windowSize.height - 12)
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

	recvTopic := viewport.New(m.windowSize.width - 44, 1)
	messageNr := viewport.New(7, 1)
	recvAt := viewport.New(m.windowSize.width - 35, 1)
	data := viewport.New(m.windowSize.width - 35, m.windowSize.height - (6 + 6))
	subItems := m.subscriptions.Items()

	if (len(subItems) > 0) {
		sub, _ := subItems[m.subscriptions.GlobalIndex()].(Subscription)
		sub.messagesMu.Lock()
		messages := *sub.messages
		messageNr.SetContent(fmt.Sprintf("%d/%d", sub.messageIdx, len(messages)))
		if (len(messages) > 0) {
			message := messages[sub.messageIdx]
			recvTopic.SetContent(string(message.recvTopic))
			recvAt.SetContent(string(message.recvAt.String()))
			data.SetContent(string(message.data))
		}
		sub.messagesMu.Unlock()
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

