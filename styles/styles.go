package styles

import "github.com/charmbracelet/lipgloss"

const MenuWidth = 40

var (
	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("38"))

	BlurredBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))
)
