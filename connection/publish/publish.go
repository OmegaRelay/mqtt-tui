package publish

import (
	"fmt"

	"github.com/OmegaRelay/mqtt-tui/connection/subscription"
	"github.com/OmegaRelay/mqtt-tui/form"
	"github.com/OmegaRelay/mqtt-tui/program"
	"github.com/OmegaRelay/mqtt-tui/styles"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type inputs struct {
	Topic   textinput.Model
	QoS     form.MultipleChoice
	Retain  bool
	Message textarea.Model
}

type Model struct {
	client mqtt.Client

	form form.Model
}

func New(cl mqtt.Client) Model {
	m := Model{
		client: cl,

		form: form.New("Publish Message", &inputs{
			QoS: form.NewMultipleChoice(subscription.QosChoices()),
		}),
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)

	switch msg.(type) {
	case form.SubmitMsg:
		i := m.form.Inputs().(*inputs)
		m.client.Publish(i.Topic.Value(), byte(i.QoS.Index()), i.Retain, i.Message.Value())
		return nil, nil
	case form.CancelMsg:
		return nil, nil
	}

	return m, cmd
}

func (m Model) View() string {
	width, height, err := term.GetSize(0)
	if err != nil {
		program.SendErrorMsg(fmt.Errorf("failed to get terminal size: %w", err))
		return ""
	}

	vp := viewport.New(width-2, height-2)
	vp.SetContent(m.form.View())

	return styles.FocusedBorderStyle.Render(vp.View())

}
