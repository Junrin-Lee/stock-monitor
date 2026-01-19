package types

import (
	"time"
)

// TimePoint 图表时间点数据
type TimePoint struct {
	Time  time.Time
	Value float64
}

// CollectionMode 数据采集模式
type CollectionMode int

const (
	CollectionModeHistorical CollectionMode = iota // 采集历史交易日数据
	CollectionModeLive                             // 采集当日实时数据
	CollectionModeComplete                         // 数据已完整，无需采集
)

// String returns the string representation of CollectionMode
func (c CollectionMode) String() string {
	switch c {
	case CollectionModeHistorical:
		return "Historical"
	case CollectionModeLive:
		return "Live"
	case CollectionModeComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// TradingState 交易状态
type TradingState int

const (
	TradingStatePreMarket  TradingState = iota // 盘前（开盘前）
	TradingStateLive                           // 交易中
	TradingStatePostMarket                     // 盘后（收盘后，当日）
	TradingStateWeekend                        // 周末
	TradingStateHoliday                        // 假日
)

// String returns the string representation of TradingState
func (t TradingState) String() string {
	switch t {
	case TradingStatePreMarket:
		return "PreMarket"
	case TradingStateLive:
		return "Live"
	case TradingStatePostMarket:
		return "PostMarket"
	case TradingStateWeekend:
		return "Weekend"
	case TradingStateHoliday:
		return "Holiday"
	default:
		return "Unknown"
	}
}

// WorkerMetadata Worker状态元数据
type WorkerMetadata struct {
	StockCode         string         // 股票代码
	TargetDate        string         // 目标日期 (YYYYMMDD)
	Mode              CollectionMode // 采集模式
	StartTime         time.Time      // Worker启动时间
	LastUpdateTime    time.Time      // 最后更新时间
	DatapointCount    int            // 已采集数据点数量
	ConsecutiveErrors int            // 连续错误次数
	ConsecutiveSkips  int            // 连续跳过次数（数据完全一致）
	IsRunning         bool           // 是否正在运行
}
