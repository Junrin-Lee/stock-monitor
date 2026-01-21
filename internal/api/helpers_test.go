package api

import (
	"testing"

	"stock-monitor/internal/types"

	"github.com/stretchr/testify/assert"
)

// TestIsChinaStock 测试中国A股识别功能
func TestIsChinaStock(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		expected bool
	}{
		// 上海股票测试
		{
			name:     "上海股票-完整格式-大写",
			symbol:   "SH601138",
			expected: true,
		},
		{
			name:     "上海股票-完整格式-小写",
			symbol:   "sh601138",
			expected: true,
		},
		{
			name:     "上海股票-简写格式",
			symbol:   "601138",
			expected: true,
		},
		{
			name:     "上海股票-带空格",
			symbol:   " SH601138 ",
			expected: true,
		},
		// 深圳股票测试
		{
			name:     "深圳股票-完整格式-大写",
			symbol:   "SZ000001",
			expected: true,
		},
		{
			name:     "深圳股票-完整格式-小写",
			symbol:   "sz000001",
			expected: true,
		},
		{
			name:     "深圳股票-0开头简写",
			symbol:   "000001",
			expected: true,
		},
		{
			name:     "深圳股票-3开头简写",
			symbol:   "300001",
			expected: true,
		},
		// 港股测试
		{
			name:     "港股-HK前缀",
			symbol:   "HK00700",
			expected: false,
		},
		{
			name:     "港股-.HK后缀",
			symbol:   "0700.HK",
			expected: false,
		},
		// 美股测试
		{
			name:     "美股-字母代码",
			symbol:   "AAPL",
			expected: false,
		},
		{
			name:     "美股-长代码",
			symbol:   "GOOGL",
			expected: false,
		},
		// 边界情况
		{
			name:     "空字符串",
			symbol:   "",
			expected: false,
		},
		{
			name:     "纯空格",
			symbol:   "   ",
			expected: false,
		},
		{
			name:     "无效代码-数字不足6位",
			symbol:   "12345",
			expected: false,
		},
		{
			name:     "无效代码-数字超过6位",
			symbol:   "1234567",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsChinaStock(tt.symbol)
			assert.Equal(t, tt.expected, result,
				"IsChinaStock(%q) 应该返回 %v", tt.symbol, tt.expected)
		})
	}
}

// TestIsHKStock 测试香港股票识别功能
func TestIsHKStock(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		expected bool
	}{
		// 港股测试
		{
			name:     "港股-HK前缀-大写",
			symbol:   "HK00700",
			expected: true,
		},
		{
			name:     "港股-HK前缀-小写",
			symbol:   "hk00700",
			expected: true,
		},
		{
			name:     "港股-.HK后缀-大写",
			symbol:   "0700.HK",
			expected: true,
		},
		{
			name:     "港股-.HK后缀-小写",
			symbol:   "0700.hk",
			expected: true,
		},
		{
			name:     "港股-带空格",
			symbol:   " HK00700 ",
			expected: true,
		},
		// 非港股测试
		{
			name:     "港股代码-无前后缀",
			symbol:   "00700",
			expected: false,
		},
		{
			name:     "A股-上海",
			symbol:   "SH601138",
			expected: false,
		},
		{
			name:     "A股-深圳",
			symbol:   "SZ000001",
			expected: false,
		},
		{
			name:     "美股",
			symbol:   "AAPL",
			expected: false,
		},
		// 边界情况
		{
			name:     "空字符串",
			symbol:   "",
			expected: false,
		},
		{
			name:     "仅HK",
			symbol:   "HK",
			expected: true, // 因为 HasPrefix 会返回 true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHKStock(tt.symbol)
			assert.Equal(t, tt.expected, result,
				"IsHKStock(%q) 应该返回 %v", tt.symbol, tt.expected)
		})
	}
}

// TestGetMarketType 测试市场类型识别功能
func TestGetMarketType(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		expected   types.MarketType
		description string
	}{
		// A股测试
		{
			name:        "A股-上海-完整格式",
			symbol:      "SH601138",
			expected:    types.MarketChina,
			description: "上海股票完整格式",
		},
		{
			name:        "A股-上海-简写",
			symbol:      "601138",
			expected:    types.MarketChina,
			description: "上海股票简写格式",
		},
		{
			name:        "A股-深圳-完整格式",
			symbol:      "SZ000001",
			expected:    types.MarketChina,
			description: "深圳股票完整格式",
		},
		{
			name:        "A股-深圳-0开头",
			symbol:      "000001",
			expected:    types.MarketChina,
			description: "深圳主板股票",
		},
		{
			name:        "A股-深圳-3开头",
			symbol:      "300001",
			expected:    types.MarketChina,
			description: "深圳创业板股票",
		},
		// 港股测试
		{
			name:        "港股-HK前缀",
			symbol:      "HK00700",
			expected:    types.MarketHongKong,
			description: "港股HK前缀",
		},
		{
			name:        "港股-.HK后缀",
			symbol:      "0700.HK",
			expected:    types.MarketHongKong,
			description: "港股.HK后缀",
		},
		// 美股测试
		{
			name:        "美股-字母代码",
			symbol:      "AAPL",
			expected:    types.MarketUS,
			description: "美股字母代码",
		},
		{
			name:        "美股-长代码",
			symbol:      "GOOGL",
			expected:    types.MarketUS,
			description: "美股长代码",
		},
		{
			name:        "美股-纯数字-非6位",
			symbol:      "1234",
			expected:    types.MarketUS,
			description: "默认为美股",
		},
		// 边界情况
		{
			name:        "空字符串-默认美股",
			symbol:      "",
			expected:    types.MarketUS,
			description: "空字符串默认为美股",
		},
		{
			name:        "未知代码-默认美股",
			symbol:      "???",
			expected:    types.MarketUS,
			description: "未知代码默认为美股",
		},
		{
			name:        "大小写混合-上海",
			symbol:      "Sh601138",
			expected:    types.MarketChina,
			description: "大小写混合应正确识别",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMarketType(tt.symbol)
			assert.Equal(t, tt.expected, result,
				"GetMarketType(%q) 应该返回 %v (%s)", tt.symbol, tt.expected, tt.description)
		})
	}
}

// TestContainsChineseChars 测试中文字符检测功能
func TestContainsChineseChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// 包含中文
		{
			name:     "全中文",
			input:    "工业富联",
			expected: true,
		},
		{
			name:     "单个中文字符",
			input:    "中",
			expected: true,
		},
		{
			name:     "中英混合",
			input:    "Apple苹果",
			expected: true,
		},
		{
			name:     "中文+数字",
			input:    "工业富联601138",
			expected: true,
		},
		{
			name:     "中文+符号",
			input:    "中国平安（集团）",
			expected: true,
		},
		// 不包含中文
		{
			name:     "纯英文",
			input:    "Apple",
			expected: false,
		},
		{
			name:     "纯数字",
			input:    "601138",
			expected: false,
		},
		{
			name:     "英文+数字",
			input:    "AAPL150",
			expected: false,
		},
		{
			name:     "符号",
			input:    "!@#$%",
			expected: false,
		},
		{
			name:     "空字符串",
			input:    "",
			expected: false,
		},
		// 边界情况
		{
			name:     "日文假名-不在范围",
			input:    "トヨタ",
			expected: false,
		},
		{
			name:     "韩文-不在范围",
			input:    "삼성",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsChineseChars(tt.input)
			assert.Equal(t, tt.expected, result,
				"ContainsChineseChars(%q) 应该返回 %v", tt.input, tt.expected)
		})
	}
}

// TestGenerateSearchKeywords 测试搜索关键词生成功能
func TestGenerateSearchKeywords(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		shouldContain    []string
		shouldHaveLength int
		desc             string
	}{
		{
			name:  "带股份后缀",
			input: "工业富联股份",
			shouldContain: []string{
				"工业富联股份", // 原始
				"工业富联",     // 去掉"股份"
				"工业富",      // 前3个字
			},
			shouldHaveLength: -1, // 不检查具体长度
			desc:             "去掉后缀并生成短关键词",
		},
		{
			name:  "带集团后缀",
			input: "中国平安集团",
			shouldContain: []string{
				"中国平安集团", // 原始
				"中国平安",    // 去掉"集团"
				"平安集团",    // 去掉"中国"前缀
			},
			shouldHaveLength: -1, // 不检查具体长度
			desc:             "去掉前缀和后缀",
		},
		{
			name:  "带中国前缀",
			input: "中国平安",
			shouldContain: []string{
				"中国平安", // 原始
				"平安",    // 去掉"中国"
			},
			shouldHaveLength: -1, // 不检查具体长度
			desc:             "去掉常见前缀",
		},
		{
			name:  "短名称",
			input: "苹果",
			shouldContain: []string{
				"苹果", // 原始(长度不足4,不生成短关键词)
			},
			shouldHaveLength: 1,
			desc:             "短名称不生成额外关键词",
		},
		{
			name:             "单字符",
			input:            "A",
			shouldContain:    []string{"A"},
			shouldHaveLength: 1,
			desc:             "单字符只返回原始",
		},
		{
			name:             "空字符串",
			input:            "",
			shouldContain:    []string{""},
			shouldHaveLength: 1,
			desc:             "空字符串返回空切片",
		},
		{
			name:  "长名称-生成多个关键词",
			input: "深圳市腾讯计算机系统",
			shouldContain: []string{
				"深圳市腾讯计算机系统", // 原始
				"深圳市",           // 前3个字
			},
			shouldHaveLength: -1, // 不检查具体长度
			desc:             "长名称生成多个关键词",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSearchKeywords(tt.input)

			// 检查必须包含的关键词
			for _, expected := range tt.shouldContain {
				assert.Contains(t, result, expected,
					"应包含关键词 %q", expected)
			}

			// 如果指定了长度,检查长度
			if tt.shouldHaveLength > 0 {
				assert.Len(t, result, tt.shouldHaveLength,
					"关键词数量应为 %d: %s", tt.shouldHaveLength, tt.desc)
			}

			// 确保第一个关键词是原始输入
			assert.Greater(t, len(result), 0, "至少应有一个关键词")
			if len(result) > 0 {
				assert.Equal(t, tt.input, result[0], "第一个关键词应为原始输入")
			}
		})
	}
}

// BenchmarkIsChinaStock 性能基准测试
func BenchmarkIsChinaStock(b *testing.B) {
	testCases := []string{
		"SH601138",
		"SZ000001",
		"601138",
		"AAPL",
		"HK00700",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, symbol := range testCases {
			IsChinaStock(symbol)
		}
	}
}

// BenchmarkGetMarketType 性能基准测试
func BenchmarkGetMarketType(b *testing.B) {
	testCases := []string{
		"SH601138",
		"SZ000001",
		"HK00700",
		"AAPL",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, symbol := range testCases {
			GetMarketType(symbol)
		}
	}
}

// BenchmarkContainsChineseChars 性能基准测试
func BenchmarkContainsChineseChars(b *testing.B) {
	testCases := []string{
		"工业富联",
		"Apple",
		"Apple苹果",
		"601138",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range testCases {
			ContainsChineseChars(input)
		}
	}
}
