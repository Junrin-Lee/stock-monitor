package api

import (
	"stock-monitor/internal/api/china"
	"stock-monitor/internal/api/us"
	"stock-monitor/internal/log"
	"stock-monitor/internal/types"
	"strings"
)

// SearchStockBySymbol 通过符号搜索股票（支持美股等国际股票）
func SearchStockBySymbol(symbol string) *types.StockData {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	log.Debug("log.api.symbolSearch", symbol)

	// 策略1: 使用TwelveData搜索API
	result := us.SearchStockByTwelveDataAPI(symbol)
	if result != nil && result.Price > 0 {
		log.Info("log.api.twelveDataSuccess", result.Name, result.Symbol)
		return result
	}

	// 策略2: 尝试腾讯API（可能支持部分国际股票）
	result = china.SearchStockByTencentAPI(symbol)
	if result != nil && result.Price > 0 {
		log.Info("log.api.tencentSuccess", result.Name, result.Symbol)
		return result
	}

	// 策略3: 尝试新浪API（可能支持部分国际股票）
	result = china.SearchStockBySinaAPI(symbol)
	if result != nil && result.Price > 0 {
		log.Info("log.api.sinaSuccess", result.Name, result.Symbol)
		return result
	}

	log.Debug("log.api.allSymbolFailed")
	return nil
}

// SearchChineseStock 通过API搜索中文股票名称
func SearchChineseStock(chineseName string) *types.StockData {
	chineseName = strings.TrimSpace(chineseName)
	log.Debug("log.api.chineseSearch", chineseName)

	// 策略1: 使用腾讯搜索API
	result := china.SearchStockByTencentAPI(chineseName)
	if result != nil && result.Price > 0 {
		log.Info("log.api.tencentSearchSuccess", result.Name, result.Symbol)
		return result
	}

	// 策略2: 尝试新浪财经搜索API
	result = china.SearchStockBySinaAPI(chineseName)
	if result != nil && result.Price > 0 {
		log.Info("log.api.sinaSearchSuccess", result.Name, result.Symbol)
		return result
	}

	// 策略3: 尝试更多的搜索关键词变形
	result = tryAdvancedSearch(chineseName)
	if result != nil && result.Price > 0 {
		log.Info("log.api.advancedSearchSuccess", result.Name, result.Symbol)
		return result
	}

	// 所有搜索策略都失败
	log.Debug("log.api.allSearchFailed")
	return nil
}

// tryAdvancedSearch 高级搜索策略：尝试多种关键词变形
func tryAdvancedSearch(chineseName string) *types.StockData {
	// 生成搜索关键词变形
	keywords := GenerateSearchKeywords(chineseName)

	for _, keyword := range keywords {
		if keyword == chineseName {
			continue // 跳过原始关键词，避免重复搜索
		}

		log.Debug("log.api.tryKeywordVariation", keyword)
		result := china.SearchStockByTencentAPI(keyword)
		if result != nil && result.Price > 0 {
			return result
		}
	}

	return nil
}
