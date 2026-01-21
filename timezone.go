package main

import (
	"time"

	"stock-monitor/internal/market"
)

// ============================================================================
// 时区相关函数 - 封装 internal/market 包
// ============================================================================

// parseTimeInMarket 在指定市场时区中解析时间字符串（带日志）
// date: "20251210", timeStr: "09:30", marketConfig: 市场配置
// 返回：该市场的本地时间（time.Time）
func parseTimeInMarket(date string, timeStr string, marketConfig MarketConfig) (time.Time, error) {
	// 调用 internal/market 的实现
	return market.ParseTimeInMarket(date, timeStr, marketConfig)
}

// isMarketOpenForConfig 检查指定市场在指定时间是否开市（带日志）
// checkTime: 要检查的时间
// marketConfig: 市场配置
// 返回：true 表示开市，false 表示休市
func isMarketOpenForConfig(checkTime time.Time, marketConfig MarketConfig) bool {
	// 调用 internal/market 的实现
	return market.IsMarketOpenForConfig(checkTime, marketConfig)
}

// getCurrentDateForMarket 获取指定市场的当前日期（考虑时区）
// market: 市场类型
// m: Model 指针（用于访问配置）
// 返回：当前日期字符串 (YYYYMMDD)
func getCurrentDateForMarket(marketType MarketType, m *Model) string {
	// 调用 internal/market 的实现，传入配置而不是整个 Model
	return market.GetCurrentDateForMarket(marketType, m.config)
}

// getMarketLocation 根据市场类型返回对应的时区Location
// marketType: 市场类型
// 返回：时区Location和可能的错误
func getMarketLocation(marketType MarketType) (*time.Location, error) {
	// 调用 internal/market 的实现
	return market.GetMarketLocation(marketType)
}
