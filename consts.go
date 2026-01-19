package main

import (
	"stock-monitor/internal/consts"
)

// 文件路径常量 - 重新导出
const (
	dataFile        = consts.DataFile
	watchlistFile   = consts.WatchlistFile
	alertFile       = consts.AlertFile
	configFile      = consts.ConfigFile
	refreshInterval = consts.RefreshInterval
)

// 语言常量 - 重新导出
const (
	Chinese = consts.Chinese
	English = consts.English
)

// 应用状态常量 - 重新导出
const (
	MainMenu                 = consts.MainMenu
	AddingStock              = consts.AddingStock
	Monitoring               = consts.Monitoring
	EditingStock             = consts.EditingStock
	SearchingStock           = consts.SearchingStock
	SearchResult             = consts.SearchResult
	LanguageSelection        = consts.LanguageSelection
	WatchlistViewing         = consts.WatchlistViewing
	SearchResultWithActions  = consts.SearchResultWithActions
	WatchlistSearchConfirm   = consts.WatchlistSearchConfirm
	WatchlistTagging         = consts.WatchlistTagging
	WatchlistTagSelect       = consts.WatchlistTagSelect
	WatchlistTagManage       = consts.WatchlistTagManage
	WatchlistTagRemoveSelect = consts.WatchlistTagRemoveSelect
	WatchlistTagEdit         = consts.WatchlistTagEdit
	WatchlistGroupSelect     = consts.WatchlistGroupSelect
	PortfolioSorting         = consts.PortfolioSorting
	WatchlistSorting         = consts.WatchlistSorting
	IntradayChartViewing     = consts.IntradayChartViewing
	AlertManage              = consts.AlertManage
	StockAlertManage         = consts.StockAlertManage
	AlertAdd                 = consts.AlertAdd
	AlertBatchMethodSelect   = consts.AlertBatchMethodSelect
	AlertBatchByTag          = consts.AlertBatchByTag
	AlertBatchByMarket       = consts.AlertBatchByMarket
	AlertBatchByStocks       = consts.AlertBatchByStocks
	SelectBatchStocks        = consts.SelectBatchStocks
	SelectBatchFromWatchlist = consts.SelectBatchFromWatchlist
	SelectBatchFromPortfolio = consts.SelectBatchFromPortfolio
	InputBatchCodes          = consts.InputBatchCodes
	AlertEdit                = consts.AlertEdit
	AlertTypeSelect          = consts.AlertTypeSelect
	AlertThresholdInput      = consts.AlertThresholdInput
	AlertConditionSelect     = consts.AlertConditionSelect
	AlertBatchAdd            = consts.AlertBatchAdd
	AlertFrequencySelect     = consts.AlertFrequencySelect
	AlertFrequencyDaysInput  = consts.AlertFrequencyDaysInput
)

// 排序字段枚举 - 重新导出
const (
	SortByCode          = consts.SortByCode
	SortByName          = consts.SortByName
	SortByPrice         = consts.SortByPrice
	SortByCostPrice     = consts.SortByCostPrice
	SortByChange        = consts.SortByChange
	SortByChangePercent = consts.SortByChangePercent
	SortByQuantity      = consts.SortByQuantity
	SortByTotalProfit   = consts.SortByTotalProfit
	SortByProfitRate    = consts.SortByProfitRate
	SortByMarketValue   = consts.SortByMarketValue
	SortByTag           = consts.SortByTag
	SortByTurnoverRate  = consts.SortByTurnoverRate
	SortByVolume        = consts.SortByVolume
)

// 排序方向枚举 - 重新导出
const (
	SortAsc  = consts.SortAsc
	SortDesc = consts.SortDesc
)
