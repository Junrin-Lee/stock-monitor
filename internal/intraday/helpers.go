package intraday

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"stock-monitor/internal/api"
	"stock-monitor/internal/types"
)

// mergeDatapoints combines existing and new datapoints, deduplicating by time
func mergeDatapoints(existing, new []IntradayDataPoint) []IntradayDataPoint {
	// Create map of datapoints by time
	dataMap := make(map[string]IntradayDataPoint)

	// Add existing datapoints
	for _, dp := range existing {
		dataMap[dp.Time] = dp
	}

	// Overlay new datapoints (overwrites duplicates)
	for _, dp := range new {
		dataMap[dp.Time] = dp
	}

	// Convert back to sorted slice
	result := make([]IntradayDataPoint, 0, len(dataMap))
	for _, dp := range dataMap {
		result = append(result, dp)
	}

	// Sort by time
	sort.Slice(result, func(i, j int) bool {
		return result[i].Time < result[j].Time
	})

	return result
}

// compareDatapoints 比较已有和新的数据点，检测价格变化
// 时间复杂度: O(n+m)
func CompareDatapoints(existing, new []IntradayDataPoint) DatapointDiffResult {
	result := DatapointDiffResult{}

	// 构建已有数据的 map (时间 -> 价格)
	existingMap := make(map[string]float64, len(existing))
	for _, dp := range existing {
		existingMap[dp.Time] = dp.Price
	}

	// 比较每个新数据点
	for _, dp := range new {
		existingPrice, exists := existingMap[dp.Time]
		if !exists {
			// 新时间点
			result.HasNewEntries = true
			result.NewEntryCount++
		} else if existingPrice != dp.Price {
			// 价格变化
			result.HasPriceChanges = true
			result.PriceChangeCount++
		}
	}

	return result
}

// ShouldSaveIntradayData 决定保存策略
func ShouldSaveIntradayData(existing, new []IntradayDataPoint) SaveDecision {
	// 首次写入
	if len(existing) == 0 {
		return SaveDecisionUpdate
	}

	diff := CompareDatapoints(existing, new)

	// 有价格变化 → 增量更新
	if diff.HasPriceChanges {
		return SaveDecisionUpdate
	}

	// 有新时间点但无价格变化 → 追加
	if diff.HasNewEntries {
		return SaveDecisionAppend
	}

	// 完全无变化 → 跳过
	return SaveDecisionSkip
}

// formatIntradayTime converts "2025-11-26 09:31:00" to "09:31"
func formatIntradayTime(fullTime string) string {
	// Try to parse various formats
	parts := strings.Fields(fullTime)
	if len(parts) < 2 {
		return ""
	}

	// Extract time part (e.g., "09:31:00")
	timePart := parts[1]
	timeComponents := strings.Split(timePart, ":")
	if len(timeComponents) < 2 {
		return ""
	}

	// Return "HH:MM"
	return timeComponents[0] + ":" + timeComponents[1]
}

// fetchAndSaveIntradayData performs one fetch-merge-save cycle for a stock
// targetDate: 目标日期 (YYYYMMDD)，如果为空则自动计算
// 返回: (SaveDecision, error) - 保存决策和错误（如果有）
func (im *IntradayManager) fetchAndSaveIntradayData(stockCode, stockName string, m ModelInterface, checkMarketHours bool, targetDate string) (SaveDecision, error) {
	// Check if market is open (only if requested)
	if checkMarketHours && !IsMarketOpen(stockCode, m) {
		return SaveDecisionSkip, nil // Not an error, just skipping
	}

	// Fetch from API
	datapoints, err := FetchIntradayDataFromAPI(stockCode)
	if err != nil {
		logDebug("log.intraday.fetchFail", stockCode, err)
		return SaveDecisionUpdate, err
	}

	if len(datapoints) == 0 {
		logDebug("log.intraday.noData", stockCode)
		return SaveDecisionUpdate, fmt.Errorf("no datapoints returned for %s", stockCode)
	}

	// 确定目标日期：优先使用传入的 targetDate，否则根据交易状态计算
	var today string
	if targetDate != "" {
		today = targetDate
	} else {
		// 降级逻辑：根据交易状态决定日期
		market := string(api.GetMarketType(stockCode))
		location, err := GetMarketLocation(market)
		if err != nil {
			// 如果无法获取时区，使用市场当前日期（向后兼容）
			today = getCurrentDateForMarket(market, m)
		} else {
			now := time.Now().In(location)
			tradingState := GetTradingState(now, market)

			// 盘前时，使用上一个交易日；盘中/盘后时，使用当天
			if tradingState == types.TradingStatePreMarket || tradingState == types.TradingStateWeekend || tradingState == types.TradingStateHoliday {
				today = FindPreviousTradingDay(stockCode, now.Format("20060102"), m)
			} else {
				today = now.Format("20060102")
			}
		}
	}

	marketDir := getMarketDirectory(stockCode)
	filePath := filepath.Join("data", "intraday", marketDir, stockCode, today+".json")

	// Ensure directory exists (using new market-based structure)
	if err := ensureIntradayDirectoryWithMarket(stockCode); err != nil {
		logDebug("log.intraday.mkdirFail", stockCode, err)
		return SaveDecisionUpdate, err
	}

	// 获取市场类型（用于保存到数据结构）
	market := string(api.GetMarketType(stockCode))

	// Read existing data (if any)
	existingData := &IntradayData{
		Code:       stockCode,
		Name:       stockName,
		Date:       today,
		Market:     market, // 保存市场类型
		Datapoints: []IntradayDataPoint{},
	}

	if fileExists(filePath) {
		data, err := os.ReadFile(filePath)
		if err == nil {
			if unmarshalErr := json.Unmarshal(data, existingData); unmarshalErr != nil {
				logDebug("log.intraday.unmarshalFail", stockCode, unmarshalErr)
				// 文件损坏时重置为空数据，而非静默使用零值覆盖
				existingData.Datapoints = []IntradayDataPoint{}
			}
		}
	}

	// 增量更新决策逻辑
	newTimestamp := time.Now().Format("2006-01-02 15:04:05")
	decision := ShouldSaveIntradayData(existingData.Datapoints, datapoints)
	diff := CompareDatapoints(existingData.Datapoints, datapoints)

	switch decision {
	case SaveDecisionSkip:
		// 完全无变化，跳过保存
		logDebug("log.intraday.skipSave", stockCode, existingData.UpdatedAt, newTimestamp)
		return SaveDecisionSkip, nil

	case SaveDecisionAppend:
		// 仅追加新时间点，更新时间戳
		existingData.Datapoints = mergeDatapoints(existingData.Datapoints, datapoints)
		existingData.UpdatedAt = newTimestamp
		logDebug("log.intraday.appendOnly", stockCode, diff.NewEntryCount)

	case SaveDecisionUpdate:
		// 有价格变化，完整更新
		existingData.Datapoints = mergeDatapoints(existingData.Datapoints, datapoints)
		existingData.UpdatedAt = newTimestamp
		logDebug("log.intraday.priceUpdate", stockCode, diff.PriceChangeCount, diff.NewEntryCount)
	}

	// NEW: 如果 existingData.PrevClose 为空，从缓存获取（单 key 查询，避免全量复制）
	if existingData.PrevClose == 0 {
		if entry := m.GetStockPriceCacheEntry(stockCode); entry != nil {
			if data, ok := entry.(interface{ GetPrevClose() float64 }); ok {
				existingData.PrevClose = data.GetPrevClose()
			}
		}

		if existingData.PrevClose > 0 {
			logDebug("log.intraday.prevCloseSet", stockCode, existingData.PrevClose)
		}
	}

	// Write back to file
	if err := SaveIntradayData(filePath, existingData); err != nil {
		logDebug("log.intraday.saveFail", stockCode, err)
		return decision, err
	}

	logDebug("log.intraday.saveSuccess", stockCode, len(existingData.Datapoints))
	return decision, nil
}
