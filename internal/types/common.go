package types

// ============================================================================
// 辅助函数 - Common Helper Functions
// ============================================================================

// MoveCursorUp 处理光标上移
// 参数:
//   - current: 当前光标位置
//
// 返回: 新的光标位置
func MoveCursorUp(current int) int {
	if current > 0 {
		return current - 1
	}
	return current
}

// MoveCursorDown 处理光标下移
// 参数:
//   - current: 当前光标位置
//   - maxIndex: 最大索引(列表长度-1)
//
// 返回: 新的光标位置
func MoveCursorDown(current, maxIndex int) int {
	if current < maxIndex {
		return current + 1
	}
	return current
}
