package main

import (
	"fmt"
	"gd-agent/pkg/llms"
	"gd-agent/pkg/ui"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	provider := llms.ProviderList["DeepSeek"]
	m := ui.NewChatBox(provider)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error: ", err.Error())
	}
}
