package types

import (
	"time"
)

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
	// 美股盘前盘后数据（A股/港股为零值）
	PreMarketPrice    float64 `json:"pre_market_price,omitempty"`
	PreMarketChange   float64 `json:"pre_market_change,omitempty"`
	PreMarketPercent  float64 `json:"pre_market_percent,omitempty"`
	PostMarketPrice   float64 `json:"post_market_price,omitempty"`
	PostMarketChange  float64 `json:"post_market_change,omitempty"`
	PostMarketPercent float64 `json:"post_market_percent,omitempty"`
}

// Portfolio 持仓组合
type Portfolio struct {
	Stocks []Stock `json:"stocks"`
}

// StockPriceCacheEntry 股价缓存条目结构
type StockPriceCacheEntry struct {
	Data       *StockData `json:"data"`        // 股价数据
	UpdateTime time.Time  `json:"update_time"` // 数据更新时间
	IsUpdating bool       `json:"is_updating"` // 是否正在更新中
}
