package main

import (
	"stock-monitor/internal/api"
	"stock-monitor/internal/ui/watchlist"
)

// ============================================================================
// Market Tag Functions (Adapters)
// ============================================================================

// getMarketTagName returns the display name for a market type
func (m *Model) getMarketTagName(market MarketType) string {
	switch market {
	case MarketChina:
		return m.getText("marketTag.china")
	case MarketUS:
		return m.getText("marketTag.us")
	case MarketHongKong:
		return m.getText("marketTag.hongkong")
	}
	return "-"
}

// isMarketTag checks if a tag is a market tag
func isMarketTag(tag string) bool {
	return watchlist.IsMarketTag(tag)
}

// ============================================================================
// Tag Management Functions (Adapters)
// ============================================================================

// renameTagForAllStocks renames a tag across all stocks
func (m *Model) renameTagForAllStocks(oldTag, newTag string) int {
	var updatedCount int
	m.watchlist.Stocks, updatedCount = watchlist.RenameTagForAllStocks(m.watchlist.Stocks, oldTag, newTag)
	return updatedCount
}

// getAvailableTags returns all available tags
func (m *Model) getAvailableTags() []string {
	return watchlist.GetAvailableTags(m.watchlist.Stocks, m.getMarketTagName)
}

// getMarketTags returns market tags in fixed order
func (m *Model) getMarketTags() []string {
	return watchlist.GetMarketTags(m.watchlist.Stocks, m.getMarketTagName)
}

// getUserTags returns user-defined tags
func (m *Model) getUserTags() []string {
	return watchlist.GetUserTags(m.watchlist.Stocks)
}

// getExistingMarkets returns markets that exist in the watchlist
func (m *Model) getExistingMarkets() map[MarketType]bool {
	return watchlist.GetExistingMarkets(m.watchlist.Stocks)
}

// getMarketOptions returns market filter options
func (m *Model) getMarketOptions() []MarketType {
	return watchlist.GetMarketOptions(m.watchlist.Stocks)
}

// getUserTagOptions returns user tag filter options
func (m *Model) getUserTagOptions() []string {
	return watchlist.GetUserTagOptions(m.watchlist.Stocks, m.getText("filter.allTags"))
}

// getTagGroups returns categorized tag groups
func (m *Model) getTagGroups() []TagGroup {
	return watchlist.GetTagGroups(
		m.watchlist.Stocks,
		m.getMarketTagName,
		m.getText("group.marketTags"),
		m.getText("group.userTags"),
	)
}

// getTagAtCursor returns the tag name at cursor position
func (m *Model) getTagAtCursor() string {
	return watchlist.GetTagAtCursor(m.tagGroups, m.cursor)
}

// getTotalTagCount returns total tag count across all groups
func (m *Model) getTotalTagCount() int {
	return watchlist.GetTotalTagCount(m.tagGroups)
}

// findTagPositionInGroups finds the position of a tag
func (m *Model) findTagPositionInGroups(tagName string) int {
	return watchlist.FindTagPositionInGroups(m.tagGroups, tagName)
}

// findMarketTagPosition finds the position of a market tag
func (m *Model) findMarketTagPosition(marketFilter MarketType) int {
	return watchlist.FindMarketTagPosition(m.watchlist.Stocks, marketFilter, m.getMarketTagName)
}

// findUserTagPosition finds the position of a user tag
func (m *Model) findUserTagPosition(tagName string) int {
	return watchlist.FindUserTagPosition(m.watchlist.Stocks, tagName)
}

// ============================================================================
// WatchlistStock Tag Methods (Adapters)
// ============================================================================

// 注意：由于 WatchlistStock 现在是类型别名，不能添加方法
// 这些功能直接通过 watchlist 包的函数提供

// ============================================================================
// Watchlist Filtering and Caching (Adapters)
// ============================================================================

// getFilteredWatchlist returns filtered watchlist
func (m *Model) getFilteredWatchlist() []WatchlistStock {
	// Check cache validity
	if m.isFilteredWatchlistValid &&
		m.cachedFilterMarket == m.selectedMarketFilter &&
		m.cachedFilterUserTag == m.selectedUserTagFilter {
		return m.cachedFilteredWatchlist
	}

	// Recalculate filtered results
	filtered := watchlist.GetFilteredWatchlist(m.watchlist.Stocks, m.selectedMarketFilter, m.selectedUserTagFilter)

	// Update cache
	m.cachedFilteredWatchlist = filtered
	m.cachedFilterMarket = m.selectedMarketFilter
	m.cachedFilterUserTag = m.selectedUserTagFilter
	m.isFilteredWatchlistValid = true

	return filtered
}

// invalidateWatchlistCache invalidates the filter cache
func (m *Model) invalidateWatchlistCache() {
	m.isFilteredWatchlistValid = false
	m.cachedFilteredWatchlist = nil
	m.cachedFilterMarket = ""
	m.cachedFilterUserTag = ""
}

// ============================================================================
// Watchlist Cursor and Scroll Management
// ============================================================================

// resetWatchlistCursor resets cursor to first stock in filtered list

// adjustWatchlistScroll adjusts scroll position
func (m *Model) adjustWatchlistScroll(filteredStocks []WatchlistStock) {
	maxWatchlistLines := m.config.Display.MaxLines
	m.watchlistScrollPos = watchlist.AdjustWatchlistScroll(
		m.watchlistCursor,
		m.watchlistScrollPos,
		len(filteredStocks),
		maxWatchlistLines,
	)
}

// ============================================================================
// Watchlist Stock Management (Adapters)
// ============================================================================

// isStockInWatchlist checks if a stock is in the watchlist
func (m *Model) isStockInWatchlist(code string) bool {
	return watchlist.IsStockInWatchlist(m.watchlist.Stocks, code)
}

// addToWatchlist adds a stock to the watchlist
func (m *Model) addToWatchlist(code, name string) bool {
	if watchlist.IsStockInWatchlist(m.watchlist.Stocks, code) {
		return false
	}

	// Detect market type
	market := api.GetMarketType(code)
	m.watchlist.Stocks = watchlist.AddToWatchlist(m.watchlist.Stocks, code, name, market)
	m.invalidateWatchlistCache()
	m.watchlistIsSorted = false
	m.saveWatchlist()
	return true
}

// removeFromWatchlist removes a stock from the watchlist
func (m *Model) removeFromWatchlist(index int) {
	m.watchlist.Stocks = watchlist.RemoveFromWatchlist(m.watchlist.Stocks, index)
	m.invalidateWatchlistCache()
	m.watchlistIsSorted = false
	m.saveWatchlist()
}

// ============================================================================
// Tag State Handlers
// ============================================================================

// handleWatchlistTagging handles tag input state

// handleWatchlistGroupSelect handles group selection state

// ============================================================================
// View Functions (Adapters)
// ============================================================================

// viewWatchlistTagging renders tag input view

// renderCurrentFilterStatus renders current filter status
func (m *Model) renderCurrentFilterStatus() string {
	return watchlist.RenderCurrentFilterStatus(
		m.selectedMarketFilter,
		m.selectedUserTagFilter,
		m.getMarketTagName,
		m.getText("watchlist.currentFilter"),
	)
}

// viewWatchlistGroupSelect renders group selection view

// getGroupHelpText returns help text based on filter selection step
func (m *Model) getGroupHelpText() string {
	if m.filterSelectionStep == 0 {
		return m.getText("group.helpText.step1")
	}
	return m.getText("group.helpText.step2")
}
