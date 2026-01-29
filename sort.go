package main

import (
	"sort"

	"stock-monitor/internal/consts"
	internalSort "stock-monitor/internal/sort"
	"stock-monitor/internal/types"
)

// StockSorter 股票排序接口 - 重新导出
type StockSorter = internalSort.StockSorter

// DefaultSorter 默认排序实现 - 重新导出
type DefaultSorter = internalSort.DefaultSorter

// NewDefaultSorter 创建默认排序器
func NewDefaultSorter() *DefaultSorter {
	return internalSort.NewDefaultSorter()
}

// 性能优化的排序函数，供Model调用
func (m *Model) optimizedSortPortfolio(field SortField, direction SortDirection) {
	sorter := NewDefaultSorter()
	// 需要转换类型
	stocks := make([]types.Stock, len(m.portfolio.Stocks))
	for i, s := range m.portfolio.Stocks {
		stocks[i] = types.Stock(s)
	}
	sorter.SortPortfolio(stocks, field, direction)
	// 转换回来
	for i, s := range stocks {
		m.portfolio.Stocks[i] = Stock(s)
	}
}

// updatePortfolioPricesFromCache 从缓存更新持股列表的价格数据
// 用于排序前确保价格数据是最新的
func (m *Model) updatePortfolioPricesFromCache() {
	for i := range m.portfolio.Stocks {
		stock := &m.portfolio.Stocks[i]
		stockData := m.getStockPriceFromCache(stock.Code)
		if stockData != nil {
			stock.Price = stockData.Price
			stock.Change = stockData.Change
			stock.ChangePercent = stockData.ChangePercent
			stock.StartPrice = stockData.StartPrice
			stock.MaxPrice = stockData.MaxPrice
			stock.MinPrice = stockData.MinPrice
			stock.PrevClose = stockData.PrevClose
		}
	}
}

func (m *Model) optimizedSortWatchlist(field SortField, direction SortDirection) {
	// 获取过滤后的股票列表
	filteredStocks := m.getFilteredWatchlist()

	// 使用高效排序（基于缓存数据，避免API调用）
	sorter := NewDefaultSorter()

	// 读取股价缓存并转换类型
	m.stockPriceMutex.RLock()
	stockCacheCopy := make(map[string]*types.StockPriceCacheEntry)
	for k, v := range m.stockPriceCache {
		// 类型转换
		entry := &types.StockPriceCacheEntry{
			UpdateTime: v.UpdateTime,
			IsUpdating: v.IsUpdating,
		}
		if v.Data != nil {
			entry.Data = &types.StockData{
				Symbol:        v.Data.Symbol,
				Name:          v.Data.Name,
				Price:         v.Data.Price,
				Change:        v.Data.Change,
				ChangePercent: v.Data.ChangePercent,
				StartPrice:    v.Data.StartPrice,
				MaxPrice:      v.Data.MaxPrice,
				MinPrice:      v.Data.MinPrice,
				PrevClose:     v.Data.PrevClose,
				TurnoverRate:  v.Data.TurnoverRate,
				Volume:        v.Data.Volume,
			}
		}
		stockCacheCopy[k] = entry
	}
	m.stockPriceMutex.RUnlock()

	// 转换 filteredStocks 到 types.WatchlistStock
	typesStocks := make([]types.WatchlistStock, len(filteredStocks))
	for i, s := range filteredStocks {
		typesStocks[i] = types.WatchlistStock{
			Code:   s.Code,
			Name:   s.Name,
			Tags:   s.Tags,
			Market: types.MarketType(s.Market),
		}
	}

	// 执行排序（使用缓存数据）
	sorter.SortWatchlist(typesStocks, stockCacheCopy, field, direction)

	// 转换回来
	for i, s := range typesStocks {
		filteredStocks[i] = WatchlistStock{
			Code:   s.Code,
			Name:   s.Name,
			Tags:   s.Tags,
			Market: MarketType(s.Market),
		}
	}

	// 将排序后的过滤列表更新回原列表
	// 如果没有过滤，直接使用排序结果
	if m.selectedMarketFilter == "" && m.selectedUserTagFilter == "" {
		m.watchlist.Stocks = filteredStocks
	} else {
		// 如果有过滤，需要将排序结果更新回原列表
		// 创建一个映射来快速查找排序后的位置
		sortedMap := make(map[string]int)
		for i, stock := range filteredStocks {
			sortedMap[stock.Code] = i
		}

		// 重新排列原列表，将过滤的股票按排序顺序放在前面
		var newStocks []WatchlistStock
		var remainingStocks []WatchlistStock

		// 先添加排序后的过滤股票
		newStocks = append(newStocks, filteredStocks...)

		// 再添加未过滤的股票
		for _, stock := range m.watchlist.Stocks {
			if _, exists := sortedMap[stock.Code]; !exists {
				remainingStocks = append(remainingStocks, stock)
			}
		}

		newStocks = append(newStocks, remainingStocks...)
		m.watchlist.Stocks = newStocks
	}
}

// ============================================================================
// Sector Sorting Functions - 板块排序
// ============================================================================

// sortSectors 板块列表排序
func sortSectors(sectors []types.Sector, field SortField, direction SortDirection) {
	if len(sectors) == 0 {
		return
	}

	// 使用 Go 标准库 sort
	sort.Slice(sectors, func(i, j int) bool {
		var less bool
		switch field {
		case consts.SortBySectorName:
			less = sectors[i].Name < sectors[j].Name
		case consts.SortBySectorChangePercent:
			less = sectors[i].ChangePercent < sectors[j].ChangePercent
		case consts.SortBySectorChange:
			less = sectors[i].Change < sectors[j].Change
		case consts.SortBySectorTurnover:
			less = sectors[i].Turnover < sectors[j].Turnover
		case consts.SortByTurnoverRate:
			less = sectors[i].TurnoverRate < sectors[j].TurnoverRate
		case consts.SortBySectorRiseCount:
			less = sectors[i].RiseCount < sectors[j].RiseCount
		default:
			less = sectors[i].Name < sectors[j].Name
		}

		if direction == consts.SortDesc {
			return !less
		}
		return less
	})
}

// sortSectorStocks 成分股列表排序
func sortSectorStocks(stocks []types.SectorStock, field SortField, direction SortDirection) {
	if len(stocks) == 0 {
		return
	}

	sort.Slice(stocks, func(i, j int) bool {
		var less bool
		switch field {
		case consts.SortByCode:
			less = stocks[i].Code < stocks[j].Code
		case consts.SortByName:
			less = stocks[i].Name < stocks[j].Name
		case consts.SortByPrice:
			less = stocks[i].Price < stocks[j].Price
		case consts.SortByChangePercent:
			less = stocks[i].ChangePercent < stocks[j].ChangePercent
		case consts.SortByChange:
			less = stocks[i].Change < stocks[j].Change
		case consts.SortByVolume:
			less = stocks[i].Volume < stocks[j].Volume
		case consts.SortBySectorTurnover:
			less = stocks[i].Turnover < stocks[j].Turnover
		case consts.SortByTurnoverRate:
			less = stocks[i].TurnoverRate < stocks[j].TurnoverRate
		default:
			less = stocks[i].Name < stocks[j].Name
		}

		if direction == consts.SortDesc {
			return !less
		}
		return less
	})
}
