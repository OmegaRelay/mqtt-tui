package form

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type keyMap struct {
	Select key.Binding
	Next   key.Binding
	Prev   key.Binding
	Cancel key.Binding
	Help   key.Binding
	Quit   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.Select, k.Cancel, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Prev}, // first column
		{k.Help, k.Quit}, // second column
	}
}

var keys = keyMap{
	Next: key.NewBinding(
		key.WithKeys("tab", "down", "j"),
		key.WithHelp("↓/tab/j", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab", "up", "k"),
		key.WithHelp("↑/shift+tab/h", "previous"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/cycle options"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
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

type Model struct {
	title  string
	inputs any
	cursor int
	keys   keyMap
	help   help.Model
}

type MultipleChoice struct {
	choices []string
	index   int
}

type SubmitMsg struct{}
type CancelMsg struct{}

func New(title string, inputs any) Model {
	mi := reflect.ValueOf(inputs).Elem()
	for i := range mi.NumField() {
		v := mi.Field(i)
		_, ok := v.Interface().(textinput.Model)
		if ok {
			v.Set(reflect.ValueOf(textinput.New()))
		}
	}

	return Model{
		title:  title,
		inputs: inputs,
		help:   help.New(),
		keys:   keys,
	}
}

func NewMultipleChoice(choices []string) MultipleChoice {
	return MultipleChoice{choices: choices}
}

func submit() tea.Msg {
	return SubmitMsg{}
}

func cancel() tea.Msg {
	return CancelMsg{}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	mi := reflect.ValueOf(m.inputs).Elem()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Select):
			nrInputs := mi.NumField()
			if m.cursor < nrInputs {
				for i := range mi.NumField() {
					v := mi.Field(i)
					switch v := v.Interface().(type) {
					case textinput.Model:
						if m.cursor == i {
							if v.Focused() {
								v.Blur()
								mi.Field(m.cursor).Set(reflect.ValueOf(v))
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
					case MultipleChoice:
						if m.cursor == i {
							v.index++
							v.index %= len(v.choices)
							mi.Field(i).Set(reflect.ValueOf(v))
						}
					}
				}
			} else if m.cursor == nrInputs { // cancel
				return m, cancel
			} else { // submit
				return m, submit
			}

		case key.Matches(msg, m.keys.Next):
			if m.cursor < mi.NumField() {
				f, ok := mi.Field(m.cursor).Interface().(textinput.Model)
				if ok && f.Focused() {
					break
				}
			}

			m.cursor++
			nrInputs := (mi.NumField() + 2)
			if m.cursor >= nrInputs {
				m.cursor = nrInputs - 1
			}

		case key.Matches(msg, m.keys.Prev):
			if m.cursor < mi.NumField() {
				f, ok := mi.Field(m.cursor).Interface().(textinput.Model)
				if ok && f.Focused() {
					break
				}
			}

			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}

		case key.Matches(msg, m.keys.Cancel):
			if m.cursor < mi.NumField() {
				f, ok := mi.Field(m.cursor).Interface().(textinput.Model)
				if ok && f.Focused() {
					f.SetValue("")
					f.Blur()
					mi.Field(m.cursor).Set(reflect.ValueOf(f))
					break
				}
			}
			return m, cancel

		}
	}
	cmds := make([]tea.Cmd, 0)
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

func (m Model) View() string {
	var content strings.Builder

	content.WriteString(m.title)
	content.WriteString("\n\n")
	mi := reflect.ValueOf(m.inputs).Elem()
	if mi.Kind() != reflect.Struct {
		panic("a forms inputs must be in a struct")
	}
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
			case MultipleChoice:
				var b strings.Builder
				b.WriteString(" >-\n")
				for i, c := range v.choices {
					if v.index == i {
						b.WriteString("     [x] ")
					} else {
						b.WriteString("     [ ] ")
					}
					b.WriteString(c)
					b.WriteString("\n")
				}
				input = b.String()
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

	content.WriteString("\n\n")
	content.WriteString(m.help.View(m.keys))
	return content.String()
}

func (m Model) Inputs() any { return m.inputs }

func (m MultipleChoice) Index() int {
	return m.index
}

func (m MultipleChoice) Selected() string {
	return m.choices[m.index]
}
