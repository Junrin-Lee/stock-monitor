package main

import (
	"fmt"
	"stock-monitor/internal/api"
	"stock-monitor/internal/ui/watchlist"
	"stock-monitor/internal/ui"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
func (m *Model) resetWatchlistCursor() {
	filteredStocks := m.getFilteredWatchlist()
	maxWatchlistLines := m.config.Display.MaxLines

	pos := watchlist.ResetWatchlistCursor(len(filteredStocks), maxWatchlistLines)
	m.watchlistCursor = pos.Cursor
	m.watchlistScrollPos = pos.ScrollPos
}

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
func (m *Model) handleWatchlistTagging(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.tagInput == "" {
			// Return to tag management view
			m.availableTags = m.getAvailableTags()
			m.state = WatchlistTagManage
			m.tagManageCursor = 0
			return m, nil
		}

		// Update tags for selected stock (based on filtered list)
		filteredStocks := m.getFilteredWatchlist()
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			stockToTag := filteredStocks[m.watchlistCursor]

			// Find the stock in original list and add tags
			for i, stock := range m.watchlist.Stocks {
				if stock.Code == stockToTag.Code {
					// Handle multiple tags (comma separated)
					newTags := strings.Split(m.tagInput, ",")
					for _, tag := range newTags {
						tag = strings.TrimSpace(tag)
						if tag != "" && tag != "-" {
							watchlist.AddTag(&m.watchlist.Stocks[i], tag)
						}
					}
					// Ensure at least default tag if no valid tags
					if len(m.watchlist.Stocks[i].Tags) == 0 {
						m.watchlist.Stocks[i].Tags = []string{"-"}
					}

					// Update current stock tags list
					m.currentStockTags = make([]string, 0)
					for _, tag := range m.watchlist.Stocks[i].Tags {
						if tag != "" && tag != "-" {
							m.currentStockTags = append(m.currentStockTags, tag)
						}
					}
					break
				}
			}

			m.invalidateWatchlistCache()
			m.saveWatchlist()

			if m.language == Chinese {
				m.message = fmt.Sprintf("已为 %s 添加标签: %s",
					stockToTag.Name, m.tagInput)
			} else {
				m.message = fmt.Sprintf("Added tags to %s: %s",
					stockToTag.Name, m.tagInput)
			}
		}

		// Return to tag management view
		m.availableTags = m.getAvailableTags()
		m.state = WatchlistTagManage
		m.tagManageCursor = 0
		m.tagInput = ""
		m.tagInputCursor = 0
		return m, nil
	case "esc", "q":
		// Return to tag management view
		m.availableTags = m.getAvailableTags()
		m.state = WatchlistTagManage
		m.tagManageCursor = 0
		m.tagInput = ""
		m.tagInputCursor = 0
		m.message = ""
		return m, nil
	case "left", "ctrl+b":
		if m.tagInputCursor > 0 {
			m.tagInputCursor--
		}
		return m, nil
	case "right", "ctrl+f":
		runes := []rune(m.tagInput)
		if m.tagInputCursor < len(runes) {
			m.tagInputCursor++
		}
		return m, nil
	case "home", "ctrl+a":
		m.tagInputCursor = 0
		return m, nil
	case "end", "ctrl+e":
		m.tagInputCursor = len([]rune(m.tagInput))
		return m, nil
	case "backspace":
		m.tagInput, m.tagInputCursor = ui.DeleteRuneBeforeCursor(m.tagInput, m.tagInputCursor)
		return m, nil
	case "delete", "ctrl+d":
		m.tagInput, m.tagInputCursor = ui.DeleteRuneAtCursor(m.tagInput, m.tagInputCursor)
		return m, nil
	default:
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !ui.IsControlKey(str) {
			m.tagInput, m.tagInputCursor = ui.InsertStringAtCursor(m.tagInput, m.tagInputCursor, str)
		}
		return m, nil
	}
}

// handleWatchlistGroupSelect handles group selection state
func (m *Model) handleWatchlistGroupSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	marketTags := m.getMarketTags()
	userTags := m.getUserTags()

	switch msg.String() {
	case "enter":
		if m.filterSelectionStep == 0 {
			// First stage: select market
			if m.cursor == 0 {
				m.selectedMarketFilter = ""
			} else if m.cursor > 0 && m.cursor <= len(marketTags) {
				selectedMarketTag := marketTags[m.cursor-1]
				// Convert market tag to MarketType
				switch selectedMarketTag {
				case m.getText("marketTag.china"):
					m.selectedMarketFilter = MarketChina
				case m.getText("marketTag.us"):
					m.selectedMarketFilter = MarketUS
				case m.getText("marketTag.hongkong"):
					m.selectedMarketFilter = MarketHongKong
				}
			}
			// Move to second stage
			m.filterSelectionStep = 1
			m.cursor = 0
			return m, nil

		} else {
			// Second stage: select user tag
			if m.cursor == 0 {
				m.selectedUserTagFilter = ""
			} else if m.cursor > 0 && m.cursor <= len(userTags) {
				m.selectedUserTagFilter = userTags[m.cursor-1]
				m.lastSelectedGroupTag = m.selectedUserTagFilter
			}

			m.invalidateWatchlistCache()
			m.filterSelectionStep = 0
			m.state = WatchlistViewing
			m.resetWatchlistCursor()
			m.message = ""
			return m, m.tickCmd()
		}

	case "esc", "q":
		m.filterSelectionStep = 0
		m.selectedMarketFilter = ""
		m.state = WatchlistViewing
		m.message = ""
		return m, m.tickCmd()

	case "c":
		// Clear all filters
		m.selectedMarketFilter = ""
		m.selectedUserTagFilter = ""
		m.invalidateWatchlistCache()
		m.filterSelectionStep = 0
		m.state = WatchlistViewing
		m.resetWatchlistCursor()
		m.message = ""
		return m, m.tickCmd()

	case "b", "backspace":
		// Return to first stage if in second stage
		if m.filterSelectionStep == 1 {
			m.filterSelectionStep = 0
			m.cursor = 0
			m.selectedMarketFilter = ""
			return m, nil
		}
		return m, nil

	case "up", "k", "w":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j", "s":
		maxCursor := 0
		if m.filterSelectionStep == 0 {
			maxCursor = len(marketTags)
		} else {
			maxCursor = len(userTags)
		}

		if m.cursor < maxCursor {
			m.cursor++
		}
		return m, nil
	}

	return m, nil
}

// ============================================================================
// View Functions (Adapters)
// ============================================================================

// viewWatchlistTagging renders tag input view
func (m *Model) viewWatchlistTagging() string {
	filteredStocks := m.getFilteredWatchlist()
	if m.watchlistCursor < 0 || m.watchlistCursor >= len(filteredStocks) {
		return ""
	}

	stock := filteredStocks[m.watchlistCursor]
	marketTag := m.getMarketTagName(stock.Market)

	params := watchlist.TaggingViewParams{
		Title:          m.getText("watchlist.setTag"),
		StockName:      stock.Name,
		StockCode:      stock.Code,
		MarketLabel:    m.getText("marketInfo"),
		MarketTag:      marketTag,
		CurrentTags:    watchlist.GetTagsDisplay(&stock, marketTag),
		InputPrompt:    m.getText("watchlist.enterTags"),
		TagInput:       m.tagInput,
		TagInputCursor: m.tagInputCursor,
		HelpText:       m.getText("watchlist.tagHelp"),
	}

	return watchlist.RenderTaggingView(params, ui.FormatTextWithCursor)
}

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
func (m *Model) viewWatchlistGroupSelect() string {
	marketTags := m.getMarketTags()
	userTags := m.getUserTags()

	params := watchlist.GroupSelectViewParams{
		Title:            m.getText("watchlist.selectGroup"),
		MarketTagsTitle:  m.getText("group.marketTags"),
		UserTagsTitle:    m.getText("group.userTags"),
		AllMarketsText:   m.getText("filter.allMarkets"),
		AllTagsText:      m.getText("filter.allTags"),
		NoTagsText:       m.getText("watchlist.noTags"),
		HelpText:         m.getGroupHelpText(),
		MarketTags:       marketTags,
		UserTags:         userTags,
		Cursor:           m.cursor,
		FilterStep:       m.filterSelectionStep,
		SelectedMarket:   m.selectedMarketFilter,
		GetMarketTagName: m.getMarketTagName,
	}

	return watchlist.RenderGroupSelectView(params)
}

// getGroupHelpText returns help text based on filter selection step
func (m *Model) getGroupHelpText() string {
	if m.filterSelectionStep == 0 {
		return m.getText("group.helpText.step1")
	}
	return m.getText("group.helpText.step2")
}
