package main

import (
	"sync"
	"time"

	"stock-monitor/internal/consts"
	"stock-monitor/internal/types"
)

// 为了向后兼容，重新导出 internal/types 包中的类型
// 注意：对于需要添加方法的类型，我们使用独立定义而非别名

type (
	// 数据配置类型 - 使用别名（无需添加方法）
	Config = types.Config
	SystemConfig             = types.SystemConfig
	DisplayConfig            = types.DisplayConfig
	UpdateConfig             = types.UpdateConfig
	TradingSession           = types.TradingSession
	MarketConfig             = types.MarketConfig
	MarketsConfig            = types.MarketsConfig
	IntradayCollectionConfig = types.IntradayCollectionConfig
	TextMap                  = types.TextMap
	TimePoint                = types.TimePoint
	WorkerMetadata           = types.WorkerMetadata
)

// 重新导出 internal/consts 包中的类型
type (
	Language      = consts.Language
	AppState      = consts.AppState
	SortField     = consts.SortField
	SortDirection = consts.SortDirection
)

// MarketType 市场类型枚举（需要和 types 包中保持一致）
type MarketType = types.MarketType

// 重新导出 MarketType 常量
const (
	MarketChina    = types.MarketChina
	MarketUS       = types.MarketUS
	MarketHongKong = types.MarketHongKong
)

// AlertType 告警类型
type AlertType = types.AlertType

// TriggerFrequency 触发频率类型
type TriggerFrequency = types.TriggerFrequency

// 重新导出告警相关常量
const (
	AlertTypePrice  = types.AlertTypePrice
	AlertTypeRate   = types.AlertTypeRate
	AlertTypeVolume = types.AlertTypeVolume

	TriggerOnce       = types.TriggerOnce
	TriggerDaily      = types.TriggerDaily
	TriggerWeekly     = types.TriggerWeekly
	TriggerMonthly    = types.TriggerMonthly
	TriggerEveryNDays = types.TriggerEveryNDays
)

// BatchStockSource 批量股票来源类型
type BatchStockSource = types.BatchStockSource

// 重新导出 BatchStockSource 常量
const (
	BatchSourceWatchlist = types.BatchSourceWatchlist
	BatchSourcePortfolio = types.BatchSourcePortfolio
	BatchSourceManual    = types.BatchSourceManual
)

// CollectionMode 数据采集模式
type CollectionMode = types.CollectionMode

// 重新导出 CollectionMode 常量
const (
	CollectionModeHistorical = types.CollectionModeHistorical
	CollectionModeLive       = types.CollectionModeLive
	CollectionModeComplete   = types.CollectionModeComplete
)

// TradingState 交易状态
type TradingState = types.TradingState

// 重新导出 TradingState 常量
const (
	TradingStatePreMarket  = types.TradingStatePreMarket
	TradingStateLive       = types.TradingStateLive
	TradingStatePostMarket = types.TradingStatePostMarket
	TradingStateWeekend    = types.TradingStateWeekend
	TradingStateHoliday    = types.TradingStateHoliday
)

// 重新导出变量和函数
var AlertConditions = types.AlertConditions

// 辅助函数重新导出
var (
	GetAlertTypeFromCursor      = types.GetAlertTypeFromCursor
	GetAlertConditionFromCursor = types.GetAlertConditionFromCursor
	CheckNumericCondition       = types.CheckNumericCondition
	MoveCursorUp                = types.MoveCursorUp
	MoveCursorDown              = types.MoveCursorDown
)

// ============================================================================
// 需要添加方法的类型 - 在 main 包中独立定义
// ============================================================================

// Stock 持仓股票数据结构
type Stock struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	CostPrice     float64 `json:"cost_price"`
	Quantity      int     `json:"quantity"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	StartPrice    float64 `json:"start_price"`
	MaxPrice      float64 `json:"max_price"`
	MinPrice      float64 `json:"min_price"`
	PrevClose     float64 `json:"prev_close"`
}

// StockData 股票市场数据（来自API）
type StockData struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	StartPrice    float64 `json:"start_price"`
	MaxPrice      float64 `json:"max_price"`
	MinPrice      float64 `json:"min_price"`
	PrevClose     float64 `json:"prev_close"` // 昨日收盘价
	TurnoverRate  float64 `json:"turnover_rate"`
	Volume        int64   `json:"volume"`
}

// WatchlistStock 自选股票数据结构 (使用别名)
type WatchlistStock = types.WatchlistStock

// Watchlist 自选股票列表 (使用别名)
type Watchlist = types.Watchlist

// TagGroup 标签分组结构 (v5.6) (使用别名)
type TagGroup = types.TagGroup

// Portfolio 持仓组合
type Portfolio struct {
	Stocks []Stock `json:"stocks"`
}

// Alert 告警规则
type Alert struct {
	ID            string           `json:"id"`             // 唯一标识符（UUID v4）
	StockCode     string           `json:"code"`           // 股票代码
	StockName     string           `json:"name"`           // 股票名称
	Type          AlertType        `json:"type"`           // 告警类型
	Condition     string           `json:"condition"`      // 条件（">" / "<" / ">=" / "<="）
	Threshold     float64          `json:"threshold"`      // 阈值（价格或百分比）
	IsActive      bool             `json:"is_active"`      // 是否启用
	Frequency     TriggerFrequency `json:"frequency"`      // 触发频率
	FrequencyDays int              `json:"frequency_days"` // 自定义天数间隔（仅 every_n_days 模式）
	CreatedAt     time.Time        `json:"created_at"`     // 创建时间
	TriggeredAt   time.Time        `json:"triggered_at"`   // 最后触发时间
	LastChecked   time.Time        `json:"last_checked"`   // 最后检查时间
	BatchTag      string           `json:"batch_tag"`      // 批量标签名（批量添加时）
}

// AlertData 告警配置文件
type AlertData struct {
	Alerts     []Alert `json:"alerts"`
	LastCheck  string  `json:"last_check"`
	AlertCount int     `json:"alert_count"`
}

// StockPriceCacheEntry 股价缓存条目结构
type StockPriceCacheEntry struct {
	Data       *StockData `json:"data"`        // 股价数据
	UpdateTime time.Time  `json:"update_time"` // 数据更新时间
	IsUpdating bool       `json:"is_updating"` // 是否正在更新中
}

// Model 应用程序主模型
type Model struct {
	state           AppState
	currentMenuItem int
	menuItems       []string
	cursor          int
	input           string
	inputCursor     int // 通用输入框光标位置
	message         string
	portfolio       Portfolio
	watchlist       Watchlist // 自选股票列表
	config          Config    // 系统配置
	language        Language

	// For stock addition
	addingStep         int
	tempCode           string
	tempCodeCursor     int // 股票代码输入光标位置
	tempCost           string
	tempCostCursor     int // 成本价输入光标位置
	tempQuantity       string
	tempQuantityCursor int // 数量输入光标位置
	stockInfo          *StockData
	fromSearch         bool     // 标记是否从搜索结果添加
	previousState      AppState // 记录进入编辑/删除前的状态

	// For stock editing
	editingStep        int
	selectedStockIndex int

	// For stock searching
	searchInput         string
	searchInputCursor   int // 搜索输入光标位置
	searchResult        *StockData
	searchFromWatchlist bool // 标记是否从自选列表进入搜索

	// For language selection
	languageCursor int

	// For monitoring
	lastUpdate time.Time

	// For scrolling
	portfolioScrollPos int // 持股列表滚动位置
	watchlistScrollPos int // 自选列表滚动位置
	portfolioCursor    int // 持股列表当前选中行
	watchlistCursor    int // 自选列表当前选中行

	// For watchlist tagging and grouping
	selectedMarketFilter  MarketType // 市场过滤条件："" 表示全部市场，或具体市场类型
	selectedUserTagFilter string     // 用户标签过滤条件："" 表示全部标签，或具体用户标签
	filterSelectionStep   int        // 过滤选择步骤：0=市场选择, 1=用户标签选择
	availableTags         []string   // 所有可用的标签列表
	tagGroups             []TagGroup // 标签分组（市场分组 + 用户标签）- v5.6
	lastSelectedGroupTag  string     // 上次在分组选择中选中的标签（用于记住位置）- v5.6
	tagInput              string     // 标签输入框内容
	tagInputCursor        int        // 标签输入光标位置
	tagSelectCursor       int        // 标签选择界面的游标位置
	currentStockTags      []string   // 当前选中股票的标签列表（用于删除管理）
	tagManageCursor       int        // 标签管理界面的游标位置
	tagRemoveCursor       int        // 标签删除选择界面的游标位置
	isInRemoveMode        bool       // 是否处于删除模式
	tagToEdit             string     // 要编辑的原标签名称
	tagEditInput          string     // 标签编辑输入框内容
	tagEditInputCursor    int        // 标签编辑输入光标位置

	// Performance optimization - cached filtered watchlist
	cachedFilteredWatchlist  []WatchlistStock // 缓存的过滤后自选列表
	cachedFilterMarket       MarketType       // 缓存的市场过滤条件
	cachedFilterUserTag      string           // 缓存的用户标签过滤条件
	isFilteredWatchlistValid bool             // 缓存是否有效

	// For sorting - 持股列表排序状态
	portfolioSortField     SortField     // 持股列表当前排序字段
	portfolioSortDirection SortDirection // 持股列表当前排序方向
	portfolioSortCursor    int           // 持股列表排序菜单光标位置
	portfolioIsSorted      bool          // 持股列表是否已经应用了排序

	// For sorting - 自选列表排序状态
	watchlistSortField     SortField     // 自选列表当前排序字段
	watchlistSortDirection SortDirection // 自选列表当前排序方向
	watchlistSortCursor    int           // 自选列表排序菜单光标位置
	watchlistIsSorted      bool          // 自选列表是否已经应用了排序

	// For stock price async data - 股价异步数据
	stockPriceCache      map[string]*StockPriceCacheEntry // 股价数据缓存
	stockPriceMutex      sync.RWMutex                     // 股价数据读写锁
	stockPriceUpdateTime time.Time                        // 上次更新股价数据的时间

	// For intraday data collection - 分时数据采集
	intradayManager *IntradayManager // 分时数据管理器

	// For intraday chart viewing - 分时图表查看
	chartViewStock        string        // 正在查看的股票代码
	chartViewStockName    string        // 股票名称
	chartViewDate         string        // 正在查看的日期 (YYYYMMDD)
	chartData             *IntradayData // 加载的分时数据
	chartLoadError        error         // 加载错误(如有)
	chartIsCollecting     bool          // 是否正在自动采集数据
	chartCollectStartTime time.Time     // 开始采集的时间

	// For search mode intraday - 搜索模式临时分时数据
	isSearchMode           bool          // 是否处于搜索模式（用于区分数据来源）
	searchIntradayData     *IntradayData // 搜索模式的临时分时数据(仅内存)
	searchIntradayWorker   chan struct{} // 临时 worker 停止信号
	searchIntradayUpdateCh chan struct{} // 数据更新通知 channel

	// For alert management - 告警管理相关字段
	alertData              AlertData        // 告警数据
	alertCursor            int              // 告警列表光标
	alertScrollPos         int              // 告警滚动位置
	alertManageStep        int              // 告警管理步骤
	alertInput             string           // 告警输入
	alertInputCursor       int              // 输入光标
	selectedAlertType      AlertType        // 选中的告警类型
	selectedAlertCondition string           // 选中的告警条件
	alertThreshold         float64          // 告警阈值
	currentAlert           Alert            // 当前操作的告警
	batchAlertTag          string           // 批量添加的标签
	stockAlertCode         string           // 当前股票的代码（StockAlertManage状态使用）
	stockAlertName         string           // 当前股票的名称（StockAlertManage状态使用）
	stockAlertCursor       int              // 股票告警列表光标
	stockAlertAlerts       []Alert          // 当前股票的所有告警（缓存）
	batchSelectedStocks    []string         // 批量选中的股票代码
	batchStockSource       BatchStockSource // 批量股票来源
	batchSelectStep        int              // 批量选择步骤
	batchCodeInput         string           // 批量输入的股票代码
	stockSelectionMap      map[string]bool  // 股票选择状态映射（code -> selected）
	batchSelectedMarket    string           // 选中的市场类型（"china"/"us"/"hongkong"）
	marketCursor           int              // 市场选择光标

	// For alert frequency selection - 告警频率选择相关字段
	selectedAlertFrequency TriggerFrequency // 选中的触发频率
	alertFrequencyDays     int              // 自定义天数间隔
	alertFrequencyCursor   int              // 频率选择光标
}

// tickMsg 定时刷新消息
type tickMsg struct{}

// stockPriceUpdateMsg 股价数据更新消息
type stockPriceUpdateMsg struct {
	Symbol string
	Data   *StockData
	Error  error
}

// checkDataAvailabilityMsg 数据可用性检查消息
type checkDataAvailabilityMsg struct {
	code string
	date string
}

// searchIntradayUpdateMsg 搜索模式分时数据更新消息
type searchIntradayUpdateMsg struct{}
