package llms


import (
	llm_types "gd-agent/pkg/llms/types"
)

// 服务提供商接口类型
type Provider interface {
	GetResponse(state *llm_types.State) (<-chan rune, error)
}


// 供外部注册使用
var ProviderList map[string]Provider
var AvailableProviderNames []string
