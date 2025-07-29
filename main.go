package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(NewCnxModel("localhost", 1883, "client_1", nil),
		tea.WithAltScreen(), tea.WithReportFocus())
	_, err := p.Run()
	if err != nil {
		panic(err)
	}
}

