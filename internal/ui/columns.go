package ui

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"stock-monitor/internal/consts"
)

// ColumnID - 列的唯一标识符
type ColumnID string

// Portfolio列ID常量
const (
	ColCursor         ColumnID = "cursor"
	ColCode           ColumnID = "code"
	ColName           ColumnID = "name"
	ColPrevClose      ColumnID = "prev_close"
	ColOpen           ColumnID = "open"
	ColHigh           ColumnID = "high"
	ColLow            ColumnID = "low"
	ColPrice          ColumnID = "price"
	ColCost           ColumnID = "cost"
	ColQuantity       ColumnID = "quantity"
	ColTodayChange    ColumnID = "today_change"
	ColPositionProfit ColumnID = "position_profit"
	ColProfitRate     ColumnID = "profit_rate"
	ColMarketValue    ColumnID = "market_value"

	// Watchlist特有列ID
	ColTag      ColumnID = "tag"
	ColTurnover ColumnID = "turnover"
	ColVolume   ColumnID = "volume"
)

// ColumnMetadata - 列的元数据
type ColumnMetadata struct {
	ID         ColumnID             // 列ID
	I18nKey    string               // 国际化翻译键
	IsRequired bool                 // 是否为必须列（不可隐藏）
	SortField  *consts.SortField    // 关联的排序字段（nil表示不可排序）
}

// ColumnRegistry - 列注册表
type ColumnRegistry struct {
	portfolioColumns map[ColumnID]*ColumnMetadata
	watchlistColumns map[ColumnID]*ColumnMetadata
}

// GlobalColumnRegistry - 全局列注册表实例
var GlobalColumnRegistry *ColumnRegistry

// InitColumnRegistry - 初始化列注册表
func InitColumnRegistry() {
	GlobalColumnRegistry = &ColumnRegistry{
		portfolioColumns: makePortfolioColumnRegistry(),
		watchlistColumns: makeWatchlistColumnRegistry(),
	}
}

// makePortfolioColumnRegistry - 创建Portfolio列注册表
func makePortfolioColumnRegistry() map[ColumnID]*ColumnMetadata {
	// 创建排序字段指针（用于元数据引用）
	sortByCode := consts.SortByCode
	sortByName := consts.SortByName
	sortByPrice := consts.SortByPrice
	sortByCostPrice := consts.SortByCostPrice
	sortByQuantity := consts.SortByQuantity
	sortByChangePercent := consts.SortByChangePercent
	sortByTotalProfit := consts.SortByTotalProfit
	sortByProfitRate := consts.SortByProfitRate
	sortByMarketValue := consts.SortByMarketValue

	return map[ColumnID]*ColumnMetadata{
		ColCursor: {
			ID:         ColCursor,
			I18nKey:    "", // 光标列无需翻译
			IsRequired: true,
			SortField:  nil, // 不可排序
		},
		ColCode: {
			ID:         ColCode,
			I18nKey:    "col.code",
			IsRequired: true,
			SortField:  &sortByCode,
		},
		ColName: {
			ID:         ColName,
			I18nKey:    "col.name",
			IsRequired: true,
			SortField:  &sortByName,
		},
		ColPrevClose: {
			ID:         ColPrevClose,
			I18nKey:    "col.prev_close",
			IsRequired: false,
			SortField:  nil,
		},
		ColOpen: {
			ID:         ColOpen,
			I18nKey:    "col.open",
			IsRequired: false,
			SortField:  nil,
		},
		ColHigh: {
			ID:         ColHigh,
			I18nKey:    "col.high",
			IsRequired: false,
			SortField:  nil,
		},
		ColLow: {
			ID:         ColLow,
			I18nKey:    "col.low",
			IsRequired: false,
			SortField:  nil,
		},
		ColPrice: {
			ID:         ColPrice,
			I18nKey:    "col.price",
			IsRequired: true,
			SortField:  &sortByPrice,
		},
		ColCost: {
			ID:         ColCost,
			I18nKey:    "col.cost",
			IsRequired: false,
			SortField:  &sortByCostPrice,
		},
		ColQuantity: {
			ID:         ColQuantity,
			I18nKey:    "col.quantity",
			IsRequired: false,
			SortField:  &sortByQuantity,
		},
		ColTodayChange: {
			ID:         ColTodayChange,
			I18nKey:    "col.today_change",
			IsRequired: false,
			SortField:  &sortByChangePercent,
		},
		ColPositionProfit: {
			ID:         ColPositionProfit,
			I18nKey:    "col.position_profit",
			IsRequired: false,
			SortField:  &sortByTotalProfit,
		},
		ColProfitRate: {
			ID:         ColProfitRate,
			I18nKey:    "col.profit_rate",
			IsRequired: false,
			SortField:  &sortByProfitRate,
		},
		ColMarketValue: {
			ID:         ColMarketValue,
			I18nKey:    "col.market_value",
			IsRequired: false,
			SortField:  &sortByMarketValue,
		},
	}
}

// makeWatchlistColumnRegistry - 创建Watchlist列注册表
func makeWatchlistColumnRegistry() map[ColumnID]*ColumnMetadata {
	// 创建排序字段指针
	sortByTag := consts.SortByTag
	sortByCode := consts.SortByCode
	sortByName := consts.SortByName
	sortByPrice := consts.SortByPrice
	sortByChangePercent := consts.SortByChangePercent
	sortByTurnoverRate := consts.SortByTurnoverRate
	sortByVolume := consts.SortByVolume

	return map[ColumnID]*ColumnMetadata{
		ColCursor: {
			ID:         ColCursor,
			I18nKey:    "",
			IsRequired: true,
			SortField:  nil,
		},
		ColTag: {
			ID:         ColTag,
			I18nKey:    "col.tag",
			IsRequired: true,
			SortField:  &sortByTag,
		},
		ColCode: {
			ID:         ColCode,
			I18nKey:    "col.code",
			IsRequired: true,
			SortField:  &sortByCode,
		},
		ColName: {
			ID:         ColName,
			I18nKey:    "col.name",
			IsRequired: true,
			SortField:  &sortByName,
		},
		ColPrice: {
			ID:         ColPrice,
			I18nKey:    "col.price",
			IsRequired: true,
			SortField:  &sortByPrice,
		},
		ColPrevClose: {
			ID:         ColPrevClose,
			I18nKey:    "col.prev_close",
			IsRequired: false,
			SortField:  nil,
		},
		ColOpen: {
			ID:         ColOpen,
			I18nKey:    "col.open",
			IsRequired: false,
			SortField:  nil,
		},
		ColHigh: {
			ID:         ColHigh,
			I18nKey:    "col.high",
			IsRequired: false,
			SortField:  nil,
		},
		ColLow: {
			ID:         ColLow,
			I18nKey:    "col.low",
			IsRequired: false,
			SortField:  nil,
		},
		ColTodayChange: {
			ID:         ColTodayChange,
			I18nKey:    "col.today_change",
			IsRequired: false,
			SortField:  &sortByChangePercent,
		},
		ColTurnover: {
			ID:         ColTurnover,
			I18nKey:    "col.turnover",
			IsRequired: false,
			SortField:  &sortByTurnoverRate,
		},
		ColVolume: {
			ID:         ColVolume,
			I18nKey:    "col.volume",
			IsRequired: false,
			SortField:  &sortByVolume,
		},
	}
}

// GetPortfolioColumns - 获取Portfolio的活跃列列表
// 这些函数需要从main.go的Model方法中调用，所以需要接收必要的参数
func GetPortfolioColumns(configuredColumns []string, getTextFunc func(string) string) []*ColumnMetadata {
	return buildColumnList(configuredColumns, GlobalColumnRegistry.portfolioColumns, getTextFunc)
}

// GetWatchlistColumns - 获取Watchlist的活跃列列表
func GetWatchlistColumns(configuredColumns []string, getTextFunc func(string) string) []*ColumnMetadata {
	return buildColumnList(configuredColumns, GlobalColumnRegistry.watchlistColumns, getTextFunc)
}

// buildColumnList - 从配置构建列元数据列表
func buildColumnList(configIDs []string, registry map[ColumnID]*ColumnMetadata, getTextFunc func(string) string) []*ColumnMetadata {
	var result []*ColumnMetadata
	for _, idStr := range configIDs {
		id := ColumnID(idStr)
		if meta, exists := registry[id]; exists {
			result = append(result, meta)
		}
		// Note: 日志记录已移除，调用方可以自行处理
	}
	return result
}

// GeneratePortfolioHeaderFunc - 生成Portfolio表头（含排序指示器）的辅助函数
// 注意：由于这些函数需要访问Model的许多字段，建议保留在main.go中作为Model的方法
// 这里提供了一个示例，但实际使用中可能需要更多参数

// GenerateHeader - 生成表头的通用函数
func GenerateHeader(
	columns []*ColumnMetadata,
	getTextFunc func(string) string,
	isSorted bool,
	sortField consts.SortField,
	sortDirection consts.SortDirection,
) table.Row {
	header := make(table.Row, len(columns))

	for i, col := range columns {
		// 基础列名
		if col.I18nKey == "" {
			header[i] = "" // cursor列无名称
		} else {
			header[i] = getTextFunc(col.I18nKey)
		}

		// 添加排序指示器
		if col.SortField != nil &&
			isSorted &&
			*col.SortField == sortField {
			indicator := "↑"
			if sortDirection == consts.SortDesc {
				indicator = "↓"
			}
			header[i] = fmt.Sprintf("%s %s", header[i], indicator)
		}
	}

	return header
}
