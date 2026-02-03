package main

import (
	"stock-monitor/internal/consts"
	"fmt"
	"stock-monitor/internal/api"
	"stock-monitor/internal/ui"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ============================================================================
// Search Stock Handlers
// ============================================================================

func (m *Model) handleSearchingStock(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.searchFromWatchlist {
			m.state = consts.WatchlistViewing
			m.resetWatchlistCursor() // Reset cursor to first stock
			m.searchFromWatchlist = false
			m.searchInput = ""
			m.searchInputCursor = 0
			m.message = ""
			return m, m.tickCmd() // Restart timer
		} else {
			m.state = consts.MainMenu
		}
		m.searchInput = ""
		m.searchInputCursor = 0
		m.message = ""
		return m, nil
	case "enter":
		if m.searchInput == "" {
			m.message = m.getText("enterSearch")[:len(m.getText("enterSearch"))-2] // Remove ": " suffix
			return m, nil
		}
		logInfo("搜索股票: %s", m.searchInput)
		m.message = m.getText("searching")
		m.searchResult = convertStockData(api.GetStockInfo(m.searchInput))
		if m.searchResult == nil || m.searchResult.Name == "" {
			logInfo("搜索失败: %s", m.searchInput)
			m.message = fmt.Sprintf(m.getText("searchNotFound"), m.searchInput)
			return m, nil
		}
		logInfo("搜索成功: %s (%s)", m.searchResult.Name, m.searchResult.Symbol)

		// Mark as search mode
		m.isSearchMode = true

		// Get smart date (current day or recent trading day)
		actualDate, _, err := GetTradingDayForCollection(m.searchResult.Symbol, m)
		if err != nil {
			// Fallback to simple logic
			actualDate = getSmartChartDate()
		}

		// Set chart parameters
		m.chartViewStock = m.searchResult.Symbol
		m.chartViewStockName = m.searchResult.Name
		m.chartViewDate = actualDate

		// Clear input
		m.searchInput = ""
		m.searchInputCursor = 0
		m.message = ""

		// Determine next state based on source
		if m.searchFromWatchlist {
			m.state = consts.WatchlistSearchConfirm
		} else {
			m.state = consts.SearchResultWithActions
		}

		// Both states start temporary worker (auto-show chart)
		return m, m.startSearchIntradayWorker(
			m.searchResult.Symbol,
			m.searchResult.Name,
			actualDate,
		)
	case "left", "ctrl+b":
		m.searchInputCursor = MoveCursorUp(m.searchInputCursor)
		return m, nil
	case "right", "ctrl+f":
		runes := []rune(m.searchInput)
		m.searchInputCursor = MoveCursorDown(m.searchInputCursor, len(runes))
		return m, nil
	case "home", "ctrl+a":
		m.searchInputCursor = 0
		return m, nil
	case "end", "ctrl+e":
		m.searchInputCursor = len([]rune(m.searchInput))
		return m, nil
	case "backspace":
		m.searchInput, m.searchInputCursor = ui.DeleteRuneBeforeCursor(m.searchInput, m.searchInputCursor)
		return m, nil
	case "delete", "ctrl+d":
		m.searchInput, m.searchInputCursor = ui.DeleteRuneAtCursor(m.searchInput, m.searchInputCursor)
		return m, nil
	default:
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !ui.IsControlKey(str) {
			m.searchInput, m.searchInputCursor = ui.InsertStringAtCursor(m.searchInput, m.searchInputCursor, str)
		}
	}
	return m, nil
}

func (m *Model) viewSearchingStock() string {
	s := m.getText("searchTitle") + "\n\n"
	s += m.getText("enterSearch") + ui.FormatTextWithCursor(m.searchInput, m.searchInputCursor) + "\n\n"
	s += m.getText("searchFormats") + "\n\n"

	if m.language == consts.Chinese {
		s += "操作: ←/→移动光标, Enter搜索, ESC返回, Home/End跳转首尾\n"
	} else {
		s += "Actions: ←/→ move cursor, Enter search, ESC back, Home/End jump\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ============================================================================
// Search Result Handlers (Legacy - Simple View)
// ============================================================================

func (m *Model) handleSearchResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = consts.MainMenu
		m.message = ""
		return m, nil
	case "r":
		m.state = consts.SearchingStock
		m.searchFromWatchlist = false
		m.message = ""
		return m, nil
	}
	return m, nil
}

func (m *Model) viewSearchResult() string {
	s := m.getText("detailTitle") + "\n\n"

	if m.searchResult == nil {
		s += m.getText("noInfo") + "\n"
		s += "\n" + m.getText("detailHelp") + "\n"
		return s
	}

	// Create horizontal table to display stock details
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// Build header and data row
	var headers []interface{}
	var values []interface{}

	// Basic info
	if m.language == consts.Chinese {
		headers = append(headers, "股票代码", "股票名称", "现价")
	} else {
		headers = append(headers, "Code", "Name", "Price")
	}
	values = append(values, m.searchResult.Symbol, m.searchResult.Name, m.formatPriceWithColorLang(m.searchResult.Price, m.searchResult.PrevClose))

	// Previous close
	if m.searchResult.PrevClose > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "昨收价")
		} else {
			headers = append(headers, "Prev Close")
		}
		values = append(values, fmt.Sprintf("%.3f", m.searchResult.PrevClose))
	}

	// Price info (show only when available)
	if m.searchResult.StartPrice > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "开盘价")
		} else {
			headers = append(headers, "Open")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.StartPrice, m.searchResult.PrevClose))
	}
	if m.searchResult.MaxPrice > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "最高价")
		} else {
			headers = append(headers, "High")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.MaxPrice, m.searchResult.PrevClose))
	}
	if m.searchResult.MinPrice > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "最低价")
		} else {
			headers = append(headers, "Low")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.MinPrice, m.searchResult.PrevClose))
	}

	// Change info
	if m.searchResult.Change != 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "涨跌额")
		} else {
			headers = append(headers, "Change")
		}
		changeStr := m.formatProfitWithColorZeroLang(m.searchResult.Change)
		values = append(values, changeStr)
	}
	if m.searchResult.ChangePercent != 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "今日涨幅")
		} else {
			headers = append(headers, "Change %")
		}
		changePercentStr := m.formatProfitRateWithColorZeroLang(m.searchResult.ChangePercent)
		values = append(values, changePercentStr)
	}

	// Turnover rate
	if m.searchResult.TurnoverRate > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "换手率")
		} else {
			headers = append(headers, "Turnover")
		}
		values = append(values, fmt.Sprintf("%.2f%%", m.searchResult.TurnoverRate))
	}

	// Volume
	if m.searchResult.Volume > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "成交量")
		} else {
			headers = append(headers, "Volume")
		}
		volumeStr := formatVolume(m.searchResult.Volume)
		values = append(values, volumeStr)
	}

	// Append header and data row
	t.AppendHeader(table.Row(headers))
	t.AppendRow(table.Row(values))

	s += t.Render() + "\n\n"
	s += m.getText("detailHelp") + "\n"

	return s
}

// ============================================================================
// Search Result With Actions Handlers (With Chart)
// ============================================================================

func (m *Model) handleSearchResultWithActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Stop search worker and clean up data
		if m.isSearchMode {
			m.stopSearchIntradayWorker()
		}

		m.state = consts.MainMenu
		m.message = ""
		return m, nil
	case "r":
		// Clean up old data when re-searching
		if m.isSearchMode {
			m.stopSearchIntradayWorker()
		}

		m.state = consts.SearchingStock
		m.searchFromWatchlist = false
		m.message = ""
		return m, nil
	case "1":
		// Add to watchlist and jump to watchlist page
		if m.searchResult != nil {
			if m.addToWatchlist(m.searchResult.Symbol, m.searchResult.Name) {
				m.message = fmt.Sprintf(m.getText("addWatchSuccess"), m.searchResult.Name, m.searchResult.Symbol)
			} else {
				m.message = fmt.Sprintf(m.getText("alreadyInWatch"), m.searchResult.Symbol)
			}

			// Stop search worker
			if m.isSearchMode {
				m.stopSearchIntradayWorker()
			}

			// Jump to watchlist page
			m.state = consts.WatchlistViewing
			m.resetWatchlistCursor() // Reset cursor to first stock
			m.cursor = 0
			m.lastUpdate = time.Now()

			// Start watchlist intraday data collection
			m.startIntradayDataCollection()
		}
		return m, m.tickCmd()
	case "2":
		// Add to portfolio (enter add flow)
		if m.searchResult != nil {
			// Stop search worker
			if m.isSearchMode {
				m.stopSearchIntradayWorker()
			}

			m.state = consts.AddingStock
			m.addingStep = 1 // Skip code input, go directly to cost input
			m.tempCode = m.searchResult.Symbol
			m.stockInfo = &StockData{
				Symbol: m.searchResult.Symbol,
				Name:   m.searchResult.Name,
				Price:  m.searchResult.Price,
			}
			m.input = ""
			m.message = ""
			m.fromSearch = true // Mark as added from search result
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) viewSearchResultWithActions() string {
	s := m.getText("detailTitle") + "\n\n"

	if m.searchResult == nil {
		s += m.getText("noInfo") + "\n"
		s += "\n" + m.getText("actionHelp") + "\n"
		return s
	}

	// Reuse original search result display logic
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// Build header and data row
	var headers []interface{}
	var values []interface{}

	// Basic info
	if m.language == consts.Chinese {
		headers = append(headers, "股票代码", "股票名称", "现价")
	} else {
		headers = append(headers, "Code", "Name", "Price")
	}
	values = append(values, m.searchResult.Symbol, m.searchResult.Name, m.formatPriceWithColorLang(m.searchResult.Price, m.searchResult.PrevClose))

	// Previous close
	if m.searchResult.PrevClose > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "昨收价")
		} else {
			headers = append(headers, "Prev Close")
		}
		values = append(values, fmt.Sprintf("%.3f", m.searchResult.PrevClose))
	}

	// Price info (show only when available)
	if m.searchResult.StartPrice > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "开盘价")
		} else {
			headers = append(headers, "Open")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.StartPrice, m.searchResult.PrevClose))
	}
	if m.searchResult.MaxPrice > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "最高价")
		} else {
			headers = append(headers, "High")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.MaxPrice, m.searchResult.PrevClose))
	}
	if m.searchResult.MinPrice > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "最低价")
		} else {
			headers = append(headers, "Low")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.MinPrice, m.searchResult.PrevClose))
	}

	// Change info
	if m.searchResult.Change != 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "涨跌额")
		} else {
			headers = append(headers, "Change")
		}
		changeStr := m.formatProfitWithColorZeroLang(m.searchResult.Change)
		values = append(values, changeStr)
	}
	if m.searchResult.ChangePercent != 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "今日涨幅")
		} else {
			headers = append(headers, "Change %")
		}
		changePercentStr := m.formatProfitRateWithColorZeroLang(m.searchResult.ChangePercent)
		values = append(values, changePercentStr)
	}

	// Turnover rate
	if m.searchResult.TurnoverRate > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "换手率")
		} else {
			headers = append(headers, "Turnover")
		}
		values = append(values, fmt.Sprintf("%.2f%%", m.searchResult.TurnoverRate))
	}

	// Volume
	if m.searchResult.Volume > 0 {
		if m.language == consts.Chinese {
			headers = append(headers, "成交量")
		} else {
			headers = append(headers, "Volume")
		}
		volumeStr := formatVolume(m.searchResult.Volume)
		values = append(values, volumeStr)
	}

	// Append header and data row
	t.AppendHeader(table.Row(headers))
	t.AppendRow(table.Row(values))

	s += t.Render() + "\n\n"

	// === New: Search mode intraday chart (auto-display) ===
	if m.isSearchMode {
		// Render chart area separator
		s += strings.Repeat("─", 80) + "\n"
		if m.language == consts.Chinese {
			s += "📈 实时分时图表 (每5秒自动刷新)\n\n"
		} else {
			s += "📈 Real-time Intraday Chart (Auto-refresh every 5s)\n\n"
		}

		// Render chart
		if m.searchIntradayData != nil && len(m.searchIntradayData.Datapoints) > 0 {
			// Create chart (use smaller embedded size)
			chartWidth := 100  // Embedded chart width
			chartHeight := 15  // Embedded chart height

			chartModel := m.createSearchIntradayChart(chartWidth, chartHeight)
			if chartModel != nil {
				s += chartModel.View() + "\n"

				// Display update info
				if m.language == consts.Chinese {
					s += fmt.Sprintf("最后更新: %s | 数据点: %d\n",
						m.searchIntradayData.UpdatedAt,
						len(m.searchIntradayData.Datapoints))
				} else {
					s += fmt.Sprintf("Last update: %s | Data points: %d\n",
						m.searchIntradayData.UpdatedAt,
						len(m.searchIntradayData.Datapoints))
				}
			} else {
				// Chart creation failed (terminal too small)
				if m.language == consts.Chinese {
					s += "终端尺寸过小，无法显示图表\n"
				} else {
					s += "Terminal size too small to display chart\n"
				}
			}
		} else {
			// Data not yet loaded
			if m.language == consts.Chinese {
				s += "正在获取分时数据...\n"
			} else {
				s += "Loading intraday data...\n"
			}
		}

		s += "\n"
	}

	// Action button hints
	s += m.getText("actionHelp") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ============================================================================
// Watchlist Search Confirm Handlers
// ============================================================================

func (m *Model) handleWatchlistSearchConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Stop search worker and clean up data
		if m.isSearchMode {
			m.stopSearchIntradayWorker()
		}

		m.state = consts.WatchlistViewing
		m.resetWatchlistCursor() // Reset cursor to first stock
		m.searchFromWatchlist = false
		m.message = ""

		// Start watchlist intraday data collection
		m.startIntradayDataCollection()

		return m, m.tickCmd() // Restart timer
	case "enter":
		// Confirm add to watchlist
		if m.searchResult != nil {
			if m.addToWatchlist(m.searchResult.Symbol, m.searchResult.Name) {
				m.message = fmt.Sprintf(m.getText("addWatchSuccess"), m.searchResult.Name, m.searchResult.Symbol)
				logInfo("添加到自选列表: %s (%s)", m.searchResult.Name, m.searchResult.Symbol)
			} else {
				m.message = fmt.Sprintf(m.getText("alreadyInWatch"), m.searchResult.Symbol)
			}

			// Stop search worker
			if m.isSearchMode {
				m.stopSearchIntradayWorker()
			}

			m.state = consts.WatchlistViewing
			m.resetWatchlistCursor() // Reset cursor to first stock
			m.searchFromWatchlist = false

			// Start watchlist intraday data collection
			m.startIntradayDataCollection()

			return m, m.tickCmd()
		}
		return m, nil
	case "r":
		// Clean up old data when re-searching
		if m.isSearchMode {
			m.stopSearchIntradayWorker()
		}

		m.state = consts.SearchingStock
		m.searchInput = ""
		m.searchResult = nil
		m.message = ""
		return m, nil
	}
	return m, nil
}

func (m *Model) viewWatchlistSearchConfirm() string {
	if m.searchResult == nil {
		return m.getText("searchNotFound")
	}

	s := m.getText("searchTitle") + "\n\n"

	// Create table to display stock info
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// Set header
	if m.language == consts.Chinese {
		t.AppendHeader(table.Row{"名称", "现价", "昨收价", "开盘", "最高", "最低", "今日涨幅", "换手率", "成交量"})
	} else {
		t.AppendHeader(table.Row{"Name", "Price", "PrevClose", "Open", "High", "Low", "Today%", "Turnover", "Volume"})
	}

	// Build data row
	var values []interface{}

	// Name
	values = append(values, m.searchResult.Name)

	// Price (with color)
	priceStr := m.formatPriceWithColorLang(m.searchResult.Price, m.searchResult.PrevClose)
	values = append(values, priceStr)

	// Previous close
	values = append(values, fmt.Sprintf("%.3f", m.searchResult.PrevClose))

	// Open
	if m.searchResult.StartPrice > 0 {
		openStr := m.formatPriceWithColorLang(m.searchResult.StartPrice, m.searchResult.PrevClose)
		values = append(values, openStr)
	} else {
		values = append(values, "-")
	}

	// High
	if m.searchResult.MaxPrice > 0 {
		highStr := m.formatPriceWithColorLang(m.searchResult.MaxPrice, m.searchResult.PrevClose)
		values = append(values, highStr)
	} else {
		values = append(values, "-")
	}

	// Low
	if m.searchResult.MinPrice > 0 {
		lowStr := m.formatPriceWithColorLang(m.searchResult.MinPrice, m.searchResult.PrevClose)
		values = append(values, lowStr)
	} else {
		values = append(values, "-")
	}

	// Today change percent
	if m.searchResult.ChangePercent != 0 {
		changePercentStr := m.formatProfitRateWithColorZeroLang(m.searchResult.ChangePercent)
		values = append(values, changePercentStr)
	} else {
		values = append(values, "-")
	}

	// Turnover rate
	if m.searchResult.TurnoverRate > 0 {
		values = append(values, fmt.Sprintf("%.2f%%", m.searchResult.TurnoverRate))
	} else {
		values = append(values, "-")
	}

	// Volume
	if m.searchResult.Volume > 0 {
		if m.searchResult.Volume >= 100000000 { // >= 100M
			values = append(values, fmt.Sprintf("%.2f亿", float64(m.searchResult.Volume)/100000000))
		} else if m.searchResult.Volume >= 10000 { // >= 10K
			values = append(values, fmt.Sprintf("%.2f万", float64(m.searchResult.Volume)/10000))
		} else {
			values = append(values, fmt.Sprintf("%d", m.searchResult.Volume))
		}
	} else {
		values = append(values, "-")
	}

	t.AppendRow(values)

	s += t.Render() + "\n\n"

	// === New: Search mode intraday chart (auto-display) ===
	if m.isSearchMode {
		// Render chart area separator
		s += strings.Repeat("─", 80) + "\n"
		if m.language == consts.Chinese {
			s += "📈 实时分时图表 (每5秒自动刷新)\n\n"
		} else {
			s += "📈 Real-time Intraday Chart (Auto-refresh every 5s)\n\n"
		}

		// Render chart
		if m.searchIntradayData != nil && len(m.searchIntradayData.Datapoints) > 0 {
			// Create chart (use smaller embedded size)
			chartWidth := 100  // Embedded chart width
			chartHeight := 15  // Embedded chart height

			chartModel := m.createSearchIntradayChart(chartWidth, chartHeight)
			if chartModel != nil {
				s += chartModel.View() + "\n"

				// Display update info
				if m.language == consts.Chinese {
					s += fmt.Sprintf("最后更新: %s | 数据点: %d\n",
						m.searchIntradayData.UpdatedAt,
						len(m.searchIntradayData.Datapoints))
				} else {
					s += fmt.Sprintf("Last update: %s | Data points: %d\n",
						m.searchIntradayData.UpdatedAt,
						len(m.searchIntradayData.Datapoints))
				}
			} else {
				// Chart creation failed (terminal too small)
				if m.language == consts.Chinese {
					s += "终端尺寸过小，无法显示图表\n"
				} else {
					s += "Terminal size too small to display chart\n"
				}
			}
		} else {
			// Data not yet loaded
			if m.language == consts.Chinese {
				s += "正在获取分时数据...\n"
			} else {
				s += "Loading intraday data...\n"
			}
		}

		s += "\n"
	}

	if m.language == consts.Chinese {
		s += "按回车键添加到自选列表，ESC键返回，R键重新搜索\n"
	} else {
		s += "Press Enter to add to watchlist, ESC to return, R to search again\n"
	}

	return s
}

// ============================================================================
// Helper Functions
// ============================================================================

// resetPortfolioCursor resets portfolio cursor to first stock
func (m *Model) resetPortfolioCursor() {
	if len(m.portfolio.Stocks) > 0 {
		m.portfolioCursor = 0
		maxPortfolioLines := m.config.Display.MaxLines
		if len(m.portfolio.Stocks) > maxPortfolioLines {
			// Display first N lines: set scroll position to show N lines starting from index 0
			m.portfolioScrollPos = len(m.portfolio.Stocks) - maxPortfolioLines
		} else {
			// Stock count doesn't exceed display lines, show all
			m.portfolioScrollPos = 0
		}
	}
}
