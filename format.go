package main

import (
	"stock-monitor/internal/api"
	"stock-monitor/internal/ui"
)

// ============================================================================
// 盈亏格式化函数 - 委托给 internal/ui.Formatter
// 中文：红涨绿跌 | 英文：绿涨红跌
// ============================================================================

// formatProfitWithColorLang 格式化盈亏金额（带颜色）
func (m *Model) formatProfitWithColorLang(profit float64) string {
	formatter := ui.NewFormatter(m.language)
	return formatter.FormatProfitWithColor(profit)
}

// formatProfitRateWithColorLang 格式化盈亏比例（带颜色）
func (m *Model) formatProfitRateWithColorLang(rate float64) string {
	formatter := ui.NewFormatter(m.language)
	return formatter.FormatProfitRateWithColor(rate)
}

// formatProfitWithColorZeroLang 格式化盈亏金额（零值显示白色）
func (m *Model) formatProfitWithColorZeroLang(profit float64) string {
	formatter := ui.NewFormatter(m.language)
	return formatter.FormatProfitWithColorZero(profit)
}

// formatProfitRateWithColorZeroLang 格式化盈亏比例（零值显示白色）
func (m *Model) formatProfitRateWithColorZeroLang(rate float64) string {
	formatter := ui.NewFormatter(m.language)
	return formatter.FormatProfitRateWithColorZero(rate)
}

// formatProfitRateWithColorZeroLangForStock 格式化盈亏率（支持股票类型检测）
func (m *Model) formatProfitRateWithColorZeroLangForStock(rate float64, symbol string) string {
	// 对于非A股（如美股），显示 "-" 表示数据不可用
	if !api.IsChinaStock(symbol) {
		return "-"
	}
	return m.formatProfitRateWithColorZeroLang(rate)
}

// formatPriceWithColorLang 格式化价格（根据涨跌显示颜色）
func (m *Model) formatPriceWithColorLang(currentPrice, prevClose float64) string {
	formatter := ui.NewFormatter(m.language)
	return formatter.FormatPriceWithColor(currentPrice, prevClose)
}

// ============================================================================
// 其他格式化函数 - 委托给 internal/ui
// ============================================================================

// formatVolume 格式化成交量（万/亿）
func formatVolume(volume int64) string {
	return ui.FormatVolume(volume)
}

// formatStockNameWithPortfolioHighlight 格式化股票名称（持仓股票高亮）
func (m *Model) formatStockNameWithPortfolioHighlight(name, code string) string {
	if m.isStockInPortfolio(code) {
		// 使用颜色工具处理背景高亮
		colorUtils := ui.NewColorUtils()
		configColor := m.config.Display.PortfolioHighlight

		logDebug("log.highlight.found", name, code, configColor)

		// 获取最终的颜色名称（仅支持go-pretty颜色名称）
		finalColorName := colorUtils.GetColorFromConfigOrDefault(configColor, "yellow") // 默认黄色背景

		logDebug("log.highlight.finalColor", finalColorName)

		// 应用背景颜色格式化
		result := colorUtils.FormatTextWithBackground(name, finalColorName)
		logDebug("log.highlight.result", result, name)

		return result
	}
	return name
}

// ============================================================================
// 辅助函数 - 委托给 internal/ui
// ============================================================================

// abs 返回浮点数的绝对值
func abs(x float64) float64 {
	return ui.Abs(x)
}
