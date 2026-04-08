package api

import (
	"stock-monitor/internal/api/china"
	"stock-monitor/internal/api/hongkong"
	"stock-monitor/internal/api/us"
	"stock-monitor/internal/log"
	"stock-monitor/internal/types"
)

// ConvertStockDataToInternal converts main package StockData to internal types.StockData
// This is a temporary bridge function until full refactoring is complete
func ConvertFromInternalStockData(data *types.StockData) interface{} {
	if data == nil {
		return nil
	}
	// Return as-is since the API already returns types.StockData
	// The calling code will need to do type conversion
	return data
}

// GetStockInfo 获取股票信息（支持中英文搜索）
func GetStockInfo(symbol string) *types.StockData {
	var stockData *types.StockData

	// 如果输入是中文，尝试通过API搜索
	if ContainsChineseChars(symbol) {
		stockData = SearchChineseStock(symbol)
	} else {
		// 对于非中文输入，先尝试直接获取价格，然后尝试搜索
		stockData = GetStockPrice(symbol)

		// 如果直接获取失败，尝试作为搜索关键词搜索
		if stockData == nil || stockData.Price <= 0 {
			log.Error("log.api.directFail", symbol)
			stockData = SearchStockBySymbol(symbol)
		}
	}

	return stockData
}

// GetStockPrice 获取股票价格（带多API降级策略）
func GetStockPrice(symbol string) *types.StockData {
	if IsChinaStock(symbol) || IsHKStock(symbol) {
		data := china.TryTencentAPI(symbol)
		if data.Price > 0 {
			if IsHKStock(symbol) && data.TurnoverRate == 0 {
				log.Debug("log.api.hkTurnoverMissing", symbol)
				turnover, volume, err := hongkong.TryEastMoneyHKTurnover(symbol)
				if err == nil {
					data.TurnoverRate = turnover
					if volume > 0 {
						data.Volume = volume
					}
					log.Debug("log.api.hkTurnoverEnhanced", symbol, turnover)
				} else {
					log.Error("log.api.hkTurnoverFallbackFail", err)
				}
			}
			return data
		}
		log.Error("log.api.tencentFail")
		// A股/港股不应穿透到 US API，避免发送无效代码和掩盖区域 API 故障
		return nil
	}

	data := tryMultiUSAPI(symbol)
	if data.Price > 0 {
		return data
	}

	log.Error("log.api.allApiFail")
	return nil
}

// tryMultiUSAPI 尝试美股API（实际上是多API降级策略）
func tryMultiUSAPI(symbol string) *types.StockData {
	// 策略1: 尝试TwelveData API
	data := us.TryTwelveDataAPI(symbol)
	if data != nil && data.Price > 0 {
		return data
	}

	// 策略2: 尝试免费的 FMP API (无需API key的基础数据)
	data = us.TryFMPFreeAPI(symbol)
	if data != nil && data.Price > 0 {
		return data
	}

	// 策略3: 尝试Yahoo Finance API
	data = us.TryYahooFinanceAPI(symbol)
	if data != nil && data.Price > 0 {
		return data
	}

	log.Error("log.api.allUsApiFail")
	return &types.StockData{Symbol: symbol, Price: 0}
}
