package main

import (
	"stock-monitor/internal/consts"
	"fmt"
	"strconv"
	"stock-monitor/internal/api"
	"stock-monitor/internal/ui"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ============================================================================
// Portfolio consts.Monitoring Handlers
// ============================================================================

// handleMonitoringNavigation handles navigation in portfolio monitoring view
func (m *Model) handleMonitoringNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "up", "k", "w":
		m.portfolioCursor = MoveCursorUp(m.portfolioCursor)
		return m, nil, true
	case "down", "j":
		m.portfolioCursor = MoveCursorDown(m.portfolioCursor, len(m.portfolio.Stocks)-1)
		return m, nil, true
	}
	return m, nil, false
}

// handleMonitoringActions handles data operations in portfolio monitoring view
func (m *Model) handleMonitoringActions(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "esc", "q", "m":
		m.stopIntradayDataCollection() // Stop intraday data collection
		m.state = consts.MainMenu
		m.message = "" // Clear message
		return m, nil, true
	case "e":
		// Edit current stock
		if len(m.portfolio.Stocks) == 0 {
			m.message = m.getText("emptyPortfolio")
			return m, nil, true
		}
		logInfo("log.action.enterEdit")
		m.previousState = m.state // Record current state
		m.state = consts.EditingStock
		m.editingStep = 1 // Start editing cost price
		m.selectedStockIndex = m.portfolioCursor
		m.tempCode = m.portfolio.Stocks[m.portfolioCursor].Code
		m.tempCost = ""
		m.tempQuantity = ""
		m.input = fmt.Sprintf("%.*f", m.config.Display.DecimalPlaces, m.portfolio.Stocks[m.portfolioCursor].CostPrice) // Prefill with current cost
		m.inputCursor = len([]rune(m.input))                                                                           // Cursor at end
		m.message = ""
		return m, nil, true
	case "d":
		// Delete stock at cursor
		if len(m.portfolio.Stocks) == 0 {
			m.message = m.getText("emptyPortfolio")
			return m, nil, true
		}
		// Remove stock at cursor
		removedStock := m.portfolio.Stocks[m.portfolioCursor]
		m.portfolio.Stocks = append(m.portfolio.Stocks[:m.portfolioCursor], m.portfolio.Stocks[m.portfolioCursor+1:]...)
		m.savePortfolio()
		// 删除后无需重新排序(相对顺序不变),保持排序状态
		// Adjust cursor position
		if m.portfolioCursor >= len(m.portfolio.Stocks) && len(m.portfolio.Stocks) > 0 {
			m.portfolioCursor = len(m.portfolio.Stocks) - 1
		}
		m.message = fmt.Sprintf(m.getText("removeSuccess"), removedStock.Name, removedStock.Code)
		return m, nil, true
	case "a":
		// Jump to add stock page
		logInfo("log.action.enterAdd")
		m.previousState = m.state // Record current state
		m.state = consts.AddingStock
		m.addingStep = 0
		m.tempCode = ""
		m.tempCost = ""
		m.tempQuantity = ""
		m.stockInfo = nil
		m.input = ""
		m.message = ""
		m.fromSearch = true // Set flag to return to monitoring after completion
		return m, nil, true
	}
	return m, nil, false
}

// handleMonitoringViews handles view switching in portfolio monitoring
func (m *Model) handleMonitoringViews(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "v":
		// View intraday chart
		if len(m.portfolio.Stocks) == 0 {
			m.message = m.getText("emptyPortfolio")
			return m, nil, true
		}
		selectedStock := m.portfolio.Stocks[m.portfolioCursor]
		m.chartViewStock = selectedStock.Code
		m.chartViewStockName = selectedStock.Name

		// 选择展示日期（与 worker 采集逻辑解耦，处理静默期等边缘情况）
		actualDate := getChartDisplayDate(selectedStock.Code, m)
		m.chartViewDate = actualDate
		m.previousState = consts.Monitoring

		logDebug("log.chart.keyV", selectedStock.Code, selectedStock.Name, m.chartViewDate)

		// Try to load data
		data, loadErr := m.loadIntradayDataForDate(
			selectedStock.Code,
			selectedStock.Name,
			actualDate,
		)

		if loadErr != nil {
			// No data - trigger collection
			logDebug("log.chart.noData", loadErr)
			m.chartData = nil
			m.chartLoadError = nil
			m.state = consts.IntradayChartViewing
			return m, m.triggerIntradayDataCollection(
				selectedStock.Code,
				selectedStock.Name,
				actualDate,
			), true
		}

		// Data exists - create chart
		logDebug("log.chart.dataLoaded", len(data.Datapoints))
		m.chartData = data
		m.chartLoadError = nil
		m.chartIsCollecting = false
		m.state = consts.IntradayChartViewing
		return m, nil, true
	case "s":
		// Enter sort menu
		logInfo("log.action.enterSort")
		m.state = consts.PortfolioSorting
		// Smart position cursor to current sort field
		m.portfolioSortCursor = m.findSortFieldIndex(m.portfolioSortField, true)
		m.message = ""
		return m, nil, true
	case "l", "L":
		// View stock alert details
		if len(m.portfolio.Stocks) == 0 {
			m.message = m.getText("emptyPortfolio")
			return m, nil, true
		}
		currentStock := m.portfolio.Stocks[m.portfolioCursor]

		// Set stock alert detail parameters
		m.stockAlertCode = currentStock.Code
		m.stockAlertName = currentStock.Name
		m.stockAlertCursor = 0
		m.previousState = consts.Monitoring // Record return state

		// Get all alerts for this stock
		m.stockAlertAlerts = m.getStockAlerts(currentStock.Code)

		logInfo("log.action.enterStockAlertManagement", currentStock.Code, currentStock.Name)
		m.state = consts.StockAlertManage
		m.message = ""
		return m, nil, true
	}
	return m, nil, false
}

func (m *Model) handleMonitoring(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Try navigation handling
	if model, cmd, handled := m.handleMonitoringNavigation(msg); handled {
		return model, cmd
	}

	// Try action handling
	if model, cmd, handled := m.handleMonitoringActions(msg); handled {
		return model, cmd
	}

	// Try view switching handling
	if model, cmd, handled := m.handleMonitoringViews(msg); handled {
		return model, cmd
	}

	return m, nil
}

func (m *Model) viewMonitoring() string {
	s := m.getText("monitoringTitle") + "\n"
	s += fmt.Sprintf(m.getText("updateTime"), m.lastUpdate.Format("2006-01-02 15:04:05")) + "\n"
	s += "\n"

	if len(m.portfolio.Stocks) == 0 {
		s += m.getText("emptyPortfolio") + "\n\n"
		s += m.getText("addStockFirst") + "\n\n"
		s += m.getText("holdingsHelp") + "\n"
		return s
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// Get header with sort indicator
	t.AppendHeader(m.GeneratePortfolioHeader())

	var totalMarketValue float64
	var totalCost float64

	// Display scroll information
	totalStocks := len(m.portfolio.Stocks)
	maxPortfolioLines := m.config.Display.MaxLines
	if totalStocks > 0 {
		currentPos := m.portfolioCursor + 1 // Display position starting from 1
		if m.language == consts.Chinese {
			s += fmt.Sprintf("📊 持股列表 (%d/%d) [↑/↓:翻页]\n", currentPos, totalStocks)
		} else {
			s += fmt.Sprintf("📊 Portfolio (%d/%d) [↑/↓:scroll]\n", currentPos, totalStocks)
		}
		s += "\n"
	}

	// Calculate range of stocks to display
	stocks := m.portfolio.Stocks
	endIndex := len(stocks) - m.portfolioScrollPos
	startIndex := endIndex - maxPortfolioLines
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(stocks) {
		endIndex = len(stocks)
	}

	// First calculate totals for all stocks (for summary row)
	for i := range m.portfolio.Stocks {
		stock := &m.portfolio.Stocks[i]
		// Get stock price from cache (non-blocking)
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

		if stock.Price > 0 {
			marketValue := stock.Price * float64(stock.Quantity)
			cost := stock.CostPrice * float64(stock.Quantity)

			totalMarketValue += marketValue
			totalCost += cost
		}
	}

	// Then display stocks in current range
	for i := startIndex; i < endIndex; i++ {
		stock := &m.portfolio.Stocks[i]

		// Use dynamic column renderer to generate row
		row := m.GeneratePortfolioRow(stock, i, startIndex, endIndex)
		t.AppendRow(row)

		// Add separator after each stock (except last in visible range)
		if i < endIndex-1 {
			t.AppendSeparator()
		}
	}

	totalPortfolioProfit := totalMarketValue - totalCost
	totalProfitRate := 0.0
	if totalCost > 0 {
		totalProfitRate = (totalPortfolioProfit / totalCost) * 100
	}

	t.AppendSeparator()
	// Use dynamic column renderer to generate total row
	totalRow := m.GeneratePortfolioTotalRow(totalPortfolioProfit, totalProfitRate, totalMarketValue)
	t.AppendRow(totalRow)

	s += t.Render() + "\n"

	// If scrolling is available, show scroll indicators
	if totalStocks > maxPortfolioLines {
		s += strings.Repeat("-", 80) + "\n"
		if m.portfolioScrollPos > 0 {
			if m.language == consts.Chinese {
				s += "↑ 有更新的股票 (按↓查看)\n"
			} else {
				s += "↑ Newer stocks available (press ↓)\n"
			}
		}
		if m.portfolioScrollPos < totalStocks-1 {
			if m.language == consts.Chinese {
				s += "↓ 有更多历史股票 (按↑查看)\n"
			} else {
				s += "↓ More stocks available (press ↑)\n"
			}
		}
	}

	s += "\n" + m.getText("holdingsHelp") + "\n"

	return s
}

// ============================================================================
// Add Stock Handlers
// ============================================================================

func (m *Model) handleAddingStock(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Determine return target based on source
		if m.fromSearch {
			// From portfolio or search result, return to respective page
			if m.previousState == consts.Monitoring {
				m.state = consts.Monitoring
				m.resetPortfolioCursor() // Reset cursor to first stock
				m.lastUpdate = time.Now()
			} else {
				m.state = consts.SearchResultWithActions
			}
			m.fromSearch = false // Reset flag
		} else {
			m.state = consts.MainMenu
		}
		m.message = ""
		m.inputCursor = 0
		return m, nil
	case "enter":
		return m.processAddingStep()
	case "left", "ctrl+b":
		m.inputCursor = MoveCursorUp(m.inputCursor)
		return m, nil
	case "right", "ctrl+f":
		runes := []rune(m.input)
		m.inputCursor = MoveCursorDown(m.inputCursor, len(runes))
		return m, nil
	case "home", "ctrl+a":
		m.inputCursor = 0
		return m, nil
	case "end", "ctrl+e":
		m.inputCursor = len([]rune(m.input))
		return m, nil
	case "backspace":
		m.input, m.inputCursor = ui.DeleteRuneBeforeCursor(m.input, m.inputCursor)
		return m, nil
	case "delete", "ctrl+d":
		m.input, m.inputCursor = ui.DeleteRuneAtCursor(m.input, m.inputCursor)
		return m, nil
	default:
		// Improved input handling: support multi-byte characters (e.g., consts.Chinese)
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !ui.IsControlKey(str) {
			m.input, m.inputCursor = ui.InsertStringAtCursor(m.input, m.inputCursor, str)
		}
	}
	return m, nil
}

func (m *Model) processAddingStep() (tea.Model, tea.Cmd) {
	switch m.addingStep {
	case 0: // Search stock
		if m.input == "" {
			m.message = m.getText("codeRequired")
			return m, nil
		}
		m.message = m.getText("searching")

		// Use search functionality
		var stockData *StockData
		if api.ContainsChineseChars(m.input) {
			stockData = api.SearchChineseStock(m.input)
		} else {
			// For non-consts.Chinese input, try direct price fetch first, then search
			stockData = api.GetStockPrice(m.input)

			// If direct fetch fails, try as search keyword
			if stockData == nil || stockData.Price <= 0 {
				logWarn("log.api.addStockFail", m.input)
				stockData = api.SearchStockBySymbol(m.input)
			}
		}

		if stockData == nil || stockData.Name == "" {
			m.message = fmt.Sprintf(m.getText("searchNotFound"), m.input)
			m.input = ""
			m.inputCursor = 0
			return m, nil
		}

		// Save search result and move to cost input step
		m.stockInfo = stockData
		m.tempCode = stockData.Symbol
		m.addingStep = 1
		m.input = ""
		m.inputCursor = 0
		m.message = ""
	case 1: // Input cost price
		if m.input == "" {
			m.message = m.getText("costRequired")
			return m, nil
		}
		if _, err := strconv.ParseFloat(m.input, 64); err != nil {
			m.message = m.getText("invalidPrice")
			m.input = ""
			m.inputCursor = 0
			return m, nil
		}
		m.tempCost = m.input
		m.addingStep = 2
		m.input = ""
		m.inputCursor = 0
		m.message = ""
	case 2: // Input quantity
		if m.input == "" {
			m.message = m.getText("quantityRequired")
			return m, nil
		}
		if _, err := strconv.Atoi(m.input); err != nil {
			m.message = m.getText("invalidQuantity")
			m.input = ""
			m.inputCursor = 0
			return m, nil
		}
		m.tempQuantity = m.input

		// Add stock
		costPrice, _ := strconv.ParseFloat(m.tempCost, 64)
		quantity, _ := strconv.Atoi(m.tempQuantity)

		stock := Stock{
			Code:      m.tempCode,
			Name:      m.stockInfo.Name,
			CostPrice: costPrice,
			Quantity:  quantity,
		}

		m.portfolio.Stocks = append(m.portfolio.Stocks, stock)
		m.savePortfolio()
		// 如果之前已排序,重新应用排序以保持顺序
		if m.portfolioIsSorted {
			m.updatePortfolioPricesFromCache()
			m.optimizedSortPortfolio(m.portfolioSortField, m.portfolioSortDirection)
		}

		// Determine jump target based on source
		if m.fromSearch {
			// From search result, jump to portfolio (monitoring) page
			m.state = consts.Monitoring
			m.resetPortfolioCursor() // Reset cursor to first stock
			m.lastUpdate = time.Now()
			m.fromSearch = false // Reset flag
			m.message = fmt.Sprintf(m.getText("addSuccess"), m.stockInfo.Name, m.tempCode)
			m.addingStep = 0
			m.input = ""
			return m, m.tickCmd() // Start timer when jumping to monitoring
		} else {
			// From main menu, return to main menu
			m.state = consts.MainMenu
			m.message = fmt.Sprintf(m.getText("addSuccess"), m.stockInfo.Name, m.tempCode)
			m.addingStep = 0
			m.input = ""
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) viewAddingStock() string {
	s := m.getText("addingTitle") + "\n\n"

	switch m.addingStep {
	case 0:
		s += m.getText("enterSearch") + ui.FormatTextWithCursor(m.input, m.inputCursor) + "\n"
		s += "\n" + m.getText("searchFormats") + "\n"
	case 1:
		s += fmt.Sprintf(m.getText("stockCode"), m.tempCode) + "\n"
		s += fmt.Sprintf(m.getText("stockName"), m.stockInfo.Name) + "\n"
		s += fmt.Sprintf(m.getText("currentPrice"), m.stockInfo.Price) + "\n\n"
		s += m.getText("enterCost") + ui.FormatTextWithCursor(m.input, m.inputCursor) + "\n"
	case 2:
		s += fmt.Sprintf(m.getText("stockCode"), m.tempCode) + "\n"
		s += fmt.Sprintf(m.getText("stockName"), m.stockInfo.Name) + "\n"
		s += fmt.Sprintf(m.getText("currentPrice"), m.stockInfo.Price) + "\n"
		s += fmt.Sprintf(m.getText("costPrice"), m.tempCost) + "\n\n"
		s += m.getText("enterQuantity") + ui.FormatTextWithCursor(m.input, m.inputCursor) + "\n"
	}

	// Add cursor operation hints
	if m.language == consts.Chinese {
		s += "\n操作: ←/→移动光标, Enter确认, ESC返回, Home/End跳转首尾\n"
	} else {
		s += "\nActions: ←/→ move cursor, Enter confirm, ESC back, Home/End jump\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ============================================================================
// Edit Stock Handlers
// ============================================================================

func (m *Model) handleEditingStock(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Determine where to return based on previous state
		if m.previousState == consts.Monitoring {
			m.state = consts.Monitoring
			m.resetPortfolioCursor() // Reset cursor to first stock
			m.lastUpdate = time.Now()
			m.message = ""
			m.inputCursor = 0
			return m, m.tickCmd()
		} else {
			m.state = consts.MainMenu
			m.message = ""
			m.inputCursor = 0
			return m, nil
		}
	case "enter", " ":
		return m.processEditingStep()
	case "left", "ctrl+b":
		m.inputCursor = MoveCursorUp(m.inputCursor)
		return m, nil
	case "right", "ctrl+f":
		runes := []rune(m.input)
		m.inputCursor = MoveCursorDown(m.inputCursor, len(runes))
		return m, nil
	case "home", "ctrl+a":
		m.inputCursor = 0
		return m, nil
	case "end", "ctrl+e":
		m.inputCursor = len([]rune(m.input))
		return m, nil
	case "backspace":
		m.input, m.inputCursor = ui.DeleteRuneBeforeCursor(m.input, m.inputCursor)
		return m, nil
	case "delete", "ctrl+d":
		m.input, m.inputCursor = ui.DeleteRuneAtCursor(m.input, m.inputCursor)
		return m, nil
	default:
		// Improved input handling: support multi-byte characters (e.g., consts.Chinese)
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !ui.IsControlKey(str) {
			m.input, m.inputCursor = ui.InsertStringAtCursor(m.input, m.inputCursor, str)
		}
	}
	return m, nil
}

func (m *Model) processEditingStep() (tea.Model, tea.Cmd) {
	switch m.editingStep {
	case 1: // Modify cost price
		if m.input == "" {
			m.message = m.getText("costRequired")
			return m, nil
		}
		if newCost, err := strconv.ParseFloat(m.input, 64); err != nil {
			m.message = m.getText("invalidPrice")
			m.input = ""
			m.inputCursor = 0
			return m, nil
		} else {
			m.portfolio.Stocks[m.selectedStockIndex].CostPrice = newCost
			m.editingStep = 2
			m.input = fmt.Sprintf("%d", m.portfolio.Stocks[m.selectedStockIndex].Quantity)
			m.inputCursor = len([]rune(m.input)) // Cursor at end
			m.message = ""
		}
	case 2: // Modify quantity
		if m.input == "" {
			m.message = m.getText("quantityRequired")
			return m, nil
		}
		if newQuantity, err := strconv.Atoi(m.input); err != nil {
			m.message = m.getText("invalidQuantity")
			m.input = ""
			m.inputCursor = 0
			return m, nil
		} else {
			m.portfolio.Stocks[m.selectedStockIndex].Quantity = newQuantity
			m.savePortfolio()
			// 如果之前已排序且排序字段可能受影响,重新排序
			if m.portfolioIsSorted {
				m.updatePortfolioPricesFromCache()
				m.optimizedSortPortfolio(m.portfolioSortField, m.portfolioSortDirection)
			}

			stockName := m.portfolio.Stocks[m.selectedStockIndex].Name
			// Determine where to return based on previous state
			if m.previousState == consts.Monitoring {
				m.state = consts.Monitoring
				m.resetPortfolioCursor() // Reset cursor to first stock
				m.lastUpdate = time.Now()
				m.message = fmt.Sprintf(m.getText("editSuccess"), stockName)
				m.editingStep = 0
				m.input = ""
				m.inputCursor = 0
				return m, m.tickCmd()
			} else {
				m.state = consts.MainMenu
				m.message = fmt.Sprintf(m.getText("editSuccess"), stockName)
				m.editingStep = 0
				m.input = ""
				m.inputCursor = 0
			}
		}
	}
	return m, nil
}

func (m *Model) viewEditingStock() string {
	s := m.getText("editTitle") + "\n\n"

	switch m.editingStep {
	case 1:
		stock := m.portfolio.Stocks[m.selectedStockIndex]
		if m.language == consts.Chinese {
			s += fmt.Sprintf("股票: %s (%s)\n", stock.Name, stock.Code)
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n", stock.Name, stock.Code)
		}
		s += fmt.Sprintf(m.getText("currentCost"), stock.CostPrice) + "\n\n"
		s += m.getText("enterNewCost") + ui.FormatTextWithCursor(m.input, m.inputCursor) + "\n"
	case 2:
		stock := m.portfolio.Stocks[m.selectedStockIndex]
		if m.language == consts.Chinese {
			s += fmt.Sprintf("股票: %s (%s)\n", stock.Name, stock.Code)
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n", stock.Name, stock.Code)
		}
		s += fmt.Sprintf(m.getText("newCost"), stock.CostPrice) + "\n"
		s += fmt.Sprintf(m.getText("currentQuantity"), stock.Quantity) + "\n\n"
		s += m.getText("enterNewQuantity") + ui.FormatTextWithCursor(m.input, m.inputCursor) + "\n"
	}

	// Add cursor operation hints
	if m.language == consts.Chinese {
		s += "\n操作: ←/→移动光标, Enter确认, ESC/Q返回, Home/End跳转首尾\n"
	} else {
		s += "\nActions: ←/→ move cursor, Enter confirm, ESC/Q back, Home/End jump\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ============================================================================
// Portfolio Sorting Handlers
// ============================================================================

func (m *Model) handlePortfolioSorting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sortFields := m.getPortfolioSortFields()

	switch msg.String() {
	case "up", "k", "w":
		m.portfolioSortCursor = MoveCursorUp(m.portfolioSortCursor)
	case "down", "j", "s":
		m.portfolioSortCursor = MoveCursorDown(m.portfolioSortCursor, len(sortFields)-1)
	case "enter", " ":
		// Toggle sort direction or apply sort
		selectedField := sortFields[m.portfolioSortCursor]
		if m.portfolioSortField == selectedField {
			// Toggle sort direction
			if m.portfolioSortDirection == consts.SortAsc {
				m.portfolioSortDirection = consts.SortDesc
			} else {
				m.portfolioSortDirection = consts.SortAsc
			}
		} else {
			// Set new sort field, default ascending
			m.portfolioSortField = selectedField
			m.portfolioSortDirection = consts.SortAsc
		}
		// Execute sort and mark as sorted
		m.optimizedSortPortfolio(m.portfolioSortField, m.portfolioSortDirection)
		m.portfolioIsSorted = true
		m.resetPortfolioCursor()
		// Return to portfolio page
		m.state = consts.Monitoring
		m.message = ""
		return m, nil
	case "c", "C":
		// Clear current sort - reload original data order
		m.portfolioIsSorted = false
		// Clear sort field and direction state
		m.portfolioSortField = consts.SortByCode  // Reset to default
		m.portfolioSortDirection = consts.SortAsc // Reset to default
		// Reload original data order
		portfolio, corrupted := loadPortfolio()
		m.portfolio = portfolio
		if corrupted {
			m.portfolioCorrupted = true
		}
		m.resetPortfolioCursor()
		// Return to portfolio page
		m.state = consts.Monitoring
		m.message = m.getText("sortCleared")
		return m, nil
	case "esc", "q":
		// Return to portfolio page
		m.state = consts.Monitoring
		m.message = ""
		return m, nil
	}
	return m, nil
}

// viewPortfolioSorting renders the portfolio sorting menu
func (m *Model) viewPortfolioSorting() string {
	s := m.getText("sortTitle") + "\n\n"
	s += m.getText("selectSortField") + "\n\n"

	sortFields := m.getPortfolioSortFields()
	for i, field := range sortFields {
		prefix := "  "
		if i == m.portfolioSortCursor {
			prefix = "► "
		}

		fieldName := m.getSortFieldName(field)
		if m.portfolioIsSorted && m.portfolioSortField == field {
			// Show current sort state (only when sorted)
			directionName := m.getSortDirectionName(m.portfolioSortDirection)
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

// Stock calculation methods
func (s *Stock) CalculatePositionProfit() float64 {
	// Use simplified weighted average cost calculation
	return (s.Price - s.CostPrice) * float64(s.Quantity)
}

func (s *Stock) CalculateWeightedAverageCost() float64 {
	return s.CostPrice // Directly return cost price
}

func (s *Stock) CalculateTotalQuantity() int {
	return s.Quantity // Directly return quantity
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(consts.RefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}
