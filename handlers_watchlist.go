package main

import (
	"fmt"
	"stock-monitor/internal/ui"
	"stock-monitor/internal/ui/watchlist"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ============================================================================
// Watchlist Viewing Handlers
// ============================================================================

func (m *Model) handleWatchlistViewing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m":
		m.stopIntradayDataCollection() // Stop intraday data collection
		m.state = MainMenu
		m.message = ""
		return m, nil
	case "d":
		// Delete stock at cursor from watchlist
		filteredStocks := m.getFilteredWatchlist()
		if len(filteredStocks) == 0 {
			m.message = m.getText("emptyWatchlist")
			return m, nil
		}

		// Get stock to remove (from filtered list)
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			stockToRemove := filteredStocks[m.watchlistCursor]

			// Find and delete stock from original list
			for i, stock := range m.watchlist.Stocks {
				if stock.Code == stockToRemove.Code {
					m.removeFromWatchlist(i)
					break
				}
			}

			// Adjust cursor position (based on filtered list)
			newFilteredStocks := m.getFilteredWatchlist()
			if m.watchlistCursor >= len(newFilteredStocks) && len(newFilteredStocks) > 0 {
				m.watchlistCursor = len(newFilteredStocks) - 1
			}

			m.message = fmt.Sprintf(m.getText("removeWatchSuccess"), stockToRemove.Name, stockToRemove.Code)
		}
		return m, nil
	case "v":
		// View intraday chart
		filteredStocks := m.getFilteredWatchlist()
		if len(filteredStocks) == 0 {
			m.message = m.getText("emptyWatchlist")
			return m, nil
		}
		selectedStock := filteredStocks[m.watchlistCursor]
		m.chartViewStock = selectedStock.Code
		m.chartViewStockName = selectedStock.Name

		// Get smart date (consistent with worker logic)
		actualDate, _, err := GetTradingDayForCollection(selectedStock.Code, m)
		if err != nil {
			// Fallback to simple logic
			actualDate = getSmartChartDate()
		}
		m.chartViewDate = actualDate
		m.previousState = WatchlistViewing

		// Try to load data
		data, loadErr := m.loadIntradayDataForDate(
			selectedStock.Code,
			selectedStock.Name,
			actualDate,
		)

		if loadErr != nil {
			// No data - trigger collection
			m.chartData = nil
			m.chartLoadError = nil
			m.state = IntradayChartViewing
			return m, m.triggerIntradayDataCollection(
				selectedStock.Code,
				selectedStock.Name,
				actualDate,
			)
		}

		// Data exists - create chart
		m.chartData = data
		m.chartLoadError = nil
		m.chartIsCollecting = false
		m.state = IntradayChartViewing
		return m, nil
	case "a":
		// Jump to stock search page
		logInfo("log.action.watchlistSearch")
		m.state = SearchingStock
		m.searchInput = ""
		m.searchResult = nil
		m.searchFromWatchlist = true
		m.message = ""
		return m, nil
	case "s":
		// Enter sort menu
		logInfo("log.action.watchlistSort")
		m.state = WatchlistSorting
		// Smart position cursor to current sort field
		m.watchlistSortCursor = m.findSortFieldIndex(m.watchlistSortField, false)
		m.message = ""
		return m, nil
	case "t":
		// Manage tags for current selected stock - enter tag management
		filteredStocks := m.getFilteredWatchlist()
		if len(filteredStocks) == 0 {
			m.message = m.getText("emptyWatchlist")
			return m, nil
		}

		// Get current selected stock's tag info
		currentStock := filteredStocks[m.watchlistCursor]
		m.currentStockTags = make([]string, 0)
		for _, tag := range currentStock.Tags {
			if tag != "" && tag != "-" {
				m.currentStockTags = append(m.currentStockTags, tag)
			}
		}

		// Get all available tags
		m.availableTags = m.getAvailableTags()
		m.state = WatchlistTagManage
		m.tagManageCursor = 0
		m.tagInput = ""
		m.isInRemoveMode = false
		m.message = ""
		return m, nil
	case "l", "L":
		// View stock alert details
		filteredStocks := m.getFilteredWatchlist()
		if len(filteredStocks) == 0 {
			m.message = m.getText("emptyWatchlist")
			return m, nil
		}

		currentStock := filteredStocks[m.watchlistCursor]

		// Set stock alert detail parameters
		m.stockAlertCode = currentStock.Code
		m.stockAlertName = currentStock.Name
		m.stockAlertCursor = 0
		m.previousState = WatchlistViewing // Record return state

		// Get all alerts for this stock
		m.stockAlertAlerts = m.getStockAlerts(currentStock.Code)

		logInfo("log.action.enterStockAlertManagement", currentStock.Code, currentStock.Name)
		m.state = StockAlertManage
		m.message = ""
		return m, nil
	case "g":
		// Group view (v5.8: two-stage selection with position memory)
		m.tagGroups = m.getTagGroups()
		totalTags := m.getTotalTagCount()

		if totalTags == 0 {
			m.message = m.getText("watchlist.noTags")
			return m, nil
		}

		// Determine initial stage and cursor position based on current filter state
		if m.selectedMarketFilter != "" || m.selectedUserTagFilter != "" {
			// Has filter: restore last selected position
			if m.selectedUserTagFilter != "" {
				// User tag selected: enter second stage, position cursor to that tag
				m.filterSelectionStep = 1
				userTagPos := m.findUserTagPosition(m.selectedUserTagFilter)
				if userTagPos >= 0 {
					m.cursor = userTagPos
				} else {
					m.cursor = 0 // Default to first item if not found
				}
			} else {
				// Only market selected: stay in first stage, position cursor to that market
				m.filterSelectionStep = 0
				marketPos := m.findMarketTagPosition(m.selectedMarketFilter)
				if marketPos >= 0 {
					m.cursor = marketPos
				} else {
					m.cursor = 0 // Default to first item if not found
				}
			}
		} else {
			// No filter: start from beginning
			m.filterSelectionStep = 0
			m.cursor = 0
		}

		m.state = WatchlistGroupSelect
		m.message = ""
		return m, nil
	case "c":
		// Clear all filters (market filter and user tag filter)
		if m.selectedMarketFilter != "" || m.selectedUserTagFilter != "" {
			m.selectedMarketFilter = ""
			m.selectedUserTagFilter = ""
			m.invalidateWatchlistCache() // Invalidate cache
			m.resetWatchlistCursor()     // Reset cursor to first stock
			if m.language == Chinese {
				m.message = "已清除所有过滤条件"
			} else {
				m.message = "All filters cleared"
			}
		}
		return m, nil
	case "up", "k", "w":
		// Get filtered list once to avoid repeated calls
		filteredStocks := m.getFilteredWatchlist()
		m.watchlistCursor = MoveCursorUp(m.watchlistCursor)
		// Adjust scroll only when cursor moves
		m.adjustWatchlistScroll(filteredStocks)
		return m, nil
	case "down", "j":
		// Get filtered list once to avoid repeated calls
		filteredStocks := m.getFilteredWatchlist()
		m.watchlistCursor = MoveCursorDown(m.watchlistCursor, len(filteredStocks)-1)
		// Adjust scroll only when cursor moves
		m.adjustWatchlistScroll(filteredStocks)
		return m, nil
	}
	return m, nil
}

func (m *Model) viewWatchlistViewing() string {
	s := m.getText("watchlistTitle") + "\n"
	s += fmt.Sprintf(m.getText("updateTime"), m.lastUpdate.Format("2006-01-02 15:04:05")) + "\n"

	// Display current combined filter status
	if m.selectedMarketFilter != "" || m.selectedUserTagFilter != "" {
		var filterParts []string
		if m.selectedMarketFilter != "" {
			filterParts = append(filterParts, m.getMarketTagName(m.selectedMarketFilter))
		}
		if m.selectedUserTagFilter != "" {
			filterParts = append(filterParts, m.selectedUserTagFilter)
		}

		if m.language == Chinese {
			s += fmt.Sprintf("当前过滤: %s\n", strings.Join(filterParts, " + "))
		} else {
			s += fmt.Sprintf("Current filter: %s\n", strings.Join(filterParts, " + "))
		}
	}
	s += "\n"

	// Get filtered stock list
	filteredStocks := m.getFilteredWatchlist()

	if len(filteredStocks) == 0 {
		if m.selectedMarketFilter != "" || m.selectedUserTagFilter != "" {
			var filterDesc string
			if m.selectedMarketFilter != "" && m.selectedUserTagFilter != "" {
				filterDesc = m.getMarketTagName(m.selectedMarketFilter) + " + " + m.selectedUserTagFilter
			} else if m.selectedMarketFilter != "" {
				filterDesc = m.getMarketTagName(m.selectedMarketFilter)
			} else {
				filterDesc = m.selectedUserTagFilter
			}

			if m.language == Chinese {
				s += fmt.Sprintf("过滤条件 '%s' 下没有股票\n\n", filterDesc)
				s += "按G键选择其他过滤条件，或按C键清除过滤\n"
			} else {
				s += fmt.Sprintf("No stocks under filter '%s'\n\n", filterDesc)
				s += "Press G to select other filters, or C to clear filter\n"
			}
		} else {
			s += m.getText("emptyWatchlist") + "\n\n"
			s += m.getText("addToWatchFirst") + "\n\n"
		}
		s += m.getText("watchlistHelp") + "\n"
		return s
	}

	// Display scroll information
	totalWatchStocks := len(filteredStocks)
	maxWatchlistLines := m.config.Display.MaxLines
	if totalWatchStocks > 0 {
		currentPos := m.watchlistCursor + 1 // Display position starting from 1
		if m.language == Chinese {
			s += fmt.Sprintf("⭐ 自选列表 (%d/%d) [↑/↓:翻页]\n", currentPos, totalWatchStocks)
		} else {
			s += fmt.Sprintf("⭐ Watchlist (%d/%d) [↑/↓:scroll]\n", currentPos, totalWatchStocks)
		}
		s += "\n"
	}

	// Create table to display watchlist
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// Get header with sort indicator
	t.AppendHeader(m.GenerateWatchlistHeader())

	// Calculate range of stocks to display
	endIndex := len(filteredStocks) - m.watchlistScrollPos
	startIndex := endIndex - maxWatchlistLines
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(filteredStocks) {
		endIndex = len(filteredStocks)
	}

	for i := startIndex; i < endIndex; i++ {
		watchStock := filteredStocks[i]
		// Get stock price from cache (non-blocking)
		stockData := m.getStockPriceFromCache(watchStock.Code)

		// Use dynamic column renderer to generate row
		row := m.GenerateWatchlistRow(&watchStock, stockData, i, startIndex, endIndex)
		t.AppendRow(row)

		// Add separator after each stock (except last in visible range)
		if i < endIndex-1 {
			t.AppendSeparator()
		}
	}

	s += t.Render() + "\n"

	// If scrolling is available, show scroll indicators
	if totalWatchStocks > maxWatchlistLines {
		s += "\n" + strings.Repeat("-", 80) + "\n"
		if m.watchlistScrollPos > 0 {
			if m.language == Chinese {
				s += "↑ 有更新的自选股票 (按↓查看)\n"
			} else {
				s += "↑ Newer watchlist stocks available (press ↓)\n"
			}
		}
		if m.watchlistScrollPos < totalWatchStocks-1 {
			if m.language == Chinese {
				s += "↓ 有更多历史自选股票 (按↑查看)\n"
			} else {
				s += "↓ More watchlist stocks available (press ↑)\n"
			}
		}
	}

	// Use unified help text
	s += "\n" + m.getText("watchlistHelp") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// resetWatchlistCursor resets watchlist cursor to first stock
func (m *Model) resetWatchlistCursor() {
	filteredStocks := m.getFilteredWatchlist()
	if len(filteredStocks) > 0 {
		m.watchlistCursor = 0
		maxWatchlistLines := m.config.Display.MaxLines
		if len(filteredStocks) > maxWatchlistLines {
			// Display first N lines: set scroll position to show N lines starting from index 0
			m.watchlistScrollPos = len(filteredStocks) - maxWatchlistLines
		} else {
			// Stock count doesn't exceed display lines, show all
			m.watchlistScrollPos = 0
		}
	}
}

// ============================================================================
// Watchlist Tag Management Handlers
// ============================================================================

func (m *Model) handleWatchlistTagManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = WatchlistViewing
		m.message = ""
		m.resetWatchlistCursor()
		return m, m.tickCmd() // Restart timer
	case "n":
		// Manually input new tag
		m.state = WatchlistTagging
		m.tagInput = ""
		return m, nil
	case "d":
		// Delete currently selected tag (if current stock has it)
		if len(m.availableTags) == 0 {
			if m.language == Chinese {
				m.message = "没有可删除的标签"
			} else {
				m.message = "No tags to remove"
			}
			return m, nil
		}

		// Get currently selected tag
		selectedTag := m.availableTags[m.tagManageCursor]

		// Check if current stock has this tag
		filteredStocks := m.getFilteredWatchlist()
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			currentStock := filteredStocks[m.watchlistCursor]

			// Find and delete tag
			stockFound := false
			for i, stock := range m.watchlist.Stocks {
				if stock.Code == currentStock.Code {
					if watchlist.HasTag(&stock, selectedTag) {
						watchlist.RemoveTag(&m.watchlist.Stocks[i], selectedTag)
						m.saveWatchlist()
						m.invalidateWatchlistCache()

						// Update current stock tag list
						m.currentStockTags = make([]string, 0)
						for _, tag := range m.watchlist.Stocks[i].Tags {
							if tag != "" && tag != "-" {
								m.currentStockTags = append(m.currentStockTags, tag)
							}
						}

						// Update available tags list
						m.availableTags = m.getAvailableTags()

						// Adjust cursor position
						if m.tagManageCursor >= len(m.availableTags) && len(m.availableTags) > 0 {
							m.tagManageCursor = len(m.availableTags) - 1
						}

						if m.language == Chinese {
							m.message = fmt.Sprintf("已删除标签: %s", selectedTag)
						} else {
							m.message = fmt.Sprintf("Removed tag: %s", selectedTag)
						}
						stockFound = true
					} else {
						if m.language == Chinese {
							m.message = fmt.Sprintf("该股票没有标签: %s", selectedTag)
						} else {
							m.message = fmt.Sprintf("Stock doesn't have tag: %s", selectedTag)
						}
						stockFound = true
					}
					break
				}
			}

			if !stockFound {
				if m.language == Chinese {
					m.message = "找不到对应的股票"
				} else {
					m.message = "Stock not found"
				}
			}
		}
		return m, nil
	case "e":
		// Edit currently selected tag
		if len(m.availableTags) == 0 {
			if m.language == Chinese {
				m.message = "没有可编辑的标签"
			} else {
				m.message = "No tags to edit"
			}
			return m, nil
		}

		// Get currently selected tag
		selectedTag := m.availableTags[m.tagManageCursor]

		// Enter tag edit state
		m.state = WatchlistTagEdit
		m.tagToEdit = selectedTag
		m.tagEditInput = selectedTag                    // Prefill with current tag name
		m.tagEditInputCursor = len([]rune(selectedTag)) // Cursor at end
		m.message = ""
		return m, nil
	case "up", "k", "w":
		if len(m.availableTags) > 0 && m.tagManageCursor > 0 {
			m.tagManageCursor--
		}
		return m, nil
	case "down", "j", "s":
		if len(m.availableTags) > 0 && m.tagManageCursor < len(m.availableTags)-1 {
			m.tagManageCursor++
		}
		return m, nil
	case "enter":
		// Add selected tag to current stock
		if len(m.availableTags) == 0 {
			if m.language == Chinese {
				m.message = "没有可添加的标签，按N键创建新标签"
			} else {
				m.message = "No tags to add, press N to create new tag"
			}
			return m, nil
		}

		selectedTag := m.availableTags[m.tagManageCursor]

		// Get currently selected stock
		filteredStocks := m.getFilteredWatchlist()
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			currentStock := filteredStocks[m.watchlistCursor]

			// Find and add tag
			stockFound := false
			for i, stock := range m.watchlist.Stocks {
				if stock.Code == currentStock.Code {
					if !watchlist.HasTag(&stock, selectedTag) {
						watchlist.AddTag(&m.watchlist.Stocks[i], selectedTag)
						m.saveWatchlist()
						m.invalidateWatchlistCache()

						// Update current stock tag list
						m.currentStockTags = make([]string, 0)
						for _, tag := range m.watchlist.Stocks[i].Tags {
							if tag != "" && tag != "-" {
								m.currentStockTags = append(m.currentStockTags, tag)
							}
						}

						if m.language == Chinese {
							m.message = fmt.Sprintf("已添加标签: %s", selectedTag)
						} else {
							m.message = fmt.Sprintf("Added tag: %s", selectedTag)
						}
					} else {
						if m.language == Chinese {
							m.message = fmt.Sprintf("该股票已有标签: %s", selectedTag)
						} else {
							m.message = fmt.Sprintf("Stock already has tag: %s", selectedTag)
						}
					}
					stockFound = true
					break
				}
			}

			if !stockFound {
				if m.language == Chinese {
					m.message = "找不到对应的股票"
				} else {
					m.message = "Stock not found"
				}
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) viewWatchlistTagManage() string {
	var s string

	if m.language == Chinese {
		s += "=== 标签管理 ===\n\n"
	} else {
		s += "=== Tag Management ===\n\n"
	}

	filteredStocks := m.getFilteredWatchlist()
	if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
		stock := filteredStocks[m.watchlistCursor]
		if m.language == Chinese {
			s += fmt.Sprintf("股票: %s (%s)\n", stock.Name, stock.Code)
			marketTag := m.getMarketTagName(stock.Market)
			s += fmt.Sprintf("当前标签: %s\n\n", watchlist.GetTagsDisplay(&stock, marketTag))
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n", stock.Name, stock.Code)
			marketTag := m.getMarketTagName(stock.Market)
			s += fmt.Sprintf("Current tags: %s\n\n", watchlist.GetTagsDisplay(&stock, marketTag))
		}

		// Display all available tags, mark those owned by current stock
		if len(m.availableTags) > 0 {
			if m.language == Chinese {
				s += "所有可用标签:\n"
			} else {
				s += "All available tags:\n"
			}

			for i, tag := range m.availableTags {
				cursor := "  "
				if i == m.tagManageCursor {
					cursor = "► "
				}

				// Check if current stock has this tag
				hasTag := watchlist.HasTag(&stock, tag)
				status := ""
				if hasTag {
					if m.language == Chinese {
						status = " ✓ (已拥有)"
					} else {
						status = " ✓ (owned)"
					}
				}

				s += fmt.Sprintf("%s%s%s\n", cursor, tag, status)
			}
			s += "\n"
		} else {
			if m.language == Chinese {
				s += "暂无可用标签，按N键创建新标签\n\n"
			} else {
				s += "No available tags, press N to create new tag\n\n"
			}
		}

		// Operation instructions
		if m.language == Chinese {
			s += "操作说明:\n"
			s += "  ↑↓ - 选择标签\n"
			s += "  Enter - 添加/切换选中标签\n"
			s += "  D - 删除选中标签(如果当前股票拥有)\n"
			s += "  E - 编辑选中标签(批量修改所有使用该标签的股票)\n"
			s += "  N - 创建新标签\n"
			s += "  ESC/Q - 返回自选列表\n"
		} else {
			s += "Actions:\n"
			s += "  ↑↓ - Select tag\n"
			s += "  Enter - Add/toggle selected tag\n"
			s += "  D - Remove selected tag (if owned by current stock)\n"
			s += "  E - Edit selected tag (batch update all stocks with this tag)\n"
			s += "  N - Create new tag\n"
			s += "  ESC/Q - Return to watchlist\n"
		}
	}

	return s
}

// ============================================================================
// Watchlist Tag Input Handlers
// ============================================================================

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

// ============================================================================
// Watchlist Tag Select Handlers (Legacy)
// ============================================================================

func (m *Model) handleWatchlistTagSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute operation based on current selection
		if m.tagSelectCursor == len(m.availableTags) {
			// Selected "manually input new tag" option
			m.state = WatchlistTagging
			m.tagInput = ""
			return m, nil
		} else if m.tagSelectCursor >= 0 && m.tagSelectCursor < len(m.availableTags) {
			// Selected existing tag
			selectedTag := m.availableTags[m.tagSelectCursor]

			// Update current selected stock's tag (based on filtered list)
			filteredStocks := m.getFilteredWatchlist()
			if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
				stockToTag := filteredStocks[m.watchlistCursor]

				// Find the stock in original list and add tag
				for i, stock := range m.watchlist.Stocks {
					if stock.Code == stockToTag.Code {
						watchlist.AddTag(&m.watchlist.Stocks[i], selectedTag)
						break
					}
				}

				m.invalidateWatchlistCache() // Invalidate cache
				m.saveWatchlist()

				if m.language == Chinese {
					m.message = fmt.Sprintf("已为 %s 添加标签: %s",
						stockToTag.Name, selectedTag)
				} else {
					m.message = fmt.Sprintf("Added tag to %s: %s",
						stockToTag.Name, selectedTag)
				}
			}

			m.state = WatchlistViewing
			m.tagInput = ""
			m.resetWatchlistCursor() // Reset cursor to first stock
			return m, m.tickCmd()    // Restart timer
		}
		return m, nil
	case "d":
		// Enter tag deletion selection mode
		filteredStocks := m.getFilteredWatchlist()
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			stockToModify := filteredStocks[m.watchlistCursor]

			// Get valid tags for this stock (exclude default tag)
			var validTags []string
			for _, stock := range m.watchlist.Stocks {
				if stock.Code == stockToModify.Code {
					for _, tag := range stock.Tags {
						if tag != "" && tag != "-" {
							validTags = append(validTags, tag)
						}
					}
					break
				}
			}

			if len(validTags) == 0 {
				if m.language == Chinese {
					m.message = fmt.Sprintf("%s 没有可删除的标签", stockToModify.Name)
				} else {
					m.message = fmt.Sprintf("%s has no tags to remove", stockToModify.Name)
				}
				return m, nil
			}

			// Set tag deletion state
			m.currentStockTags = validTags
			m.tagRemoveCursor = 0
			m.state = WatchlistTagRemoveSelect
			return m, nil
		}
		return m, nil
	case "esc", "q":
		m.state = WatchlistViewing
		m.tagInput = ""
		m.message = ""
		m.resetWatchlistCursor() // Reset cursor to first stock
		return m, m.tickCmd()    // Restart timer
	case "up", "k", "w":
		m.tagSelectCursor = MoveCursorUp(m.tagSelectCursor)
		return m, nil
	case "down", "j", "s":
		maxCursor := len(m.availableTags) // Include "manually input new tag" option
		m.tagSelectCursor = MoveCursorDown(m.tagSelectCursor, maxCursor)
		return m, nil
	}
	return m, nil
}

func (m *Model) viewWatchlistTagSelect() string {
	var s string

	if m.language == Chinese {
		s += "=== 管理标签 ===\n\n"
	} else {
		s += "=== Manage Tags ===\n\n"
	}

	filteredStocks := m.getFilteredWatchlist()
	if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
		stock := filteredStocks[m.watchlistCursor]
		if m.language == Chinese {
			s += fmt.Sprintf("股票: %s (%s)\n", stock.Name, stock.Code)
			marketTag := m.getMarketTagName(stock.Market)
			s += fmt.Sprintf("当前标签: %s\n\n", watchlist.GetTagsDisplay(&stock, marketTag))

			// Display stock's tags for deletion
			if len(stock.Tags) > 0 {
				hasValidTags := false
				for _, tag := range stock.Tags {
					if tag != "" && tag != "-" {
						hasValidTags = true
						break
					}
				}
				if hasValidTags {
					s += "当前标签(按D键删除):\n"
					for _, tag := range stock.Tags {
						if tag != "" && tag != "-" {
							s += fmt.Sprintf("  • %s\n", tag)
						}
					}
					s += "\n"
				}
			}
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n", stock.Name, stock.Code)
			marketTag := m.getMarketTagName(stock.Market)
			s += fmt.Sprintf("Current tags: %s\n\n", watchlist.GetTagsDisplay(&stock, marketTag))

			// Display stock's tags for deletion
			if len(stock.Tags) > 0 {
				hasValidTags := false
				for _, tag := range stock.Tags {
					if tag != "" && tag != "-" {
						hasValidTags = true
						break
					}
				}
				if hasValidTags {
					s += "Current tags (press D to remove):\n"
					for _, tag := range stock.Tags {
						if tag != "" && tag != "-" {
							s += fmt.Sprintf("  • %s\n", tag)
						}
					}
					s += "\n"
				}
			}
		}
	}

	// Display existing tag options
	if len(m.availableTags) > 0 {
		if m.language == Chinese {
			s += "可添加的系统标签:\n"
		} else {
			s += "Available system tags to add:\n"
		}

		for i, tag := range m.availableTags {
			cursor := "  "
			if i == m.tagSelectCursor {
				cursor = "► "
			}
			s += fmt.Sprintf("%s%s\n", cursor, tag)
		}
		s += "\n"
	}

	// Add "manually input new tag" option
	cursor := "  "
	if m.tagSelectCursor == len(m.availableTags) {
		cursor = "► "
	}
	if m.language == Chinese {
		s += fmt.Sprintf("%s手动输入新标签\n\n", cursor)
		s += "操作: ↑↓选择 Enter添加标签 D进入删除模式 ESC/Q取消"
	} else {
		s += fmt.Sprintf("%sManually enter new tag\n\n", cursor)
		s += "Actions: ↑↓ select, Enter add tag, D enter remove mode, ESC/Q cancel"
	}

	return s
}

// ============================================================================
// Watchlist Tag Remove Handlers
// ============================================================================

func (m *Model) handleWatchlistTagRemoveSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = WatchlistTagManage
		return m, nil
	case "enter":
		if m.tagRemoveCursor >= 0 && m.tagRemoveCursor < len(m.currentStockTags) {
			tagToRemove := m.currentStockTags[m.tagRemoveCursor]

			// Remove selected tag from current stock
			filteredStocks := m.getFilteredWatchlist()
			if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
				stockToModify := filteredStocks[m.watchlistCursor]

				// Find the stock in original list and remove specified tag
				for i, stock := range m.watchlist.Stocks {
					if stock.Code == stockToModify.Code {
						watchlist.RemoveTag(&m.watchlist.Stocks[i], tagToRemove)
						// If no tags remain, add default tag
						if len(m.watchlist.Stocks[i].Tags) == 0 {
							m.watchlist.Stocks[i].Tags = []string{"-"}
						}
						break
					}
				}

				m.invalidateWatchlistCache()
				m.saveWatchlist()

				// Update current stock tags list
				m.currentStockTags = make([]string, 0)
				for _, stock := range m.watchlist.Stocks {
					if stock.Code == stockToModify.Code {
						for _, tag := range stock.Tags {
							if tag != "" && tag != "-" {
								m.currentStockTags = append(m.currentStockTags, tag)
							}
						}
						break
					}
				}

				if m.language == Chinese {
					m.message = fmt.Sprintf("已从 %s 删除标签: %s", stockToModify.Name, tagToRemove)
				} else {
					m.message = fmt.Sprintf("Removed tag from %s: %s", stockToModify.Name, tagToRemove)
				}

				// If no more tags to delete, return to tag management
				if len(m.currentStockTags) == 0 {
					m.state = WatchlistTagManage
				} else {
					// Adjust cursor position
					if m.tagRemoveCursor >= len(m.currentStockTags) {
						m.tagRemoveCursor = len(m.currentStockTags) - 1
					}
				}
			}
		}
		return m, nil
	case "up", "k", "w":
		m.tagRemoveCursor = MoveCursorUp(m.tagRemoveCursor)
		return m, nil
	case "down", "j", "s":
		m.tagRemoveCursor = MoveCursorDown(m.tagRemoveCursor, len(m.currentStockTags)-1)
		return m, nil
	}
	return m, nil
}

func (m *Model) viewWatchlistTagRemoveSelect() string {
	var s string

	if m.language == Chinese {
		s += "=== 选择要删除的标签 ===\n\n"
	} else {
		s += "=== Select Tag to Remove ===\n\n"
	}

	filteredStocks := m.getFilteredWatchlist()
	if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
		stock := filteredStocks[m.watchlistCursor]
		if m.language == Chinese {
			s += fmt.Sprintf("股票: %s (%s)\n\n", stock.Name, stock.Code)
			s += "请选择要删除的标签:\n\n"
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n\n", stock.Name, stock.Code)
			s += "Select tag to remove:\n\n"
		}

		// Display deletable tags
		for i, tag := range m.currentStockTags {
			cursor := "  "
			if i == m.tagRemoveCursor {
				cursor = "► "
			}
			s += fmt.Sprintf("%s%s\n", cursor, tag)
		}

		s += "\n"
		if m.language == Chinese {
			s += "操作: ↑↓选择标签 Enter删除 ESC/Q取消"
		} else {
			s += "Actions: ↑↓ select tag, Enter remove, ESC/Q cancel"
		}
	}

	return s
}

// ============================================================================
// Watchlist Tag Edit Handlers
// ============================================================================

func (m *Model) handleWatchlistTagEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Cancel edit, return to tag management
		m.state = WatchlistTagManage
		m.message = m.getText("tagEditCanceled")
		m.tagEditInput = ""
		m.tagEditInputCursor = 0
		m.tagToEdit = ""
		return m, nil
	case "enter":
		// Confirm edit
		newTagName := strings.TrimSpace(m.tagEditInput)

		// Validate new tag name
		if newTagName == "" {
			m.message = m.getText("tagNameRequired")
			return m, nil
		}

		// Check if same as original tag
		if newTagName == m.tagToEdit {
			m.message = m.getText("tagNameUnchanged")
			return m, nil
		}

		// Batch update all stocks using this tag
		updatedCount := m.renameTagForAllStocks(m.tagToEdit, newTagName)

		// Save updates
		m.invalidateWatchlistCache()
		m.saveWatchlist()

		// Update available tags list
		m.availableTags = m.getAvailableTags()

		// If current filtered user tag is the modified tag, update filter tag
		if m.selectedUserTagFilter == m.tagToEdit {
			m.selectedUserTagFilter = newTagName
		}

		// Display success message
		m.message = fmt.Sprintf(m.getText("tagEditSuccess"), m.tagToEdit, newTagName, updatedCount)

		// Return to tag management
		m.state = WatchlistTagManage
		m.tagEditInput = ""
		m.tagEditInputCursor = 0
		m.tagToEdit = ""

		return m, nil
	case "left", "ctrl+b":
		// Move cursor left
		m.tagEditInputCursor = MoveCursorUp(m.tagEditInputCursor)
		return m, nil
	case "right", "ctrl+f":
		// Move cursor right
		runes := []rune(m.tagEditInput)
		m.tagEditInputCursor = MoveCursorDown(m.tagEditInputCursor, len(runes))
		return m, nil
	case "home", "ctrl+a":
		// Move cursor to start
		m.tagEditInputCursor = 0
		return m, nil
	case "end", "ctrl+e":
		// Move cursor to end
		m.tagEditInputCursor = len([]rune(m.tagEditInput))
		return m, nil
	case "backspace":
		// Delete character before cursor
		m.tagEditInput, m.tagEditInputCursor = ui.DeleteRuneBeforeCursor(m.tagEditInput, m.tagEditInputCursor)
		return m, nil
	case "delete", "ctrl+d":
		// Delete character at cursor
		m.tagEditInput, m.tagEditInputCursor = ui.DeleteRuneAtCursor(m.tagEditInput, m.tagEditInputCursor)
		return m, nil
	default:
		// Handle text input
		if len(msg.String()) == 1 || (len(msg.String()) > 1 && msg.Type == tea.KeyRunes) {
			m.tagEditInput, m.tagEditInputCursor = ui.InsertStringAtCursor(m.tagEditInput, m.tagEditInputCursor, msg.String())
		}
		return m, nil
	}
}

func (m *Model) viewWatchlistTagEdit() string {
	var s string

	s += m.getText("editTagTitle") + "\n\n"
	s += fmt.Sprintf(m.getText("editingTag"), m.tagToEdit) + "\n\n"
	s += m.getText("enterNewTagName") + ui.FormatTextWithCursor(m.tagEditInput, m.tagEditInputCursor) + "\n\n"

	if m.language == Chinese {
		s += "提示: 修改后将更新所有使用此标签的股票\n"
		s += "操作: ←/→移动光标, Enter确认, ESC/Q取消, Home/End跳转首尾"
	} else {
		s += "Note: All stocks using this tag will be updated\n"
		s += "Actions: ←/→ move cursor, Enter confirm, ESC/Q cancel, Home/End jump"
	}

	if m.message != "" {
		s += "\n\n" + m.message
	}

	return s
}

// ============================================================================
// Watchlist Group Select Handlers
// ============================================================================

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

func (m *Model) viewWatchlistGroupSelect() string {
	marketTags := m.getMarketTags()
	userTags := m.getUserTags()

	params := watchlist.GroupSelectViewParams{
		Title:            m.getText("watchlist.selectGroup"),
		MarketTagsTitle:  m.getText("group.marketTags"),
		UserTagsTitle:    m.getText("group.userTags"),
		AllMarketsText:   m.getText("filter.allMarkets"),
		AllTagsText:      m.getText("filter.allTags"),
		MarketTags:       marketTags,
		UserTags:         userTags,
		Cursor:           m.cursor,
		FilterStep:       m.filterSelectionStep,
		HelpText:         m.getText("group.help"),
	}

	return watchlist.RenderGroupSelectView(params)
}

// ============================================================================
// Watchlist Sorting Handlers
// ============================================================================

func (m *Model) handleWatchlistSorting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sortFields := m.getWatchlistSortFields()

	switch msg.String() {
	case "up", "k", "w":
		m.watchlistSortCursor = MoveCursorUp(m.watchlistSortCursor)
	case "down", "j", "s":
		m.watchlistSortCursor = MoveCursorDown(m.watchlistSortCursor, len(sortFields)-1)
	case "enter", " ":
		// Toggle sort direction or apply sort
		selectedField := sortFields[m.watchlistSortCursor]
		if m.watchlistSortField == selectedField {
			// Toggle sort direction
			if m.watchlistSortDirection == SortAsc {
				m.watchlistSortDirection = SortDesc
			} else {
				m.watchlistSortDirection = SortAsc
			}
		} else {
			// Set new sort field, default ascending
			m.watchlistSortField = selectedField
			m.watchlistSortDirection = SortAsc
		}
		// Execute sort and mark as sorted
		m.optimizedSortWatchlist(m.watchlistSortField, m.watchlistSortDirection)
		m.watchlistIsSorted = true
		m.resetWatchlistCursor()
		// Return to watchlist page
		m.state = WatchlistViewing
		m.message = ""
		return m, m.tickCmd() // Restart timer
	case "c", "C":
		// Clear current sort - reload original data order
		m.watchlistIsSorted = false
		// Clear sort field and direction state
		m.watchlistSortField = SortByCode  // Reset to default
		m.watchlistSortDirection = SortAsc // Reset to default
		// Reload original data order
		m.watchlist = loadWatchlist()
		m.resetWatchlistCursor()
		// Return to watchlist page
		m.state = WatchlistViewing
		m.message = m.getText("sortCleared")
		return m, m.tickCmd() // Restart timer
	case "esc", "q":
		// Return to watchlist page
		m.state = WatchlistViewing
		m.message = ""
		return m, m.tickCmd() // Restart timer
	}
	return m, nil
}

func (m *Model) viewWatchlistSorting() string {
	s := m.getText("sortTitle") + "\n\n"
	s += m.getText("selectSortField") + "\n\n"

	sortFields := m.getWatchlistSortFields()
	for i, field := range sortFields {
		prefix := "  "
		if i == m.watchlistSortCursor {
			prefix = "► "
		}

		fieldName := m.getSortFieldName(field)
		if m.watchlistIsSorted && m.watchlistSortField == field {
			// Show current sort state (only when sorted)
			directionName := m.getSortDirectionName(m.watchlistSortDirection)
			s += fmt.Sprintf("%s%s (%s)\n", prefix, fieldName, directionName)
		} else {
			s += fmt.Sprintf("%s%s\n", prefix, fieldName)
		}
	}

	s += "\n" + m.getText("sortHelp") + "\n"
	return s
}

// ============================================================================
// Helper Functions
// ============================================================================

// isStockInPortfolio checks if stock is in portfolio
func (m *Model) isStockInPortfolio(code string) bool {
	for _, stock := range m.portfolio.Stocks {
		if stock.Code == code {
			return true
		}
	}
	return false
}

// getSortFieldName gets display name for sort field
func (m *Model) getSortFieldName(field SortField) string {
	switch field {
	case SortByCode:
		return m.getText("sortCode")
	case SortByName:
		return m.getText("sortName")
	case SortByPrice:
		return m.getText("sortPrice")
	case SortByCostPrice:
		return m.getText("sortCostPrice")
	case SortByChange:
		return m.getText("sortChange")
	case SortByChangePercent:
		return m.getText("sortChangePercent")
	case SortByQuantity:
		return m.getText("sortQuantity")
	case SortByTotalProfit:
		return m.getText("sortTotalProfit")
	case SortByProfitRate:
		return m.getText("sortProfitRate")
	case SortByMarketValue:
		return m.getText("sortMarketValue")
	case SortByTag:
		return m.getText("sortTag")
	case SortByTurnoverRate:
		return m.getText("sortTurnoverRate")
	case SortByVolume:
		return m.getText("sortVolume")
	default:
		return "Unknown"
	}
}

// getSortDirectionName gets display name for sort direction
func (m *Model) getSortDirectionName(direction SortDirection) string {
	if direction == SortAsc {
		return m.getText("sortAsc")
	}
	return m.getText("sortDesc")
}

// getWatchlistSortFields gets available sort fields for watchlist
func (m *Model) getWatchlistSortFields() []SortField {
	return []SortField{
		SortByCode, SortByName, SortByPrice, SortByTag,
		SortByChangePercent, SortByTurnoverRate, SortByVolume,
	}
}

// getPortfolioSortFields gets available sort fields for portfolio
func (m *Model) getPortfolioSortFields() []SortField {
	return []SortField{
		SortByCode, SortByName, SortByPrice, SortByCostPrice,
		SortByChange, SortByChangePercent, SortByQuantity,
		SortByTotalProfit, SortByProfitRate, SortByMarketValue,
	}
}

// findSortFieldIndex finds the index of sort field in field list, returns 0 if not found
func (m *Model) findSortFieldIndex(field SortField, isPortfolio bool) int {
	var fields []SortField
	if isPortfolio {
		fields = m.getPortfolioSortFields()
	} else {
		fields = m.getWatchlistSortFields()
	}

	for i, f := range fields {
		if f == field {
			return i
		}
	}

	// If current sort field not found, return 0 (first field)
	return 0
}
