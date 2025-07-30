package main

import (
	"github.com/OmegaRelay/mqtt-tui/connection"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(connection.NewModel("localhost", 1883, "client_1", nil),
		tea.WithAltScreen(), tea.WithReportFocus())
	_, err := p.Run()
	if err != nil {
		panic(err)
	}
}
