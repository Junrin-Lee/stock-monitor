package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPadHKStockCode 测试港股代码补齐功能
func TestPadHKStockCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
		desc     string
	}{
		{
			name:     "三位数补齐",
			code:     "700",
			expected: "00700",
			desc:     "腾讯股票代码",
		},
		{
			name:     "四位数补齐",
			code:     "9626",
			expected: "09626",
			desc:     "哔哩哔哩股票代码",
		},
		{
			name:     "已经五位",
			code:     "00700",
			expected: "00700",
			desc:     "已补齐的代码",
		},
		{
			name:     "超长代码-不补齐",
			code:     "000700",
			expected: "000700",
			desc:     "长度超过5位不补齐",
		},
		{
			name:     "单位数补齐",
			code:     "1",
			expected: "00001",
			desc:     "补齐4个零",
		},
		{
			name:     "两位数补齐",
			code:     "88",
			expected: "00088",
			desc:     "补齐3个零",
		},
		{
			name:     "空字符串",
			code:     "",
			expected: "00000",
			desc:     "空字符串补齐为5个零",
		},
		{
			name:     "带前导空格",
			code:     "  700",
			expected: "00700",
			desc:     "去除空格后补齐",
		},
		{
			name:     "带后导空格",
			code:     "700  ",
			expected: "00700",
			desc:     "去除空格后补齐",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadHKStockCode(tt.code)
			assert.Equal(t, tt.expected, result,
				"PadHKStockCode(%q) 应该返回 %q (%s)", tt.code, tt.expected, tt.desc)
			// 验证结果长度
			if len(tt.code) < 5 {
				assert.Len(t, result, 5, "结果应为5位")
			}
		})
	}
}

// TestConvertJSONCodeToStandard 测试JSON格式代码转换
func TestConvertJSONCodeToStandard(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
		desc     string
	}{
		// 已经是标准格式
		{
			name:     "已标准-上海",
			code:     "SH601138",
			expected: "SH601138",
			desc:     "已是标准格式无需转换",
		},
		{
			name:     "已标准-深圳",
			code:     "SZ000001",
			expected: "SZ000001",
			desc:     "已是标准格式无需转换",
		},
		{
			name:     "已标准-港股",
			code:     "HK00700",
			expected: "HK00700",
			desc:     "已是标准格式无需转换",
		},
		// 需要转换的格式
		{
			name:     "上海股票-6开头",
			code:     "601138",
			expected: "SH601138",
			desc:     "6开头添加SH前缀",
		},
		{
			name:     "深圳股票-0开头",
			code:     "000001",
			expected: "SZ000001",
			desc:     "0开头添加SZ前缀",
		},
		{
			name:     "深圳股票-3开头",
			code:     "300001",
			expected: "SZ300001",
			desc:     "3开头添加SZ前缀",
		},
		// 边界情况
		{
			name:     "非6位数字",
			code:     "12345",
			expected: "12345",
			desc:     "非6位数字不转换",
		},
		{
			name:     "字母代码",
			code:     "AAPL",
			expected: "AAPL",
			desc:     "字母代码不转换",
		},
		{
			name:     "空字符串",
			code:     "",
			expected: "",
			desc:     "空字符串返回空",
		},
		{
			name:     "带空格",
			code:     " 601138 ",
			expected: "SH601138",
			desc:     "去除空格后转换",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertJSONCodeToStandard(tt.code)
			assert.Equal(t, tt.expected, result,
				"ConvertJSONCodeToStandard(%q) 应该返回 %q (%s)", tt.code, tt.expected, tt.desc)
		})
	}
}

// TestConvertToStandardCode 测试腾讯格式代码转换
func TestConvertToStandardCode(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		shortCode string
		expected  string
		desc      string
	}{
		// 上海股票
		{
			name:      "上海股票-小写",
			code:      "sh601138",
			shortCode: "601138",
			expected:  "SH601138",
			desc:      "腾讯sh前缀转SH",
		},
		{
			name:      "上海股票-大写",
			code:      "SH601138",
			shortCode: "601138",
			expected:  "SH601138",
			desc:      "已是大写格式",
		},
		// 深圳股票
		{
			name:      "深圳股票-小写",
			code:      "sz000001",
			shortCode: "000001",
			expected:  "SZ000001",
			desc:      "腾讯sz前缀转SZ",
		},
		{
			name:      "深圳股票-大写",
			code:      "SZ000001",
			shortCode: "000001",
			expected:  "SZ000001",
			desc:      "已是大写格式",
		},
		// 港股
		{
			name:      "港股-小写",
			code:      "hk00700",
			shortCode: "00700",
			expected:  "HK00700",
			desc:      "腾讯hk前缀转HK",
		},
		{
			name:      "港股-大写",
			code:      "HK00700",
			shortCode: "00700",
			expected:  "HK00700",
			desc:      "已是大写格式",
		},
		// 大小写混合
		{
			name:      "大小写混合-Sh",
			code:      "Sh601138",
			shortCode: "601138",
			expected:  "SH601138",
			desc:      "混合大小写正确转换",
		},
		// 无法识别的格式
		{
			name:      "无法识别-返回原值",
			code:      "unknown123",
			shortCode: "123",
			expected:  "unknown123",
			desc:      "无法识别的格式返回原值",
		},
		{
			name:      "美股代码",
			code:      "AAPL",
			shortCode: "AAPL",
			expected:  "aapl", // ConvertToStandardCode 会转为小写
			desc:      "美股代码返回小写",
		},
		// 边界情况
		{
			name:      "空字符串",
			code:      "",
			shortCode: "",
			expected:  "",
			desc:      "空字符串返回空",
		},
		{
			name:      "带空格",
			code:      " sh601138 ",
			shortCode: "601138",
			expected:  "SH601138",
			desc:      "去除空格后转换",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToStandardCode(tt.code, tt.shortCode)
			assert.Equal(t, tt.expected, result,
				"ConvertToStandardCode(%q, %q) 应该返回 %q (%s)",
				tt.code, tt.shortCode, tt.expected, tt.desc)
		})
	}
}

// TestConvertSinaCodeToStandard 测试新浪格式代码转换
func TestConvertSinaCodeToStandard(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
		desc     string
	}{
		// 已经是标准格式
		{
			name:     "已标准-上海-大写",
			code:     "SH601138",
			expected: "SH601138",
			desc:     "已是标准格式",
		},
		{
			name:     "已标准-深圳-大写",
			code:     "SZ000001",
			expected: "SZ000001",
			desc:     "已是标准格式",
		},
		// 新浪格式转换
		{
			name:     "新浪格式-sh前缀",
			code:     "sh601138",
			expected: "SH601138",
			desc:     "sh前缀转大写SH",
		},
		{
			name:     "新浪格式-sz前缀",
			code:     "sz000001",
			expected: "SZ000001",
			desc:     "sz前缀转大写SZ",
		},
		{
			name:     "新浪格式-hk前缀",
			code:     "hk00700",
			expected: "HK00700",
			desc:     "hk前缀转大写HK",
		},
		// 纯数字代码
		{
			name:     "6开头6位数",
			code:     "601138",
			expected: "SH601138",
			desc:     "6开头添加SH",
		},
		{
			name:     "0开头6位数",
			code:     "000001",
			expected: "SZ000001",
			desc:     "0开头添加SZ",
		},
		{
			name:     "3开头6位数",
			code:     "300001",
			expected: "SZ300001",
			desc:     "3开头添加SZ",
		},
		// 其他情况
		{
			name:     "非6位数字",
			code:     "12345",
			expected: "12345",
			desc:     "非6位数字转大写",
		},
		{
			name:     "字母代码",
			code:     "aapl",
			expected: "AAPL",
			desc:     "字母代码转大写",
		},
		{
			name:     "空字符串",
			code:     "",
			expected: "",
			desc:     "空字符串返回空",
		},
		// 边界情况
		{
			name:     "带空格-需要转换",
			code:     " sh601138 ",
			expected: "SH601138",
			desc:     "去除空格后转换",
		},
		{
			name:     "大小写混合",
			code:     "Sh601138",
			expected: "SH601138",
			desc:     "混合大小写正确处理",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSinaCodeToStandard(tt.code)
			assert.Equal(t, tt.expected, result,
				"ConvertSinaCodeToStandard(%q) 应该返回 %q (%s)", tt.code, tt.expected, tt.desc)
		})
	}
}

// TestMin 测试最小值函数
func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "a小于b",
			a:        5,
			b:        10,
			expected: 5,
		},
		{
			name:     "b小于a",
			a:        10,
			b:        5,
			expected: 5,
		},
		{
			name:     "a等于b",
			a:        7,
			b:        7,
			expected: 7,
		},
		{
			name:     "负数比较",
			a:        -5,
			b:        -10,
			expected: -10,
		},
		{
			name:     "零和正数",
			a:        0,
			b:        5,
			expected: 0,
		},
		{
			name:     "零和负数",
			a:        0,
			b:        -5,
			expected: -5,
		},
		{
			name:     "大数比较",
			a:        1000000,
			b:        999999,
			expected: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result,
				"Min(%d, %d) 应该返回 %d", tt.a, tt.b, tt.expected)
		})
	}
}

// BenchmarkPadHKStockCode 性能基准测试
func BenchmarkPadHKStockCode(b *testing.B) {
	testCases := []string{
		"700",
		"9626",
		"00700",
		"1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, code := range testCases {
			PadHKStockCode(code)
		}
	}
}

// BenchmarkConvertToStandardCode 性能基准测试
func BenchmarkConvertToStandardCode(b *testing.B) {
	testCases := []struct {
		code      string
		shortCode string
	}{
		{"sh601138", "601138"},
		{"sz000001", "000001"},
		{"hk00700", "00700"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			ConvertToStandardCode(tc.code, tc.shortCode)
		}
	}
}

// BenchmarkConvertSinaCodeToStandard 性能基准测试
func BenchmarkConvertSinaCodeToStandard(b *testing.B) {
	testCases := []string{
		"sh601138",
		"sz000001",
		"601138",
		"AAPL",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, code := range testCases {
			ConvertSinaCodeToStandard(code)
		}
	}
}
