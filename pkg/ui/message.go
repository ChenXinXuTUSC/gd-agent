package ui

// 流式收到一个字符
type streamRuneMsg rune

// 流结束
type streamDoneMsg struct{}

// 流出错
type streamErrMsg struct{ err error }
