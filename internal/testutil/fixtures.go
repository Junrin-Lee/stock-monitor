package testutil

import (
	"stock-monitor/internal/types"
	"time"
)

// NewTestStock 创建测试用的持仓股票数据
func NewTestStock() types.Stock {
	return types.Stock{
		Code:          "SH601138",
		Name:          "工业富联",
		Price:         10.5,
		CostPrice:     10.0,
		Quantity:      100,
		Change:        0.5,
		ChangePercent: 5.0,
		StartPrice:    10.0,
		MaxPrice:      10.8,
		MinPrice:      9.8,
		PrevClose:     10.0,
	}
}

// NewTestStockData 创建测试用的实时股价数据
func NewTestStockData(symbol, name string, price float64) *types.StockData {
	return &types.StockData{
		Symbol:        symbol,
		Name:          name,
		Price:         price,
		Change:        0.5,
		ChangePercent: 5.0,
		StartPrice:    price - 0.5,
		MaxPrice:      price + 0.3,
		MinPrice:      price - 0.8,
		PrevClose:     price - 0.5,
		TurnoverRate:  2.5,
		Volume:        1000000,
	}
}

// NewTestChinaStock 创建测试用的A股数据
func NewTestChinaStock() types.Stock {
	return types.Stock{
		Code:          "SH601138",
		Name:          "工业富联",
		Price:         10.5,
		CostPrice:     10.0,
		Quantity:      100,
		Change:        0.5,
		ChangePercent: 5.0,
	}
}

// NewTestUSStock 创建测试用的美股数据
func NewTestUSStock() types.Stock {
	return types.Stock{
		Code:          "AAPL",
		Name:          "Apple Inc.",
		Price:         150.0,
		CostPrice:     140.0,
		Quantity:      50,
		Change:        10.0,
		ChangePercent: 7.14,
	}
}

// NewTestHKStock 创建测试用的港股数据
func NewTestHKStock() types.Stock {
	return types.Stock{
		Code:          "HK00700",
		Name:          "腾讯控股",
		Price:         380.0,
		CostPrice:     350.0,
		Quantity:      20,
		Change:        30.0,
		ChangePercent: 8.57,
	}
}

// NewTestWatchlist 创建测试用的自选列表
func NewTestWatchlist() types.Watchlist {
	return types.Watchlist{
		Stocks: []types.WatchlistStock{
			{
				Code:   "SH601138",
				Name:   "工业富联",
				Market: types.MarketChina,
				Tags:   []string{"科技", "AI"},
			},
			{
				Code:   "AAPL",
				Name:   "Apple",
				Market: types.MarketUS,
				Tags:   []string{"科技"},
			},
			{
				Code:   "HK00700",
				Name:   "腾讯控股",
				Market: types.MarketHongKong,
				Tags:   []string{"互联网", "科技"},
			},
		},
	}
}

// NewTestWatchlistStock 创建测试用的自选股票
func NewTestWatchlistStock(code, name string, market types.MarketType, tags []string) types.WatchlistStock {
	return types.WatchlistStock{
		Code:   code,
		Name:   name,
		Market: market,
		Tags:   tags,
	}
}

// NewTestAlert 创建测试用的告警配置
func NewTestAlert(alertType types.AlertType) types.Alert {
	now := time.Now()
	return types.Alert{
		ID:          "test-alert-1",
		StockCode:   "SH601138",
		StockName:   "工业富联",
		Type:        alertType,
		Condition:   ">",
		Threshold:   11.0,
		IsActive:    true,
		Frequency:   types.TriggerDaily,
		CreatedAt:   now,
		LastChecked: now,
	}
}

// NewTestAlertWithFrequency 创建带特定频率的测试告警
func NewTestAlertWithFrequency(frequency types.TriggerFrequency, customDays int) types.Alert {
	alert := NewTestAlert(types.AlertTypePrice)
	alert.Frequency = frequency
	alert.FrequencyDays = customDays
	return alert
}

// NewTestPortfolio 创建测试用的投资组合
func NewTestPortfolio() types.Portfolio {
	return types.Portfolio{
		Stocks: []types.Stock{
			NewTestChinaStock(),
			NewTestUSStock(),
			NewTestHKStock(),
		},
	}
}

// NewTestConfig 创建测试用的配置
func NewTestConfig() types.Config {
	return types.Config{
		System: types.SystemConfig{
			Language:      "zh",
			AutoStart:     true,
			StartupModule: "portfolio",
			LogLevel:      "info",
		},
		Display: types.DisplayConfig{
			ColorScheme:        "professional",
			DecimalPlaces:      3,
			TableStyle:         "light",
			MaxLines:           10,
			PortfolioHighlight: "yellow",
		},
		Update: types.UpdateConfig{
			RefreshInterval: 5,
			AutoUpdate:      true,
		},
	}
}

// MustParseTime 解析时间字符串,用于测试
func MustParseTime(timeStr, timezone string) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, loc)
	if err != nil {
		panic(err)
	}
	return t
}

// TimePtr 创建时间指针,用于测试
func TimePtr(t time.Time) *time.Time {
	return &t
}
