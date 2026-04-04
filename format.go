package main

import (
	"fmt"
	"strings"
	"time"

	"stock-monitor/internal/api"
	"stock-monitor/internal/intraday"
	"stock-monitor/internal/ui"
	"stock-monitor/internal/ui/sparkline"
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

// ============================================================================
// Sparkline 趋势图
// ============================================================================

// getSparklineForStock 获取指定股票的 Sparkline 趋势图字符串（带缓存）
// TTL 与 config.Update.RefreshInterval 对齐，保持价格和趋势图刷新频率一致
func (m *Model) getSparklineForStock(code string) string {
	ttl := time.Duration(m.config.Update.RefreshInterval) * time.Second
	if m.sparklineCache != nil {
		if cached, ok := m.sparklineCache[code]; ok {
			if cacheTime, ok := m.sparklineCacheTime[code]; ok && time.Since(cacheTime) < ttl {
				return cached
			}
		}
	}

	prices := intraday.LoadLatestPrices(code)
	if len(prices) < 3 {
		return sparkline.Generate(nil, 12, "", "", "") // 返回占位符
	}

	isAShare := strings.HasPrefix(code, "SH") || strings.HasPrefix(code, "SZ")
	result := sparkline.GenerateWithDefaults(prices, isAShare)

	if m.sparklineCache == nil {
		m.sparklineCache = make(map[string]string)
	}
	if m.sparklineCacheTime == nil {
		m.sparklineCacheTime = make(map[string]time.Time)
	}
	m.sparklineCache[code] = result
	m.sparklineCacheTime[code] = time.Now()

	return result
}

// getStockPriceCacheEntry 从股价缓存中读取指定股票的 StockData（并发安全）
func (m *Model) getStockPriceCacheEntry(code string) *StockData {
	m.stockPriceMutex.RLock()
	defer m.stockPriceMutex.RUnlock()
	if entry, ok := m.stockPriceCache[code]; ok && entry != nil {
		return entry.Data
	}
	return nil
}

// formatPrePostChange 格式化盘前/盘后价格（带颜色和涨跌幅）
func (m *Model) formatPrePostChange(price, percent, prevClose float64) string {
	formatter := ui.NewFormatter(m.language)
	direction := ""
	if price > prevClose {
		direction = "+"
	}
	colorStr := formatter.FormatProfitRateWithColor(percent)
	_ = colorStr
	// 格式：价格 (+涨跌幅%)，颜色由 Formatter 决定
	return formatter.FormatPriceWithColor(price, prevClose) + fmt.Sprintf(" %s%.2f%%", direction, percent)
}
