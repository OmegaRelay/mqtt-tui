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
	return gProgram
}

func SendErrorMsg(err error) {
	gProgram.Send(ErrorMsg{Err: err})
}
