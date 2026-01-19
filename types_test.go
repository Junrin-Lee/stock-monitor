package main

import (
	"testing"
)

// TestMoveCursorUp 测试光标上移函数
func TestMoveCursorUp(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		expected int
	}{
		{"从中间位置上移", 5, 4},
		{"从第一个位置上移(边界)", 0, 0},
		{"从第二个位置上移", 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MoveCursorUp(tt.current)
			if result != tt.expected {
				t.Errorf("MoveCursorUp(%d) = %d, expected %d", tt.current, result, tt.expected)
			}
		})
	}
}

// TestMoveCursorDown 测试光标下移函数
func TestMoveCursorDown(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		maxIndex int
		expected int
	}{
		{"从中间位置下移", 5, 10, 6},
		{"到达最大位置(边界)", 10, 10, 10},
		{"从倒数第二个位置下移", 9, 10, 10},
		{"空列表边界", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MoveCursorDown(tt.current, tt.maxIndex)
			if result != tt.expected {
				t.Errorf("MoveCursorDown(%d, %d) = %d, expected %d",
					tt.current, tt.maxIndex, result, tt.expected)
			}
		})
	}
}

// TestGetAlertTypeFromCursor 测试根据光标获取警报类型
func TestGetAlertTypeFromCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		expected AlertType
	}{
		{"第一个选项 - 价格警报", 0, AlertTypePrice},
		{"第二个选项 - 涨跌幅警报", 1, AlertTypeRate},
		{"第三个选项 - 成交量警报", 2, AlertTypeVolume},
		{"越界值 - 返回默认值", 10, AlertTypePrice},
		{"负值 - 返回默认值", -1, AlertTypePrice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertTypeFromCursor(tt.cursor)
			if result != tt.expected {
				t.Errorf("GetAlertTypeFromCursor(%d) = %v, expected %v",
					tt.cursor, result, tt.expected)
			}
		})
	}
}

// TestGetAlertConditionFromCursor 测试根据光标获取条件符号
func TestGetAlertConditionFromCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		expected string
	}{
		{"第一个选项 - 大于", 0, ">"},
		{"第二个选项 - 小于", 1, "<"},
		{"第三个选项 - 大于等于", 2, ">="},
		{"第四个选项 - 小于等于", 3, "<="},
		{"越界值 - 返回默认值", 10, ">"},
		{"负值 - 返回默认值", -1, ">"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertConditionFromCursor(tt.cursor)
			if result != tt.expected {
				t.Errorf("GetAlertConditionFromCursor(%d) = %s, expected %s",
					tt.cursor, result, tt.expected)
			}
		})
	}
}

// TestCheckNumericCondition 测试数值条件检查
func TestCheckNumericCondition(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		threshold float64
		condition string
		expected  bool
	}{
		// 大于条件
		{"100 > 50 应为true", 100, 50, ">", true},
		{"50 > 100 应为false", 50, 100, ">", false},
		{"100 > 100 应为false", 100, 100, ">", false},

		// 小于条件
		{"50 < 100 应为true", 50, 100, "<", true},
		{"100 < 50 应为false", 100, 50, "<", false},
		{"100 < 100 应为false", 100, 100, "<", false},

		// 大于等于条件
		{"100 >= 50 应为true", 100, 50, ">=", true},
		{"100 >= 100 应为true", 100, 100, ">=", true},
		{"50 >= 100 应为false", 50, 100, ">=", false},

		// 小于等于条件
		{"50 <= 100 应为true", 50, 100, "<=", true},
		{"100 <= 100 应为true", 100, 100, "<=", true},
		{"100 <= 50 应为false", 100, 50, "<=", false},

		// 无效条件
		{"无效条件符号应返回false", 100, 50, "==", false},
		{"空条件符号应返回false", 100, 50, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckNumericCondition(tt.value, tt.threshold, tt.condition)
			if result != tt.expected {
				t.Errorf("CheckNumericCondition(%.2f, %.2f, %s) = %v, expected %v",
					tt.value, tt.threshold, tt.condition, result, tt.expected)
			}
		})
	}
}
