package ui

import "charm.land/lipgloss/v2"

var (
	ChatWindow = lipgloss.NewStyle().
			Width(100).
			Height(50)

	InputBox = lipgloss.NewStyle().
			Width(100).
			Height(5)

	UserLabel = lipgloss.NewStyle().
			Background(lipgloss.Color("2")).
			Foreground(lipgloss.Color("10")).
			Padding(0, 1)

	AssistantLabel = lipgloss.NewStyle().
			Background(lipgloss.Color("4")).
			Foreground(lipgloss.Color("12")).
			Padding(0, 1)

	MessageBubble = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder())
)
