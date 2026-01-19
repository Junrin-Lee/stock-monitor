package intraday

// IntradayDataPoint represents a single minute's price data
type IntradayDataPoint struct {
	Time  string  `json:"time"`  // Format: "09:31" (HH:MM)
	Price float64 `json:"price"` // Closing price for that minute
}

// IntradayData represents the complete intraday data for a stock on a given day
type IntradayData struct {
	Code       string              `json:"code"`                 // e.g., "SH600000"
	Name       string              `json:"name"`                 // e.g., "浦发银行"
	Date       string              `json:"date"`                 // Format: "20251126"
	Market     string              `json:"market,omitempty"`     // 市场类型 (向后兼容)
	Datapoints []IntradayDataPoint `json:"datapoints"`           // Minute-by-minute data
	UpdatedAt  string              `json:"updated_at"`           // Format: "2025-11-26 15:00:00"
	PrevClose  float64             `json:"prev_close,omitempty"` // 昨日收盘价（向后兼容）
}

// DatapointDiffResult 表示数据点比较结果
type DatapointDiffResult struct {
	HasPriceChanges  bool // 是否有价格变化
	HasNewEntries    bool // 是否有新时间点
	PriceChangeCount int  // 价格变化数量
	NewEntryCount    int  // 新时间点数量
}

// SaveDecision 表示保存决策
type SaveDecision int

const (
	SaveDecisionSkip   SaveDecision = iota // 跳过保存（无任何变化）
	SaveDecisionAppend                     // 追加新数据（仅新时间点）
	SaveDecisionUpdate                     // 增量更新（有价格变化）
)
