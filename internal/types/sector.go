package types

import "time"

// SectorType 板块类型
type SectorType string

const (
	SectorTypeIndustry SectorType = "industry" // 行业板块
	SectorTypeConcept  SectorType = "concept"  // 概念板块
)

// Sector 板块数据
type Sector struct {
	Code          string     `json:"code"`           // 板块代码 (BK0493)
	Name          string     `json:"name"`           // 板块名称
	Type          SectorType `json:"type"`           // 行业/概念
	ChangePercent float64    `json:"change_percent"` // 涨跌幅
	Change        float64    `json:"change"`         // 涨跌额
	Turnover      float64    `json:"turnover"`       // 成交额
	TurnoverRate  float64    `json:"turnover_rate"`  // 换手率
	RiseCount     int        `json:"rise_count"`     // 上涨家数
	FallCount     int        `json:"fall_count"`     // 下跌家数
	LeaderCode    string     `json:"leader_code"`    // 领涨股代码
	LeaderName    string     `json:"leader_name"`    // 领涨股名称
	LeaderChange  float64    `json:"leader_change"`  // 领涨股涨幅
	UpdatedAt     time.Time  `json:"updated_at"`     // 更新时间
}

// SectorStock 板块成分股
type SectorStock struct {
	Code          string    `json:"code"`           // 股票代码
	Name          string    `json:"name"`           // 股票名称
	Price         float64   `json:"price"`          // 现价
	ChangePercent float64   `json:"change_percent"` // 涨跌幅
	Change        float64   `json:"change"`         // 涨跌额
	Volume        int64     `json:"volume"`         // 成交量
	Turnover      float64   `json:"turnover"`       // 成交额
	TurnoverRate  float64   `json:"turnover_rate"`  // 换手率
	UpdatedAt     time.Time `json:"updated_at"`     // 更新时间
}

// SectorCacheEntry 板块缓存条目
type SectorCacheEntry struct {
	Sectors   []Sector  `json:"sectors"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SectorStockCacheEntry 成分股缓存条目
type SectorStockCacheEntry struct {
	Stocks    []SectorStock `json:"stocks"`
	UpdatedAt time.Time     `json:"updated_at"`
}
