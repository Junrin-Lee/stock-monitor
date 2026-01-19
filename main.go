package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"stock-monitor/internal/api"
	"stock-monitor/internal/types"
	"stock-monitor/internal/ui"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jedib0t/go-pretty/v6/table"
)

// globalModel 全局模型引用，用于日志记录的 i18n 支持
var globalModel *Model

// convertStockData converts types.StockData to main.StockData
func convertStockData(data *types.StockData) *StockData {
	if data == nil {
		return nil
	}
	return &StockData{
		Symbol:        data.Symbol,
		Name:          data.Name,
		Price:         data.Price,
		Change:        data.Change,
		ChangePercent: data.ChangePercent,
		StartPrice:    data.StartPrice,
		MaxPrice:      data.MaxPrice,
		MinPrice:      data.MinPrice,
		PrevClose:     data.PrevClose,
		TurnoverRate:  data.TurnoverRate,
		Volume:        data.Volume,
	}
}

// 获取主菜单项
// i18n 相关函数已移动到 i18n.go

// 获取主菜单项
func (m *Model) getMenuItems() []string {
	return []string{
		m.getText("stockList"),
		m.getText("watchlist"),
		m.getText("stockSearch"),
		m.getText("alertManagement"),
		m.getText("language"),
		m.getText("exit"),
	}
}

func main() {
	// 确保目录存在
	os.MkdirAll("data", 0755)
	os.MkdirAll("cmd/conf", 0755)
	os.MkdirAll("i18n", 0755)

	// 初始化日志系统（在所有初始化之前）
	logDir := filepath.Join(".", "data", "logs")
	logLevel := LogInfo // 默认 INFO 级别
	if err := InitLogger(logDir, logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer globalLogger.Sync()

	// 启动节假日数据同步 worker（异步执行，不阻塞主流程）
	StartHolidayWorker()

	// 加载 i18n 文件
	loadI18nFiles()

	// 初始化列注册表
	ui.InitColumnRegistry()

	// 加载配置文件
	config := loadConfig()
	portfolio := loadPortfolio()
	watchlist := loadWatchlist()

	// 根据配置和是否有股票数据决定初始状态
	initialState := MainMenu
	var lastUpdate time.Time
	if config.System.AutoStart {
		// 根据startup_module配置决定启动哪个模块
		switch config.System.StartupModule {
		case "portfolio":
			// 启动持股模块，需要有持股数据
			if len(portfolio.Stocks) > 0 {
				initialState = Monitoring
				lastUpdate = time.Now()
			}
		case "watchlist":
			// 启动自选模块，需要有自选数据
			if len(watchlist.Stocks) > 0 {
				initialState = WatchlistViewing
				lastUpdate = time.Now()
			}
		default:
			// 默认行为：如果有持股数据则进入持股模块
			if len(portfolio.Stocks) > 0 {
				initialState = Monitoring
				lastUpdate = time.Now()
			}
		}
	}

	// 根据配置文件设置语言
	language := English // 默认英文
	if config.System.Language == "zh" {
		language = Chinese
	}

	m := Model{
		state:              initialState,
		currentMenuItem:    0,
		portfolio:          portfolio,
		watchlist:          watchlist,
		config:             config,
		language:           language,
		lastUpdate:         lastUpdate,
		alertData:          loadAlertData(), // 启动时加载告警数据
		portfolioScrollPos: 0,               // 持股列表滚动位置
		watchlistScrollPos: 0,               // 自选列表滚动位置
		portfolioCursor:    0,               // 持股列表游标
		watchlistCursor:    0,               // 自选列表游标
		portfolioIsSorted:  false,           // 持股列表默认未排序状态
		watchlistIsSorted:  false,           // 自选列表默认未排序状态
		// 股价缓存初始化
		stockPriceCache:      make(map[string]*StockPriceCacheEntry),
		stockPriceUpdateTime: time.Time{}, // 初始化为零时间
	}

	// 根据语言设置菜单项
	m.menuItems = m.getMenuItems()

	// 设置全局模型引用用于调试日志
	globalModel = &m

	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func (m *Model) Init() tea.Cmd {
	if m.state == Monitoring || m.state == WatchlistViewing {
		return m.tickCmd()
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var newModel tea.Model
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// 持股列表和自选列表滚动快捷键
		if m.state == Monitoring || m.state == WatchlistViewing {
			keyStr := msg.String()
			switch keyStr {
			case "up":
				if m.state == Monitoring {
					m.scrollPortfolioUp()
				} else {
					m.scrollWatchlistUp()
				}
				return m, nil
			case "down":
				if m.state == Monitoring {
					m.scrollPortfolioDown()
				} else {
					m.scrollWatchlistDown()
				}
				return m, nil
			}
		}

		// 处理各状态的正常按键
		switch m.state {
		case MainMenu:
			newModel, cmd = m.handleMainMenu(msg)
		case AddingStock:
			newModel, cmd = m.handleAddingStock(msg)
		case Monitoring:
			newModel, cmd = m.handleMonitoring(msg)
		case EditingStock:
			newModel, cmd = m.handleEditingStock(msg)
		case SearchingStock:
			newModel, cmd = m.handleSearchingStock(msg)
		case SearchResult:
			newModel, cmd = m.handleSearchResult(msg)
		case SearchResultWithActions:
			newModel, cmd = m.handleSearchResultWithActions(msg)
		case WatchlistSearchConfirm:
			newModel, cmd = m.handleWatchlistSearchConfirm(msg)
		case LanguageSelection:
			newModel, cmd = m.handleLanguageSelection(msg)
		case WatchlistViewing:
			newModel, cmd = m.handleWatchlistViewing(msg)
		case WatchlistTagging:
			newModel, cmd = m.handleWatchlistTagging(msg)
		case WatchlistTagSelect:
			newModel, cmd = m.handleWatchlistTagSelect(msg)
		case WatchlistTagManage:
			newModel, cmd = m.handleWatchlistTagManage(msg)
		case WatchlistTagRemoveSelect:
			newModel, cmd = m.handleWatchlistTagRemoveSelect(msg)
		case WatchlistTagEdit:
			newModel, cmd = m.handleWatchlistTagEdit(msg)
		case WatchlistGroupSelect:
			newModel, cmd = m.handleWatchlistGroupSelect(msg)
		case PortfolioSorting:
			newModel, cmd = m.handlePortfolioSorting(msg)
		case WatchlistSorting:
			newModel, cmd = m.handleWatchlistSorting(msg)
		case IntradayChartViewing:
			newModel, cmd = m.handleIntradayChartViewing(msg)
		case AlertManage:
			newModel, cmd = m.handleAlertManage(msg)
		case StockAlertManage:
			newModel, cmd = m.handleStockAlertManage(msg)
		case AlertAdd:
			newModel, cmd = m.handleAlertAdd(msg)
		case AlertEdit:
			newModel, cmd = m.handleAlertEdit(msg)
		case AlertBatchMethodSelect:
			newModel, cmd = m.handleAlertBatchMethodSelect(msg)
		case AlertBatchByTag:
			newModel, cmd = m.handleAlertBatchByTag(msg)
		case AlertBatchByMarket:
			newModel, cmd = m.handleAlertBatchByMarket(msg)
		case SelectBatchStocks:
			newModel, cmd = m.handleSelectBatchStocks(msg)
		case SelectBatchFromWatchlist:
			newModel, cmd = m.handleSelectBatchFromWatchlist(msg)
		case SelectBatchFromPortfolio:
			newModel, cmd = m.handleSelectBatchFromPortfolio(msg)
		case InputBatchCodes:
			newModel, cmd = m.handleInputBatchCodes(msg)
		case AlertBatchAdd:
			newModel, cmd = m.handleAlertBatchAdd(msg)
		default:
			newModel, cmd = m, nil
		}
	case tickMsg:
		if m.state == Monitoring || m.state == WatchlistViewing {
			m.lastUpdate = time.Now()

			// 启动异步数据更新
			var cmds []tea.Cmd
			cmds = append(cmds, m.tickCmd())

			// 启动股价数据更新（持股和自选页面都需要）
			if stockPriceCmd := m.startStockPriceUpdates(); stockPriceCmd != nil {
				cmds = append(cmds, stockPriceCmd)
			}

			// 【新增】检查告警
			if len(m.alertData.Alerts) > 0 {
				cmds = append(cmds, func() tea.Msg {
					return checkAlertsMsg{}
				})
			}

			newModel, cmd = m, tea.Batch(cmds...)
		} else {
			newModel, cmd = m, nil
		}
	case fetchStockPriceTriggerMsg:
		// 触发单个股票的价格获取（两阶段更新模式）
		newModel, cmd = m, fetchStockPriceCmd(msg.symbol)
	case checkAlertsMsg:
		// 处理告警检查
		newModel, cmd = m.handleCheckAlerts(msg)
	case stockPriceUpdateMsg:
		// 处理股价数据更新
		if msg.Error == nil && msg.Data != nil {
			// 更新缓存
			m.stockPriceMutex.Lock()
			if entry, exists := m.stockPriceCache[msg.Symbol]; exists {
				entry.Data = msg.Data
				entry.UpdateTime = time.Now()
				entry.IsUpdating = false
			} else {
				m.stockPriceCache[msg.Symbol] = &StockPriceCacheEntry{
					Data:       msg.Data,
					UpdateTime: time.Now(),
					IsUpdating: false,
				}
			}
			m.stockPriceMutex.Unlock()
			logDebug("log.cache.updated", msg.Symbol)

			// 如果当前在自选列表且已启用排序，重新应用排序以保持顺序正确
			if m.state == WatchlistViewing && m.watchlistIsSorted {
				m.optimizedSortWatchlist(m.watchlistSortField, m.watchlistSortDirection)
			}

			// 如果当前在持股列表且已启用排序，先更新价格数据再重新排序
			if m.state == Monitoring && m.portfolioIsSorted {
				m.updatePortfolioPricesFromCache()
				m.optimizedSortPortfolio(m.portfolioSortField, m.portfolioSortDirection)
			}
		} else {
			// 更新失败，标记为未更新状态
			m.stockPriceMutex.Lock()
			if entry, exists := m.stockPriceCache[msg.Symbol]; exists {
				entry.IsUpdating = false
			}
			m.stockPriceMutex.Unlock()
			logError("log.cache.error", msg.Symbol, msg.Error)
		}
		newModel, cmd = m, nil
	case checkDataAvailabilityMsg:
		// 处理数据可用性检查during auto-collection
		if m.state == IntradayChartViewing && m.chartIsCollecting {
			data, err := m.loadIntradayDataForDate(msg.code, m.chartViewStockName, msg.date)
			if err == nil {
				// 数据现在可用!
				m.chartData = data
				m.chartIsCollecting = false
				m.chartLoadError = nil
				newModel, cmd = m, nil
			} else {
				// 仍在等待 - 2 秒后再次检查 (最多 30 秒超时)
				if time.Since(m.chartCollectStartTime) < 30*time.Second {
					newModel, cmd = m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
						return checkDataAvailabilityMsg{code: msg.code, date: msg.date}
					})
				} else {
					// 超时 - 显示错误
					m.chartLoadError = fmt.Errorf("data collection timeout")
					m.chartIsCollecting = false
					newModel, cmd = m, nil
				}
			}
		} else {
			newModel, cmd = m, nil
		}
	case searchIntradayUpdateMsg:
		// 搜索模式分时数据更新，触发 UI 重新渲染
		// 继续监听下一次更新
		newModel, cmd = m, m.waitForSearchIntradayUpdate()
	default:
		newModel, cmd = m, nil
	}

	// 更新全局模型引用以保持调试日志同步
	if newModel != nil {
		if modelPtr, ok := newModel.(*Model); ok {
			globalModel = modelPtr
		}
	}

	return newModel, cmd
}

func (m *Model) View() string {
	var mainContent string
	switch m.state {
	case MainMenu:
		mainContent = m.viewMainMenu()
	case AddingStock:
		mainContent = m.viewAddingStock()
	case Monitoring:
		mainContent = m.viewMonitoring()
	case EditingStock:
		mainContent = m.viewEditingStock()
	case SearchingStock:
		mainContent = m.viewSearchingStock()
	case SearchResult:
		mainContent = m.viewSearchResult()
	case SearchResultWithActions:
		mainContent = m.viewSearchResultWithActions()
	case WatchlistSearchConfirm:
		mainContent = m.viewWatchlistSearchConfirm()
	case LanguageSelection:
		mainContent = m.viewLanguageSelection()
	case WatchlistViewing:
		mainContent = m.viewWatchlistViewing()
	case WatchlistTagging:
		mainContent = m.viewWatchlistTagging()
	case WatchlistTagSelect:
		mainContent = m.viewWatchlistTagSelect()
	case WatchlistTagManage:
		mainContent = m.viewWatchlistTagManage()
	case WatchlistTagRemoveSelect:
		mainContent = m.viewWatchlistTagRemoveSelect()
	case WatchlistTagEdit:
		mainContent = m.viewWatchlistTagEdit()
	case WatchlistGroupSelect:
		mainContent = m.viewWatchlistGroupSelect()
	case PortfolioSorting:
		mainContent = m.viewPortfolioSorting()
	case WatchlistSorting:
		mainContent = m.viewWatchlistSorting()
	case IntradayChartViewing:
		// 获取终端尺寸 - 使用合理的默认值
		termWidth := 120
		termHeight := 30
		mainContent = m.viewIntradayChart(termWidth, termHeight)
	case AlertManage:
		mainContent = m.viewAlertManage()
	case StockAlertManage:
		mainContent = m.viewStockAlertManage()
	case AlertAdd:
		mainContent = m.viewAlertAdd()
	case AlertEdit:
		mainContent = m.viewAlertEdit()
	case AlertBatchMethodSelect:
		mainContent = m.viewAlertBatchMethodSelect()
	case AlertBatchByTag:
		mainContent = m.viewAlertBatchByTag()
	case AlertBatchByMarket:
		mainContent = m.viewAlertBatchByMarket()
	case SelectBatchStocks:
		mainContent = m.viewSelectBatchStocks()
	case SelectBatchFromWatchlist:
		mainContent = m.viewSelectBatchFromWatchlist()
	case SelectBatchFromPortfolio:
		mainContent = m.viewSelectBatchFromPortfolio()
	case InputBatchCodes:
		mainContent = m.viewInputBatchCodes()
	case AlertBatchAdd:
		mainContent = m.viewAlertBatchAdd()
	default:
		mainContent = ""
	}

	return mainContent
}

func (m *Model) handleMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "w":
		m.currentMenuItem = MoveCursorUp(m.currentMenuItem)
		m.message = "" // 清除消息
	case "down", "j", "s":
		m.currentMenuItem = MoveCursorDown(m.currentMenuItem, len(m.menuItems)-1)
		m.message = "" // 清除消息
	case "enter", " ":
		return m.executeMenuItem()
	case "q", "ctrl+c":
		m.savePortfolio()
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) executeMenuItem() (tea.Model, tea.Cmd) {
	m.message = "" // 清除之前的消息
	switch m.currentMenuItem {
	case 0: // 股票列表
		logInfo("log.action.enterPortfolio")
		m.state = Monitoring
		m.resetPortfolioCursor() // 重置游标到第一只股票
		m.lastUpdate = time.Now()

		// 启动分时数据采集
		m.startIntradayDataCollection()

		return m, m.tickCmd()
	case 1: // 自选股票
		logInfo("log.action.enterWatchlist")
		m.state = WatchlistViewing
		m.resetWatchlistCursor() // 重置游标到第一只股票
		m.cursor = 0
		m.message = ""
		m.lastUpdate = time.Now()

		// 启动分时数据采集
		m.startIntradayDataCollection()

		// 立即启动数据更新，而不等待定时器
		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())

		// 强制启动股价数据更新
		if stockPriceCmd := m.startStockPriceUpdates(); stockPriceCmd != nil {
			cmds = append(cmds, stockPriceCmd)
		}

		return m, tea.Batch(cmds...)
	case 2: // 股票搜索
		logInfo("log.action.enterSearch")
		m.state = SearchingStock
		m.searchInput = ""
		m.searchResult = nil
		m.searchFromWatchlist = false
		m.message = ""
		return m, nil
	case 3: // 告警管理
		logInfo("log.action.enterAlertManagement")
		m.state = AlertManage
		m.alertCursor = 0
		m.alertData = loadAlertData()
		return m, nil
	case 4: // 语言选择页面
		logInfo("log.action.enterLanguage")
		m.state = LanguageSelection
		m.languageCursor = 0
		if m.language == English {
			m.languageCursor = 1
		}
		return m, nil
	case 5: // 退出
		logInfo("log.action.exit")
		m.savePortfolio()
		m.saveWatchlist()
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) viewMainMenu() string {
	s := m.getText("title") + "\n\n"

	for i, item := range m.menuItems {
		prefix := "  "
		if i == m.currentMenuItem {
			prefix = "► "
		}

		if i == 4 { // 语言选择
			langStatus := m.getText("english")
			if m.language == Chinese {
				langStatus = m.getText("chinese")
			}
			s += fmt.Sprintf("%s%s: %s\n", prefix, item, langStatus)
		} else {
			s += fmt.Sprintf("%s%s\n", prefix, item)
		}
	}

	s += "\n"
	if runtime.GOOS == "windows" {
		s += m.getText("keyHelpWin") + "\n"
	} else {
		s += m.getText("keyHelp") + "\n"
	}
	s += "==================================================\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

func (m *Model) handleAddingStock(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// 根据来源决定返回目标
		if m.fromSearch {
			// 从持股列表或搜索结果进入，返回相应页面
			if m.previousState == Monitoring {
				m.state = Monitoring
				m.resetPortfolioCursor() // 重置游标到第一只股票
				m.lastUpdate = time.Now()
			} else {
				m.state = SearchResultWithActions
			}
			m.fromSearch = false // 重置标志
		} else {
			m.state = MainMenu
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
		m.input, m.inputCursor = deleteRuneBeforeCursor(m.input, m.inputCursor)
		return m, nil
	case "delete", "ctrl+d":
		m.input, m.inputCursor = deleteRuneAtCursor(m.input, m.inputCursor)
		return m, nil
	default:
		// 改进的输入处理：支持多字节字符（如中文）
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !isControlKey(str) {
			m.input, m.inputCursor = insertStringAtCursor(m.input, m.inputCursor, str)
		}
	}
	return m, nil
}

func (m *Model) processAddingStep() (tea.Model, tea.Cmd) {
	switch m.addingStep {
	case 0: // 搜索股票
		if m.input == "" {
			m.message = m.getText("codeRequired")
			return m, nil
		}
		m.message = m.getText("searching")

		// 使用搜索功能
		var stockData *StockData
		if api.ContainsChineseChars(m.input) {
			stockData = convertStockData(api.SearchChineseStock(m.input))
		} else {
			// 对于非中文输入，先尝试直接获取价格，然后尝试搜索
			stockData = convertStockData(api.GetStockPrice(m.input))

			// 如果直接获取失败，尝试作为搜索关键词搜索
			if stockData == nil || stockData.Price <= 0 {
				logWarn("log.api.addStockFail", m.input)
				stockData = convertStockData(api.SearchStockBySymbol(m.input))
			}
		}

		if stockData == nil || stockData.Name == "" {
			m.message = fmt.Sprintf(m.getText("searchNotFound"), m.input)
			m.input = ""
			m.inputCursor = 0
			return m, nil
		}

		// 保存搜索结果并转到输入成本价步骤
		m.stockInfo = stockData
		m.tempCode = stockData.Symbol
		m.addingStep = 1
		m.input = ""
		m.inputCursor = 0
		m.message = ""
	case 1: // 输入成本价
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
	case 2: // 输入数量
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

		// 添加股票
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
		m.portfolioIsSorted = false // 添加股票后重置持股列表排序状态

		// 根据来源决定跳转目标
		if m.fromSearch {
			// 从搜索结果添加，跳转到持股列表（监控）页面
			m.state = Monitoring
			m.resetPortfolioCursor() // 重置游标到第一只股票
			m.lastUpdate = time.Now()
			m.fromSearch = false // 重置标志
			m.message = fmt.Sprintf(m.getText("addSuccess"), m.stockInfo.Name, m.tempCode)
			m.addingStep = 0
			m.input = ""
			return m, m.tickCmd() // 跳转到监控页面时启动定时器
		} else {
			// 从主菜单添加，返回主菜单
			m.state = MainMenu
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
		s += m.getText("enterSearch") + formatTextWithCursor(m.input, m.inputCursor) + "\n"
		s += "\n" + m.getText("searchFormats") + "\n"
	case 1:
		s += fmt.Sprintf(m.getText("stockCode"), m.tempCode) + "\n"
		s += fmt.Sprintf(m.getText("stockName"), m.stockInfo.Name) + "\n"
		s += fmt.Sprintf(m.getText("currentPrice"), m.stockInfo.Price) + "\n\n"
		s += m.getText("enterCost") + formatTextWithCursor(m.input, m.inputCursor) + "\n"
	case 2:
		s += fmt.Sprintf(m.getText("stockCode"), m.tempCode) + "\n"
		s += fmt.Sprintf(m.getText("stockName"), m.stockInfo.Name) + "\n"
		s += fmt.Sprintf(m.getText("currentPrice"), m.stockInfo.Price) + "\n"
		s += fmt.Sprintf(m.getText("costPrice"), m.tempCost) + "\n\n"
		s += m.getText("enterQuantity") + formatTextWithCursor(m.input, m.inputCursor) + "\n"
	}

	// 添加光标操作提示
	if m.language == Chinese {
		s += "\n操作: ←/→移动光标, Enter确认, ESC返回, Home/End跳转首尾\n"
	} else {
		s += "\nActions: ←/→ move cursor, Enter confirm, ESC back, Home/End jump\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// handleMonitoringNavigation 处理投资组合监控页面的导航操作
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

// handleMonitoringActions 处理投资组合监控页面的数据操作
func (m *Model) handleMonitoringActions(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "esc", "q", "m":
		m.stopIntradayDataCollection() // 停止分时数据采集
		m.state = MainMenu
		m.message = "" // 清除消息
		return m, nil, true
	case "e":
		// 编辑当前光标指向的股票
		if len(m.portfolio.Stocks) == 0 {
			m.message = m.getText("emptyPortfolio")
			return m, nil, true
		}
		logInfo("log.action.enterEdit")
		m.previousState = m.state // 记录当前状态
		m.state = EditingStock
		m.editingStep = 1 // 开始编辑成本价
		m.selectedStockIndex = m.portfolioCursor
		m.tempCode = m.portfolio.Stocks[m.portfolioCursor].Code
		m.tempCost = ""
		m.tempQuantity = ""
		m.input = fmt.Sprintf("%.*f", m.config.Display.DecimalPlaces, m.portfolio.Stocks[m.portfolioCursor].CostPrice) // 预填充当前成本价,使用配置的小数位数
		m.inputCursor = len([]rune(m.input))                                                                           // 光标放到末尾
		m.message = ""
		return m, nil, true
	case "d":
		// 直接删除光标指向的股票
		if len(m.portfolio.Stocks) == 0 {
			m.message = m.getText("emptyPortfolio")
			return m, nil, true
		}
		// 删除当前光标指向的股票
		removedStock := m.portfolio.Stocks[m.portfolioCursor]
		m.portfolio.Stocks = append(m.portfolio.Stocks[:m.portfolioCursor], m.portfolio.Stocks[m.portfolioCursor+1:]...)
		m.savePortfolio()
		m.portfolioIsSorted = false // 删除股票后重置持股列表排序状态
		// 调整光标位置
		if m.portfolioCursor >= len(m.portfolio.Stocks) && len(m.portfolio.Stocks) > 0 {
			m.portfolioCursor = len(m.portfolio.Stocks) - 1
		}
		m.message = fmt.Sprintf(m.getText("removeSuccess"), removedStock.Name, removedStock.Code)
		return m, nil, true
	case "a":
		// 跳转到添加股票页面
		logInfo("log.action.enterAdd")
		m.previousState = m.state // 记录当前状态
		m.state = AddingStock
		m.addingStep = 0
		m.tempCode = ""
		m.tempCost = ""
		m.tempQuantity = ""
		m.stockInfo = nil
		m.input = ""
		m.message = ""
		m.fromSearch = true // 设置标志,表示从持股列表进入,完成后应该回到监控页面
		return m, nil, true
	}
	return m, nil, false
}

// handleMonitoringViews 处理投资组合监控页面的视图切换
func (m *Model) handleMonitoringViews(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "v":
		// 查看分时图表
		if len(m.portfolio.Stocks) == 0 {
			m.message = m.getText("emptyPortfolio")
			return m, nil, true
		}
		selectedStock := m.portfolio.Stocks[m.portfolioCursor]
		m.chartViewStock = selectedStock.Code
		m.chartViewStockName = selectedStock.Name

		// 获取智能日期(与 worker 采集逻辑一致)
		actualDate, _, err := GetTradingDayForCollection(selectedStock.Code, m)
		if err != nil {
			// 如果获取失败,降级为简单逻辑
			actualDate = getSmartChartDate()
		}
		m.chartViewDate = actualDate
		m.previousState = Monitoring

		logDebug("log.chart.keyV", selectedStock.Code, selectedStock.Name, m.chartViewDate)

		// 尝试加载数据
		data, loadErr := m.loadIntradayDataForDate(
			selectedStock.Code,
			selectedStock.Name,
			actualDate,
		)

		if loadErr != nil {
			// 无数据 - 触发采集
			logDebug("log.chart.noData", loadErr)
			m.chartData = nil
			m.chartLoadError = nil
			m.state = IntradayChartViewing
			return m, m.triggerIntradayDataCollection(
				selectedStock.Code,
				selectedStock.Name,
				actualDate,
			), true
		}

		// 数据存在 - 创建图表
		logDebug("log.chart.dataLoaded", len(data.Datapoints))
		m.chartData = data
		m.chartLoadError = nil
		m.chartIsCollecting = false
		m.state = IntradayChartViewing
		return m, nil, true
	case "s":
		// 进入排序菜单
		logInfo("log.action.enterSort")
		m.state = PortfolioSorting
		// 智能定位光标到当前排序字段
		m.portfolioSortCursor = m.findSortFieldIndex(m.portfolioSortField, true)
		m.message = ""
		return m, nil, true
	case "l", "L":
		// 查看股票告警详情
		if len(m.portfolio.Stocks) == 0 {
			m.message = m.getText("emptyPortfolio")
			return m, nil, true
		}
		currentStock := m.portfolio.Stocks[m.portfolioCursor]

		// 设置股票告警详情参数
		m.stockAlertCode = currentStock.Code
		m.stockAlertName = currentStock.Name
		m.stockAlertCursor = 0
		m.previousState = Monitoring // 记录返回状态

		// 获取该股票的所有告警
		m.stockAlertAlerts = m.getStockAlerts(currentStock.Code)

		logInfo("log.action.enterStockAlertManagement", currentStock.Code, currentStock.Name)
		m.state = StockAlertManage
		m.message = ""
		return m, nil, true
	}
	return m, nil, false
}

func (m *Model) handleMonitoring(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 尝试导航处理
	if model, cmd, handled := m.handleMonitoringNavigation(msg); handled {
		return model, cmd
	}

	// 尝试操作处理
	if model, cmd, handled := m.handleMonitoringActions(msg); handled {
		return model, cmd
	}

	// 尝试视图切换处理
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

	// 获取带排序指示器的表头
	t.AppendHeader(m.GeneratePortfolioHeader())

	var totalMarketValue float64
	var totalCost float64

	// 显示滚动信息
	totalStocks := len(m.portfolio.Stocks)
	maxPortfolioLines := m.config.Display.MaxLines
	if totalStocks > 0 {
		currentPos := m.portfolioCursor + 1 // 显示从1开始的位置
		if m.language == Chinese {
			s += fmt.Sprintf("📊 持股列表 (%d/%d) [↑/↓:翻页]\n", currentPos, totalStocks)
		} else {
			s += fmt.Sprintf("📊 Portfolio (%d/%d) [↑/↓:scroll]\n", currentPos, totalStocks)
		}
		s += "\n"
	}

	// 计算要显示的股票范围
	stocks := m.portfolio.Stocks
	endIndex := len(stocks) - m.portfolioScrollPos
	startIndex := endIndex - maxPortfolioLines
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(stocks) {
		endIndex = len(stocks)
	}

	// 首先计算所有股票的总计（用于汇总行）
	for i := range m.portfolio.Stocks {
		stock := &m.portfolio.Stocks[i]
		// 从缓存获取股价数据（非阻塞）
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

	// 然后显示当前范围内的股票
	for i := startIndex; i < endIndex; i++ {
		stock := &m.portfolio.Stocks[i]

		// 使用动态列渲染器生成行
		row := m.GeneratePortfolioRow(stock, i, startIndex, endIndex)
		t.AppendRow(row)

		// 在每个股票后添加分隔线（除了显示范围内的最后一个）
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
	// 使用动态列渲染器生成总计行
	totalRow := m.GeneratePortfolioTotalRow(totalPortfolioProfit, totalProfitRate, totalMarketValue)
	t.AppendRow(totalRow)

	s += t.Render() + "\n"

	// 如果可以滚动，显示滚动指示
	if totalStocks > maxPortfolioLines {
		s += strings.Repeat("-", 80) + "\n"
		if m.portfolioScrollPos > 0 {
			if m.language == Chinese {
				s += "↑ 有更新的股票 (按↓查看)\n"
			} else {
				s += "↑ Newer stocks available (press ↓)\n"
			}
		}
		if m.portfolioScrollPos < totalStocks-1 {
			if m.language == Chinese {
				s += "↓ 有更多历史股票 (按↑查看)\n"
			} else {
				s += "↓ More stocks available (press ↑)\n"
			}
		}
	}

	s += "\n" + m.getText("holdingsHelp") + "\n"

	return s
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// savePortfolio, getDefaultConfig, loadConfig, saveConfig 已移动到 persistence.go

// 计算股票的持仓盈亏
func (s *Stock) CalculatePositionProfit() float64 {
	// 使用简化的加权平均成本价计算
	return (s.Price - s.CostPrice) * float64(s.Quantity)
}

// 计算股票的加权平均成本价
func (s *Stock) CalculateWeightedAverageCost() float64 {
	return s.CostPrice // 直接返回成本价
}

// 计算总持股数量
func (s *Stock) CalculateTotalQuantity() int {
	return s.Quantity // 直接返回持股数量
}

// loadPortfolio 已移动到 persistence.go

// 格式化函数 (formatProfitWithColorLang, formatPriceWithColorLang, abs 等) 已移动到 format.go

// API 相关函数 (getStockInfo, getStockPrice, searchStock*, tryXXXAPI 等) 已移动到 api.go
// 缓存相关函数 (getStockPriceFromCache, startStockPriceUpdates) 已移动到 cache.go
// scroll 相关函数 (scrollPortfolioUp/Down, scrollWatchlistUp/Down) 已移动到 scroll.go

func (m *Model) handleEditingStock(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// 根据之前的状态决定返回到哪里
		if m.previousState == Monitoring {
			m.state = Monitoring
			m.resetPortfolioCursor() // 重置游标到第一只股票
			m.lastUpdate = time.Now()
			m.message = ""
			m.inputCursor = 0
			return m, m.tickCmd()
		} else {
			m.state = MainMenu
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
		m.input, m.inputCursor = deleteRuneBeforeCursor(m.input, m.inputCursor)
		return m, nil
	case "delete", "ctrl+d":
		m.input, m.inputCursor = deleteRuneAtCursor(m.input, m.inputCursor)
		return m, nil
	default:
		// 改进的输入处理：支持多字节字符（如中文）
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !isControlKey(str) {
			m.input, m.inputCursor = insertStringAtCursor(m.input, m.inputCursor, str)
		}
	}
	return m, nil
}

func (m *Model) processEditingStep() (tea.Model, tea.Cmd) {
	switch m.editingStep {
	case 1: // 修改成本价
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
			m.inputCursor = len([]rune(m.input)) // 光标放到末尾
			m.message = ""
		}
	case 2: // 修改数量
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
			m.portfolioIsSorted = false // 修改股票后重置持股列表排序状态

			stockName := m.portfolio.Stocks[m.selectedStockIndex].Name
			// 根据之前的状态决定返回到哪里
			if m.previousState == Monitoring {
				m.state = Monitoring
				m.resetPortfolioCursor() // 重置游标到第一只股票
				m.lastUpdate = time.Now()
				m.message = fmt.Sprintf(m.getText("editSuccess"), stockName)
				m.editingStep = 0
				m.input = ""
				m.inputCursor = 0
				return m, m.tickCmd()
			} else {
				m.state = MainMenu
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
		if m.language == Chinese {
			s += fmt.Sprintf("股票: %s (%s)\n", stock.Name, stock.Code)
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n", stock.Name, stock.Code)
		}
		s += fmt.Sprintf(m.getText("currentCost"), stock.CostPrice) + "\n\n"
		s += m.getText("enterNewCost") + formatTextWithCursor(m.input, m.inputCursor) + "\n"
	case 2:
		stock := m.portfolio.Stocks[m.selectedStockIndex]
		if m.language == Chinese {
			s += fmt.Sprintf("股票: %s (%s)\n", stock.Name, stock.Code)
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n", stock.Name, stock.Code)
		}
		s += fmt.Sprintf(m.getText("newCost"), stock.CostPrice) + "\n"
		s += fmt.Sprintf(m.getText("currentQuantity"), stock.Quantity) + "\n\n"
		s += m.getText("enterNewQuantity") + formatTextWithCursor(m.input, m.inputCursor) + "\n"
	}

	// 添加光标操作提示
	if m.language == Chinese {
		s += "\n操作: ←/→移动光标, Enter确认, ESC/Q返回, Home/End跳转首尾\n"
	} else {
		s += "\nActions: ←/→ move cursor, Enter confirm, ESC/Q back, Home/End jump\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

func (m *Model) handleSearchingStock(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.searchFromWatchlist {
			m.state = WatchlistViewing
			m.resetWatchlistCursor() // 重置游标到第一只股票
			m.searchFromWatchlist = false
			m.searchInput = ""
			m.searchInputCursor = 0
			m.message = ""
			return m, m.tickCmd() // 重启定时器
		} else {
			m.state = MainMenu
		}
		m.searchInput = ""
		m.searchInputCursor = 0
		m.message = ""
		return m, nil
	case "enter":
		if m.searchInput == "" {
			m.message = m.getText("enterSearch")[:len(m.getText("enterSearch"))-2] // 去掉": "后缀
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

		// 标记为搜索模式
		m.isSearchMode = true

		// 获取智能日期（当日或最近交易日）
		actualDate, _, err := GetTradingDayForCollection(m.searchResult.Symbol, m)
		if err != nil {
			// 降级为简单逻辑
			actualDate = getSmartChartDate()
		}

		// 设置图表参数
		m.chartViewStock = m.searchResult.Symbol
		m.chartViewStockName = m.searchResult.Name
		m.chartViewDate = actualDate

		// 清理输入
		m.searchInput = ""
		m.searchInputCursor = 0
		m.message = ""

		// 根据来源决定下一个状态
		if m.searchFromWatchlist {
			m.state = WatchlistSearchConfirm
		} else {
			m.state = SearchResultWithActions
		}

		// 两种状态都启动临时 Worker（自动显示图表）
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
		m.searchInput, m.searchInputCursor = deleteRuneBeforeCursor(m.searchInput, m.searchInputCursor)
		return m, nil
	case "delete", "ctrl+d":
		m.searchInput, m.searchInputCursor = deleteRuneAtCursor(m.searchInput, m.searchInputCursor)
		return m, nil
	default:
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !isControlKey(str) {
			m.searchInput, m.searchInputCursor = insertStringAtCursor(m.searchInput, m.searchInputCursor, str)
		}
	}
	return m, nil
}

func (m *Model) handleSearchResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = MainMenu
		m.message = ""
		return m, nil
	case "r":
		m.state = SearchingStock
		m.searchFromWatchlist = false
		m.message = ""
		return m, nil
	}
	return m, nil
}

func (m *Model) viewSearchingStock() string {
	s := m.getText("searchTitle") + "\n\n"
	s += m.getText("enterSearch") + formatTextWithCursor(m.searchInput, m.searchInputCursor) + "\n\n"
	s += m.getText("searchFormats") + "\n\n"

	if m.language == Chinese {
		s += "操作: ←/→移动光标, Enter搜索, ESC返回, Home/End跳转首尾\n"
	} else {
		s += "Actions: ←/→ move cursor, Enter search, ESC back, Home/End jump\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

func (m *Model) viewSearchResult() string {
	s := m.getText("detailTitle") + "\n\n"

	if m.searchResult == nil {
		s += m.getText("noInfo") + "\n"
		s += "\n" + m.getText("detailHelp") + "\n"
		return s
	}

	// 创建横向表格显示股票详细信息
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// 构建表头和数据行
	var headers []interface{}
	var values []interface{}

	// 基本信息
	if m.language == Chinese {
		headers = append(headers, "股票代码", "股票名称", "现价")
	} else {
		headers = append(headers, "Code", "Name", "Price")
	}
	values = append(values, m.searchResult.Symbol, m.searchResult.Name, m.formatPriceWithColorLang(m.searchResult.Price, m.searchResult.PrevClose))

	// 昨收价
	if m.searchResult.PrevClose > 0 {
		if m.language == Chinese {
			headers = append(headers, "昨收价")
		} else {
			headers = append(headers, "Prev Close")
		}
		values = append(values, fmt.Sprintf("%.3f", m.searchResult.PrevClose))
	}

	// 价格信息（有数据时才显示）
	if m.searchResult.StartPrice > 0 {
		if m.language == Chinese {
			headers = append(headers, "开盘价")
		} else {
			headers = append(headers, "Open")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.StartPrice, m.searchResult.PrevClose))
	}
	if m.searchResult.MaxPrice > 0 {
		if m.language == Chinese {
			headers = append(headers, "最高价")
		} else {
			headers = append(headers, "High")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.MaxPrice, m.searchResult.PrevClose))
	}
	if m.searchResult.MinPrice > 0 {
		if m.language == Chinese {
			headers = append(headers, "最低价")
		} else {
			headers = append(headers, "Low")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.MinPrice, m.searchResult.PrevClose))
	}

	// 涨跌信息
	if m.searchResult.Change != 0 {
		if m.language == Chinese {
			headers = append(headers, "涨跌额")
		} else {
			headers = append(headers, "Change")
		}
		changeStr := m.formatProfitWithColorZeroLang(m.searchResult.Change)
		values = append(values, changeStr)
	}
	if m.searchResult.ChangePercent != 0 {
		if m.language == Chinese {
			headers = append(headers, "今日涨幅")
		} else {
			headers = append(headers, "Change %")
		}
		changePercentStr := m.formatProfitRateWithColorZeroLang(m.searchResult.ChangePercent)
		values = append(values, changePercentStr)
	}

	// 换手率
	if m.searchResult.TurnoverRate > 0 {
		if m.language == Chinese {
			headers = append(headers, "换手率")
		} else {
			headers = append(headers, "Turnover")
		}
		values = append(values, fmt.Sprintf("%.2f%%", m.searchResult.TurnoverRate))
	}

	// 买入量（成交量）
	if m.searchResult.Volume > 0 {
		if m.language == Chinese {
			headers = append(headers, "成交量")
		} else {
			headers = append(headers, "Volume")
		}
		volumeStr := formatVolume(m.searchResult.Volume)
		values = append(values, volumeStr)
	}

	// 添加表头和数据行
	t.AppendHeader(table.Row(headers))
	t.AppendRow(table.Row(values))

	s += t.Render() + "\n\n"
	s += m.getText("detailHelp") + "\n"

	return s
}

// formatVolume, isControlKey 已移动到 format.go 和 ui_utils.go

func (m *Model) handleLanguageSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = MainMenu
		m.message = "" // 清除消息
		return m, nil
	case "up", "k", "w":
		m.languageCursor = MoveCursorUp(m.languageCursor)
	case "down", "j", "s":
		m.languageCursor = MoveCursorDown(m.languageCursor, 1) // 0: Chinese, 1: English
	case "enter", " ":
		// 选择语言
		if m.languageCursor == 0 {
			m.language = Chinese
			m.config.System.Language = "zh"
		} else {
			m.language = English
			m.config.System.Language = "en"
		}
		// 保存配置到文件
		if err := saveConfig(m.config); err != nil {
			m.message = fmt.Sprintf("Warning: Failed to save config: %v", err)
		}
		// 更新菜单项
		m.menuItems = m.getMenuItems()
		m.state = MainMenu
		m.message = ""
		return m, nil
	}
	return m, nil
}

func (m *Model) viewLanguageSelection() string {
	s := m.getText("languageTitle") + "\n\n"
	s += m.getText("selectLanguage") + "\n\n"

	// 语言选项
	languages := []string{
		"中文简体",
		"English",
	}

	for i, lang := range languages {
		prefix := "  "
		if i == m.languageCursor {
			prefix = "► "
		}
		s += fmt.Sprintf("%s%s\n", prefix, lang)
	}

	s += "\n" + m.getText("languageHelp") + "\n"

	return s
}

// ========== 自选股票相关功能 ==========

// WatchlistStockLegacy, WatchlistLegacy, loadWatchlist, saveWatchlist 已移动到 persistence.go

// 标签管理函数 (renameTagForAllStocks, getAvailableTags, hasTag, addTag, removeTag, getTagsDisplay, getFilteredWatchlist, invalidateWatchlistCache) 已移动到 watchlist.go

// 重置持股列表游标到第一只股票
func (m *Model) resetPortfolioCursor() {
	if len(m.portfolio.Stocks) > 0 {
		m.portfolioCursor = 0
		maxPortfolioLines := m.config.Display.MaxLines
		if len(m.portfolio.Stocks) > maxPortfolioLines {
			// 显示前N条：滚动位置设置为显示从索引0开始的N条
			m.portfolioScrollPos = len(m.portfolio.Stocks) - maxPortfolioLines
		} else {
			// 股票数量不超过显示行数，显示全部
			m.portfolioScrollPos = 0
		}
	}
}

// 处理自选股票标签选择
func (m *Model) handleWatchlistTagSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// 根据当前选择的选项来执行操作
		if m.tagSelectCursor == len(m.availableTags) {
			// 选择了"手动输入新标签"选项
			m.state = WatchlistTagging
			m.tagInput = ""
			return m, nil
		} else if m.tagSelectCursor >= 0 && m.tagSelectCursor < len(m.availableTags) {
			// 选择了现有标签
			selectedTag := m.availableTags[m.tagSelectCursor]

			// 更新当前选中股票的标签（基于过滤后的列表）
			filteredStocks := m.getFilteredWatchlist()
			if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
				stockToTag := filteredStocks[m.watchlistCursor]

				// 在原始列表中找到该股票并添加标签
				for i, stock := range m.watchlist.Stocks {
					if stock.Code == stockToTag.Code {
						m.watchlist.Stocks[i].addTag(selectedTag)
						break
					}
				}

				m.invalidateWatchlistCache() // 使缓存失效
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
			m.resetWatchlistCursor() // 重置游标到第一只股票
			return m, m.tickCmd()    // 重启定时器
		}
		return m, nil
	case "d":
		// 进入标签删除选择模式
		filteredStocks := m.getFilteredWatchlist()
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			stockToModify := filteredStocks[m.watchlistCursor]

			// 获取该股票的有效标签（排除默认标签）
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

			// 设置删除标签的状态
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
		m.resetWatchlistCursor() // 重置游标到第一只股票
		return m, m.tickCmd()    // 重启定时器
	case "up", "k", "w":
		m.tagSelectCursor = MoveCursorUp(m.tagSelectCursor)
		return m, nil
	case "down", "j", "s":
		maxCursor := len(m.availableTags) // 包括"手动输入新标签"选项
		m.tagSelectCursor = MoveCursorDown(m.tagSelectCursor, maxCursor)
		return m, nil
	}
	return m, nil
}

// 标签选择视图
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
			s += fmt.Sprintf("当前标签: %s\n\n", stock.getTagsDisplay(m))

			// 显示该股票的标签，供删除使用
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
			s += fmt.Sprintf("Current tags: %s\n\n", stock.getTagsDisplay(m))

			// 显示该股票的标签，供删除使用
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

	// 显示现有标签选项
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

	// 添加"手动输入新标签"选项
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

// ========== 新的标签管理界面 ==========

// 处理标签管理界面
func (m *Model) handleWatchlistTagManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = WatchlistViewing
		m.message = ""
		m.resetWatchlistCursor()
		return m, m.tickCmd() // 重启定时器
	case "n":
		// 手动输入新标签
		m.state = WatchlistTagging
		m.tagInput = ""
		return m, nil
	case "d":
		// 删除当前选中的标签（如果当前股票拥有该标签）
		if len(m.availableTags) == 0 {
			if m.language == Chinese {
				m.message = "没有可删除的标签"
			} else {
				m.message = "No tags to remove"
			}
			return m, nil
		}

		// 获取当前选中的标签
		selectedTag := m.availableTags[m.tagManageCursor]

		// 检查当前股票是否拥有这个标签
		filteredStocks := m.getFilteredWatchlist()
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			currentStock := filteredStocks[m.watchlistCursor]

			// 查找并删除标签
			stockFound := false
			for i, stock := range m.watchlist.Stocks {
				if stock.Code == currentStock.Code {
					if stock.hasTag(selectedTag) {
						m.watchlist.Stocks[i].removeTag(selectedTag)
						m.saveWatchlist()
						m.invalidateWatchlistCache()

						// 更新当前股票标签列表
						m.currentStockTags = make([]string, 0)
						for _, tag := range m.watchlist.Stocks[i].Tags {
							if tag != "" && tag != "-" {
								m.currentStockTags = append(m.currentStockTags, tag)
							}
						}

						// 更新可用标签列表
						m.availableTags = m.getAvailableTags()

						// 调整光标位置
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
		// 编辑当前选中的标签
		if len(m.availableTags) == 0 {
			if m.language == Chinese {
				m.message = "没有可编辑的标签"
			} else {
				m.message = "No tags to edit"
			}
			return m, nil
		}

		// 获取当前选中的标签
		selectedTag := m.availableTags[m.tagManageCursor]

		// 进入标签编辑状态
		m.state = WatchlistTagEdit
		m.tagToEdit = selectedTag
		m.tagEditInput = selectedTag                    // 预填充当前标签名称
		m.tagEditInputCursor = len([]rune(selectedTag)) // 光标放在末尾
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
		// 为当前股票添加选中的标签
		if len(m.availableTags) == 0 {
			if m.language == Chinese {
				m.message = "没有可添加的标签，按N键创建新标签"
			} else {
				m.message = "No tags to add, press N to create new tag"
			}
			return m, nil
		}

		selectedTag := m.availableTags[m.tagManageCursor]

		// 获取当前选中的股票
		filteredStocks := m.getFilteredWatchlist()
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			currentStock := filteredStocks[m.watchlistCursor]

			// 查找并添加标签
			stockFound := false
			for i, stock := range m.watchlist.Stocks {
				if stock.Code == currentStock.Code {
					if !stock.hasTag(selectedTag) {
						m.watchlist.Stocks[i].addTag(selectedTag)
						m.saveWatchlist()
						m.invalidateWatchlistCache()

						// 更新当前股票标签列表
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

// 处理标签删除选择界面
func (m *Model) handleWatchlistTagRemoveSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = WatchlistTagManage
		return m, nil
	case "enter":
		if m.tagRemoveCursor >= 0 && m.tagRemoveCursor < len(m.currentStockTags) {
			tagToRemove := m.currentStockTags[m.tagRemoveCursor]

			// 从当前股票中删除选中的标签
			filteredStocks := m.getFilteredWatchlist()
			if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
				stockToModify := filteredStocks[m.watchlistCursor]

				// 在原始列表中找到该股票并删除指定标签
				for i, stock := range m.watchlist.Stocks {
					if stock.Code == stockToModify.Code {
						m.watchlist.Stocks[i].removeTag(tagToRemove)
						// 如果删除后没有标签，添加默认标签
						if len(m.watchlist.Stocks[i].Tags) == 0 {
							m.watchlist.Stocks[i].Tags = []string{"-"}
						}
						break
					}
				}

				m.invalidateWatchlistCache()
				m.saveWatchlist()

				// 更新当前股票标签列表
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

				// 如果没有更多标签可删除，返回标签管理界面
				if len(m.currentStockTags) == 0 {
					m.state = WatchlistTagManage
				} else {
					// 调整光标位置
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

// 处理标签编辑界面
func (m *Model) handleWatchlistTagEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// 取消编辑，返回标签管理界面
		m.state = WatchlistTagManage
		m.message = m.getText("tagEditCanceled")
		m.tagEditInput = ""
		m.tagEditInputCursor = 0
		m.tagToEdit = ""
		return m, nil
	case "enter":
		// 确认编辑
		newTagName := strings.TrimSpace(m.tagEditInput)

		// 验证新标签名称
		if newTagName == "" {
			m.message = m.getText("tagNameRequired")
			return m, nil
		}

		// 检查是否与原标签相同
		if newTagName == m.tagToEdit {
			m.message = m.getText("tagNameUnchanged")
			return m, nil
		}

		// 批量更新所有使用该标签的股票
		updatedCount := m.renameTagForAllStocks(m.tagToEdit, newTagName)

		// 保存更新
		m.invalidateWatchlistCache()
		m.saveWatchlist()

		// 更新可用标签列表
		m.availableTags = m.getAvailableTags()

		// 如果当前过滤的用户标签是被修改的标签，更新过滤标签
		if m.selectedUserTagFilter == m.tagToEdit {
			m.selectedUserTagFilter = newTagName
		}

		// 显示成功消息
		m.message = fmt.Sprintf(m.getText("tagEditSuccess"), m.tagToEdit, newTagName, updatedCount)

		// 返回标签管理界面
		m.state = WatchlistTagManage
		m.tagEditInput = ""
		m.tagEditInputCursor = 0
		m.tagToEdit = ""

		return m, nil
	case "left", "ctrl+b":
		// 光标左移
		m.tagEditInputCursor = MoveCursorUp(m.tagEditInputCursor)
		return m, nil
	case "right", "ctrl+f":
		// 光标右移
		runes := []rune(m.tagEditInput)
		m.tagEditInputCursor = MoveCursorDown(m.tagEditInputCursor, len(runes))
		return m, nil
	case "home", "ctrl+a":
		// 光标移到开头
		m.tagEditInputCursor = 0
		return m, nil
	case "end", "ctrl+e":
		// 光标移到末尾
		m.tagEditInputCursor = len([]rune(m.tagEditInput))
		return m, nil
	case "backspace":
		// 删除光标前的字符
		m.tagEditInput, m.tagEditInputCursor = deleteRuneBeforeCursor(m.tagEditInput, m.tagEditInputCursor)
		return m, nil
	case "delete", "ctrl+d":
		// 删除光标处的字符
		m.tagEditInput, m.tagEditInputCursor = deleteRuneAtCursor(m.tagEditInput, m.tagEditInputCursor)
		return m, nil
	default:
		// 处理文本输入
		if len(msg.String()) == 1 || (len(msg.String()) > 1 && msg.Type == tea.KeyRunes) {
			m.tagEditInput, m.tagEditInputCursor = insertStringAtCursor(m.tagEditInput, m.tagEditInputCursor, msg.String())
		}
		return m, nil
	}
}

// 标签管理界面视图
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
			s += fmt.Sprintf("当前标签: %s\n\n", stock.getTagsDisplay(m))
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n", stock.Name, stock.Code)
			s += fmt.Sprintf("Current tags: %s\n\n", stock.getTagsDisplay(m))
		}

		// 显示所有可用标签，标记当前股票拥有的标签
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

				// 检查当前股票是否拥有这个标签
				hasTag := stock.hasTag(tag)
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

		// 操作提示
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

// 标签删除选择界面视图
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

		// 显示可删除的标签
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

// 标签编辑界面视图
func (m *Model) viewWatchlistTagEdit() string {
	var s string

	s += m.getText("editTagTitle") + "\n\n"
	s += fmt.Sprintf(m.getText("editingTag"), m.tagToEdit) + "\n\n"
	s += m.getText("enterNewTagName") + formatTextWithCursor(m.tagEditInput, m.tagEditInputCursor) + "\n\n"

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

// 文本编辑辅助函数 (insertRuneAtCursor, deleteRuneBeforeCursor, handleTextInput 等) 已移动到 ui_utils.go

// isStockInWatchlist 已移动到 watchlist.go

// 检查股票是否在持仓中
func (m *Model) isStockInPortfolio(code string) bool {
	for _, stock := range m.portfolio.Stocks {
		if stock.Code == code {
			return true
		}
	}
	return false
}

// formatStockNameWithPortfolioHighlight 已移动到 format.go
// addToWatchlist, removeFromWatchlist 已移动到 watchlist.go

// ========== 搜索结果带操作按钮处理 ==========

func (m *Model) handleSearchResultWithActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// 停止搜索 worker 并清理数据
		if m.isSearchMode {
			m.stopSearchIntradayWorker()
		}

		m.state = MainMenu
		m.message = ""
		return m, nil
	case "r":
		// 重新搜索时也要清理旧数据
		if m.isSearchMode {
			m.stopSearchIntradayWorker()
		}

		m.state = SearchingStock
		m.searchFromWatchlist = false
		m.message = ""
		return m, nil
	case "1":
		// 添加到自选列表并跳转到自选页面
		if m.searchResult != nil {
			if m.addToWatchlist(m.searchResult.Symbol, m.searchResult.Name) {
				m.message = fmt.Sprintf(m.getText("addWatchSuccess"), m.searchResult.Name, m.searchResult.Symbol)
			} else {
				m.message = fmt.Sprintf(m.getText("alreadyInWatch"), m.searchResult.Symbol)
			}

			// 停止搜索 worker
			if m.isSearchMode {
				m.stopSearchIntradayWorker()
			}

			// 跳转到自选列表页面
			m.state = WatchlistViewing
			m.resetWatchlistCursor() // 重置游标到第一只股票
			m.cursor = 0
			m.lastUpdate = time.Now()

			// 启动自选列表的分时数据采集
			m.startIntradayDataCollection()
		}
		return m, m.tickCmd()
	case "2":
		// 添加到持股列表（进入添加流程）
		if m.searchResult != nil {
			// 停止搜索 worker
			if m.isSearchMode {
				m.stopSearchIntradayWorker()
			}

			m.state = AddingStock
			m.addingStep = 1 // 跳过代码输入，直接到成本价输入
			m.tempCode = m.searchResult.Symbol
			m.stockInfo = &StockData{
				Symbol: m.searchResult.Symbol,
				Name:   m.searchResult.Name,
				Price:  m.searchResult.Price,
			}
			m.input = ""
			m.message = ""
			m.fromSearch = true // 标记从搜索结果添加
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

	// 复用原有的搜索结果显示逻辑
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// 构建表头和数据行
	var headers []interface{}
	var values []interface{}

	// 基本信息
	if m.language == Chinese {
		headers = append(headers, "股票代码", "股票名称", "现价")
	} else {
		headers = append(headers, "Code", "Name", "Price")
	}
	values = append(values, m.searchResult.Symbol, m.searchResult.Name, m.formatPriceWithColorLang(m.searchResult.Price, m.searchResult.PrevClose))

	// 昨收价
	if m.searchResult.PrevClose > 0 {
		if m.language == Chinese {
			headers = append(headers, "昨收价")
		} else {
			headers = append(headers, "Prev Close")
		}
		values = append(values, fmt.Sprintf("%.3f", m.searchResult.PrevClose))
	}

	// 价格信息（有数据时才显示）
	if m.searchResult.StartPrice > 0 {
		if m.language == Chinese {
			headers = append(headers, "开盘价")
		} else {
			headers = append(headers, "Open")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.StartPrice, m.searchResult.PrevClose))
	}
	if m.searchResult.MaxPrice > 0 {
		if m.language == Chinese {
			headers = append(headers, "最高价")
		} else {
			headers = append(headers, "High")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.MaxPrice, m.searchResult.PrevClose))
	}
	if m.searchResult.MinPrice > 0 {
		if m.language == Chinese {
			headers = append(headers, "最低价")
		} else {
			headers = append(headers, "Low")
		}
		values = append(values, m.formatPriceWithColorLang(m.searchResult.MinPrice, m.searchResult.PrevClose))
	}

	// 涨跌信息
	if m.searchResult.Change != 0 {
		if m.language == Chinese {
			headers = append(headers, "涨跌额")
		} else {
			headers = append(headers, "Change")
		}
		changeStr := m.formatProfitWithColorZeroLang(m.searchResult.Change)
		values = append(values, changeStr)
	}
	if m.searchResult.ChangePercent != 0 {
		if m.language == Chinese {
			headers = append(headers, "今日涨幅")
		} else {
			headers = append(headers, "Change %")
		}
		changePercentStr := m.formatProfitRateWithColorZeroLang(m.searchResult.ChangePercent)
		values = append(values, changePercentStr)
	}

	// 换手率
	if m.searchResult.TurnoverRate > 0 {
		if m.language == Chinese {
			headers = append(headers, "换手率")
		} else {
			headers = append(headers, "Turnover")
		}
		values = append(values, fmt.Sprintf("%.2f%%", m.searchResult.TurnoverRate))
	}

	// 买入量（成交量）
	if m.searchResult.Volume > 0 {
		if m.language == Chinese {
			headers = append(headers, "成交量")
		} else {
			headers = append(headers, "Volume")
		}
		volumeStr := formatVolume(m.searchResult.Volume)
		values = append(values, volumeStr)
	}

	// 添加表头和数据行
	t.AppendHeader(table.Row(headers))
	t.AppendRow(table.Row(values))

	s += t.Render() + "\n\n"

	// === 新增：搜索模式分时图表（自动展示） ===
	if m.isSearchMode {
		// 渲染图表区域分隔线
		s += strings.Repeat("─", 80) + "\n"
		if m.language == Chinese {
			s += "📈 实时分时图表 (每5秒自动刷新)\n\n"
		} else {
			s += "📈 Real-time Intraday Chart (Auto-refresh every 5s)\n\n"
		}

		// 渲染图表
		if m.searchIntradayData != nil && len(m.searchIntradayData.Datapoints) > 0 {
			// 创建图表（使用较小的嵌入式尺寸）
			chartWidth := 100 // 嵌入式图表宽度
			chartHeight := 15 // 嵌入式图表高度

			chartModel := m.createSearchIntradayChart(chartWidth, chartHeight)
			if chartModel != nil {
				s += chartModel.View() + "\n"

				// 显示更新信息
				if m.language == Chinese {
					s += fmt.Sprintf("最后更新: %s | 数据点: %d\n",
						m.searchIntradayData.UpdatedAt,
						len(m.searchIntradayData.Datapoints))
				} else {
					s += fmt.Sprintf("Last update: %s | Data points: %d\n",
						m.searchIntradayData.UpdatedAt,
						len(m.searchIntradayData.Datapoints))
				}
			} else {
				// 图表创建失败（终端太小）
				if m.language == Chinese {
					s += "终端尺寸过小，无法显示图表\n"
				} else {
					s += "Terminal size too small to display chart\n"
				}
			}
		} else {
			// 数据尚未加载
			if m.language == Chinese {
				s += "正在获取分时数据...\n"
			} else {
				s += "Loading intraday data...\n"
			}
		}

		s += "\n"
	}

	// 操作按钮提示
	s += m.getText("actionHelp") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ========== 自选股票查看处理 ==========

func (m *Model) handleWatchlistViewing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m":
		m.stopIntradayDataCollection() // 停止分时数据采集
		m.state = MainMenu
		m.message = ""
		return m, nil
	case "d":
		// 直接删除光标指向的自选股票
		filteredStocks := m.getFilteredWatchlist()
		if len(filteredStocks) == 0 {
			m.message = m.getText("emptyWatchlist")
			return m, nil
		}

		// 获取要删除的股票（从过滤列表中）
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			stockToRemove := filteredStocks[m.watchlistCursor]

			// 在原始列表中找到该股票并删除
			for i, stock := range m.watchlist.Stocks {
				if stock.Code == stockToRemove.Code {
					m.removeFromWatchlist(i)
					break
				}
			}

			// 调整光标位置（基于过滤后的列表）
			newFilteredStocks := m.getFilteredWatchlist()
			if m.watchlistCursor >= len(newFilteredStocks) && len(newFilteredStocks) > 0 {
				m.watchlistCursor = len(newFilteredStocks) - 1
			}

			m.message = fmt.Sprintf(m.getText("removeWatchSuccess"), stockToRemove.Name, stockToRemove.Code)
		}
		return m, nil
	case "v":
		// 查看分时图表
		filteredStocks := m.getFilteredWatchlist()
		if len(filteredStocks) == 0 {
			m.message = m.getText("emptyWatchlist")
			return m, nil
		}
		selectedStock := filteredStocks[m.watchlistCursor]
		m.chartViewStock = selectedStock.Code
		m.chartViewStockName = selectedStock.Name

		// 获取智能日期（与 worker 采集逻辑一致）
		actualDate, _, err := GetTradingDayForCollection(selectedStock.Code, m)
		if err != nil {
			// 如果获取失败，降级为简单逻辑
			actualDate = getSmartChartDate()
		}
		m.chartViewDate = actualDate
		m.previousState = WatchlistViewing

		// 尝试加载数据
		data, loadErr := m.loadIntradayDataForDate(
			selectedStock.Code,
			selectedStock.Name,
			actualDate,
		)

		if loadErr != nil {
			// 无数据 - 触发采集
			m.chartData = nil
			m.chartLoadError = nil
			m.state = IntradayChartViewing
			return m, m.triggerIntradayDataCollection(
				selectedStock.Code,
				selectedStock.Name,
				actualDate,
			)
		}

		// 数据存在 - 创建图表
		m.chartData = data
		m.chartLoadError = nil
		m.chartIsCollecting = false
		m.state = IntradayChartViewing
		return m, nil
	case "a":
		// 跳转到股票搜索页面
		logInfo("log.action.watchlistSearch")
		m.state = SearchingStock
		m.searchInput = ""
		m.searchResult = nil
		m.searchFromWatchlist = true
		m.message = ""
		return m, nil
	case "s":
		// 进入排序菜单
		logInfo("log.action.watchlistSort")
		m.state = WatchlistSorting
		// 智能定位光标到当前排序字段
		m.watchlistSortCursor = m.findSortFieldIndex(m.watchlistSortField, false)
		m.message = ""
		return m, nil
	case "t":
		// 给当前选中的股票管理标签 - 进入标签管理界面
		filteredStocks := m.getFilteredWatchlist()
		if len(filteredStocks) == 0 {
			m.message = m.getText("emptyWatchlist")
			return m, nil
		}

		// 获取当前选中股票的标签信息
		currentStock := filteredStocks[m.watchlistCursor]
		m.currentStockTags = make([]string, 0)
		for _, tag := range currentStock.Tags {
			if tag != "" && tag != "-" {
				m.currentStockTags = append(m.currentStockTags, tag)
			}
		}

		// 获取所有可用标签
		m.availableTags = m.getAvailableTags()
		m.state = WatchlistTagManage
		m.tagManageCursor = 0
		m.tagInput = ""
		m.isInRemoveMode = false
		m.message = ""
		return m, nil
	case "l", "L":
		// 查看股票告警详情
		filteredStocks := m.getFilteredWatchlist()
		if len(filteredStocks) == 0 {
			m.message = m.getText("emptyWatchlist")
			return m, nil
		}

		currentStock := filteredStocks[m.watchlistCursor]

		// 设置股票告警详情参数
		m.stockAlertCode = currentStock.Code
		m.stockAlertName = currentStock.Name
		m.stockAlertCursor = 0
		m.previousState = WatchlistViewing // 记录返回状态

		// 获取该股票的所有告警
		m.stockAlertAlerts = m.getStockAlerts(currentStock.Code)

		logInfo("log.action.enterStockAlertManagement", currentStock.Code, currentStock.Name)
		m.state = StockAlertManage
		m.message = ""
		return m, nil
	case "g":
		// 分组查看 (v5.8: 两阶段选择，支持位置记忆)
		m.tagGroups = m.getTagGroups()
		totalTags := m.getTotalTagCount()

		if totalTags == 0 {
			m.message = m.getText("watchlist.noTags")
			return m, nil
		}

		// 根据当前过滤状态决定初始阶段和光标位置
		if m.selectedMarketFilter != "" || m.selectedUserTagFilter != "" {
			// 有过滤条件：恢复上次选择的位置
			if m.selectedUserTagFilter != "" {
				// 已选择了用户标签：进入第二阶段，光标定位到该标签
				m.filterSelectionStep = 1
				userTagPos := m.findUserTagPosition(m.selectedUserTagFilter)
				if userTagPos >= 0 {
					m.cursor = userTagPos
				} else {
					m.cursor = 0 // 未找到则默认第一项
				}
			} else {
				// 只选择了市场：停在第一阶段，光标定位到该市场
				m.filterSelectionStep = 0
				marketPos := m.findMarketTagPosition(m.selectedMarketFilter)
				if marketPos >= 0 {
					m.cursor = marketPos
				} else {
					m.cursor = 0 // 未找到则默认第一项
				}
			}
		} else {
			// 无过滤条件：从头开始
			m.filterSelectionStep = 0
			m.cursor = 0
		}

		m.state = WatchlistGroupSelect
		m.message = ""
		return m, nil
	case "c":
		// 清除所有过滤（市场过滤和用户标签过滤）
		if m.selectedMarketFilter != "" || m.selectedUserTagFilter != "" {
			m.selectedMarketFilter = ""
			m.selectedUserTagFilter = ""
			m.invalidateWatchlistCache() // 使缓存失效
			m.resetWatchlistCursor()     // 重置游标到第一只股票
			if m.language == Chinese {
				m.message = "已清除所有过滤条件"
			} else {
				m.message = "All filters cleared"
			}
		}
		return m, nil
	case "up", "k", "w":
		// 获取一次过滤后的列表,避免重复调用
		filteredStocks := m.getFilteredWatchlist()
		m.watchlistCursor = MoveCursorUp(m.watchlistCursor)
		// 只在光标移动时调整滚动
		m.adjustWatchlistScroll(filteredStocks)
		return m, nil
	case "down", "j":
		// 获取一次过滤后的列表,避免重复调用
		filteredStocks := m.getFilteredWatchlist()
		m.watchlistCursor = MoveCursorDown(m.watchlistCursor, len(filteredStocks)-1)
		// 只在光标移动时调整滚动
		m.adjustWatchlistScroll(filteredStocks)
		return m, nil
	}
	return m, nil
}

func (m *Model) viewWatchlistViewing() string {
	s := m.getText("watchlistTitle") + "\n"
	s += fmt.Sprintf(m.getText("updateTime"), m.lastUpdate.Format("2006-01-02 15:04:05")) + "\n"

	// 显示当前组合过滤状态
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

	// 获取过滤后的股票列表
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

	// 显示滚动信息
	totalWatchStocks := len(filteredStocks)
	maxWatchlistLines := m.config.Display.MaxLines
	if totalWatchStocks > 0 {
		currentPos := m.watchlistCursor + 1 // 显示从1开始的位置
		if m.language == Chinese {
			s += fmt.Sprintf("⭐ 自选列表 (%d/%d) [↑/↓:翻页]\n", currentPos, totalWatchStocks)
		} else {
			s += fmt.Sprintf("⭐ Watchlist (%d/%d) [↑/↓:scroll]\n", currentPos, totalWatchStocks)
		}
		s += "\n"
	}

	// 创建表格显示自选股票列表
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// 获取带排序指示器的表头
	t.AppendHeader(m.GenerateWatchlistHeader())

	// 计算要显示的股票范围
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
		// 从缓存获取股价数据（非阻塞）
		stockData := m.getStockPriceFromCache(watchStock.Code)

		// 使用动态列渲染器生成行
		row := m.GenerateWatchlistRow(&watchStock, stockData, i, startIndex, endIndex)
		t.AppendRow(row)

		// 在每个股票后添加分隔线（除了显示范围内的最后一个）
		if i < endIndex-1 {
			t.AppendSeparator()
		}
	}

	s += t.Render() + "\n"

	// 如果可以滚动，显示滚动指示
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

	// 使用统一的帮助文本
	s += "\n" + m.getText("watchlistHelp") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// gbkToUtf8 已移动到 ui_utils.go

// ========== 自选股票搜索确认处理 ==========

func (m *Model) handleWatchlistSearchConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// 停止搜索 worker 并清理数据
		if m.isSearchMode {
			m.stopSearchIntradayWorker()
		}

		m.state = WatchlistViewing
		m.resetWatchlistCursor() // 重置游标到第一只股票
		m.searchFromWatchlist = false
		m.message = ""

		// 启动自选列表的分时数据采集
		m.startIntradayDataCollection()

		return m, m.tickCmd() // 重启定时器
	case "enter":
		// 确认添加到自选列表
		if m.searchResult != nil {
			if m.addToWatchlist(m.searchResult.Symbol, m.searchResult.Name) {
				m.message = fmt.Sprintf(m.getText("addWatchSuccess"), m.searchResult.Name, m.searchResult.Symbol)
				logInfo("添加到自选列表: %s (%s)", m.searchResult.Name, m.searchResult.Symbol)
			} else {
				m.message = fmt.Sprintf(m.getText("alreadyInWatch"), m.searchResult.Symbol)
			}

			// 停止搜索 worker
			if m.isSearchMode {
				m.stopSearchIntradayWorker()
			}

			m.state = WatchlistViewing
			m.resetWatchlistCursor() // 重置游标到第一只股票
			m.searchFromWatchlist = false

			// 启动自选列表的分时数据采集
			m.startIntradayDataCollection()

			return m, m.tickCmd()
		}
		return m, nil
	case "r":
		// 重新搜索时也要清理旧数据
		if m.isSearchMode {
			m.stopSearchIntradayWorker()
		}

		m.state = SearchingStock
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

	// 创建表格显示股票信息
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// 设置表头
	if m.language == Chinese {
		t.AppendHeader(table.Row{"名称", "现价", "昨收价", "开盘", "最高", "最低", "今日涨幅", "换手率", "成交量"})
	} else {
		t.AppendHeader(table.Row{"Name", "Price", "PrevClose", "Open", "High", "Low", "Today%", "Turnover", "Volume"})
	}

	// 构建数据行
	var values []interface{}

	// 名称
	values = append(values, m.searchResult.Name)

	// 现价 (带颜色)
	priceStr := m.formatPriceWithColorLang(m.searchResult.Price, m.searchResult.PrevClose)
	values = append(values, priceStr)

	// 昨收价
	values = append(values, fmt.Sprintf("%.3f", m.searchResult.PrevClose))

	// 开盘价
	if m.searchResult.StartPrice > 0 {
		openStr := m.formatPriceWithColorLang(m.searchResult.StartPrice, m.searchResult.PrevClose)
		values = append(values, openStr)
	} else {
		values = append(values, "-")
	}

	// 最高价
	if m.searchResult.MaxPrice > 0 {
		highStr := m.formatPriceWithColorLang(m.searchResult.MaxPrice, m.searchResult.PrevClose)
		values = append(values, highStr)
	} else {
		values = append(values, "-")
	}

	// 最低价
	if m.searchResult.MinPrice > 0 {
		lowStr := m.formatPriceWithColorLang(m.searchResult.MinPrice, m.searchResult.PrevClose)
		values = append(values, lowStr)
	} else {
		values = append(values, "-")
	}

	// 今日涨幅
	if m.searchResult.ChangePercent != 0 {
		changePercentStr := m.formatProfitRateWithColorZeroLang(m.searchResult.ChangePercent)
		values = append(values, changePercentStr)
	} else {
		values = append(values, "-")
	}

	// 换手率
	if m.searchResult.TurnoverRate > 0 {
		values = append(values, fmt.Sprintf("%.2f%%", m.searchResult.TurnoverRate))
	} else {
		values = append(values, "-")
	}

	// 成交量
	if m.searchResult.Volume > 0 {
		if m.searchResult.Volume >= 100000000 { // 大于等于1亿
			values = append(values, fmt.Sprintf("%.2f亿", float64(m.searchResult.Volume)/100000000))
		} else if m.searchResult.Volume >= 10000 { // 大于等于1万
			values = append(values, fmt.Sprintf("%.2f万", float64(m.searchResult.Volume)/10000))
		} else {
			values = append(values, fmt.Sprintf("%d", m.searchResult.Volume))
		}
	} else {
		values = append(values, "-")
	}

	t.AppendRow(values)

	s += t.Render() + "\n\n"

	// === 新增：搜索模式分时图表（自动展示） ===
	if m.isSearchMode {
		// 渲染图表区域分隔线
		s += strings.Repeat("─", 80) + "\n"
		if m.language == Chinese {
			s += "📈 实时分时图表 (每5秒自动刷新)\n\n"
		} else {
			s += "📈 Real-time Intraday Chart (Auto-refresh every 5s)\n\n"
		}

		// 渲染图表
		if m.searchIntradayData != nil && len(m.searchIntradayData.Datapoints) > 0 {
			// 创建图表（使用较小的嵌入式尺寸）
			chartWidth := 100 // 嵌入式图表宽度
			chartHeight := 15 // 嵌入式图表高度

			chartModel := m.createSearchIntradayChart(chartWidth, chartHeight)
			if chartModel != nil {
				s += chartModel.View() + "\n"

				// 显示更新信息
				if m.language == Chinese {
					s += fmt.Sprintf("最后更新: %s | 数据点: %d\n",
						m.searchIntradayData.UpdatedAt,
						len(m.searchIntradayData.Datapoints))
				} else {
					s += fmt.Sprintf("Last update: %s | Data points: %d\n",
						m.searchIntradayData.UpdatedAt,
						len(m.searchIntradayData.Datapoints))
				}
			} else {
				// 图表创建失败（终端太小）
				if m.language == Chinese {
					s += "终端尺寸过小，无法显示图表\n"
				} else {
					s += "Terminal size too small to display chart\n"
				}
			}
		} else {
			// 数据尚未加载
			if m.language == Chinese {
				s += "正在获取分时数据...\n"
			} else {
				s += "Loading intraday data...\n"
			}
		}

		s += "\n"
	}

	if m.language == Chinese {
		s += "按回车键添加到自选列表，ESC键返回，R键重新搜索\n"
	} else {
		s += "Press Enter to add to watchlist, ESC to return, R to search again\n"
	}

	return s
}

// 获取排序字段的显示名称
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

// 获取排序方向的显示名称
func (m *Model) getSortDirectionName(direction SortDirection) string {
	if direction == SortAsc {
		return m.getText("sortAsc")
	}
	return m.getText("sortDesc")
}

// 获取持股列表可用的排序字段
func (m *Model) getPortfolioSortFields() []SortField {
	return []SortField{
		SortByCode, SortByName, SortByPrice, SortByCostPrice,
		SortByChange, SortByChangePercent, SortByQuantity,
		SortByTotalProfit, SortByProfitRate, SortByMarketValue,
	}
}

// 获取自选列表可用的排序字段
func (m *Model) getWatchlistSortFields() []SortField {
	return []SortField{
		SortByCode, SortByName, SortByPrice, SortByTag,
		SortByChangePercent, SortByTurnoverRate, SortByVolume,
	}
}

// 查找排序字段在字段列表中的索引，如果找不到返回0
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

	// 如果没找到当前排序字段，返回0（第一个字段）
	return 0
}

// 处理持股列表排序
func (m *Model) handlePortfolioSorting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sortFields := m.getPortfolioSortFields()

	switch msg.String() {
	case "up", "k", "w":
		m.portfolioSortCursor = MoveCursorUp(m.portfolioSortCursor)
	case "down", "j", "s":
		m.portfolioSortCursor = MoveCursorDown(m.portfolioSortCursor, len(sortFields)-1)
	case "enter", " ":
		// 切换排序方向或应用排序
		selectedField := sortFields[m.portfolioSortCursor]
		if m.portfolioSortField == selectedField {
			// 切换排序方向
			if m.portfolioSortDirection == SortAsc {
				m.portfolioSortDirection = SortDesc
			} else {
				m.portfolioSortDirection = SortAsc
			}
		} else {
			// 设置新的排序字段，默认升序
			m.portfolioSortField = selectedField
			m.portfolioSortDirection = SortAsc
		}
		// 执行排序并标记为已排序状态
		m.optimizedSortPortfolio(m.portfolioSortField, m.portfolioSortDirection)
		m.portfolioIsSorted = true
		m.resetPortfolioCursor()
		// 返回持股列表页面
		m.state = Monitoring
		m.message = ""
		return m, nil
	case "c", "C":
		// 清除当前排序 - 重新加载原始数据顺序
		m.portfolioIsSorted = false
		// 清除排序字段和方向状态
		m.portfolioSortField = SortByCode  // 重置为默认值
		m.portfolioSortDirection = SortAsc // 重置为默认值
		// 重新加载原始数据顺序
		m.portfolio = loadPortfolio()
		m.resetPortfolioCursor()
		// 返回持股列表页面
		m.state = Monitoring
		m.message = m.getText("sortCleared")
		return m, nil
	case "esc", "q":
		// 返回持股列表页面
		m.state = Monitoring
		m.message = ""
		return m, nil
	}
	return m, nil
}

// 处理自选列表排序
func (m *Model) handleWatchlistSorting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sortFields := m.getWatchlistSortFields()

	switch msg.String() {
	case "up", "k", "w":
		m.watchlistSortCursor = MoveCursorUp(m.watchlistSortCursor)
	case "down", "j", "s":
		m.watchlistSortCursor = MoveCursorDown(m.watchlistSortCursor, len(sortFields)-1)
	case "enter", " ":
		// 切换排序方向或应用排序
		selectedField := sortFields[m.watchlistSortCursor]
		if m.watchlistSortField == selectedField {
			// 切换排序方向
			if m.watchlistSortDirection == SortAsc {
				m.watchlistSortDirection = SortDesc
			} else {
				m.watchlistSortDirection = SortAsc
			}
		} else {
			// 设置新的排序字段，默认升序
			m.watchlistSortField = selectedField
			m.watchlistSortDirection = SortAsc
		}
		// 执行排序并标记为已排序状态
		m.optimizedSortWatchlist(m.watchlistSortField, m.watchlistSortDirection)
		m.watchlistIsSorted = true
		m.resetWatchlistCursor()
		// 返回自选列表页面
		m.state = WatchlistViewing
		m.message = ""
		return m, m.tickCmd() // 重启定时器
	case "c", "C":
		// 清除当前排序 - 重新加载原始数据顺序
		m.watchlistIsSorted = false
		// 清除排序字段和方向状态
		m.watchlistSortField = SortByCode  // 重置为默认值
		m.watchlistSortDirection = SortAsc // 重置为默认值
		// 重新加载原始数据顺序
		m.watchlist = loadWatchlist()
		m.resetWatchlistCursor()
		// 返回自选列表页面
		m.state = WatchlistViewing
		m.message = m.getText("sortCleared")
		return m, m.tickCmd() // 重启定时器
	case "esc", "q":
		// 返回自选列表页面
		m.state = WatchlistViewing
		m.message = ""
		return m, m.tickCmd() // 重启定时器
	}
	return m, nil
}

// 排序菜单视图 - 持股列表
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
			// 显示当前排序状态（只有在已排序时才显示）
			directionName := m.getSortDirectionName(m.portfolioSortDirection)
			s += fmt.Sprintf("%s%s (%s)\n", prefix, fieldName, directionName)
		} else {
			s += fmt.Sprintf("%s%s\n", prefix, fieldName)
		}
	}

	s += "\n" + m.getText("sortHelp") + "\n"
	return s
}

// 排序菜单视图 - 自选列表
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
			// 显示当前排序状态（只有在已排序时才显示）
			directionName := m.getSortDirectionName(m.watchlistSortDirection)
			s += fmt.Sprintf("%s%s (%s)\n", prefix, fieldName, directionName)
		} else {
			s += fmt.Sprintf("%s%s\n", prefix, fieldName)
		}
	}

	s += "\n" + m.getText("sortHelp") + "\n"
	return s
}

// 分时数据采集和图表功能已移动到 intraday_chart.go
// 包含: startIntradayDataCollection, stopIntradayDataCollection, loadIntradayDataForDate,
// parseIntradayTime, calculateAdaptiveMargin, getSmartChartDate, findPreviousTradingDayFromDate,
// createFixedTimeRange, createIntradayChart, triggerIntradayDataCollection, formatDate,
// isWeekend, findPreviousTradingDay, findNextTradingDay, handleIntradayChartViewing, viewIntradayChart

// ============================================================================
// Column Management Methods (delegating to internal/ui)
// ============================================================================

// GetPortfolioColumns - 获取Portfolio的活跃列列表
func (m *Model) GetPortfolioColumns() []*ui.ColumnMetadata {
	return ui.GetPortfolioColumns(m.config.Display.PortfolioColumns, m.getText)
}

// GetWatchlistColumns - 获取Watchlist的活跃列列表
func (m *Model) GetWatchlistColumns() []*ui.ColumnMetadata {
	return ui.GetWatchlistColumns(m.config.Display.WatchlistColumns, m.getText)
}

// GeneratePortfolioHeader - 生成Portfolio表头（含排序指示器）
func (m *Model) GeneratePortfolioHeader() table.Row {
	columns := m.GetPortfolioColumns()
	return ui.GenerateHeader(
		columns,
		m.getText,
		m.portfolioIsSorted,
		m.portfolioSortField,
		m.portfolioSortDirection,
	)
}

// GenerateWatchlistHeader - 生成Watchlist表头（含排序指示器）
func (m *Model) GenerateWatchlistHeader() table.Row {
	columns := m.GetWatchlistColumns()
	return ui.GenerateHeader(
		columns,
		m.getText,
		m.watchlistIsSorted,
		m.watchlistSortField,
		m.watchlistSortDirection,
	)
}

// GeneratePortfolioRow - 生成Portfolio数据行
func (m *Model) GeneratePortfolioRow(stock *Stock, rowIndex, startIndex, endIndex int) table.Row {
	columns := m.GetPortfolioColumns()
	row := make(table.Row, len(columns))

	for i, col := range columns {
		switch col.ID {
		case ui.ColCursor:
			// 光标列特殊处理
			if m.portfolioCursor >= startIndex && m.portfolioCursor < endIndex && rowIndex == m.portfolioCursor {
				row[i] = "►"
			} else {
				row[i] = ""
			}
		case ui.ColCode:
			row[i] = stock.Code
		case ui.ColName:
			row[i] = stock.Name
		case ui.ColPrevClose:
			row[i] = fmt.Sprintf("%.3f", stock.PrevClose)
		case ui.ColOpen:
			row[i] = m.formatPriceWithColorLang(stock.StartPrice, stock.PrevClose)
		case ui.ColHigh:
			row[i] = m.formatPriceWithColorLang(stock.MaxPrice, stock.PrevClose)
		case ui.ColLow:
			row[i] = m.formatPriceWithColorLang(stock.MinPrice, stock.PrevClose)
		case ui.ColPrice:
			row[i] = m.formatPriceWithColorLang(stock.Price, stock.PrevClose)
		case ui.ColCost:
			row[i] = fmt.Sprintf("%.3f", stock.CostPrice)
		case ui.ColQuantity:
			row[i] = stock.Quantity
		case ui.ColTodayChange:
			if stock.Price > 0 && stock.PrevClose > 0 {
				row[i] = m.formatProfitRateWithColorZeroLang(stock.ChangePercent)
			} else {
				row[i] = "-"
			}
		case ui.ColPositionProfit:
			if stock.Price > 0 {
				profit := stock.CalculatePositionProfit()
				row[i] = m.formatProfitWithColorZeroLang(profit)
			} else {
				row[i] = "-"
			}
		case ui.ColProfitRate:
			if stock.Price > 0 && stock.CostPrice > 0 {
				profitRate := ((stock.Price - stock.CostPrice) / stock.CostPrice) * 100
				row[i] = m.formatProfitRateWithColorZeroLang(profitRate)
			} else {
				row[i] = "-"
			}
		case ui.ColMarketValue:
			marketValue := float64(stock.Quantity) * stock.Price
			row[i] = fmt.Sprintf("%.2f", marketValue)
		default:
			row[i] = "-"
		}
	}

	return row
}

// GeneratePortfolioTotalRow - 生成Portfolio总计行
func (m *Model) GeneratePortfolioTotalRow(totalProfit, totalProfitRate, totalMarketValue float64) table.Row {
	columns := m.GetPortfolioColumns()
	row := make(table.Row, len(columns))

	for i, col := range columns {
		switch col.ID {
		case ui.ColName:
			row[i] = m.getText("total")
		case ui.ColPositionProfit:
			row[i] = m.formatProfitWithColorLang(totalProfit)
		case ui.ColProfitRate:
			row[i] = m.formatProfitRateWithColorLang(totalProfitRate)
		case ui.ColMarketValue:
			row[i] = fmt.Sprintf("%.2f", totalMarketValue)
		default:
			row[i] = ""
		}
	}

	return row
}

// GenerateWatchlistRow - 生成Watchlist数据行
func (m *Model) GenerateWatchlistRow(watchStock *WatchlistStock, stockData *StockData, rowIndex, startIndex, endIndex int) table.Row {
	columns := m.GetWatchlistColumns()
	row := make(table.Row, len(columns))

	for i, col := range columns {
		switch col.ID {
		case ui.ColCursor:
			// 光标列特殊处理
			if m.watchlistCursor >= startIndex && m.watchlistCursor < endIndex && rowIndex == m.watchlistCursor {
				row[i] = "►"
			} else {
				row[i] = ""
			}
		case ui.ColTag:
			row[i] = watchStock.getTagsDisplay(m)
		case ui.ColCode:
			row[i] = watchStock.Code
		case ui.ColName:
			// Watchlist的name列需要portfolio高亮
			row[i] = m.formatStockNameWithPortfolioHighlight(watchStock.Name, watchStock.Code)
		case ui.ColPrice:
			if stockData != nil && stockData.Price > 0 && stockData.PrevClose > 0 {
				row[i] = m.formatPriceWithColorLang(stockData.Price, stockData.PrevClose)
			} else {
				row[i] = "-"
			}
		case ui.ColPrevClose:
			if stockData != nil {
				row[i] = fmt.Sprintf("%.3f", stockData.PrevClose)
			} else {
				row[i] = "-"
			}
		case ui.ColOpen:
			if stockData != nil && stockData.Price > 0 && stockData.PrevClose > 0 {
				row[i] = m.formatPriceWithColorLang(stockData.StartPrice, stockData.PrevClose)
			} else {
				row[i] = "-"
			}
		case ui.ColHigh:
			if stockData != nil && stockData.Price > 0 && stockData.PrevClose > 0 {
				row[i] = m.formatPriceWithColorLang(stockData.MaxPrice, stockData.PrevClose)
			} else {
				row[i] = "-"
			}
		case ui.ColLow:
			if stockData != nil && stockData.Price > 0 && stockData.PrevClose > 0 {
				row[i] = m.formatPriceWithColorLang(stockData.MinPrice, stockData.PrevClose)
			} else {
				row[i] = "-"
			}
		case ui.ColTodayChange:
			if stockData != nil && stockData.Price > 0 && stockData.PrevClose > 0 {
				row[i] = m.formatProfitRateWithColorZeroLang(stockData.ChangePercent)
			} else {
				row[i] = "-"
			}
		case ui.ColTurnover:
			if stockData != nil {
				row[i] = fmt.Sprintf("%.2f%%", stockData.TurnoverRate)
			} else {
				row[i] = "-"
			}
		case ui.ColVolume:
			if stockData != nil {
				row[i] = formatVolume(stockData.Volume)
			} else {
				row[i] = "-"
			}
		default:
			row[i] = "-"
		}
	}

	return row
}
