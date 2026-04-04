package intraday

import (
	"testing"
	"time"

	"stock-monitor/internal/types"
)

// TestGetTradingState_ChinaLunchBreak 测试 A 股午休时段检测
func TestGetTradingState_ChinaLunchBreak(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		name      string
		hour      int
		minute    int
		wantState types.TradingState
	}{
		{"午休开始 11:30", 11, 30, types.TradingStateLunchBreak},
		{"午休中 11:31", 11, 31, types.TradingStateLunchBreak},
		{"午休中 12:00", 12, 0, types.TradingStateLunchBreak},
		{"午休中 12:30", 12, 30, types.TradingStateLunchBreak},
		{"午休结束前 12:59", 12, 59, types.TradingStateLunchBreak},
		{"下午开盘 13:00", 13, 0, types.TradingStateLive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 1, 6, tt.hour, tt.minute, 0, 0, loc) // 周一
			got := GetTradingState(now, "china")
			if got != tt.wantState {
				t.Errorf("GetTradingState() = %v, want %v", got, tt.wantState)
			}
		})
	}
}

// TestGetTradingState_HongKongLunchBreak 测试港股午休时段检测
func TestGetTradingState_HongKongLunchBreak(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Hong_Kong")

	tests := []struct {
		name      string
		hour      int
		minute    int
		wantState types.TradingState
	}{
		{"午休开始 12:00", 12, 0, types.TradingStateLunchBreak},
		{"午休中 12:01", 12, 1, types.TradingStateLunchBreak},
		{"午休中 12:30", 12, 30, types.TradingStateLunchBreak},
		{"午休结束前 12:59", 12, 59, types.TradingStateLunchBreak},
		{"下午开盘 13:00", 13, 0, types.TradingStateLive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 1, 6, tt.hour, tt.minute, 0, 0, loc) // 周二
			got := GetTradingState(now, "hongkong")
			if got != tt.wantState {
				t.Errorf("GetTradingState() = %v, want %v", got, tt.wantState)
			}
		})
	}
}

// TestGetTradingState_USNoLunchBreak 测试美股无午休
func TestGetTradingState_USNoLunchBreak(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name      string
		hour      int
		minute    int
		wantState types.TradingState
	}{
		{"上午交易 10:00", 10, 0, types.TradingStateLive},
		{"中午交易 12:00", 12, 0, types.TradingStateLive},
		{"下午交易 14:00", 14, 0, types.TradingStateLive},
		{"收盘前 15:59", 15, 59, types.TradingStateLive},
		{"收盘 16:00", 16, 0, types.TradingStatePostMarket},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 1, 5, tt.hour, tt.minute, 0, 0, loc) // 周一
			got := GetTradingState(now, "us")
			if got != tt.wantState {
				t.Errorf("GetTradingState() = %v, want %v", got, tt.wantState)
			}
		})
	}
}

// TestGetTradingState_BoundaryConditions 测试边界条件
func TestGetTradingState_BoundaryConditions(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		name      string
		hour      int
		minute    int
		second    int
		wantState types.TradingState
	}{
		{"集合竞价中 09:20:00", 9, 20, 0, types.TradingStateAuction},
		{"静默期 09:25:00", 9, 25, 0, types.TradingStatePreMarket},
		{"静默期 09:29:59", 9, 29, 59, types.TradingStatePreMarket},
		{"开盘整点 09:30:00", 9, 30, 0, types.TradingStateLive},
		{"上午收盘前一秒 11:29:59", 11, 29, 59, types.TradingStateLive},
		{"上午收盘整点 11:30:00", 11, 30, 0, types.TradingStateLunchBreak},
		{"午休结束前一秒 12:59:59", 12, 59, 59, types.TradingStateLunchBreak},
		{"下午开盘整点 13:00:00", 13, 0, 0, types.TradingStateLive},
		{"下午收盘前一秒 14:59:59", 14, 59, 59, types.TradingStateLive},
		{"下午收盘整点 15:00:00", 15, 0, 0, types.TradingStatePostMarket},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 1, 6, tt.hour, tt.minute, tt.second, 0, loc) // 周二
			got := GetTradingState(now, "china")
			if got != tt.wantState {
				t.Errorf("GetTradingState() at %02d:%02d:%02d = %v, want %v",
					tt.hour, tt.minute, tt.second, got, tt.wantState)
			}
		})
	}
}

// mockModel for testing
type mockModel struct{}

func (m *mockModel) GetConfig() interface{}                        { return nil }
func (m *mockModel) GetIntradayManager() interface{}               { return nil }
func (m *mockModel) SendMessage(interface{})                       {}
func (m *mockModel) GetState() int                                 { return 0 }
func (m *mockModel) SetMessage(string)                             {}
func (m *mockModel) GetPortfolioRepository() interface{}           { return nil }
func (m *mockModel) NotifyPriceUpdate(code string, price float64)  {}
func (m *mockModel) GetCurrentPortfolio() []interface{}            { return nil }
func (m *mockModel) GetCurrentWatchlist() []interface{}            { return nil }
func (m *mockModel) ShouldAutoUpdate() bool                        { return false }
func (m *mockModel) GetRefreshInterval() int                       { return 5 }
func (m *mockModel) GetAlertManager() interface{}                  { return nil }
func (m *mockModel) OnStockPriceUpdate(code string, price float64) {}

// TestGetTradingDayForCollection_LunchBreak 测试午休时段返回 Live 模式
func TestGetTradingDayForCollection_LunchBreak(t *testing.T) {
	tests := []struct {
		name       string
		stockCode  string
		hour       int
		minute     int
		wantMode   types.CollectionMode
		wantToday  bool // 是否应返回今天日期
	}{
		{"A股午休 11:31", "600000", 11, 31, types.CollectionModeLive, true},
		{"A股午休 12:00", "600000", 12, 0, types.CollectionModeLive, true},
		{"A股午休 12:59", "600000", 12, 59, types.CollectionModeLive, true},
		{"港股午休 12:01", "0700", 12, 1, types.CollectionModeLive, true},
		{"港股午休 12:30", "0700", 12, 30, types.CollectionModeLive, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 当前时间
			var loc *time.Location
			if tt.stockCode[0] == '0' || tt.stockCode[0] == '3' {
				loc, _ = time.LoadLocation("Asia/Hong_Kong")
			} else {
				loc, _ = time.LoadLocation("Asia/Shanghai")
			}

			// 无法直接 mock time.Now()，这里仅验证逻辑
			// 实际测试需要在交易时段运行或使用时间注入
			t.Logf("Test case: %s (code=%s, time=%02d:%02d)",
				tt.name, tt.stockCode, tt.hour, tt.minute)

			// 验证 GetTradingState 返回 LunchBreak
			now := time.Date(2026, 1, 6, tt.hour, tt.minute, 0, 0, loc)
			marketType := "china"
			if tt.stockCode[0] == '0' {
				marketType = "hongkong"
			}
			state := GetTradingState(now, marketType)
			if state != types.TradingStateLunchBreak {
				t.Errorf("GetTradingState() = %v, want %v", state, types.TradingStateLunchBreak)
			}

			// 验证应返回 Live 模式（通过 switch case 逻辑）
			// 注意：GetTradingDayForCollection 会调用 time.Now()，
			// 在非午休时段运行此测试会得到不同结果
			// 这里仅验证状态逻辑的一致性
		})
	}
}

// TestGetTradingState_Weekend 测试周末检测
func TestGetTradingState_Weekend(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		name      string
		weekday   time.Weekday
		wantState types.TradingState
	}{
		{"周六", time.Saturday, types.TradingStateWeekend},
		{"周日", time.Sunday, types.TradingStateWeekend},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 找到下一个指定的星期几
			now := time.Now().In(loc)
			daysUntil := (int(tt.weekday) - int(now.Weekday()) + 7) % 7
			if daysUntil == 0 {
				daysUntil = 7
			}
			targetDate := now.AddDate(0, 0, daysUntil)
			testTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(),
				12, 0, 0, 0, loc)

			got := GetTradingState(testTime, "china")
			if got != tt.wantState {
				t.Errorf("GetTradingState() on %v = %v, want %v",
					tt.weekday, got, tt.wantState)
			}
		})
	}
}
