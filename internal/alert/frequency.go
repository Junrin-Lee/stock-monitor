package alert

import (
	"time"

	"github.com/dromara/carbon/v2"

	"stock-monitor/internal/types"
)

// CanTriggerInCurrentPeriod 检查告警是否可以在当前周期触发
// 返回 true 表示距离上次触发已经过了足够的时间，可以再次触发
func CanTriggerInCurrentPeriod(alert types.Alert) bool {
	// 如果从未触发过，允许触发
	if alert.TriggeredAt.IsZero() {
		return true
	}

	now := carbon.Now()
	lastTriggered := carbon.CreateFromStdTime(alert.TriggeredAt)

	switch alert.Frequency {
	case types.TriggerOnce:
		// 一次性告警已经触发过，不再触发
		return false

	case types.TriggerDaily:
		// 如果不是同一天，可以触发
		return !IsSameDay(now, lastTriggered)

	case types.TriggerWeekly:
		// 如果不是同一周，可以触发
		return !IsSameWeek(now, lastTriggered)

	case types.TriggerMonthly:
		// 如果不是同一月，可以触发
		return !IsSameMonth(now, lastTriggered)

	case types.TriggerEveryNDays:
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

// IsSameDay 判断两个时间是否是同一天
func IsSameDay(a, b *carbon.Carbon) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// IsSameWeek 判断两个时间是否是同一周
func IsSameWeek(a, b *carbon.Carbon) bool {
	aYear, aWeek := a.StdTime().ISOWeek()
	bYear, bWeek := b.StdTime().ISOWeek()
	return aYear == bYear && aWeek == bWeek
}

// IsSameMonth 判断两个时间是否是同一月
func IsSameMonth(a, b *carbon.Carbon) bool {
	return a.Year() == b.Year() && a.Month() == b.Month()
}

// GetNextTriggerTime 获取下次可触发的时间
// 用于 UI 显示目的
func GetNextTriggerTime(alert types.Alert) time.Time {
	if alert.TriggeredAt.IsZero() {
		return time.Now()
	}

	lastTriggered := carbon.CreateFromStdTime(alert.TriggeredAt)

	switch alert.Frequency {
	case types.TriggerOnce:
		return time.Time{} // 永不再触发

	case types.TriggerDaily:
		return lastTriggered.AddDay().StartOfDay().StdTime()

	case types.TriggerWeekly:
		return lastTriggered.AddWeeks(1).StartOfWeek().StdTime()

	case types.TriggerMonthly:
		return lastTriggered.AddMonth().StartOfMonth().StdTime()

	case types.TriggerEveryNDays:
		if alert.FrequencyDays <= 0 {
			return time.Time{}
		}
		return lastTriggered.AddDays(alert.FrequencyDays).StartOfDay().StdTime()

	default:
		return time.Time{}
	}
}

// GetFrequencyOptions 获取所有可用的触发频率选项
func GetFrequencyOptions() []types.TriggerFrequency {
	return []types.TriggerFrequency{
		types.TriggerOnce,
		types.TriggerDaily,
		types.TriggerWeekly,
		types.TriggerMonthly,
		types.TriggerEveryNDays,
	}
}
