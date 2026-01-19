package ui

// ScrollState 滚动状态
type ScrollState struct {
	Cursor    int // 当前选中行
	ScrollPos int // 滚动位置
	MaxLines  int // 每页最大显示行数
	TotalLen  int // 列表总长度
}

// NewScrollState 创建滚动状态
func NewScrollState(maxLines, totalLen int) *ScrollState {
	return &ScrollState{
		Cursor:    0,
		ScrollPos: 0,
		MaxLines:  maxLines,
		TotalLen:  totalLen,
	}
}

// ScrollUp 向上滚动
func (s *ScrollState) ScrollUp() {
	if s.Cursor > 0 {
		s.Cursor--
	}
	s.adjustScroll()
}

// ScrollDown 向下滚动
func (s *ScrollState) ScrollDown() {
	if s.Cursor < s.TotalLen-1 {
		s.Cursor++
	}
	s.adjustScroll()
}

// SetCursor 设置光标位置
func (s *ScrollState) SetCursor(cursor int) {
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= s.TotalLen {
		cursor = s.TotalLen - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	s.Cursor = cursor
	s.adjustScroll()
}

// SetTotalLen 更新列表总长度
func (s *ScrollState) SetTotalLen(totalLen int) {
	s.TotalLen = totalLen
	// 确保光标不超出范围
	if s.Cursor >= s.TotalLen {
		s.Cursor = s.TotalLen - 1
	}
	if s.Cursor < 0 {
		s.Cursor = 0
	}
	s.adjustScroll()
}

// adjustScroll 调整滚动位置以确保光标可见
func (s *ScrollState) adjustScroll() {
	if s.TotalLen == 0 {
		s.ScrollPos = 0
		return
	}

	// 计算可见范围
	endIndex := s.TotalLen - s.ScrollPos
	startIndex := endIndex - s.MaxLines
	if startIndex < 0 {
		startIndex = 0
	}

	// 如果光标超出可见范围的上边界，调整滚动位置
	if s.Cursor < startIndex {
		s.ScrollPos = s.TotalLen - s.Cursor - s.MaxLines
		if s.ScrollPos < 0 {
			s.ScrollPos = 0
		}
	}

	// 如果光标超出可见范围的下边界，调整滚动位置
	if s.Cursor >= endIndex {
		s.ScrollPos = s.TotalLen - s.Cursor - 1
		if s.ScrollPos < 0 {
			s.ScrollPos = 0
		}
	}
}

// GetVisibleRange 获取可见范围 [startIndex, endIndex)
func (s *ScrollState) GetVisibleRange() (int, int) {
	if s.TotalLen == 0 {
		return 0, 0
	}

	endIndex := s.TotalLen - s.ScrollPos
	startIndex := endIndex - s.MaxLines
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > s.TotalLen {
		endIndex = s.TotalLen
	}

	return startIndex, endIndex
}

// IsCursorVisible 检查光标是否在可见范围内
func (s *ScrollState) IsCursorVisible() bool {
	startIndex, endIndex := s.GetVisibleRange()
	return s.Cursor >= startIndex && s.Cursor < endIndex
}

// PageUp 向上翻页
func (s *ScrollState) PageUp() {
	newCursor := s.Cursor - s.MaxLines
	if newCursor < 0 {
		newCursor = 0
	}
	s.Cursor = newCursor
	s.adjustScroll()
}

// PageDown 向下翻页
func (s *ScrollState) PageDown() {
	newCursor := s.Cursor + s.MaxLines
	if newCursor >= s.TotalLen {
		newCursor = s.TotalLen - 1
	}
	if newCursor < 0 {
		newCursor = 0
	}
	s.Cursor = newCursor
	s.adjustScroll()
}

// GoToTop 跳转到顶部
func (s *ScrollState) GoToTop() {
	s.Cursor = 0
	s.ScrollPos = 0
}

// GoToBottom 跳转到底部
func (s *ScrollState) GoToBottom() {
	if s.TotalLen > 0 {
		s.Cursor = s.TotalLen - 1
	} else {
		s.Cursor = 0
	}
	s.adjustScroll()
}
