package ui

import (
	"fmt"
	"gd-agent/pkg/llms"
	llm_types "gd-agent/pkg/llms/types"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

const (
	viewportHeight = 5
	viewportWidth  = 20
	inputBoxHeight = 3
	inputBoxWidth  = 10
)

type ChatBox struct {
	viewport viewport.Model
	inputbox textarea.Model

	state llm_types.State

	transcript []string // 渲染好的对话
	streaming  []rune   // 正在流式接收模型输出的字符缓冲

	isStreaming bool

	provider      llms.Provider
	currentRuneCh <-chan rune // 流式传输的只读接受通道
}

func NewChatBox(provider llms.Provider) *ChatBox {
	// 用户消息输入框
	ta := textarea.New()
	ta.Placeholder = "press enter to send any message, Shift+Enter to put a new line"
	ta.SetWidth(inputBoxWidth)
	ta.SetHeight(inputBoxHeight)
	ta.DynamicHeight = false
	ta.Focus()

	// 历史消息展示框
	vp := viewport.New()
	vp.SetWidth(viewportWidth)
	vp.SetHeight(viewportHeight)
	vp.SetContent("welcome to use chatbox")

	return &ChatBox{
		viewport: vp,
		inputbox: ta,
		state:    llm_types.State{Stream: true},
		provider: provider,
	}
}

// --- Init ---
func (m *ChatBox) Init() tea.Cmd {
	return textarea.Blink
}

// --- update ---
func (m *ChatBox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds = []tea.Cmd{}

	// 类型选择时注意重新赋值接口转换过后得类型
	// 不然 switch 里面用的 msg 还是原始 msg
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width - 4)
		m.inputbox.SetWidth(msg.Width - 4)
		viewportStyle.Width(msg.Width - 2)
		inputboxStyle.Width(msg.Width - 2)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			if m.isStreaming {
				break // 模型正在输出，什么也不做
			}
			userText := strings.TrimSpace(m.inputbox.Value())
			if userText == "" {
				break // 用户什么也没输入，什么也不做
			}
			m.inputbox.Reset()
			cmds = append(cmds, m.sendMessage(userText))

		default:
			if !m.isStreaming {
				var inputCmd tea.Cmd
				m.inputbox, inputCmd = m.inputbox.Update(msg)
				cmds = append(cmds, inputCmd)
			}
		}

	case streamRuneMsg:
		m.streaming = append(m.streaming, rune(msg))
		m.refreshViewport()
		cmds = append(cmds, m.waitNextRune(m.currentRuneCh))

	case streamDoneMsg:
		content := string(m.streaming)
		m.state.Messages = append(m.state.Messages, llm_types.Message{
			Role:    "assistant",
			Content: content,
		})

		// 将流式输出消息片段固化到 transcript
		m.streaming = nil     // 清空流式输出接收区
		m.currentRuneCh = nil // 空引用回收管道
		m.isStreaming = false
		m.refreshViewport()

	case streamErrMsg:
		m.isStreaming = false
		m.transcript = append(m.transcript, fmt.Sprintf("error: %v", msg.err.Error()))
		m.refreshViewport()
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)
	return m, tea.Batch(cmds...)
}

// --- view ---
func (m *ChatBox) View() tea.View {
	vp := viewportStyle.Render(m.viewport.View())
	in := inputboxStyle.Render(m.inputbox.View())
	hint := "ctrl+c to quit"
	if m.isStreaming {
		hint = "⏳ generating response..."
	}
	body := strings.Join([]string{vp, in, hint}, "\n")

	v := tea.NewView(body)
    v.AltScreen = true // 在新的屏幕缓冲区中渲染（清屏）
    v.MouseMode = tea.MouseModeCellMotion // 捕获屏幕的鼠标事件
    return v
}

func (m *ChatBox) sendMessage(text string) tea.Cmd {
	m.isStreaming = true
	m.state.Messages = append(m.state.Messages, llm_types.Message{
		Role:    "user",
		Content: text,
	})

	// 展示区消息追加
	m.transcript = append(m.transcript,
		userLabelStyle.Render("🧑 User"),
		text,
		"",
	)
	m.refreshViewport()

	// 返回 CMD 调用 LLM ，拿到流失传输 channel，触发第一次读取
	return func() tea.Msg {
		ch, err := m.provider.GetResponse(&m.state)
		if err != nil {
			return streamErrMsg{err}
		}

		m.currentRuneCh = ch

		m.transcript = append(m.transcript, assistantLabelStyle.Render("🤖 Assistant"))
		return m.waitNextRune(ch)()
	}
}

func (m *ChatBox) waitNextRune(ch <-chan rune) tea.Cmd {
	return func() tea.Msg {
		r, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		return streamRuneMsg(r)
	}
}

func (m *ChatBox) refreshViewport() {
	lines := append([]string{}, m.transcript...)
	if len(m.streaming) > 0 {
		lines = append(lines, string(m.streaming))
	}
	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}
