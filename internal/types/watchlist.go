package types

// WatchlistStock 自选股票数据结构
type WatchlistStock struct {
	Code   string     `json:"code"`
	Name   string     `json:"name"`
	Tags   []string   `json:"tags"`             // 标签字段，仅存储用户自定义标签
	Market MarketType `json:"market,omitempty"` // 市场类型标识（A股/美股/港股）
}

// Watchlist 自选股票列表
type Watchlist struct {
	Stocks []WatchlistStock `json:"stocks"`
}

// MarketType 市场类型枚举
type MarketType string

const (
	MarketChina    MarketType = "china"
	MarketUS       MarketType = "us"
	MarketHongKong MarketType = "hongkong"
)

// TagGroup 标签分组结构 (v5.6)
type TagGroup struct {
	Name string   // 分组名称 (如 "市场分组", "自定义标签")
	Tags []string // 该分组下的标签列表
}
