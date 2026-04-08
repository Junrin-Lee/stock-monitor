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
// 如果数据因损坏/不可读而为空，跳过保存以防止覆盖原始数据
func (m *Model) savePortfolio() {
	if m.portfolioCorrupted && len(m.portfolio.Stocks) == 0 {
		logErrorDirect("Skipping portfolio save: data was corrupted on load and is still empty")
		return
	}
	portfolio := types.Portfolio{
		Stocks: convertStocksToTypes(m.portfolio.Stocks),
	}
	if err := data.SavePortfolio(portfolio); err != nil {
		logErrorDirect("Failed to save portfolio: %v", err)
		return
	}
	m.portfolioCorrupted = false // 成功保存后清除损坏标记
}

// loadPortfolio 从文件加载持仓数据（使用 internal/data）
// 返回加载的数据和是否发生了数据损坏
func loadPortfolio() (Portfolio, bool) {
	portfolio, err := data.LoadPortfolio()
	corrupted := err != nil
	if corrupted {
		logErrorDirect("Portfolio load error: %v", err)
	}
	return Portfolio{
		Stocks: convertStocksFromTypes(portfolio.Stocks),
	}, corrupted
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
// 返回加载的数据和是否发生了数据损坏
func loadWatchlist() (Watchlist, bool) {
	watchlist, err := data.LoadWatchlist(api.GetMarketType, isMarketTag)
	corrupted := err != nil
	if corrupted {
		logErrorDirect("Watchlist load error: %v", err)
	}
	return Watchlist{
		Stocks: convertWatchlistStocksFromTypes(watchlist.Stocks),
	}, corrupted
}

// saveWatchlist 保存自选股票列表（使用 internal/data）
// 如果数据因损坏/不可读而为空，跳过保存以防止覆盖原始数据
func (m *Model) saveWatchlist() {
	if m.watchlistCorrupted && len(m.watchlist.Stocks) == 0 {
		logErrorDirect("Skipping watchlist save: data was corrupted on load and is still empty")
		return
	}
	watchlist := types.Watchlist{
		Stocks: convertWatchlistStocksToTypes(m.watchlist.Stocks),
	}
	if err := data.SaveWatchlist(watchlist); err != nil {
		logErrorDirect("Failed to save watchlist: %v", err)
		return
	}
	m.watchlistCorrupted = false // 成功保存后清除损坏标记
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
// 返回加载的数据和是否发生了数据损坏
func loadAlertData() (AlertData, bool) {
	alertData, err := data.LoadAlertData()
	corrupted := err != nil
	if corrupted {
		logErrorDirect("Alert data load error: %v", err)
	}
	return AlertData{
		Alerts:     convertAlertsFromTypes(alertData.Alerts),
		LastCheck:  alertData.LastCheck,
		AlertCount: alertData.AlertCount,
	}, corrupted
}

// saveAlertData 保存告警数据到文件（使用 internal/data）
// 如果数据因损坏/不可读而为空，跳过保存以防止覆盖原始数据
func (m *Model) saveAlertData() {
	if m.alertDataCorrupted && len(m.alertData.Alerts) == 0 {
		logErrorDirect("Skipping alert data save: data was corrupted on load and is still empty")
		return
	}
	alertData := types.AlertData{
		Alerts:     convertAlertsToTypes(m.alertData.Alerts),
		LastCheck:  m.alertData.LastCheck,
		AlertCount: m.alertData.AlertCount,
	}
	if err := data.SaveAlertData(alertData); err != nil {
		logError("log.alert.saveFailed", err)
		return
	}
	m.alertDataCorrupted = false // 成功保存后清除损坏标记
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
