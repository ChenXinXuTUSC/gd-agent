package llms


type State struct {
	Messages []*Message
	Stream   bool
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
