package market

import (
	"testing"
	"time"

	"stock-monitor/internal/testutil"
	"stock-monitor/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTimeInMarket 测试市场时区时间解析
func TestParseTimeInMarket(t *testing.T) {
	chinaConfig := types.MarketConfig{
		Timezone: "Asia/Shanghai",
	}

	usConfig := types.MarketConfig{
		Timezone: "America/New_York",
	}

	tests := []struct {
		name         string
		date         string
		timeStr      string
		marketConfig types.MarketConfig
		wantErr      bool
		validate     func(*testing.T, time.Time)
		desc         string
	}{
		{
			name:         "有效中国时间",
			date:         "20260120",
			timeStr:      "09:30",
			marketConfig: chinaConfig,
			wantErr:      false,
			validate: func(t *testing.T, result time.Time) {
				assert.Equal(t, 2026, result.Year())
				assert.Equal(t, time.January, result.Month())
				assert.Equal(t, 20, result.Day())
				assert.Equal(t, 9, result.Hour())
				assert.Equal(t, 30, result.Minute())
			},
			desc: "中国时区时间解析",
		},
		{
			name:         "有效美国时间",
			date:         "20260120",
			timeStr:      "16:00",
			marketConfig: usConfig,
			wantErr:      false,
			validate: func(t *testing.T, result time.Time) {
				assert.Equal(t, 2026, result.Year())
				assert.Equal(t, 16, result.Hour())
			},
			desc: "美国时区时间解析",
		},
		{
			name:         "无效日期格式-太短",
			date:         "2026120",
			timeStr:      "09:30",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "日期格式必须是8位YYYYMMDD",
		},
		{
			name:         "无效日期格式-太长",
			date:         "202601201",
			timeStr:      "09:30",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "日期格式过长",
		},
		{
			name:         "无效年份",
			date:         "ABCD0120",
			timeStr:      "09:30",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "年份包含非数字字符",
		},
		{
			name:         "无效月份",
			date:         "2026XX20",
			timeStr:      "09:30",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "月份包含非数字字符",
		},
		{
			name:         "无效日期",
			date:         "202601YY",
			timeStr:      "09:30",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "日期包含非数字字符",
		},
		{
			name:         "无效时间格式-缺少冒号",
			date:         "20260120",
			timeStr:      "0930",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "时间格式必须是HH:MM",
		},
		{
			name:         "无效时间格式-冒号过多",
			date:         "20260120",
			timeStr:      "09:30:00",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "时间只能包含一个冒号",
		},
		{
			name:         "无效小时",
			date:         "20260120",
			timeStr:      "XX:30",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "小时包含非数字字符",
		},
		{
			name:         "无效分钟",
			date:         "20260120",
			timeStr:      "09:YY",
			marketConfig: chinaConfig,
			wantErr:      true,
			desc:         "分钟包含非数字字符",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTimeInMarket(tt.date, tt.timeStr, tt.marketConfig)

			if tt.wantErr {
				assert.Error(t, err, tt.desc)
			} else {
				require.NoError(t, err, tt.desc)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

// TestIsMarketOpenForConfig 测试市场开市检查
func TestIsMarketOpenForConfig(t *testing.T) {
	chinaConfig := types.MarketConfig{
		Timezone: "Asia/Shanghai",
		TradingSessions: []types.TradingSession{
			{StartTime: "09:30", EndTime: "11:30"},
			{StartTime: "13:00", EndTime: "15:00"},
		},
		Weekdays: []int{1, 2, 3, 4, 5}, // 周一到周五
	}

	usConfig := types.MarketConfig{
		Timezone: "America/New_York",
		TradingSessions: []types.TradingSession{
			{StartTime: "09:30", EndTime: "16:00"},
		},
		Weekdays: []int{1, 2, 3, 4, 5},
	}

	tests := []struct {
		name         string
		checkTime    time.Time
		marketConfig types.MarketConfig
		expected     bool
		desc         string
	}{
		// 中国市场测试
		{
			name:         "中国-早盘开始",
			checkTime:    testutil.MustParseTime("2026-01-20 09:30:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     true,
			desc:         "09:30应该开市",
		},
		{
			name:         "中国-早盘进行中",
			checkTime:    testutil.MustParseTime("2026-01-20 10:30:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     true,
			desc:         "10:30应该开市",
		},
		{
			name:         "中国-早盘结束",
			checkTime:    testutil.MustParseTime("2026-01-20 11:30:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     true,
			desc:         "11:30应该开市",
		},
		{
			name:         "中国-午休时间",
			checkTime:    testutil.MustParseTime("2026-01-20 12:00:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     false,
			desc:         "12:00午休应该休市",
		},
		{
			name:         "中国-下午盘开始",
			checkTime:    testutil.MustParseTime("2026-01-20 13:00:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     true,
			desc:         "13:00应该开市",
		},
		{
			name:         "中国-下午盘进行中",
			checkTime:    testutil.MustParseTime("2026-01-20 14:30:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     true,
			desc:         "14:30应该开市",
		},
		{
			name:         "中国-收盘时间",
			checkTime:    testutil.MustParseTime("2026-01-20 15:00:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     true,
			desc:         "15:00应该开市",
		},
		{
			name:         "中国-收盘后",
			checkTime:    testutil.MustParseTime("2026-01-20 15:01:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     false,
			desc:         "15:01收盘后应该休市",
		},
		{
			name:         "中国-盘前",
			checkTime:    testutil.MustParseTime("2026-01-20 09:00:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     false,
			desc:         "09:00盘前应该休市",
		},
		// 周末测试
		{
			name:         "中国-周六",
			checkTime:    testutil.MustParseTime("2026-01-24 10:00:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     false,
			desc:         "周六应该休市",
		},
		{
			name:         "中国-周日",
			checkTime:    testutil.MustParseTime("2026-01-25 10:00:00", "Asia/Shanghai"),
			marketConfig: chinaConfig,
			expected:     false,
			desc:         "周日应该休市",
		},
		// 美国市场测试
		{
			name:         "美国-交易中",
			checkTime:    testutil.MustParseTime("2026-01-20 10:00:00", "America/New_York"),
			marketConfig: usConfig,
			expected:     true,
			desc:         "美国市场10:00应该开市",
		},
		{
			name:         "美国-开盘时间",
			checkTime:    testutil.MustParseTime("2026-01-20 09:30:00", "America/New_York"),
			marketConfig: usConfig,
			expected:     true,
			desc:         "美国市场09:30应该开市",
		},
		{
			name:         "美国-收盘时间",
			checkTime:    testutil.MustParseTime("2026-01-20 16:00:00", "America/New_York"),
			marketConfig: usConfig,
			expected:     true,
			desc:         "美国市场16:00应该开市",
		},
		{
			name:         "美国-收盘后",
			checkTime:    testutil.MustParseTime("2026-01-20 16:01:00", "America/New_York"),
			marketConfig: usConfig,
			expected:     false,
			desc:         "美国市场16:01应该休市",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMarketOpenForConfig(tt.checkTime, tt.marketConfig)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

// TestGetMarketLocation 测试市场时区获取
func TestGetMarketLocation(t *testing.T) {
	tests := []struct {
		name       string
		marketType types.MarketType
		expectedTZ string
		wantErr    bool
		desc       string
	}{
		{
			name:       "中国市场",
			marketType: types.MarketChina,
			expectedTZ: "Asia/Shanghai",
			wantErr:    false,
			desc:       "中国市场应返回上海时区",
		},
		{
			name:       "美国市场",
			marketType: types.MarketUS,
			expectedTZ: "America/New_York",
			wantErr:    false,
			desc:       "美国市场应返回纽约时区",
		},
		{
			name:       "香港市场",
			marketType: types.MarketHongKong,
			expectedTZ: "Asia/Hong_Kong",
			wantErr:    false,
			desc:       "香港市场应返回香港时区",
		},
		{
			name:       "未知市场",
			marketType: types.MarketType("unknown"),
			expectedTZ: "",
			wantErr:    true,
			desc:       "未知市场应返回错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location, err := GetMarketLocation(tt.marketType)

			if tt.wantErr {
				assert.Error(t, err, tt.desc)
				assert.Nil(t, location)
			} else {
				require.NoError(t, err, tt.desc)
				assert.NotNil(t, location)
				assert.Equal(t, tt.expectedTZ, location.String(), tt.desc)
			}
		})
	}
}

// TestGetCurrentDateForMarket 测试获取市场当前日期
func TestGetCurrentDateForMarket(t *testing.T) {
	config := testutil.NewTestConfig()
	config.Markets.China.Timezone = "Asia/Shanghai"
	config.Markets.US.Timezone = "America/New_York"
	config.Markets.HongKong.Timezone = "Asia/Hong_Kong"

	tests := []struct {
		name       string
		marketType types.MarketType
		validate   func(*testing.T, string)
		desc       string
	}{
		{
			name:       "中国市场",
			marketType: types.MarketChina,
			validate: func(t *testing.T, result string) {
				assert.Len(t, result, 8, "日期应为8位YYYYMMDD")
				assert.Regexp(t, `^\d{8}$`, result, "日期应全为数字")
			},
			desc: "中国市场日期格式验证",
		},
		{
			name:       "美国市场",
			marketType: types.MarketUS,
			validate: func(t *testing.T, result string) {
				assert.Len(t, result, 8, "日期应为8位YYYYMMDD")
				assert.Regexp(t, `^\d{8}$`, result, "日期应全为数字")
			},
			desc: "美国市场日期格式验证",
		},
		{
			name:       "香港市场",
			marketType: types.MarketHongKong,
			validate: func(t *testing.T, result string) {
				assert.Len(t, result, 8, "日期应为8位YYYYMMDD")
				assert.Regexp(t, `^\d{8}$`, result, "日期应全为数字")
			},
			desc: "香港市场日期格式验证",
		},
		{
			name:       "未知市场-使用默认",
			marketType: types.MarketType("unknown"),
			validate: func(t *testing.T, result string) {
				assert.Len(t, result, 8, "未知市场应返回8位日期")
			},
			desc: "未知市场使用本地时区",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCurrentDateForMarket(tt.marketType, config)
			tt.validate(t, result)
		})
	}
}

// TestGetMarketConfigForType 测试根据类型获取市场配置
func TestGetMarketConfigForType(t *testing.T) {
	config := testutil.NewTestConfig()
	config.Markets.China.Timezone = "Asia/Shanghai"
	config.Markets.US.Timezone = "America/New_York"
	config.Markets.HongKong.Timezone = "Asia/Hong_Kong"

	tests := []struct {
		name       string
		marketType types.MarketType
		expectedTZ string
		desc       string
	}{
		{
			name:       "中国市场配置",
			marketType: types.MarketChina,
			expectedTZ: "Asia/Shanghai",
			desc:       "应返回中国市场配置",
		},
		{
			name:       "美国市场配置",
			marketType: types.MarketUS,
			expectedTZ: "America/New_York",
			desc:       "应返回美国市场配置",
		},
		{
			name:       "香港市场配置",
			marketType: types.MarketHongKong,
			expectedTZ: "Asia/Hong_Kong",
			desc:       "应返回香港市场配置",
		},
		{
			name:       "未知市场-默认中国",
			marketType: types.MarketType("unknown"),
			expectedTZ: "Asia/Shanghai",
			desc:       "未知市场应返回中国配置",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMarketConfigForType(tt.marketType, config)
			assert.Equal(t, tt.expectedTZ, result.Timezone, tt.desc)
		})
	}
}

// TestGetTimezoneForMarket 测试获取市场时区字符串
func TestGetTimezoneForMarket(t *testing.T) {
	tests := []struct {
		name       string
		marketType types.MarketType
		expected   string
	}{
		{
			name:       "中国市场",
			marketType: types.MarketChina,
			expected:   "Asia/Shanghai",
		},
		{
			name:       "美国市场",
			marketType: types.MarketUS,
			expected:   "America/New_York",
		},
		{
			name:       "香港市场",
			marketType: types.MarketHongKong,
			expected:   "Asia/Hong_Kong",
		},
		{
			name:       "未知市场-默认上海",
			marketType: types.MarketType("unknown"),
			expected:   "Asia/Shanghai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTimezoneForMarket(tt.marketType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsMarketOpenForConfig_EdgeCases 测试边界情况
func TestIsMarketOpenForConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		marketConfig types.MarketConfig
		checkTime    time.Time
		expected     bool
		desc         string
	}{
		{
			name: "无效时区-应返回false",
			marketConfig: types.MarketConfig{
				Timezone: "Invalid/Timezone",
				TradingSessions: []types.TradingSession{
					{StartTime: "09:30", EndTime: "16:00"},
				},
				Weekdays: []int{1, 2, 3, 4, 5},
			},
			checkTime: testutil.MustParseTime("2026-01-20 10:00:00", "UTC"),
			expected:  false,
			desc:      "无效时区应返回false",
		},
		{
			name: "无交易时段",
			marketConfig: types.MarketConfig{
				Timezone:        "Asia/Shanghai",
				TradingSessions: []types.TradingSession{},
				Weekdays:        []int{1, 2, 3, 4, 5},
			},
			checkTime: testutil.MustParseTime("2026-01-20 10:00:00", "Asia/Shanghai"),
			expected:  false,
			desc:      "无交易时段应返回false",
		},
		{
			name: "无工作日配置",
			marketConfig: types.MarketConfig{
				Timezone: "Asia/Shanghai",
				TradingSessions: []types.TradingSession{
					{StartTime: "09:30", EndTime: "16:00"},
				},
				Weekdays: []int{},
			},
			checkTime: testutil.MustParseTime("2026-01-20 10:00:00", "Asia/Shanghai"),
			expected:  false,
			desc:      "无工作日配置应返回false",
		},
		{
			name: "交易时段格式错误",
			marketConfig: types.MarketConfig{
				Timezone: "Asia/Shanghai",
				TradingSessions: []types.TradingSession{
					{StartTime: "invalid", EndTime: "invalid"},
				},
				Weekdays: []int{1, 2, 3, 4, 5},
			},
			checkTime: testutil.MustParseTime("2026-01-20 10:00:00", "Asia/Shanghai"),
			expected:  false,
			desc:      "交易时段格式错误应返回false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMarketOpenForConfig(tt.checkTime, tt.marketConfig)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

// BenchmarkIsMarketOpenForConfig 性能基准测试
func BenchmarkIsMarketOpenForConfig(b *testing.B) {
	config := types.MarketConfig{
		Timezone: "Asia/Shanghai",
		TradingSessions: []types.TradingSession{
			{StartTime: "09:30", EndTime: "11:30"},
			{StartTime: "13:00", EndTime: "15:00"},
		},
		Weekdays: []int{1, 2, 3, 4, 5},
	}

	checkTime := testutil.MustParseTime("2026-01-20 10:00:00", "Asia/Shanghai")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsMarketOpenForConfig(checkTime, config)
	}
}

// BenchmarkGetMarketLocation 性能基准测试
func BenchmarkGetMarketLocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetMarketLocation(types.MarketChina)
	}
}

// BenchmarkParseTimeInMarket 性能基准测试
func BenchmarkParseTimeInMarket(b *testing.B) {
	config := types.MarketConfig{
		Timezone: "Asia/Shanghai",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseTimeInMarket("20260120", "09:30", config)
	}
}
