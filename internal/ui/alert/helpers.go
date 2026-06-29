package alert

import (
	"stock-monitor/internal/types"
	"strings"
)

// GetStockAlerts returns all alerts for a specific stock
func GetStockAlerts(alerts []types.Alert, stockCode string) []types.Alert {
	var result []types.Alert
	for _, alert := range alerts {
		if alert.StockCode == stockCode {
			result = append(result, alert)
		}
	}
	return result
}

// GetStocksByTag returns all stock codes with the specified tag
func GetStocksByTag(watchlist types.Watchlist, tag string) []string {
	codes := []string{}
	seen := make(map[string]bool)

	for _, stock := range watchlist.Stocks {
		for _, stockTag := range stock.Tags {
			if stockTag == tag {
				if !seen[stock.Code] {
					codes = append(codes, stock.Code)
					seen[stock.Code] = true
				}
				break
			}
		}
	}

	return codes
}

// GetStocksByMarket returns all stock codes in the specified market
func GetStocksByMarket(watchlist types.Watchlist, portfolio types.Portfolio, marketType types.MarketType) []string {
	codes := []string{}
	seen := make(map[string]bool)

	// From watchlist
	for _, stock := range watchlist.Stocks {
		if stock.Market == marketType {
			if !seen[stock.Code] {
				codes = append(codes, stock.Code)
				seen[stock.Code] = true
			}
		}
	}

	// From portfolio
	for _, stock := range portfolio.Stocks {
		if isMarketType(stock.Code, marketType) {
			if !seen[stock.Code] {
				codes = append(codes, stock.Code)
				seen[stock.Code] = true
			}
		}
	}

	return codes
}

// isMarketType checks if a stock code belongs to a specific market
func isMarketType(stockCode string, marketType types.MarketType) bool {
	switch marketType {
	case types.MarketChina:
		return strings.HasPrefix(stockCode, "SH") || strings.HasPrefix(stockCode, "SZ")
	case types.MarketUS:
		return !strings.HasPrefix(stockCode, "SH") && !strings.HasPrefix(stockCode, "SZ") && !strings.HasPrefix(stockCode, "HK")
	case types.MarketHongKong:
		return strings.HasPrefix(stockCode, "HK")
	default:
		return false
	}
}

// ParseStockCodes parses stock codes from input (supports comma and newline delimiters)
func ParseStockCodes(input string) []string {
	codes := []string{}

	// Split by newline first, then by comma
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ",")
		for _, part := range parts {
			code := strings.TrimSpace(part)
			if code != "" {
				codes = append(codes, code)
			}
		}
	}

	return codes
}

// GetAlertTypeFromCursor converts cursor position to AlertType
func GetAlertTypeFromCursor(cursor int) types.AlertType {
	switch cursor {
	case 0:
		return types.AlertTypePrice
	case 1:
		return types.AlertTypeRate
	case 2:
		return types.AlertTypeVolume
	default:
		return types.AlertTypePrice
	}
}

// GetAlertConditionFromCursor converts cursor position to alert condition
func GetAlertConditionFromCursor(cursor int) string {
	conditions := []string{">", "<", ">=", "<="}
	if cursor >= 0 && cursor < len(conditions) {
		return conditions[cursor]
	}
	return ">"
}

// GetFrequencyOptions returns available frequency trigger options
func GetFrequencyOptions() []types.TriggerFrequency {
	return []types.TriggerFrequency{
		types.TriggerOnce,
		types.TriggerDaily,
		types.TriggerWeekly,
		types.TriggerMonthly,
		types.TriggerEveryNDays,
	}
}
