package types

// Config 系统配置结构
type Config struct {
	System             SystemConfig             `yaml:"system"`              // 系统设置
	Display            DisplayConfig            `yaml:"display"`             // 显示设置
	Update             UpdateConfig             `yaml:"update"`              // 更新设置
	Markets            MarketsConfig            `yaml:"markets"`             // 市场配置
	IntradayCollection IntradayCollectionConfig `yaml:"intraday_collection"` // 分时数据采集配置
}

// SystemConfig 系统设置
type SystemConfig struct {
	Language      string `yaml:"language"`       // 默认语言 "zh" 或 "en"
	AutoStart     bool   `yaml:"auto_start"`     // 有数据时自动进入监控模式
	StartupModule string `yaml:"startup_module"` // 启动模块 "portfolio"(持股) 或 "watchlist"(自选)
	LogLevel      string `yaml:"log_level"`      // 日志级别 "debug", "info", "warn", "error"
}

// DisplayConfig 显示设置
type DisplayConfig struct {
	ColorScheme        string   `yaml:"color_scheme"`        // 颜色方案 "professional", "simple"
	DecimalPlaces      int      `yaml:"decimal_places"`      // 价格显示小数位数
	TableStyle         string   `yaml:"table_style"`         // 表格样式 "light", "bold", "simple"
	MaxLines           int      `yaml:"max_lines"`           // 列表每页最大显示行数
	PortfolioHighlight string   `yaml:"portfolio_highlight"` // 自选列表中持仓股票的背景高亮颜色
	PortfolioColumns   []string `yaml:"portfolio_columns"`   // 持股列表显示的列（按顺序）
	WatchlistColumns   []string `yaml:"watchlist_columns"`   // 自选列表显示的列（按顺序）
}

// UpdateConfig 更新设置
type UpdateConfig struct {
	RefreshInterval int  `yaml:"refresh_interval"` // 刷新间隔（秒）
	AutoUpdate      bool `yaml:"auto_update"`      // 是否自动更新
}

// TradingSession 交易时段
type TradingSession struct {
	StartTime string `yaml:"start_time"` // "09:30"
	EndTime   string `yaml:"end_time"`   // "11:30"
}

// MarketConfig 市场配置
type MarketConfig struct {
	Timezone        string           `yaml:"timezone"`         // "Asia/Shanghai"
	TradingSessions []TradingSession `yaml:"trading_sessions"` // 交易时段列表
	Weekdays        []int            `yaml:"weekdays"`         // [1,2,3,4,5] (周一到周五)
}

// MarketsConfig 所有市场配置
type MarketsConfig struct {
	China    MarketConfig `yaml:"china"`
	US       MarketConfig `yaml:"us"`
	HongKong MarketConfig `yaml:"hongkong"`
}

// IntradayCollectionConfig 分时数据采集配置
type IntradayCollectionConfig struct {
	EnableAutoStop        bool    `yaml:"enable_auto_stop"`       // 启用自动停止
	CompletenessThreshold float64 `yaml:"completeness_threshold"` // 完整性阈值 (百分比)
	MaxConsecutiveErrors  int     `yaml:"max_consecutive_errors"` // 最大连续错误次数
	MinDatapoints         int     `yaml:"min_datapoints"`         // 最小数据点数量
}

// TextMap 文本映射结构（用于i18n）
type TextMap map[string]string
