package types

// ConfigProvider 配置提供者接口
// 用于解耦模块与 Model 之间的依赖
type ConfigProvider interface {
	// GetLanguage 获取当前语言设置
	GetLanguage() string
	// GetDisplayConfig 获取显示配置
	GetDisplayConfig() DisplayConfig
	// GetConfig 获取完整配置
	GetConfig() Config
}

// TextProvider 文本提供者接口（i18n）
type TextProvider interface {
	// GetText 根据 key 获取翻译文本
	GetText(key string) string
}

// CacheProvider 缓存提供者接口
type CacheProvider interface {
	// GetStockPrice 从缓存获取股票价格
	GetStockPrice(symbol string) *StockData
	// SetStockPrice 设置股票价格到缓存
	SetStockPrice(symbol string, data *StockData)
	// IsStockPriceExpired 检查缓存是否过期
	IsStockPriceExpired(symbol string) bool
}

// PortfolioProvider 持仓数据提供者接口
type PortfolioProvider interface {
	// GetPortfolio 获取持仓列表
	GetPortfolio() Portfolio
	// IsStockInPortfolio 检查股票是否在持仓中
	IsStockInPortfolio(code string) bool
}

// WatchlistProvider 自选列表提供者接口
type WatchlistProvider interface {
	// GetWatchlist 获取自选列表
	GetWatchlist() Watchlist
	// GetFilteredWatchlist 获取过滤后的自选列表
	GetFilteredWatchlist() []WatchlistStock
}

// AlertProvider 告警数据提供者接口
type AlertProvider interface {
	// GetAlerts 获取告警列表
	GetAlerts() []Alert
	// GetAlertsByStock 获取指定股票的告警
	GetAlertsByStock(code string) []Alert
}

// ColorFormatter 颜色格式化接口
type ColorFormatter interface {
	// FormatProfit 格式化盈亏（带颜色）
	FormatProfit(value float64) string
	// FormatProfitRate 格式化盈亏率（带颜色）
	FormatProfitRate(rate float64) string
	// FormatPrice 格式化价格（根据涨跌显示颜色）
	FormatPrice(currentPrice, prevClose float64) string
}
