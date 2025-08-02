package connection

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/Broderick-Westrope/charmutils"
	"github.com/OmegaRelay/mqtt-tui/form"
	"github.com/OmegaRelay/mqtt-tui/program"
	"github.com/OmegaRelay/mqtt-tui/styles"
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
	connectionStateConnecting int = iota
	connectionStateReconnecting
	connectionStateConnected
	connectionStateDisconnected
)

var qosChoices = []string{
	"At most once",
	"At least once",
	"Exactly once",
}

type connectionStateChangeMsg struct {
	connectionState int
}

type NewSubMsg subscription.Model

type newSubInputs struct {
	Name   textinput.Model
	Topic  textinput.Model
	Qos    form.MultipleChoice
	Format form.MultipleChoice
}

type newSubModel struct {
	form form.Model
}

type Data struct {
	Id           string // UUID used to store subscriptions for persistence
	Name         string
	Broker       string
	Port         int
	ClientId     string
	Username     string
	Password     string
	UseTls       bool
	Authenticate bool
	KeyFilePath  string
	CertFilePath string
	CaFilePath   string
}

type Model struct {
	data         Data
	brokerUrl    string
	saveFileName string

	client mqtt.Client

	connectionState int
	newSub          tea.Model
	subscriptions   list.Model
	messageIdx      int
	spinner         spinner.Model
	subscriptionIdx int
}

var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

func NewModel(data Data) Model {
	delegate := list.NewDefaultDelegate()
	items := make([]list.Item, 0)

	m := Model{
		data:          data,
		subscriptions: list.New(items, delegate, 10, 10),
		spinner:       spinner.New(spinner.WithSpinner(spinner.Ellipsis), spinner.WithStyle(spinnerStyle)),
	}
	m.subscriptions.Title = "Subscriptions"

	var err error
	m.saveFileName, err = initSaveFile(data.Id)
	if err != nil {
		panic(err)
	}

	opts := mqtt.NewClientOptions()

	protocol := "mqtt://"
	if data.UseTls {
		protocol = "mqtts://"
	}
	m.brokerUrl = fmt.Sprintf("%s%s:%d", protocol, m.data.Broker, m.data.Port)
	opts.AddBroker(m.brokerUrl)
	opts.SetClientID(m.data.ClientId)
	opts.SetDefaultPublishHandler(m.onPubHandler)
	if data.Username != "" {
		opts.SetUsername(data.Username)
	}
	if data.Password != "" {
		opts.SetPassword(data.Password)
	}

	if data.UseTls {
		tlsCfg := &tls.Config{}
		if data.KeyFilePath != "" && data.CertFilePath != "" {
			tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
			cert, _ := tls.LoadX509KeyPair(data.CertFilePath, data.KeyFilePath)
			tlsCfg = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS13,
			}
		}
		if data.CaFilePath != "" {
			caPem, err := os.ReadFile(data.CaFilePath)
			if err == nil && caPem != nil {
				pool := x509.NewCertPool()
				ok := pool.AppendCertsFromPEM(caPem)
				if ok {
					tlsCfg.ClientCAs = pool
				}
			}
		}
		if !data.Authenticate {
			tlsCfg.InsecureSkipVerify = true
		}
		opts.SetTLSConfig(tlsCfg)
	}

	opts.OnConnect = m.onConnectHandler
	opts.OnConnectionLost = m.onConnectionLostHandler
	opts.OnReconnecting = m.onReconnectingHandler
	opts.OnConnectAttempt = m.onConnectAttemptHandler
	opts.ConnectRetry = true
	opts.AutoReconnect = true
	m.client = mqtt.NewClient(opts)

	subsData, err := os.ReadFile(m.saveFileName)
	if err != nil {
		return m
	}

	var subs []subscription.Data
	err = json.Unmarshal(subsData, &subs)
	if err != nil {
		panic(err)
	}

	for _, sub := range subs {
		newSub := subscription.NewModel(sub)
		items := m.subscriptions.Items()
		items = append(items, newSub)
		m.subscriptions.SetItems(items)
	}

	return m
}

func (m Model) saveSubscriptions() {
	subscriptionsData := make([]subscription.Data, 0)
	for _, v := range m.subscriptions.Items() {
		v, ok := v.(subscription.Model)
		if !ok {
			continue
		}
		subscriptionsData = append(subscriptionsData, v.Data())
	}

	data, _ := json.MarshalIndent(subscriptionsData, "", "  ")
	os.WriteFile(m.saveFileName, data, 0660)
}

func initSaveFile(id string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("could not get user cache directory: %w", err)
	}
	cacheDir = path.Join(cacheDir, "mqtt-tui")
	err = os.MkdirAll(cacheDir, 0777)
	if err != nil {
		return "", fmt.Errorf("could not create cache directory: %w", err)
	}
	saveFilePath := path.Join(cacheDir, fmt.Sprintf("%s.json", id))
	return saveFilePath, nil
}

func (m Model) Title() string       { return m.data.Name }
func (m Model) Description() string { return fmt.Sprintf("%s:%d", m.data.Broker, m.data.Port) }
func (m Model) FilterValue() string { return m.data.Name }

func (m Model) onPubHandler(client mqtt.Client, msg mqtt.Message) {
	go program.Program().Send(subscription.ReceivedCmd)
}

func (m Model) onConnectHandler(client mqtt.Client) {
	go program.Program().Send(connectionStateChangeMsg{connectionState: connectionStateConnected})

}

func (m Model) onConnectionLostHandler(client mqtt.Client, err error) {
	go program.Program().Send(connectionStateChangeMsg{connectionState: connectionStateDisconnected})
}

func (m Model) onReconnectingHandler(client mqtt.Client, opts *mqtt.ClientOptions) {
	go program.Program().Send(connectionStateChangeMsg{connectionState: connectionStateReconnecting})
}

func (m Model) onConnectAttemptHandler(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
	go program.Program().Send(connectionStateChangeMsg{connectionState: connectionStateConnecting})
	return tlsCfg
}

func (m Model) Data() Data {
	return m.data
}

func (m Model) Init() tea.Cmd {
	m.client.Connect()
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.newSub != nil {
		var cmd tea.Cmd
		m.newSub, cmd = m.newSub.Update(msg)
		return m, cmd
	}

	m.subscriptions.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "a":
			m.newSub = NewSubModel()
			return m, m.newSub.Init()
		case "r":
			items := m.subscriptions.Items()
			if len(items) == 0 {
				break
			}
			sub := items[m.subscriptions.GlobalIndex()].(subscription.Model)
			m.client.Unsubscribe(sub.Data().Topic)
			m.subscriptions.RemoveItem(m.subscriptions.GlobalIndex())
			m.saveSubscriptions()
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
		case "esc":
			return nil, nil
		}
	case NewSubMsg:
		newSub := subscription.Model(msg)
		m.client.Subscribe(newSub.Data().Topic, newSub.Data().Qos, newSub.OnPubHandler)
		items := m.subscriptions.Items()
		items = append(items, newSub)
		m.subscriptions.SetItems(items)
		m.saveSubscriptions()
	case connectionStateChangeMsg:
		m.connectionState = msg.connectionState
		if msg.connectionState == connectionStateConnected {
			for _, model := range m.subscriptions.Items() {
				sub, ok := model.(subscription.Model)
				if ok {
					m.client.Subscribe(sub.Data().Topic, sub.Data().Qos, sub.OnPubHandler)
				}
			}
		}
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) View() string {

	if m.connectionState == connectionStateConnecting {
		return m.connectingView()
	} else {
		return m.defaultView()
	}
}

func (m Model) connectingView() string {
	return fmt.Sprintf("Connecting to broker on %s%s", m.brokerUrl, m.spinner.View())
}

func (m Model) defaultView() string {
	var borderStyle lipgloss.Style
	width, height, _ := term.GetSize(0)

	isBg := false
	if m.newSub != nil {
		isBg = true
	}

	if isBg {
		borderStyle = styles.BlurredBorderStyle
	} else {
		borderStyle = styles.FocusedBorderStyle
	}

	m.subscriptions.SetSize(styles.MenuWidth, height-10)
	subListWidget := viewport.New(styles.MenuWidth, height-10)
	subListWidget.SetContent(m.subscriptions.View())
	subsListView := borderStyle.Render(subListWidget.View())

	l := lipgloss.Place(styles.MenuWidth, 6, lipgloss.Left, lipgloss.Top, "")
	l = borderStyle.Render(l)

	broker := viewport.New(styles.MenuWidth, 1)
	broker.SetContent(m.brokerUrl)
	brokerView := borderStyle.Render(broker.View())
	clientId := viewport.New(styles.MenuWidth, 1)
	clientId.SetContent(m.data.ClientId)
	clientIdView := borderStyle.Render(clientId.View())
	leftView := lipgloss.JoinVertical(lipgloss.Top, brokerView, clientIdView, subsListView)

	recvTopic := viewport.New(width-(styles.MenuWidth+18), 1)
	messageNr := viewport.New(7, 1)
	recvAt := viewport.New(width-(styles.MenuWidth+9), 1)
	data := viewport.New(width-(styles.MenuWidth+9), height-12)
	subItems := m.subscriptions.Items()

	if len(subItems) > 0 {
		sub, ok := subItems[m.subscriptions.GlobalIndex()].(subscription.Model)
		if ok {
			messages := sub.Messages()
			messageNr.SetContent(fmt.Sprintf("%d/%d", min(m.messageIdx+1, len(messages)), len(messages)))
			if len(messages) > 0 {
				message := messages[m.messageIdx]
				recvTopic.SetContent(string(message.RecvTopic()))
				recvAt.SetContent(string(message.RecvAt().String()))
				data.SetContent(string(message.Data()))
			}

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
		if m.newSub != nil {
			s, _ = charmutils.OverlayCenter(s, m.newSub.View(), false)
		}
	}

	return styles.FocusedBorderStyle.Render(s)
}

func NewSubModel() newSubModel {
	m := newSubModel{}
	n := newSubInputs{
		Qos:    form.NewMultipleChoice(qosChoices),
		Format: form.NewMultipleChoice(subscription.FormatChoices),
	}
	m.form = form.New("New Subscription", &n)
	return m
}

func (m newSubModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m newSubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case form.SubmitMsg:
		return nil, m.newSubCmd
	case form.CancelMsg:
		return nil, nil
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m newSubModel) View() string {
	content := m.form.View()
	width, _, _ := term.GetSize(0)
	widget := viewport.New(width-4, 20)
	widget.SetContent(content)
	return styles.FocusedBorderStyle.Render(widget.View())
}

func (m newSubModel) newSubCmd() tea.Msg {
	inputs := m.form.Inputs().(*newSubInputs)
	sub := subscription.NewModel(subscription.Data{
		Name:   inputs.Name.Value(),
		Topic:  inputs.Topic.Value(),
		Qos:    byte(inputs.Qos.Index()),
		Format: inputs.Format.Selected(),
	})
	return NewSubMsg(sub)
}
