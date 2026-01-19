package ui

import (
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// InsertRuneAtCursor 在光标位置插入字符
func InsertRuneAtCursor(text string, cursor int, r rune) (string, int) {
	runes := []rune(text)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}

	// 在光标位置插入字符
	newRunes := make([]rune, len(runes)+1)
	copy(newRunes[:cursor], runes[:cursor])
	newRunes[cursor] = r
	copy(newRunes[cursor+1:], runes[cursor:])

	return string(newRunes), cursor + 1
}

// InsertStringAtCursor 在光标位置插入字符串
func InsertStringAtCursor(text string, cursor int, insert string) (string, int) {
	runes := []rune(text)
	insertRunes := []rune(insert)

	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}

	// 在光标位置插入字符串
	newRunes := make([]rune, len(runes)+len(insertRunes))
	copy(newRunes[:cursor], runes[:cursor])
	copy(newRunes[cursor:cursor+len(insertRunes)], insertRunes)
	copy(newRunes[cursor+len(insertRunes):], runes[cursor:])

	return string(newRunes), cursor + len(insertRunes)
}

// DeleteRuneBeforeCursor 删除光标前的字符（退格键）
func DeleteRuneBeforeCursor(text string, cursor int) (string, int) {
	runes := []rune(text)
	if cursor <= 0 || len(runes) == 0 {
		return text, cursor
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}

	// 删除光标前的字符
	newRunes := make([]rune, len(runes)-1)
	copy(newRunes[:cursor-1], runes[:cursor-1])
	copy(newRunes[cursor-1:], runes[cursor:])

	return string(newRunes), cursor - 1
}

// DeleteRuneAtCursor 删除光标处的字符（Delete键）
func DeleteRuneAtCursor(text string, cursor int) (string, int) {
	runes := []rune(text)
	if cursor < 0 || cursor >= len(runes) || len(runes) == 0 {
		return text, cursor
	}

	// 删除光标处的字符
	newRunes := make([]rune, len(runes)-1)
	copy(newRunes[:cursor], runes[:cursor])
	copy(newRunes[cursor:], runes[cursor+1:])

	return string(newRunes), cursor
}

// FormatTextWithCursor 格式化带光标的文本用于显示
func FormatTextWithCursor(text string, cursor int) string {
	runes := []rune(text)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}

	// 在光标位置插入光标符号
	if cursor == len(runes) {
		return text + "|"
	}

	before := string(runes[:cursor])
	after := string(runes[cursor:])
	return before + "|" + after
}

// HandleTextInput 通用输入处理函数：处理光标移动和文本编辑
// 返回 true 表示输入被处理
func HandleTextInput(msg tea.KeyMsg, text *string, cursor *int) bool {
	switch msg.String() {
	case "left", "ctrl+b":
		if *cursor > 0 {
			*cursor--
		}
		return true
	case "right", "ctrl+f":
		runes := []rune(*text)
		if *cursor < len(runes) {
			*cursor++
		}
		return true
	case "home", "ctrl+a":
		*cursor = 0
		return true
	case "end", "ctrl+e":
		*cursor = len([]rune(*text))
		return true
	case "backspace":
		*text, *cursor = DeleteRuneBeforeCursor(*text, *cursor)
		return true
	case "delete", "ctrl+d":
		*text, *cursor = DeleteRuneAtCursor(*text, *cursor)
		return true
	default:
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !IsControlKey(str) {
			*text, *cursor = InsertStringAtCursor(*text, *cursor, str)
			return true
		}
	}
	return false
}

// IsControlKey 检查是否为控制键
func IsControlKey(str string) bool {
	if len(str) == 0 {
		return true
	}

	// 检查常见的控制键序列
	controlKeys := []string{
		"ctrl+c", "ctrl+d", "ctrl+z", "ctrl+l", "ctrl+r",
		"alt+", "cmd+", "shift+", "ctrl+",
		"up", "down", "left", "right",
		"home", "end", "pgup", "pgdown",
		"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
		"insert", "delete", "tab", "enter", "backspace", "esc",
	}

	for _, key := range controlKeys {
		if strings.HasPrefix(strings.ToLower(str), key) {
			return true
		}
	}

	// 检查单个字符的控制字符（ASCII < 32，除了可打印字符）
	if len(str) == 1 {
		r := rune(str[0])
		if r < 32 && r != '\t' {
			return true
		}
	}

	return false
}

// GbkToUtf8 将GBK编码转换为UTF-8
func GbkToUtf8(data []byte) (string, error) {
	reader := transform.NewReader(strings.NewReader(string(data)), simplifiedchinese.GBK.NewDecoder())
	utf8Data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(utf8Data), nil
}
