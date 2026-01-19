package ui

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/text"

	"stock-monitor/internal/consts"
)

// Formatter 格式化器，处理盈亏、价格的颜色格式化
type Formatter struct {
	language consts.Language
}

// NewFormatter 创建格式化器
func NewFormatter(lang consts.Language) *Formatter {
	return &Formatter{language: lang}
}

// SetLanguage 设置语言
func (f *Formatter) SetLanguage(lang consts.Language) {
	f.language = lang
}

// FormatProfitWithColor 格式化盈亏金额（带颜色）
// 中文：红涨绿跌 | 英文：绿涨红跌
func (f *Formatter) FormatProfitWithColor(profit float64) string {
	if f.language == consts.English {
		// 英文：绿色盈利，红色亏损
		if profit >= 0 {
			return text.FgGreen.Sprintf("+%.2f", profit)
		}
		return text.FgRed.Sprintf("%.2f", profit)
	}
	// 中文：红色盈利，绿色亏损
	if profit >= 0 {
		return text.FgRed.Sprintf("+%.2f", profit)
	}
	return text.FgGreen.Sprintf("%.2f", profit)
}

// FormatProfitRateWithColor 格式化盈亏比例（带颜色）
func (f *Formatter) FormatProfitRateWithColor(rate float64) string {
	if f.language == consts.English {
		// 英文：绿色盈利，红色亏损
		if rate >= 0 {
			return text.FgGreen.Sprintf("+%.2f%%", rate)
		}
		return text.FgRed.Sprintf("%.2f%%", rate)
	}
	// 中文：红色盈利，绿色亏损
	if rate >= 0 {
		return text.FgRed.Sprintf("+%.2f%%", rate)
	}
	return text.FgGreen.Sprintf("%.2f%%", rate)
}

// FormatProfitWithColorZero 格式化盈亏金额（零值显示白色）
func (f *Formatter) FormatProfitWithColorZero(profit float64) string {
	// 当数值接近0时（考虑浮点数精度），显示白色（无颜色）
	if Abs(profit) < 0.001 {
		return fmt.Sprintf("%.2f", profit)
	}
	// 否则使用语言相关颜色逻辑
	return f.FormatProfitWithColor(profit)
}

// FormatProfitRateWithColorZero 格式化盈亏比例（零值显示白色）
func (f *Formatter) FormatProfitRateWithColorZero(rate float64) string {
	// 当数值接近0时（考虑浮点数精度），显示白色（无颜色）
	if Abs(rate) < 0.001 {
		return fmt.Sprintf("%.2f%%", rate)
	}
	// 否则使用语言相关颜色逻辑
	return f.FormatProfitRateWithColor(rate)
}

// FormatPriceWithColor 格式化价格（根据涨跌显示颜色）
func (f *Formatter) FormatPriceWithColor(currentPrice, prevClose float64) string {
	if prevClose == 0 {
		// 如果昨收价为0，直接显示价格不加颜色
		return fmt.Sprintf("%.3f", currentPrice)
	}

	if currentPrice > prevClose {
		if f.language == consts.English {
			// 英文：高于昨收价显示绿色
			return text.FgGreen.Sprintf("%.3f", currentPrice)
		}
		// 中文：高于昨收价显示红色
		return text.FgRed.Sprintf("%.3f", currentPrice)
	} else if currentPrice < prevClose {
		if f.language == consts.English {
			// 英文：低于昨收价显示红色
			return text.FgRed.Sprintf("%.3f", currentPrice)
		}
		// 中文：低于昨收价显示绿色
		return text.FgGreen.Sprintf("%.3f", currentPrice)
	}
	// 等于昨收价显示白色（无颜色）
	return fmt.Sprintf("%.3f", currentPrice)
}

// FormatVolume 格式化成交量（万/亿）
func FormatVolume(volume int64) string {
	if volume >= 1000000000 {
		return fmt.Sprintf("%.2f B", float64(volume)/1000000000)
	} else if volume >= 100000000 {
		return fmt.Sprintf("%.2f Y", float64(volume)/100000000)
	} else if volume >= 10000 {
		return fmt.Sprintf("%.2f W", float64(volume)/10000)
	}
	return fmt.Sprintf("%d", volume)
}

// FormatVolumeZh 格式化成交量（中文：万/亿）
func FormatVolumeZh(volume int64) string {
	if volume >= 1000000000 {
		return fmt.Sprintf("%.2f十亿", float64(volume)/1000000000)
	} else if volume >= 100000000 {
		return fmt.Sprintf("%.2f亿", float64(volume)/100000000)
	} else if volume >= 10000 {
		return fmt.Sprintf("%.2f万", float64(volume)/10000)
	}
	return fmt.Sprintf("%d", volume)
}

// Abs 返回浮点数的绝对值
func Abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
