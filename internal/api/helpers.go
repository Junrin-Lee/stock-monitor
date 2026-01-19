package api

import (
	"stock-monitor/internal/types"
	"strings"
)

// ContainsChineseChars 检查字符串是否包含中文字符
func ContainsChineseChars(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// IsChinaStock 判断是否为中国A股
func IsChinaStock(symbol string) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	return strings.HasPrefix(symbol, "SH") || strings.HasPrefix(symbol, "SZ") ||
		(len(symbol) == 6 && (strings.HasPrefix(symbol, "0") || strings.HasPrefix(symbol, "3") || strings.HasPrefix(symbol, "6")))
}

// IsHKStock 判断是否为香港股票
func IsHKStock(symbol string) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	return strings.HasPrefix(symbol, "HK") || strings.HasSuffix(symbol, ".HK")
}

// GetMarketType 根据股票代码识别市场类型
func GetMarketType(symbol string) types.MarketType {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// A股识别 (上海、深圳)
	if strings.HasPrefix(symbol, "SH") || strings.HasPrefix(symbol, "SZ") ||
		(len(symbol) == 6 && (strings.HasPrefix(symbol, "0") ||
			strings.HasPrefix(symbol, "3") ||
			strings.HasPrefix(symbol, "6"))) {
		return types.MarketChina
	}

	// 港股识别
	if strings.HasPrefix(symbol, "HK") || strings.HasSuffix(symbol, ".HK") {
		return types.MarketHongKong
	}

	// 默认为美股
	return types.MarketUS
}

// GenerateSearchKeywords 生成搜索关键词变形
func GenerateSearchKeywords(name string) []string {
	var keywords []string

	// 原始关键词
	keywords = append(keywords, name)

	// 如果名称包含"股份"、"集团"等后缀，尝试去掉
	suffixes := []string{"股份", "集团", "公司", "有限公司", "科技", "实业"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			shortName := strings.TrimSuffix(name, suffix)
			if len(shortName) > 1 {
				keywords = append(keywords, shortName)
			}
		}
	}

	// 如果名称包含"中国"、"上海"等前缀，尝试去掉
	prefixes := []string{"中国", "上海", "北京", "广东", "深圳", "天津"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix)+1 {
			shortName := strings.TrimPrefix(name, prefix)
			if len(shortName) > 1 {
				keywords = append(keywords, shortName)
			}
		}
	}

	// 如果名称较长，尝试取前几个字符作为关键词
	if len([]rune(name)) > 4 {
		runes := []rune(name)
		// 取前3个字符
		if len(runes) >= 3 {
			keywords = append(keywords, string(runes[:3]))
		}
		// 取前4个字符
		if len(runes) >= 4 {
			keywords = append(keywords, string(runes[:4]))
		}
	}

	return keywords
}
