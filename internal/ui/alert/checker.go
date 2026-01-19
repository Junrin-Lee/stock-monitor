package alert

import (
	"stock-monitor/internal/types"
)

// CheckAlerts checks all alert conditions and returns triggered alerts
func CheckAlerts(alerts []types.Alert, priceCache map[string]*types.StockPriceCacheEntry, canTrigger func(types.Alert) bool) []types.Alert {
	triggeredAlerts := []types.Alert{}

	for i, alert := range alerts {
		if !alert.IsActive {
			continue
		}

		// Check if it's time to trigger
		if !canTrigger(alert) {
			continue
		}

		// Get current stock price from cache
		cacheEntry, exists := priceCache[alert.StockCode]
		if !exists || cacheEntry.Data == nil {
			continue
		}

		stockData := cacheEntry.Data

		// Check alert condition
		isTriggered := false
		switch alert.Type {
		case types.AlertTypePrice:
			isTriggered = CheckPriceAlert(stockData, alert)
		case types.AlertTypeRate:
			isTriggered = CheckRateAlert(stockData, alert)
		case types.AlertTypeVolume:
			isTriggered = CheckVolumeAlert(stockData, alert)
		}

		if isTriggered {
			triggeredAlerts = append(triggeredAlerts, alerts[i])
		}
	}

	return triggeredAlerts
}

// CheckPriceAlert checks price-based alert conditions
func CheckPriceAlert(stockData *types.StockData, alert types.Alert) bool {
	return types.CheckNumericCondition(stockData.Price, alert.Threshold, alert.Condition)
}

// CheckRateAlert checks rate-of-change alert conditions
func CheckRateAlert(stockData *types.StockData, alert types.Alert) bool {
	return types.CheckNumericCondition(stockData.ChangePercent, alert.Threshold, alert.Condition)
}

// CheckVolumeAlert checks volume-based alert conditions
func CheckVolumeAlert(stockData *types.StockData, alert types.Alert) bool {
	volumeFloat := float64(stockData.Volume)
	return types.CheckNumericCondition(volumeFloat, alert.Threshold, alert.Condition)
}
