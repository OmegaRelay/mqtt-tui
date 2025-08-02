package program

import tea "github.com/charmbracelet/bubbletea"

var gProgram *tea.Program

func SetProgram(program *tea.Program) {
	gProgram = program
}

func Program() *tea.Program {
	return gProgram
}
