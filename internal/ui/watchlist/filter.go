package watchlist

import "stock-monitor/internal/types"

// ============================================================================
// 自选列表过滤逻辑 - 框架无关
// ============================================================================

// FilterState 过滤状态（用于缓存）
type FilterState struct {
	MarketFilter  types.MarketType
	UserTagFilter string
	IsValid       bool
}

// GetFilteredWatchlist 根据市场和用户标签过滤自选股票（支持组合过滤，使用 AND 逻辑）
func GetFilteredWatchlist(stocks []types.WatchlistStock, marketFilter types.MarketType, userTagFilter string) []types.WatchlistStock {
	// 如果没有任何过滤条件，直接返回完整列表
	if marketFilter == "" && userTagFilter == "" {
		return stocks
	}

	// 应用过滤条件（AND 逻辑）
	var filtered []types.WatchlistStock
	for _, stock := range stocks {
		// 检查市场过滤条件
		if marketFilter != "" && stock.Market != marketFilter {
			continue // 市场不匹配，跳过
		}

		// 检查用户标签过滤条件
		if userTagFilter != "" && !HasTag(&stock, userTagFilter) {
			continue // 用户标签不匹配，跳过
		}

		// 两个条件都满足（或未设置），加入过滤结果
		filtered = append(filtered, stock)
	}

	return filtered
}

// ============================================================================
// 自选列表光标和滚动管理 - 框架无关的计算逻辑
// ============================================================================

// ScrollPosition 滚动位置计算结果
type ScrollPosition struct {
	Cursor    int
	ScrollPos int
}

// ResetWatchlistCursor 重置自选列表游标到第一只股票（基于过滤后的列表）
func ResetWatchlistCursor(filteredCount, maxLines int) ScrollPosition {
	if filteredCount > 0 {
		scrollPos := 0
		if filteredCount > maxLines {
			// 显示前N条：滚动位置设置为显示从索引0开始的N条
			scrollPos = filteredCount - maxLines
		}
		return ScrollPosition{Cursor: 0, ScrollPos: scrollPos}
	}

	// 没有股票时重置
	return ScrollPosition{Cursor: 0, ScrollPos: 0}
}

// AdjustWatchlistScroll 调整自选列表滚动位置（基于过滤后的列表）
func AdjustWatchlistScroll(cursor, scrollPos, filteredCount, maxLines int) int {
	if filteredCount <= maxLines {
		return 0
	}

	// 确保光标在可见范围内
	endIndex := filteredCount - scrollPos
	startIndex := endIndex - maxLines
	if startIndex < 0 {
		startIndex = 0
	}

	// 如果光标超出可见范围的上边界，调整滚动位置
	if cursor < startIndex {
		newScrollPos := filteredCount - cursor - maxLines
		if newScrollPos < 0 {
			newScrollPos = 0
		}
		return newScrollPos
	}

	// 如果光标超出可见范围的下边界，调整滚动位置
	if cursor >= endIndex {
		newScrollPos := filteredCount - cursor - 1
		if newScrollPos < 0 {
			newScrollPos = 0
		}
		return newScrollPos
	}

	return scrollPos
}

// ============================================================================
// WatchlistStock 标签方法 - 框架无关
// ============================================================================

// HasTag 检查股票是否包含指定标签
func HasTag(stock *types.WatchlistStock, tag string) bool {
	for _, t := range stock.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag 添加标签到股票（避免重复）
func AddTag(stock *types.WatchlistStock, tag string) {
	if tag == "" || tag == "-" {
		return
	}
	if !HasTag(stock, tag) {
		stock.Tags = append(stock.Tags, tag)
	}
}

// RemoveTag 移除股票的标签
func RemoveTag(stock *types.WatchlistStock, tag string) {
	for i, t := range stock.Tags {
		if t == tag {
			stock.Tags = append(stock.Tags[:i], stock.Tags[i+1:]...)
			break
		}
	}
}

// ============================================================================
// 自选列表股票管理 - 框架无关
// ============================================================================

// IsStockInWatchlist 检查股票是否已在自选列表中
func IsStockInWatchlist(stocks []types.WatchlistStock, code string) bool {
	for _, stock := range stocks {
		if stock.Code == code {
			return true
		}
	}
	return false
}

// AddToWatchlist 添加股票到自选列表（返回新列表）
func AddToWatchlist(stocks []types.WatchlistStock, code, name string, market types.MarketType) []types.WatchlistStock {
	if IsStockInWatchlist(stocks, code) {
		return stocks // 已在列表中
	}

	watchStock := types.WatchlistStock{
		Code:   code,
		Name:   name,
		Market: market,     // 保存市场类型
		Tags:   []string{}, // 初始为空，不包含市场标签
	}
	// 将新股票插入到列表首位，而不是末尾
	return append([]types.WatchlistStock{watchStock}, stocks...)
}

// RemoveFromWatchlist 从自选列表删除股票（返回新列表）
func RemoveFromWatchlist(stocks []types.WatchlistStock, index int) []types.WatchlistStock {
	if index >= 0 && index < len(stocks) {
		return append(stocks[:index], stocks[index+1:]...)
	}
	return stocks
}
