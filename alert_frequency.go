package main

import (
	"fmt"
	"time"

	"github.com/dromara/carbon/v2"

	internalAlert "stock-monitor/internal/alert"
	"stock-monitor/internal/api"
	"stock-monitor/internal/types"
)

// canTriggerInCurrentPeriod 检查告警是否可以在当前周期触发
// 包装 internal/alert 包的函数，用于类型转换
func canTriggerInCurrentPeriod(alert Alert) bool {
	// 转换为 types.Alert
	typesAlert := types.Alert{
		ID:            alert.ID,
		StockCode:     alert.StockCode,
		StockName:     alert.StockName,
		Type:          types.AlertType(alert.Type),
		Condition:     alert.Condition,
		Threshold:     alert.Threshold,
		IsActive:      alert.IsActive,
		Frequency:     types.TriggerFrequency(alert.Frequency),
		FrequencyDays: alert.FrequencyDays,
		CreatedAt:     alert.CreatedAt,
		TriggeredAt:   alert.TriggeredAt,
		LastChecked:   alert.LastChecked,
		BatchTag:      alert.BatchTag,
	}
	timezone := getMarketTimezone(alert.StockCode)
	return internalAlert.CanTriggerInCurrentPeriod(typesAlert, timezone)
}

// isSameDay 判断两个时间是否是同一天
func isSameDay(a, b *carbon.Carbon) bool {
	return internalAlert.IsSameDay(a, b)
}

// isSameWeek 判断两个时间是否是同一周
func isSameWeek(a, b *carbon.Carbon) bool {
	return internalAlert.IsSameWeek(a, b)
}

// isSameMonth 判断两个时间是否是同一月
func isSameMonth(a, b *carbon.Carbon) bool {
	return internalAlert.IsSameMonth(a, b)
}

// getNextTriggerTime 获取下次可触发的时间
func getNextTriggerTime(alert Alert) time.Time {
	// 转换为 types.Alert
	typesAlert := types.Alert{
		ID:            alert.ID,
		StockCode:     alert.StockCode,
		StockName:     alert.StockName,
		Type:          types.AlertType(alert.Type),
		Condition:     alert.Condition,
		Threshold:     alert.Threshold,
		IsActive:      alert.IsActive,
		Frequency:     types.TriggerFrequency(alert.Frequency),
		FrequencyDays: alert.FrequencyDays,
		CreatedAt:     alert.CreatedAt,
		TriggeredAt:   alert.TriggeredAt,
		LastChecked:   alert.LastChecked,
		BatchTag:      alert.BatchTag,
	}
	return internalAlert.GetNextTriggerTime(typesAlert)
}

// getMarketTimezone 根据股票代码返回对应市场的 IANA 时区字符串
func getMarketTimezone(stockCode string) string {
	switch api.GetMarketType(stockCode) {
	case types.MarketChina:
		return "Asia/Shanghai"
	case types.MarketHongKong:
		return "Asia/Hong_Kong"
	case types.MarketUS:
		return "America/New_York"
	default:
		return "" // 使用本地时区
	}
}

// getFrequencyDisplayText 获取频率的显示文本
func (m *Model) getFrequencyDisplayText(frequency TriggerFrequency, days int) string {
	switch frequency {
	case TriggerOnce:
		return m.getText("alert.frequency.once")
	case TriggerDaily:
		return m.getText("alert.frequency.daily")
	case TriggerWeekly:
		return m.getText("alert.frequency.weekly")
	case TriggerMonthly:
		return m.getText("alert.frequency.monthly")
	case TriggerEveryNDays:
		return fmt.Sprintf(m.getText("alert.frequency.everyNDays"), days)
	default:
		// 向后兼容：未设置频率时显示一次性
		return m.getText("alert.frequency.once")
	}
}

// getNextTriggerDisplayText 获取下次触发时间的显示文本
func (m *Model) getNextTriggerDisplayText(alert Alert) string {
	// 一次性告警
	if alert.Frequency == TriggerOnce || alert.Frequency == "" {
		if !alert.TriggeredAt.IsZero() {
			return m.getText("alert.frequency.neverAgain")
		}
		return m.getText("alert.frequency.pending")
	}

	// 从未触发过
	if alert.TriggeredAt.IsZero() {
		return m.getText("alert.frequency.pending")
	}

	// 获取下次触发时间
	nextTime := getNextTriggerTime(alert)
	if nextTime.IsZero() {
		return m.getText("alert.frequency.neverAgain")
	}

	now := time.Now()
	if now.After(nextTime) || now.Equal(nextTime) {
		return m.getText("alert.frequency.canTrigger")
	}

	// 显示人类可读的时间差
	nextCarbon := carbon.CreateFromStdTime(nextTime)
	return nextCarbon.DiffForHumans()
}

// getFrequencyOptions 获取所有可用的触发频率选项
func getFrequencyOptions() []TriggerFrequency {
	options := internalAlert.GetFrequencyOptions()
	result := make([]TriggerFrequency, len(options))
	for i, opt := range options {
		result[i] = TriggerFrequency(opt)
	}
	return result
}

// getFrequencyCursorFromValue 根据频率类型返回对应的光标位置
// 用于编辑告警时初始化频率选择器的光标位置
func getFrequencyCursorFromValue(frequency TriggerFrequency) int {
	switch frequency {
	case TriggerOnce:
		return 0
	case TriggerDaily:
		return 1
	case TriggerWeekly:
		return 2
	case TriggerMonthly:
		return 3
	case TriggerEveryNDays:
		return 4
	default:
		return 0 // 默认返回 "一次性"
	}
}
