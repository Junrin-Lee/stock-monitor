package main

import (
	"stock-monitor/internal/api"
	"stock-monitor/internal/data"
	"stock-monitor/internal/types"
)

// ============================================================================
// Portfolio 持仓数据持久化（适配器方法）
// ============================================================================

// savePortfolio 保存持仓数据到文件（使用 internal/data）
func (m *Model) savePortfolio() {
	portfolio := types.Portfolio{
		Stocks: convertStocksToTypes(m.portfolio.Stocks),
	}
	if err := data.SavePortfolio(portfolio); err != nil {
		logErrorDirect("Failed to save portfolio: %v", err)
	}
}

// loadPortfolio 从文件加载持仓数据（使用 internal/data）
func loadPortfolio() Portfolio {
	portfolio := data.LoadPortfolio()
	return Portfolio{
		Stocks: convertStocksFromTypes(portfolio.Stocks),
	}
}

// convertStocksToTypes 将 main.Stock 转换为 types.Stock
func convertStocksToTypes(stocks []Stock) []types.Stock {
	result := make([]types.Stock, len(stocks))
	for i, s := range stocks {
		result[i] = types.Stock{
			Code:          s.Code,
			Name:          s.Name,
			Price:         s.Price,
			CostPrice:     s.CostPrice,
			Quantity:      s.Quantity,
			Change:        s.Change,
			ChangePercent: s.ChangePercent,
			StartPrice:    s.StartPrice,
			MaxPrice:      s.MaxPrice,
			MinPrice:      s.MinPrice,
			PrevClose:     s.PrevClose,
		}
	}
	return result
}

// convertStocksFromTypes 将 types.Stock 转换为 main.Stock
func convertStocksFromTypes(stocks []types.Stock) []Stock {
	result := make([]Stock, len(stocks))
	for i, s := range stocks {
		result[i] = Stock{
			Code:          s.Code,
			Name:          s.Name,
			Price:         s.Price,
			CostPrice:     s.CostPrice,
			Quantity:      s.Quantity,
			Change:        s.Change,
			ChangePercent: s.ChangePercent,
			StartPrice:    s.StartPrice,
			MaxPrice:      s.MaxPrice,
			MinPrice:      s.MinPrice,
			PrevClose:     s.PrevClose,
		}
	}
	return result
}

// ============================================================================
// Watchlist 自选股数据持久化（适配器方法）
// ============================================================================

// loadWatchlist 加载自选股票列表（使用 internal/data）
func loadWatchlist() Watchlist {
	watchlist := data.LoadWatchlist(api.GetMarketType, isMarketTag)
	return Watchlist{
		Stocks: convertWatchlistStocksFromTypes(watchlist.Stocks),
	}
}

// saveWatchlist 保存自选股票列表（使用 internal/data）
func (m *Model) saveWatchlist() {
	watchlist := types.Watchlist{
		Stocks: convertWatchlistStocksToTypes(m.watchlist.Stocks),
	}
	if err := data.SaveWatchlist(watchlist); err != nil {
		logErrorDirect("Failed to save watchlist: %v", err)
	}
}

// convertWatchlistStocksToTypes 将 main.WatchlistStock 转换为 types.WatchlistStock
func convertWatchlistStocksToTypes(stocks []WatchlistStock) []types.WatchlistStock {
	result := make([]types.WatchlistStock, len(stocks))
	for i, s := range stocks {
		result[i] = types.WatchlistStock{
			Code:   s.Code,
			Name:   s.Name,
			Tags:   s.Tags,
			Market: s.Market,
		}
	}
	return result
}

// convertWatchlistStocksFromTypes 将 types.WatchlistStock 转换为 main.WatchlistStock
func convertWatchlistStocksFromTypes(stocks []types.WatchlistStock) []WatchlistStock {
	result := make([]WatchlistStock, len(stocks))
	for i, s := range stocks {
		result[i] = WatchlistStock{
			Code:   s.Code,
			Name:   s.Name,
			Tags:   s.Tags,
			Market: s.Market,
		}
	}
	return result
}

// ============================================================================
// Config 配置文件持久化（适配器方法）
// ============================================================================

// loadConfig 加载配置文件（使用 internal/data）
func loadConfig() Config {
	// 直接使用 internal/data 的类型，因为 Config 已经是类型别名
	return data.LoadConfig()
}

// saveConfig 保存配置文件（使用 internal/data）
func saveConfig(config Config) error {
	return data.SaveConfig(config)
}

// ============================================================================
// Alert 告警数据持久化（适配器方法）
// ============================================================================

// loadAlertData 从文件加载告警数据（使用 internal/data）
func loadAlertData() AlertData {
	alertData := data.LoadAlertData()
	return AlertData{
		Alerts:     convertAlertsFromTypes(alertData.Alerts),
		LastCheck:  alertData.LastCheck,
		AlertCount: alertData.AlertCount,
	}
}

// saveAlertData 保存告警数据到文件（使用 internal/data）
func (m *Model) saveAlertData() {
	alertData := types.AlertData{
		Alerts:     convertAlertsToTypes(m.alertData.Alerts),
		LastCheck:  m.alertData.LastCheck,
		AlertCount: m.alertData.AlertCount,
	}
	if err := data.SaveAlertData(alertData); err != nil {
		logError("log.alert.saveFailed", err)
	}
}

// convertAlertsToTypes 将 main.Alert 转换为 types.Alert
func convertAlertsToTypes(alerts []Alert) []types.Alert {
	result := make([]types.Alert, len(alerts))
	for i, a := range alerts {
		result[i] = types.Alert{
			ID:            a.ID,
			StockCode:     a.StockCode,
			StockName:     a.StockName,
			Type:          a.Type,
			Condition:     a.Condition,
			Threshold:     a.Threshold,
			Frequency:     a.Frequency,
			FrequencyDays: a.FrequencyDays,
			IsActive:      a.IsActive,
			CreatedAt:     a.CreatedAt,
			UpdatedAt:     a.UpdatedAt,
			TriggeredAt:   a.TriggeredAt,
			LastChecked:   a.LastChecked,
			BatchTag:      a.BatchTag,
		}
	}
	return result
}

// convertAlertsFromTypes 将 types.Alert 转换为 main.Alert
func convertAlertsFromTypes(alerts []types.Alert) []Alert {
	result := make([]Alert, len(alerts))
	for i, a := range alerts {
		result[i] = Alert{
			ID:            a.ID,
			StockCode:     a.StockCode,
			StockName:     a.StockName,
			Type:          a.Type,
			Condition:     a.Condition,
			Threshold:     a.Threshold,
			Frequency:     a.Frequency,
			FrequencyDays: a.FrequencyDays,
			IsActive:      a.IsActive,
			CreatedAt:     a.CreatedAt,
			UpdatedAt:     a.UpdatedAt,
			TriggeredAt:   a.TriggeredAt,
			LastChecked:   a.LastChecked,
			BatchTag:      a.BatchTag,
		}
	}
	return result
}
