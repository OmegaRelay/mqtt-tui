package form

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	title  string
	inputs any
	cursor int

	isTextInsert bool

	keysInsert keyMapInsert
	keysNormal keyMapNormal
	help       help.Model
}

type MultipleChoice struct {
	choices []string
	index   int
}

type SubmitMsg struct{}
type CancelMsg struct{}

func New(title string, inputs any) Model {
	if inputs != nil {
		mi := reflect.ValueOf(inputs).Elem()
		for i := range mi.NumField() {
			v := mi.Field(i)
			switch v.Interface().(type) {
			case textinput.Model:
				v.Set(reflect.ValueOf(textinput.New()))
			case textarea.Model:
				v.Set(reflect.ValueOf(textarea.New()))
			}
		}
	}

	return Model{
		title:      title,
		inputs:     inputs,
		help:       help.New(),
		keysNormal: keysNormal,
		keysInsert: keysInsert,
	}
}

func (m *Model) SetInputs(inputs any) {
	m.inputs = inputs
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

	cmds := make([]tea.Cmd, 0)
	for i := range mi.NumField() {
		switch v := mi.Field(i).Interface().(type) {
		case textinput.Model:
			var cmd tea.Cmd
			v, cmd = v.Update(msg)
			cmds = append(cmds, cmd)
			mi.Field(i).Set(reflect.ValueOf(v))
		case textarea.Model:
			var cmd tea.Cmd
			v, cmd = v.Update(msg)
			cmds = append(cmds, cmd)
			mi.Field(i).Set(reflect.ValueOf(v))
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isTextInsert {
			switch {
			case key.Matches(msg, m.keysInsert.Exit):
				v := mi.Field(m.cursor)
				switch v := v.Interface().(type) {
				case textinput.Model:
					v.Blur()
					mi.Field(m.cursor).Set(reflect.ValueOf(v))
				case textarea.Model:
					v.Blur()
					mi.Field(m.cursor).Set(reflect.ValueOf(v))
				}
				m.isTextInsert = false
			}
		} else {
			switch {
			case key.Matches(msg, m.keysNormal.Insert):
				nrInputs := mi.NumField()
				if m.cursor < nrInputs {
					for i := range mi.NumField() {
						v := mi.Field(i)
						switch v := v.Interface().(type) {
						case textinput.Model:
							if m.cursor == i {
								v.Focus()
								m.isTextInsert = true
							}
							mi.Field(i).Set(reflect.ValueOf(v))
						case bool:
							if m.cursor == i {
								tmp := !v
								mi.Field(i).SetBool(tmp)
							}
						case MultipleChoice:
							if m.cursor == i {
								v.index++
								v.index %= len(v.choices)
								mi.Field(i).Set(reflect.ValueOf(v))
							}

						case textarea.Model:
							if m.cursor == i {
								v.Focus()
								m.isTextInsert = true
							}
							mi.Field(i).Set(reflect.ValueOf(v))
						}
					}
				} else if m.cursor == nrInputs { // cancel
					return m, cancel
				} else { // submit
					return m, submit
				}

			case key.Matches(msg, m.keysNormal.Next):
				m.cursor++
				nrInputs := (mi.NumField() + 2)
				if m.cursor >= nrInputs {
					m.cursor = nrInputs - 1
				}

			case key.Matches(msg, m.keysNormal.Prev):
				m.cursor--
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
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
			case textarea.Model:
				input = ">-\n"
				input += v.View()
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
	if m.isTextInsert {
		content.WriteString(m.help.View(m.keysInsert))
	} else {
		content.WriteString(m.help.View(m.keysNormal))
	}
	return content.String()
}

func (m Model) Inputs() any { return m.inputs }

func (m MultipleChoice) Index() int {
	return m.index
}

func (m MultipleChoice) Selected() string {
	return m.choices[m.index]
}
