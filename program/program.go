package program

import tea "github.com/charmbracelet/bubbletea"

var gProgram *tea.Program

type ErrorMsg struct {
	Err error
}

func SetProgram(program *tea.Program) {
	gProgram = program
}

func Program() *tea.Program {
	if gProgram == nil {
		panic("get program has been called before setting program")
	}
	return gProgram
}

func SendErrorMsg(err error) {
	if gProgram == nil {
		panic("program is used before being set")
	}
	gProgram.Send(ErrorMsg{Err: err})
}
