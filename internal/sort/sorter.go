package sort

import (
	"sort"
	"strings"

	"stock-monitor/internal/consts"
	"stock-monitor/internal/types"
)

// StockSorter 股票排序接口
type StockSorter interface {
	SortPortfolio(stocks []types.Stock, field consts.SortField, direction consts.SortDirection)
	SortWatchlist(stocks []types.WatchlistStock, stockCache map[string]*types.StockPriceCacheEntry, field consts.SortField, direction consts.SortDirection)
}

// DefaultSorter 默认排序实现（使用Go标准库的sort包）
type DefaultSorter struct{}

// NewDefaultSorter 创建默认排序器
func NewDefaultSorter() *DefaultSorter {
	return &DefaultSorter{}
}

// SortPortfolio 排序持股列表
func (s *DefaultSorter) SortPortfolio(stocks []types.Stock, field consts.SortField, direction consts.SortDirection) {
	sort.Slice(stocks, func(i, j int) bool {
		var result bool

		switch field {
		case consts.SortByCode:
			result = stocks[i].Code < stocks[j].Code
		case consts.SortByName:
			result = stocks[i].Name < stocks[j].Name
		case consts.SortByPrice:
			result = stocks[i].Price < stocks[j].Price
		case consts.SortByCostPrice:
			result = stocks[i].CostPrice < stocks[j].CostPrice
		case consts.SortByChange:
			result = stocks[i].Change < stocks[j].Change
		case consts.SortByChangePercent:
			result = stocks[i].ChangePercent < stocks[j].ChangePercent
		case consts.SortByQuantity:
			result = stocks[i].Quantity < stocks[j].Quantity
		case consts.SortByTotalProfit:
			profitI := float64(stocks[i].Quantity) * (stocks[i].Price - stocks[i].CostPrice)
			profitJ := float64(stocks[j].Quantity) * (stocks[j].Price - stocks[j].CostPrice)
			result = profitI < profitJ
		case consts.SortByProfitRate:
			rateI := (stocks[i].Price - stocks[i].CostPrice) / stocks[i].CostPrice * 100
			rateJ := (stocks[j].Price - stocks[j].CostPrice) / stocks[j].CostPrice * 100
			result = rateI < rateJ
		case consts.SortByMarketValue:
			valueI := stocks[i].Price * float64(stocks[i].Quantity)
			valueJ := stocks[j].Price * float64(stocks[j].Quantity)
			result = valueI < valueJ
		default:
			result = stocks[i].Code < stocks[j].Code
		}

		if direction == consts.SortDesc {
			result = !result
		}

		return result
	})
}

// SortWatchlist 排序自选列表（使用缓存的股价数据，避免API调用）
func (s *DefaultSorter) SortWatchlist(stocks []types.WatchlistStock, stockCache map[string]*types.StockPriceCacheEntry, field consts.SortField, direction consts.SortDirection) {
	sort.Slice(stocks, func(i, j int) bool {
		var result bool

		// 获取缓存的股价数据
		stockDataI := s.getStockDataFromCache(stocks[i].Code, stockCache)
		stockDataJ := s.getStockDataFromCache(stocks[j].Code, stockCache)

		switch field {
		case consts.SortByCode:
			result = stocks[i].Code < stocks[j].Code
		case consts.SortByName:
			result = stocks[i].Name < stocks[j].Name
		case consts.SortByTag:
			tagsI := s.GetTagsDisplay(stocks[i].Tags)
			tagsJ := s.GetTagsDisplay(stocks[j].Tags)
			result = tagsI < tagsJ
		case consts.SortByPrice:
			priceI := s.getPrice(stockDataI)
			priceJ := s.getPrice(stockDataJ)
			result = priceI < priceJ
		case consts.SortByChangePercent:
			changeI := s.getChangePercent(stockDataI)
			changeJ := s.getChangePercent(stockDataJ)
			result = changeI < changeJ
		case consts.SortByTurnoverRate:
			turnoverI := s.getTurnoverRate(stockDataI)
			turnoverJ := s.getTurnoverRate(stockDataJ)
			result = turnoverI < turnoverJ
		case consts.SortByVolume:
			volumeI := s.getVolume(stockDataI)
			volumeJ := s.getVolume(stockDataJ)
			result = volumeI < volumeJ
		default:
			result = stocks[i].Code < stocks[j].Code
		}

		if direction == consts.SortDesc {
			result = !result
		}

		return result
	})
}

// 辅助函数：从缓存获取股价数据
func (s *DefaultSorter) getStockDataFromCache(code string, stockCache map[string]*types.StockPriceCacheEntry) *types.StockData {
	if entry, exists := stockCache[code]; exists && entry.Data != nil {
		return entry.Data
	}
	return nil
}

// 辅助函数：获取价格
func (s *DefaultSorter) getPrice(data *types.StockData) float64 {
	if data != nil {
		return data.Price
	}
	return 0
}

// 辅助函数：获取涨跌幅
func (s *DefaultSorter) getChangePercent(data *types.StockData) float64 {
	if data != nil {
		return data.ChangePercent
	}
	return 0
}

// 辅助函数：获取换手率
func (s *DefaultSorter) getTurnoverRate(data *types.StockData) float64 {
	if data != nil {
		return data.TurnoverRate
	}
	return 0
}

// 辅助函数：获取成交量
func (s *DefaultSorter) getVolume(data *types.StockData) int64 {
	if data != nil {
		return data.Volume
	}
	return 0
}

// GetTagsDisplay 获取标签显示文本
func (s *DefaultSorter) GetTagsDisplay(tags []string) string {
	var validTags []string
	for _, tag := range tags {
		if tag != "" && tag != "-" {
			validTags = append(validTags, tag)
		}
	}

	if len(validTags) == 0 {
		return "-"
	}

	return strings.Join(validTags, ",")
}
