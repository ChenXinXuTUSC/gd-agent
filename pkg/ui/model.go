package ui

import (
	"fmt"
	// "os"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"gd-agent/pkg/llms"
	"gd-agent/pkg/provider"
)

const (
	viewportHeight = 20
	viewportWidth  = 20
	inputBoxHeight = 3
	inputBoxWidth  = 10
)

type ChatBox struct {
	state    llms.State        // 核心属性状态
	provider provider.Provider // 大模型服务提供商

	// 以下为 CLI 界面相关状态与数据
	viewport viewport.Model
	inputbox textarea.Model

	messages  []ChatMsg // 对话及渲染结果
	streaming []rune    // 正在流式接收模型输出的字符缓冲

	isStreaming   bool
	currentRuneCh <-chan rune // 流式传输的只读接受通道
}

type ChatMsg struct {
	raw      *llms.Message
	rendered string
}

func NewChatBox(provider provider.Provider) *ChatBox {
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

	// r, err := glamour.NewTermRenderer(glamour.WithStylePath("dark"))
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "create chatbox error: %v", err)
	// 	return nil
	// }

	return &ChatBox{
		viewport: vp,
		inputbox: ta,
		state:    llms.State{Stream: true},
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
		m.inputbox.SetWidth(msg.Width - 2)
		viewBorder = viewBorder.Width(msg.Width)
		inputBorder = inputBorder.Width(msg.Width)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
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
			m.refreshViewport()

		case "shift+enter":
			// 输入换行
			if !m.isStreaming {
				current := m.inputbox.Value()
				m.inputbox.SetValue(current + "\n")
			}

		default:
			if !m.isStreaming {
				var inputCmd tea.Cmd
				m.inputbox, inputCmd = m.inputbox.Update(msg)
				cmds = append(cmds, inputCmd)
			}
		}

	case streamRuneMsg:
		m.streaming = append(m.streaming, rune(msg))
		if m.isStreaming && len(m.streaming) > 0 {
			lastMsg := &llms.Message{
				Role:    "assistant",
				Content: string(m.streaming),
			}
			m.messages[len(m.messages)-1].rendered = m.renderBubble(lastMsg, m.viewport.Width(), false)
		}
		m.refreshViewport()
		cmds = append(cmds, m.waitNextRune())

	case streamDoneMsg:
		content := strings.TrimSpace(string(m.streaming))
		lastMsg := &llms.Message{
			Role:    "assistant",
			Content: content,
		}
		m.state.Messages = append(m.state.Messages, lastMsg)
		// 将流式输出消息片段固化到消息列表
		// 流式传输完成的时候换成 markdown 渲染
		m.messages[len(m.messages)-1] = ChatMsg{
			raw:      lastMsg,
			rendered: m.renderBubble(lastMsg, m.viewport.Width(), true),
		}

		m.streaming = nil     // 清空流式输出接收区
		m.currentRuneCh = nil // 空引用回收管道
		m.isStreaming = false
		m.refreshViewport()

	case streamErrMsg:
		m.isStreaming = false
		errMsg := &llms.Message{
			Role:    "system",
			Content: fmt.Sprintf("error: %v", msg.err.Error()),
		}
		m.messages = append(m.messages, ChatMsg{
			raw:      errMsg,
			rendered: m.renderBubble(errMsg, m.viewport.Width(), false),
		})
		m.refreshViewport()
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)
	return m, tea.Batch(cmds...)
}

// --- view ---
func (m *ChatBox) View() tea.View {
	vp := viewBorder.Render(m.viewport.View())
	in := inputBorder.Render(m.inputbox.View())
	hint := "ctrl+c to quit"
	if m.isStreaming {
		hint = "⏳ generating response..."
	}
	body := strings.Join([]string{vp, in, hint}, "\n")

	v := tea.NewView(body)
	v.AltScreen = true                    // 在新的屏幕缓冲区中渲染（清屏）
	v.MouseMode = tea.MouseModeCellMotion // 捕获屏幕的鼠标事件
	return v
}

func (m *ChatBox) sendMessage(text string) tea.Cmd {
	m.isStreaming = true
	text = strings.TrimSpace(text)
	msg := &llms.Message{
		Role:    "user",
		Content: text,
	}
	m.state.Messages = append(m.state.Messages, msg)
	// 展示区消息追加
	m.messages = append(m.messages, ChatMsg{
		raw:      msg,
		rendered: m.renderBubble(msg, m.viewport.Width(), true),
	})

	// 展示区的正在接受流式传输响应的消息占位符
	m.messages = append(m.messages, ChatMsg{
		raw: &llms.Message{
			Role:    "assistant",
			Content: "",
		},
	})

	// 返回 CMD 调用 LLM ，拿到流失传输 channel，触发第一次读取
	return func() tea.Msg {
		ch, err := m.provider.GetResponse(&m.state)
		if err != nil {
			return streamErrMsg{err}
		}
		m.currentRuneCh = ch
		return m.waitNextRune()()
	}
}

func (m *ChatBox) waitNextRune() tea.Cmd {
	return func() tea.Msg {
		r, ok := <-m.currentRuneCh
		if !ok {
			return streamDoneMsg{}
		}
		return streamRuneMsg(r)
	}
}

func (m *ChatBox) refreshViewport() {
	var lines []string

	for _, msg := range m.messages {
		lines = append(lines, msg.rendered)
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoBottom()
}

// 抽出自定义渲染器构造函数
func newCustomRenderer(wordWrap int) (*glamour.TermRenderer, error) {
	style := styles.DarkStyleConfig

	// 取消默认终端 markdown 渲染的前后左右间距，这里我们是消息气泡渲染
	zero := uint(0)
	empty := ""
	style.Document.Margin = &zero
	style.Document.BlockPrefix = empty
	style.Document.BlockSuffix = empty

	// 其他 markdown 样式支持自定义
	// 水平线：用 ─ (U+2500 Box Drawing Light Horizontal) 填满整行
    hrChar := "─"
    hrColor := "240" // 灰色
    style.HorizontalRule.Format = hrChar
    style.HorizontalRule.Color = &hrColor
	return glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(wordWrap),
	)
}

// 去掉每行尾部填充空格，返回清理后的字符串和最宽行的宽度
//
//	func trimAndMeasure(s string) (string, int) {
//		lines := strings.Split(s, "\n")
//		maxW := 0
//		for i, line := range lines {
//			w0 := lipgloss.Width(lines[i])
//			lines[i] = strings.TrimRight(line, " ")
//			w1 := lipgloss.Width(lines[i])
//			lines[i] = strings.Map(func(r rune) rune {
//				if unicode.IsSpace(r) {
//					return '.' // 如果是空白符，替换
//				}
//				return r // 否则保持原样
//			}, lines[i])
//			if w := lipgloss.Width(lines[i]); w > maxW {
//				maxW = w
//			}
//			lines[i] += fmt.Sprintf(" [width: %d/%d]", w0, w1)
//		}
//		return strings.Join(lines, "\n"), maxW
//	}
func trimAndMeasure(s string) (string, int) {
	lines := strings.Split(s, "\n")
	maxW := 0
	for i, line := range lines {
		// 先 strip ANSI 序列，再去尾部空格，测量真实可见宽度
		stripped := ansi.Strip(line)
		trimmed := strings.TrimRight(stripped, " ")
		visibleW := lipgloss.Width(trimmed)

		// 用 ansi.Truncate 按可见宽度截断原始带颜色的行
		// 这样保留了 ANSI 颜色，但去掉了尾部填充空格
		lines[i] = ansi.Truncate(line, visibleW, "")

		if visibleW > maxW {
			maxW = visibleW
		}
	}
	return strings.Join(lines, "\n"), maxW
}

func (m *ChatBox) renderBubble(msg *llms.Message, containerWidth int, useMdRender bool) string {
	bubbleMaxW := containerWidth
	contentMaxW := bubbleMaxW - 4
	content := msg.Content
	actualWidth := 0
	if useMdRender {
		r, err := newCustomRenderer(contentMaxW)
		if err == nil {
			if mdContent, mdErr := r.Render(content); mdErr == nil {
				content = strings.Trim(mdContent, "\n")
				content, actualWidth = trimAndMeasure(content)
			}
		}
		// glamour 渲染失败时 fallback
		// if actualWidth == 0 {
		// 	actualWidth = lipgloss.Width(content)
		// }
	} else {
		actualWidth = lipgloss.Width(msg.Content)
	}

	// 不要再用 wrapStyle 二次换行 glamour 的输出
	// glamour 已经处理了换行，直接用就行
	switch msg.Role {
	case "user":
		label := userLabelStyle.Render("🧑 User")
		// 对非 md 内容才需要手动限宽换行
		if !useMdRender {
			content = lipgloss.NewStyle().Width(min(contentMaxW, actualWidth)).Render(content)
		}
		bubble := userBubbleStyle.Width(min(bubbleMaxW, actualWidth+4)).Render(content)
		block := lipgloss.JoinVertical(lipgloss.Right, label, bubble)
		return lipgloss.NewStyle().Width(containerWidth).Align(lipgloss.Right).Render(block)

	case "assistant":
		label := assistantLabelStyle.Render("🤖 Assistant")
		if !useMdRender {
			content = lipgloss.NewStyle().Width(min(contentMaxW, actualWidth)).Render(content)
		}
		bubble := assistantBubbleStyle.Width(min(bubbleMaxW, actualWidth+4)).Render(content)
		block := lipgloss.JoinVertical(lipgloss.Left, label, bubble)
		return lipgloss.NewStyle().Width(containerWidth).Align(lipgloss.Left).Render(block)

	case "system":
		return systemBubbleStyle.Width(containerWidth).Render(content)
	}
	return ""
}
