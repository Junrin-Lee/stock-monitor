package main

import (
	"fmt"
	"time"

	"github.com/dromara/carbon/v2"
)

// canTriggerInCurrentPeriod 检查告警是否可以在当前周期触发
// 返回 true 表示距离上次触发已经过了足够的时间，可以再次触发
func canTriggerInCurrentPeriod(alert Alert) bool {
	// 如果从未触发过，允许触发
	if alert.TriggeredAt.IsZero() {
		return true
	}

	now := carbon.Now()
	lastTriggered := carbon.CreateFromStdTime(alert.TriggeredAt)

	switch alert.Frequency {
	case TriggerOnce:
		// 一次性告警已经触发过，不再触发
		return false

	case TriggerDaily:
		// 如果不是同一天，可以触发
		return !isSameDay(now, lastTriggered)

	case TriggerWeekly:
		// 如果不是同一周，可以触发
		return !isSameWeek(now, lastTriggered)

	case TriggerMonthly:
		// 如果不是同一月，可以触发
		return !isSameMonth(now, lastTriggered)

	case TriggerEveryNDays:
		if alert.FrequencyDays <= 0 {
			return false // 无效配置
		}
		// 计算距离上次触发的天数
		daysSinceTrigger := now.DiffAbsInDays(lastTriggered)
		return daysSinceTrigger >= int64(alert.FrequencyDays)

	default:
		// 未知频率或空值，视为一次性告警（向后兼容）
		return false
	}
}

// isSameDay 判断两个时间是否是同一天
func isSameDay(a, b *carbon.Carbon) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// isSameWeek 判断两个时间是否是同一周
func isSameWeek(a, b *carbon.Carbon) bool {
	aYear, aWeek := a.StdTime().ISOWeek()
	bYear, bWeek := b.StdTime().ISOWeek()
	return aYear == bYear && aWeek == bWeek
}

// isSameMonth 判断两个时间是否是同一月
func isSameMonth(a, b *carbon.Carbon) bool {
	return a.Year() == b.Year() && a.Month() == b.Month()
}

// getNextTriggerTime 获取下次可触发的时间
// 用于 UI 显示目的
func getNextTriggerTime(alert Alert) time.Time {
	if alert.TriggeredAt.IsZero() {
		return time.Now()
	}

	lastTriggered := carbon.CreateFromStdTime(alert.TriggeredAt)

	switch alert.Frequency {
	case TriggerOnce:
		return time.Time{} // 永不再触发

	case TriggerDaily:
		return lastTriggered.AddDay().StartOfDay().StdTime()

	case TriggerWeekly:
		return lastTriggered.AddWeeks(1).StartOfWeek().StdTime()

	case TriggerMonthly:
		return lastTriggered.AddMonth().StartOfMonth().StdTime()

	case TriggerEveryNDays:
		if alert.FrequencyDays <= 0 {
			return time.Time{}
		}
		return lastTriggered.AddDays(alert.FrequencyDays).StartOfDay().StdTime()

	default:
		return time.Time{}
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
	return []TriggerFrequency{
		TriggerOnce,
		TriggerDaily,
		TriggerWeekly,
		TriggerMonthly,
		TriggerEveryNDays,
	}
}
