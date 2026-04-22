package types

type State struct {
	Messages []Message
	Stream   bool
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ProviderInfo struct {
	BaseUrl      string   `yaml:"base_url"`
	ApiKey       string   `yaml:"api_key"`
	ModelList    []string `yaml:"model_list"`
	DefaultModel string   `yaml:"default_model"`
}
