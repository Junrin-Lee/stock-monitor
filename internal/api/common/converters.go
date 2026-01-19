package common

import (
	"strings"
)

// PadHKStockCode 将港股代码补齐为5位数字
// 例如: "700" -> "00700", "2020" -> "02020", "00700" -> "00700"
func PadHKStockCode(code string) string {
	// 移除可能的前导零后的纯数字部分
	code = strings.TrimSpace(code)
	if len(code) >= 5 {
		return code
	}
	// 补齐到5位
	return strings.Repeat("0", 5-len(code)) + code
}

// ConvertJSONCodeToStandard 转换JSON格式的股票代码为标准格式
func ConvertJSONCodeToStandard(code string) string {
	code = strings.TrimSpace(code)

	// 如果已经是标准格式，直接返回
	if strings.HasPrefix(code, "SH") || strings.HasPrefix(code, "SZ") || strings.HasPrefix(code, "HK") {
		return code
	}

	// 根据数字开头判断市场
	if len(code) == 6 {
		if strings.HasPrefix(code, "6") {
			return "SH" + code
		} else if strings.HasPrefix(code, "0") || strings.HasPrefix(code, "3") {
			return "SZ" + code
		}
	}

	return code
}

// ConvertToStandardCode 将腾讯的股票代码转换为标准格式
func ConvertToStandardCode(code, shortCode string) string {
	code = strings.ToLower(strings.TrimSpace(code))

	if strings.HasPrefix(code, "sh") {
		return "SH" + shortCode
	} else if strings.HasPrefix(code, "sz") {
		return "SZ" + shortCode
	} else if strings.HasPrefix(code, "hk") {
		return "HK" + shortCode
	}

	// 如果无法识别，返回原始代码
	return code
}

// ConvertSinaCodeToStandard 转换新浪的股票代码为标准格式
func ConvertSinaCodeToStandard(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))

	// 如果已经是标准格式，直接返回
	if strings.HasPrefix(strings.ToUpper(code), "SH") || strings.HasPrefix(strings.ToUpper(code), "SZ") {
		return strings.ToUpper(code)
	}

	if strings.HasPrefix(code, "sh") {
		return "SH" + strings.TrimPrefix(code, "sh")
	} else if strings.HasPrefix(code, "sz") {
		return "SZ" + strings.TrimPrefix(code, "sz")
	} else if strings.HasPrefix(code, "hk") {
		return "HK" + strings.TrimPrefix(code, "hk")
	}

	// 如果是6位数字，根据开头判断市场
	if len(code) == 6 {
		if strings.HasPrefix(code, "6") {
			return "SH" + code
		} else if strings.HasPrefix(code, "0") || strings.HasPrefix(code, "3") {
			return "SZ" + code
		}
	}

	return strings.ToUpper(code)
}

// Min 返回两个整数中的较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
