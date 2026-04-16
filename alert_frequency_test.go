package main

import (
	"stock-monitor/internal/data"
	"stock-monitor/internal/types"
	"testing"
	"time"

	internalAlert "stock-monitor/internal/alert"
	"github.com/dromara/carbon/v2"
)

// TestCanTriggerInCurrentPeriod_NeverTriggered 测试从未触发过的告警
func TestCanTriggerInCurrentPeriod_NeverTriggered(t *testing.T) {
	alert := Alert{
		Frequency:   TriggerOnce,
		TriggeredAt: time.Time{}, // 零值，从未触发
	}

	if !canTriggerInCurrentPeriod(alert) {
		t.Error("从未触发过的告警应该可以触发")
	}
}

// TestCanTriggerInCurrentPeriod_Once 测试一次性告警
func TestCanTriggerInCurrentPeriod_Once(t *testing.T) {
	// 已触发过的一次性告警不应再触发
	alert := Alert{
		Frequency:   TriggerOnce,
		TriggeredAt: time.Now().Add(-time.Hour), // 1小时前触发过
	}

	if canTriggerInCurrentPeriod(alert) {
		t.Error("已触发的一次性告警不应该再次触发")
	}
}

// TestCanTriggerInCurrentPeriod_Daily 测试每天一次告警
func TestCanTriggerInCurrentPeriod_Daily(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name        string
		triggeredAt time.Time
		shouldFire  bool
	}{
		{
			name:        "今天已触发",
			triggeredAt: now.Add(-time.Hour), // 今天1小时前
			shouldFire:  false,
		},
		{
			name:        "昨天触发的",
			triggeredAt: now.Add(-25 * time.Hour), // 25小时前（确保跨天）
			shouldFire:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			alert := Alert{
				Frequency:   TriggerDaily,
				TriggeredAt: tc.triggeredAt,
			}

			result := canTriggerInCurrentPeriod(alert)
			if result != tc.shouldFire {
				t.Errorf("期望 %v, 得到 %v", tc.shouldFire, result)
			}
		})
	}
}

// TestCanTriggerInCurrentPeriod_Weekly 测试每周一次告警
func TestCanTriggerInCurrentPeriod_Weekly(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name        string
		triggeredAt time.Time
		shouldFire  bool
	}{
		{
			name:        "本周已触发",
			triggeredAt: now.Add(-time.Hour), // 今天
			shouldFire:  false,
		},
		{
			name:        "上周触发的",
			triggeredAt: now.Add(-8 * 24 * time.Hour), // 8天前（确保跨周）
			shouldFire:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			alert := Alert{
				Frequency:   TriggerWeekly,
				TriggeredAt: tc.triggeredAt,
			}

			result := canTriggerInCurrentPeriod(alert)
			if result != tc.shouldFire {
				t.Errorf("期望 %v, 得到 %v", tc.shouldFire, result)
			}
		})
	}
}

// TestCanTriggerInCurrentPeriod_Monthly 测试每月一次告警
func TestCanTriggerInCurrentPeriod_Monthly(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name        string
		triggeredAt time.Time
		shouldFire  bool
	}{
		{
			name:        "本月已触发",
			triggeredAt: now.Add(-time.Hour), // 今天
			shouldFire:  false,
		},
		{
			name:        "上月触发的",
			triggeredAt: now.Add(-35 * 24 * time.Hour), // 35天前（确保跨月）
			shouldFire:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			alert := Alert{
				Frequency:   TriggerMonthly,
				TriggeredAt: tc.triggeredAt,
			}

			result := canTriggerInCurrentPeriod(alert)
			if result != tc.shouldFire {
				t.Errorf("期望 %v, 得到 %v", tc.shouldFire, result)
			}
		})
	}
}

// TestCanTriggerInCurrentPeriod_EveryNDays 测试每N天一次告警
func TestCanTriggerInCurrentPeriod_EveryNDays(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name          string
		triggeredAt   time.Time
		frequencyDays int
		shouldFire    bool
	}{
		{
			name:          "3天间隔-2天前触发",
			triggeredAt:   now.Add(-2 * 24 * time.Hour),
			frequencyDays: 3,
			shouldFire:    false,
		},
		{
			name:          "3天间隔-4天前触发",
			triggeredAt:   now.Add(-4 * 24 * time.Hour),
			frequencyDays: 3,
			shouldFire:    true,
		},
		{
			name:          "无效天数配置",
			triggeredAt:   now.Add(-10 * 24 * time.Hour),
			frequencyDays: 0, // 无效
			shouldFire:    false,
		},
		{
			name:          "刚好满足间隔",
			triggeredAt:   now.Add(-5 * 24 * time.Hour),
			frequencyDays: 5,
			shouldFire:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			alert := Alert{
				Frequency:     TriggerEveryNDays,
				FrequencyDays: tc.frequencyDays,
				TriggeredAt:   tc.triggeredAt,
			}

			result := canTriggerInCurrentPeriod(alert)
			if result != tc.shouldFire {
				t.Errorf("期望 %v, 得到 %v", tc.shouldFire, result)
			}
		})
	}
}

// TestMigrateAlertFrequency 测试数据迁移函数
func TestMigrateAlertFrequency(t *testing.T) {
	// 使用 types.Alert 进行测试（internal/data 使用的类型）
	typeAlerts := []types.Alert{
		{
			StockCode: "SH600000",
			Frequency: "", // 旧数据，未设置频率
		},
		{
			StockCode: "SZ000001",
			Frequency: types.TriggerDaily, // 已设置频率
		},
		{
			StockCode:     "SH601138",
			Frequency:     types.TriggerEveryNDays,
			FrequencyDays: 0, // 无效天数
		},
	}

	// 使用 internal/data 包的迁移函数
	migrated := data.MigrateAlertFrequency(typeAlerts)

	// 检查空频率迁移为 TriggerOnce
	if migrated[0].Frequency != types.TriggerOnce {
		t.Errorf("空频率应迁移为 TriggerOnce，得到 %s", migrated[0].Frequency)
	}

	// 检查已设置频率不变
	if migrated[1].Frequency != types.TriggerDaily {
		t.Errorf("已设置频率不应改变，得到 %s", migrated[1].Frequency)
	}

	// 检查无效天数被修正
	if migrated[2].FrequencyDays != 1 {
		t.Errorf("无效天数应被修正为1，得到 %d", migrated[2].FrequencyDays)
	}
}

// TestGetNextTriggerTime 测试下次触发时间计算
func TestGetNextTriggerTime(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name        string
		alert       Alert
		expectZero  bool
		description string
	}{
		{
			name: "从未触发",
			alert: Alert{
				Frequency:   TriggerDaily,
				TriggeredAt: time.Time{},
			},
			expectZero:  false,
			description: "从未触发应返回当前时间",
		},
		{
			name: "一次性已触发",
			alert: Alert{
				Frequency:   TriggerOnce,
				TriggeredAt: now.Add(-time.Hour),
			},
			expectZero:  true,
			description: "一次性告警已触发应返回零值",
		},
		{
			name: "每日触发",
			alert: Alert{
				Frequency:   TriggerDaily,
				TriggeredAt: now.Add(-time.Hour),
			},
			expectZero:  false,
			description: "每日触发应返回明天开始时间",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getNextTriggerTime(tc.alert)
			isZero := result.IsZero()
			if isZero != tc.expectZero {
				t.Errorf("%s: 期望零值=%v, 得到零值=%v", tc.description, tc.expectZero, isZero)
			}
		})
	}
}

// TestGetFrequencyOptions 测试频率选项列表
func TestGetFrequencyOptions(t *testing.T) {
	options := getFrequencyOptions()

	// 应该有5个选项
	if len(options) != 5 {
		t.Errorf("期望5个频率选项，得到 %d", len(options))
	}

	// 检查所有预期选项都存在
	expected := map[TriggerFrequency]bool{
		TriggerOnce:       false,
		TriggerDaily:      false,
		TriggerWeekly:     false,
		TriggerMonthly:    false,
		TriggerEveryNDays: false,
	}

	for _, opt := range options {
		expected[opt] = true
	}

	for freq, found := range expected {
		if !found {
			t.Errorf("缺少频率选项: %s", freq)
		}
	}
}

// TestIsSameDay 测试同一天判断
func TestIsSameDay(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		time1    time.Time
		time2    time.Time
		expected bool
	}{
		{
			name:     "同一天不同时间",
			time1:    now,
			time2:    now.Add(time.Hour),
			expected: true,
		},
		{
			name:     "不同天",
			time1:    now,
			time2:    now.Add(25 * time.Hour),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			from1 := carbon.CreateFromStdTime(tc.time1)
			from2 := carbon.CreateFromStdTime(tc.time2)

			result := internalAlert.IsSameDay(from1, from2)
			if result != tc.expected {
				t.Errorf("期望 %v, 得到 %v", tc.expected, result)
			}
		})
	}
}

// TestIsSameWeek 测试同一周判断
func TestIsSameWeek(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		time1    time.Time
		time2    time.Time
		expected bool
	}{
		{
			name:     "同一周内",
			time1:    now,
			time2:    now.Add(time.Hour),
			expected: true,
		},
		{
			name:     "不同周",
			time1:    now,
			time2:    now.Add(8 * 24 * time.Hour), // 8天后
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			from1 := carbon.CreateFromStdTime(tc.time1)
			from2 := carbon.CreateFromStdTime(tc.time2)

			result := internalAlert.IsSameWeek(from1, from2)
			if result != tc.expected {
				t.Errorf("期望 %v, 得到 %v", tc.expected, result)
			}
		})
	}
}

// TestIsSameMonth 测试同一月判断
func TestIsSameMonth(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		time1    time.Time
		time2    time.Time
		expected bool
	}{
		{
			name:     "同一月内",
			time1:    now,
			time2:    now.Add(time.Hour),
			expected: true,
		},
		{
			name:     "不同月",
			time1:    now,
			time2:    now.Add(35 * 24 * time.Hour), // 35天后
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			from1 := carbon.CreateFromStdTime(tc.time1)
			from2 := carbon.CreateFromStdTime(tc.time2)

			result := internalAlert.IsSameMonth(from1, from2)
			if result != tc.expected {
				t.Errorf("期望 %v, 得到 %v", tc.expected, result)
			}
		})
	}
}
