package main

import "time"

// 文件路径常量
const (
	dataFile        = "data/portfolio.json"
	watchlistFile   = "data/watchlist.json"
	alertFile       = "data/alert.json"
	configFile      = "cmd/conf/config.yml"
	refreshInterval = 5 * time.Second
)

// 语言常量
type Language string

const (
	Chinese Language = "zh"
	English Language = "en"
)

// 应用状态常量
type AppState int

const (
	MainMenu AppState = iota
	AddingStock
	Monitoring
	EditingStock
	SearchingStock
	SearchResult
	LanguageSelection
	WatchlistViewing
	SearchResultWithActions
	WatchlistSearchConfirm
	WatchlistTagging         // 自选股票打标签状态
	WatchlistTagSelect       // 自选股票标签选择状态
	WatchlistTagManage       // 自选股票标签管理状态（显示当前股票的所有标签）
	WatchlistTagRemoveSelect // 自选股票标签删除选择状态
	WatchlistTagEdit         // 自选股票标签编辑状态（修改标签名称）
	WatchlistGroupSelect     // 自选股票分组选择状态
	PortfolioSorting         // 持股列表排序状态
	WatchlistSorting         // 自选列表排序状态
	IntradayChartViewing     // 分时图表查看状态
	AlertManage              // 告警管理状态（直接显示列表）
	StockAlertManage         // 股票告警详情状态（单个股票的告警管理）
	AlertAdd                 // 添加告警状态
	AlertBatchMethodSelect   // 批量添加方式选择状态
	AlertBatchByTag          // 按标签批量添加状态
	AlertBatchByMarket       // 按市场批量添加状态
	AlertBatchByStocks       // 按股票列表批量添加状态
	SelectBatchStocks        // 选择批量股票状态
	SelectBatchFromWatchlist // 从自选列表选择股票状态
	SelectBatchFromPortfolio // 从持股列表选择股票状态
	InputBatchCodes          // 手动输入股票代码状态
	AlertEdit                // 编辑告警状态
	AlertTypeSelect          // 选择告警类型状态
	AlertThresholdInput      // 输入阈值状态
	AlertConditionSelect     // 选择条件状态
	AlertBatchAdd            // 批量添加告警状态
	AlertFrequencySelect     // 选择告警触发频率状态
	AlertFrequencyDaysInput  // 输入自定义天数状态
)

// 排序字段枚举
type SortField int

const (
	SortByCode          SortField = iota // 股票代码
	SortByName                           // 股票名称
	SortByPrice                          // 现价
	SortByCostPrice                      // 成本价
	SortByChange                         // 涨跌额
	SortByChangePercent                  // 涨跌幅
	SortByQuantity                       // 持股数量
	SortByTotalProfit                    // 持仓盈亏
	SortByProfitRate                     // 盈亏率
	SortByMarketValue                    // 市值
	SortByTag                            // 标签 (仅自选列表)
	SortByTurnoverRate                   // 换手率 (仅自选列表)
	SortByVolume                         // 成交量 (仅自选列表)
)

// 排序方向枚举
type SortDirection int

const (
	SortAsc  SortDirection = iota // 升序
	SortDesc                      // 降序
)
