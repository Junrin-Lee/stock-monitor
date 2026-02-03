package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"stock-monitor/internal/app"
	"stock-monitor/internal/consts"
	"stock-monitor/internal/i18n"
	"stock-monitor/internal/types"
	"stock-monitor/internal/ui"
	"stock-monitor/internal/ui/watchlist"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jedib0t/go-pretty/v6/table"
)

//go:embed i18n/*.json
var embeddedI18nFS embed.FS

// Version information, injected at build time via ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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

// getMenuItems 获取主菜单项列表
func getMenuItems(lang Language) []MenuItem {
	return []MenuItem{
		{Key: "stockList", Label: getTextForLang("stockList", lang)},
		{Key: "watchlist", Label: getTextForLang("watchlist", lang)},
		{Key: "stockSearch", Label: getTextForLang("stockSearch", lang)},
		{Key: "alertManagement", Label: getTextForLang("alertManagement", lang)},
		{Key: "sector.title", Label: getTextForLang("sector.title", lang)},
		{Key: "language", Label: getTextForLang("language", lang)},
		{Key: "exit", Label: getTextForLang("exit", lang)},
	}
}

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("stock-monitor %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// 初始化应用程序（创建目录和UI组件）
	if err := app.InitializeApp(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志系统
	logDir := filepath.Join(".", "data", "logs")
	logLevel := LogInfo // 默认 INFO 级别
	if err := InitLogger(logDir, logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer globalLogger.Sync()

	// 启动节假日数据同步 worker（异步执行，不阻塞主流程）
	StartHolidayWorker()

	// 设置嵌入的 i18n 文件系统
	i18n.SetEmbeddedFS(embeddedI18nFS)

	// 加载 i18n 文件
	loadI18nFiles()

	// 加载配置文件
	config := loadConfig()
	portfolio := loadPortfolio()
	watchlist := loadWatchlist()

	// 根据配置和是否有股票数据决定初始状态
	initialState := consts.MainMenu
	var lastUpdate time.Time
	if config.System.AutoStart {
		// 根据startup_module配置决定启动哪个模块
		switch config.System.StartupModule {
		case "portfolio":
			// 启动持股模块，需要有持股数据
			if len(portfolio.Stocks) > 0 {
				initialState = consts.Monitoring
				lastUpdate = time.Now()
			}
		case "watchlist":
			// 启动自选模块，需要有自选数据
			if len(watchlist.Stocks) > 0 {
				initialState = consts.WatchlistViewing
				lastUpdate = time.Now()
			}
		default:
			// 默认行为：如果有持股数据则进入持股模块
			if len(portfolio.Stocks) > 0 {
				initialState = consts.Monitoring
				lastUpdate = time.Now()
			}
		}
	}

	// 根据配置文件设置语言
	language := consts.English // 默认英文
	if config.System.Language == "zh" {
		language = consts.Chinese
	}

	m := Model{
		state:              initialState,
		currentMenuItem:    0,
		portfolio:          portfolio,
		watchlist:          watchlist,
		config:             config,
		language:           language,
		maxLines:           config.Display.MaxLines, // 从配置读取每页显示行数
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
		// 板块缓存初始化
		sectorCache:    make(map[types.SectorType]*types.SectorCacheEntry),
		sectorType:     types.SectorTypeIndustry, // 默认行业板块
		sectorIsSorted: false,
	}

	// 根据语言设置菜单项
	m.menuItems = getMenuItems(m.language)

	// 设置全局模型引用用于调试日志
	globalModel = &m

	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func (m *Model) Init() tea.Cmd {
	if m.state == consts.Monitoring || m.state == consts.WatchlistViewing {
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
		if m.state == consts.Monitoring || m.state == consts.WatchlistViewing {
			keyStr := msg.String()
			switch keyStr {
			case "up":
				if m.state == consts.Monitoring {
					m.scrollPortfolioUp()
				} else {
					m.scrollWatchlistUp()
				}
				return m, nil
			case "down":
				if m.state == consts.Monitoring {
					m.scrollPortfolioDown()
				} else {
					m.scrollWatchlistDown()
				}
				return m, nil
			}
		}

		// 处理各状态的正常按键
		switch m.state {
		case consts.MainMenu:
			newModel, cmd = m.handleMainMenu(msg)
		case consts.AddingStock:
			newModel, cmd = m.handleAddingStock(msg)
		case consts.Monitoring:
			newModel, cmd = m.handleMonitoring(msg)
		case consts.EditingStock:
			newModel, cmd = m.handleEditingStock(msg)
		case consts.SearchingStock:
			newModel, cmd = m.handleSearchingStock(msg)
		case consts.SearchResult:
			newModel, cmd = m.handleSearchResult(msg)
		case consts.SearchResultWithActions:
			newModel, cmd = m.handleSearchResultWithActions(msg)
		case consts.WatchlistSearchConfirm:
			newModel, cmd = m.handleWatchlistSearchConfirm(msg)
		case consts.LanguageSelection:
			newModel, cmd = m.handleLanguageSelection(msg)
		case consts.WatchlistViewing:
			newModel, cmd = m.handleWatchlistViewing(msg)
		case consts.WatchlistTagging:
			newModel, cmd = m.handleWatchlistTagging(msg)
		case consts.WatchlistTagSelect:
			newModel, cmd = m.handleWatchlistTagSelect(msg)
		case consts.WatchlistTagManage:
			newModel, cmd = m.handleWatchlistTagManage(msg)
		case consts.WatchlistTagRemoveSelect:
			newModel, cmd = m.handleWatchlistTagRemoveSelect(msg)
		case consts.WatchlistTagEdit:
			newModel, cmd = m.handleWatchlistTagEdit(msg)
		case consts.WatchlistGroupSelect:
			newModel, cmd = m.handleWatchlistGroupSelect(msg)
		case consts.PortfolioSorting:
			newModel, cmd = m.handlePortfolioSorting(msg)
		case consts.WatchlistSorting:
			newModel, cmd = m.handleWatchlistSorting(msg)
		case consts.IntradayChartViewing:
			newModel, cmd = m.handleIntradayChartViewing(msg)
		case consts.AlertManage:
			newModel, cmd = m.handleAlertManage(msg)
		case consts.StockAlertManage:
			newModel, cmd = m.handleStockAlertManage(msg)
		case consts.AlertAdd:
			newModel, cmd = m.handleAlertAdd(msg)
		case consts.AlertEdit:
			newModel, cmd = m.handleAlertEdit(msg)
		case consts.AlertBatchMethodSelect:
			newModel, cmd = m.handleAlertBatchMethodSelect(msg)
		case consts.AlertBatchByTag:
			newModel, cmd = m.handleAlertBatchByTag(msg)
		case consts.AlertBatchByMarket:
			newModel, cmd = m.handleAlertBatchByMarket(msg)
		case consts.SelectBatchStocks:
			newModel, cmd = m.handleSelectBatchStocks(msg)
		case consts.SelectBatchFromWatchlist:
			newModel, cmd = m.handleSelectBatchFromWatchlist(msg)
		case consts.SelectBatchFromPortfolio:
			newModel, cmd = m.handleSelectBatchFromPortfolio(msg)
		case consts.InputBatchCodes:
			newModel, cmd = m.handleInputBatchCodes(msg)
		case consts.AlertBatchAdd:
			newModel, cmd = m.handleAlertBatchAdd(msg)
		case consts.AlertTypeSelect:
			newModel, cmd = m.handleAlertTypeSelect(msg)
		case consts.AlertThresholdInput:
			newModel, cmd = m.handleAlertThresholdInput(msg)
		case consts.AlertConditionSelect:
			newModel, cmd = m.handleAlertConditionSelect(msg)
		case consts.AlertFrequencySelect:
			newModel, cmd = m.handleAlertFrequencySelectStep(msg)
		case consts.AlertFrequencyDaysInput:
			newModel, cmd = m.handleAlertFrequencyDaysInput(msg)
		case consts.SectorViewing:
			newModel, cmd = m.handleSectorViewing(msg)
		case consts.SectorStockList:
			newModel, cmd = m.handleSectorStockList(msg)
		case consts.SectorSorting:
			newModel, cmd = m.handleSectorSorting(msg)
		default:
			newModel, cmd = m, nil
		}
	case tickMsg:
		if m.state == consts.Monitoring || m.state == consts.WatchlistViewing {
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
		} else if m.state == consts.SectorViewing || m.state == consts.SectorStockList {
			// 板块页面5秒刷新
			var cmds []tea.Cmd
			cmds = append(cmds, m.tickCmd())

			// 刷新板块数据
			if m.state == consts.SectorViewing {
				cmds = append(cmds, m.fetchSectorListCmd(m.sectorType))
			} else if m.state == consts.SectorStockList && m.currentSectorCode != "" {
				cmds = append(cmds, m.fetchSectorStocksCmd(m.currentSectorCode))
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
			if m.state == consts.WatchlistViewing && m.watchlistIsSorted {
				m.optimizedSortWatchlist(m.watchlistSortField, m.watchlistSortDirection)
			}

			// 如果当前在持股列表且已启用排序，先更新价格数据再重新排序
			if m.state == consts.Monitoring && m.portfolioIsSorted {
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
		if m.state == consts.IntradayChartViewing && m.chartIsCollecting {
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
	case SectorListMsg:
		// 板块列表数据返回
		if msg.Err == nil && msg.Sectors != nil {
			m.sectorList = msg.Sectors
			// 默认按名称升序排序
			if !m.sectorIsSorted {
				m.sectorSortField = consts.SortBySectorName
				m.sectorSortDirection = consts.SortAsc
				m.sortSectorList()
			}
		} else if msg.Err != nil {
			logError("获取板块列表失败: %v", msg.Err)
			m.message = fmt.Sprintf("获取板块列表失败: %v", msg.Err)
		}
		newModel, cmd = m, nil
	case SectorStocksMsg:
		// 成分股列表数据返回
		if msg.Err == nil && msg.Stocks != nil {
			m.sectorStocks = msg.Stocks
			// 默认按名称升序排序
			if !m.sectorIsSorted {
				m.sectorSortField = consts.SortByName
				m.sectorSortDirection = consts.SortAsc
				m.sortSectorStocks()
			}
		} else if msg.Err != nil {
			logError("获取成分股列表失败: %v", msg.Err)
			m.message = fmt.Sprintf("获取成分股列表失败: %v", msg.Err)
		}
		newModel, cmd = m, nil
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
	case consts.MainMenu:
		mainContent = m.viewMainMenu()
	case consts.AddingStock:
		mainContent = m.viewAddingStock()
	case consts.Monitoring:
		mainContent = m.viewMonitoring()
	case consts.EditingStock:
		mainContent = m.viewEditingStock()
	case consts.SearchingStock:
		mainContent = m.viewSearchingStock()
	case consts.SearchResult:
		mainContent = m.viewSearchResult()
	case consts.SearchResultWithActions:
		mainContent = m.viewSearchResultWithActions()
	case consts.WatchlistSearchConfirm:
		mainContent = m.viewWatchlistSearchConfirm()
	case consts.LanguageSelection:
		mainContent = m.viewLanguageSelection()
	case consts.WatchlistViewing:
		mainContent = m.viewWatchlistViewing()
	case consts.WatchlistTagging:
		mainContent = m.viewWatchlistTagging()
	case consts.WatchlistTagSelect:
		mainContent = m.viewWatchlistTagSelect()
	case consts.WatchlistTagManage:
		mainContent = m.viewWatchlistTagManage()
	case consts.WatchlistTagRemoveSelect:
		mainContent = m.viewWatchlistTagRemoveSelect()
	case consts.WatchlistTagEdit:
		mainContent = m.viewWatchlistTagEdit()
	case consts.WatchlistGroupSelect:
		mainContent = m.viewWatchlistGroupSelect()
	case consts.PortfolioSorting:
		mainContent = m.viewPortfolioSorting()
	case consts.WatchlistSorting:
		mainContent = m.viewWatchlistSorting()
	case consts.IntradayChartViewing:
		// 获取终端尺寸 - 使用合理的默认值
		termWidth := 120
		termHeight := 30
		mainContent = m.viewIntradayChart(termWidth, termHeight)
	case consts.AlertManage:
		mainContent = m.viewAlertManage()
	case consts.StockAlertManage:
		mainContent = m.viewStockAlertManage()
	case consts.AlertAdd:
		mainContent = m.viewAlertAdd()
	case consts.AlertEdit:
		mainContent = m.viewAlertEdit()
	case consts.AlertBatchMethodSelect:
		mainContent = m.viewAlertBatchMethodSelect()
	case consts.AlertBatchByTag:
		mainContent = m.viewAlertBatchByTag()
	case consts.AlertBatchByMarket:
		mainContent = m.viewAlertBatchByMarket()
	case consts.SelectBatchStocks:
		mainContent = m.viewSelectBatchStocks()
	case consts.SelectBatchFromWatchlist:
		mainContent = m.viewSelectBatchFromWatchlist()
	case consts.SelectBatchFromPortfolio:
		mainContent = m.viewSelectBatchFromPortfolio()
	case consts.InputBatchCodes:
		mainContent = m.viewInputBatchCodes()
	case consts.AlertBatchAdd:
		mainContent = m.viewAlertBatchAdd()
	case consts.AlertTypeSelect:
		mainContent = m.viewAlertAdd() // consts.AlertTypeSelect复用viewAlertAdd
	case consts.AlertThresholdInput:
		mainContent = m.viewAlertAdd() // consts.AlertThresholdInput复用viewAlertAdd
	case consts.AlertConditionSelect:
		mainContent = m.viewAlertAdd() // consts.AlertConditionSelect复用viewAlertAdd
	case consts.AlertFrequencySelect:
		mainContent = m.viewAlertAdd() // consts.AlertFrequencySelect复用viewAlertAdd
	case consts.AlertFrequencyDaysInput:
		mainContent = m.viewAlertAdd() // consts.AlertFrequencyDaysInput复用viewAlertAdd
	case consts.SectorViewing:
		mainContent = m.viewSectorViewing()
	case consts.SectorStockList:
		mainContent = m.viewSectorStockList()
	case consts.SectorSorting:
		mainContent = m.viewSectorSorting()
	default:
		mainContent = ""
	}

	return mainContent
}

// Menu handlers moved to handlers_menu.go

// handleMonitoringNavigation 处理投资组合监控页面的导航操作

// handleMonitoringActions 处理投资组合监控页面的数据操作

// handleMonitoringViews 处理投资组合监控页面的视图切换

// savePortfolio, getDefaultConfig, loadConfig, saveConfig 已移动到 persistence.go

// 计算股票的持仓盈亏

// 计算股票的加权平均成本价

// 计算总持股数量

// loadPortfolio 已移动到 persistence.go

// 格式化函数 (formatProfitWithColorLang, formatPriceWithColorLang, abs 等) 已移动到 format.go

// API 相关函数 (getStockInfo, getStockPrice, searchStock*, tryXXXAPI 等) 已移动到 api.go
// 缓存相关函数 (getStockPriceFromCache, startStockPriceUpdates) 已移动到 cache.go
// scroll 相关函数 (scrollPortfolioUp/Down, scrollWatchlistUp/Down) 已移动到 scroll.go

// formatVolume, ui.IsControlKey 已移动到 format.go 和 ui_utils.go

// Language selection handlers moved to handlers_menu.go

// ========== 自选股票相关功能 ==========

// WatchlistStockLegacy, WatchlistLegacy, loadWatchlist, saveWatchlist 已移动到 persistence.go

// 标签管理函数 (renameTagForAllStocks, getAvailableTags, hasTag, addTag, removeTag, getTagsDisplay, getFilteredWatchlist, invalidateWatchlistCache) 已移动到 watchlist.go

// 重置持股列表游标到第一只股票

// 处理自选股票标签选择

// 标签选择视图

// ========== 新的标签管理界面 ==========

// 处理标签管理界面

// 处理标签删除选择界面

// 处理标签编辑界面

// 标签管理界面视图

// 标签删除选择界面视图

// 标签编辑界面视图

// 文本编辑辅助函数 (ui.InsertRuneAtCursor, ui.DeleteRuneBeforeCursor, ui.HandleTextInput 等) 已移动到 ui_utils.go

// isStockInWatchlist 已移动到 watchlist.go

// 检查股票是否在持仓中

// formatStockNameWithPortfolioHighlight 已移动到 format.go
// addToWatchlist, removeFromWatchlist 已移动到 watchlist.go

// ========== 搜索结果带操作按钮处理 ==========

// ========== 自选股票查看处理 ==========

// gbkToUtf8 已移动到 ui_utils.go

// ========== 自选股票搜索确认处理 ==========

// 获取排序字段的显示名称

// 获取排序方向的显示名称

// 获取持股列表可用的排序字段

// 获取自选列表可用的排序字段

// 查找排序字段在字段列表中的索引，如果找不到返回0

// 处理持股列表排序

// 处理自选列表排序

// 排序菜单视图 - 持股列表

// 排序菜单视图 - 自选列表

// 分时数据采集和图表功能已移动到 intraday_chart.go
// 包含: startIntradayDataCollection, stopIntradayDataCollection, loadIntradayDataForDate,
// parseIntradayTime, calculateAdaptiveMargin, getSmartChartDate, findPreviousTradingDayFromDate,
// createFixedTimeRange, createIntradayChart, triggerIntradayDataCollection, formatDate,
// isWeekend, findPreviousTradingDay, findNextTradingDay, handleIntradayChartViewing, viewIntradayChart

// ============================================================================
// Model Interface Methods (for internal/intraday package)
// ============================================================================

// GetConfig returns the configuration for market access
func (m *Model) GetConfig() interface{} {
	return m.config
}

// GetStockPriceCache returns the stock price cache
func (m *Model) GetStockPriceCache() map[string]interface{} {
	cache := make(map[string]interface{})
	for k, v := range m.stockPriceCache {
		cache[k] = v
	}
	return cache
}

// GetStockPriceMutex returns the mutex protecting the stock price cache
func (m *Model) GetStockPriceMutex() *sync.RWMutex {
	return &m.stockPriceMutex
}

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
			marketTag := m.getMarketTagName(watchStock.Market)
			row[i] = watchlist.GetTagsDisplay(watchStock, marketTag)
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
