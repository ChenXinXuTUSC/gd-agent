package main

import (
	"fmt"
	"gd-agent/pkg/ui"
	"os"

	tea "charm.land/bubbletea/v2"

	"gd-agent/pkg/provider"
	_ "gd-agent/pkg/providerimp" // 用于自动注册已实现接口的提供商服务
)

func init() {
	provider.InitProviderList() // 此时所有 Register() 已执行完毕
}

func main() {
	providerName := "deepseek"
	provider, ok := provider.ProviderList[providerName]
	if !ok {
		panic(fmt.Sprintf("provider [%s] not implemented", providerName))
	}

	m := ui.NewChatBox(provider)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error: ", err.Error())
	}
}
