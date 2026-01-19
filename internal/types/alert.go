package types

import (
	"time"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypePrice  AlertType = "price"  // 价格告警
	AlertTypeRate   AlertType = "rate"   // 涨跌幅告警
	AlertTypeVolume AlertType = "volume" // 成交量告警
)

// TriggerFrequency 触发频率类型
type TriggerFrequency string

const (
	TriggerOnce       TriggerFrequency = "once"         // 一次性
	TriggerDaily      TriggerFrequency = "daily"        // 每天一次
	TriggerWeekly     TriggerFrequency = "weekly"       // 每周一次
	TriggerMonthly    TriggerFrequency = "monthly"      // 每月一次
	TriggerEveryNDays TriggerFrequency = "every_n_days" // 每 N 天一次
)

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

// BatchStockSource 批量股票来源类型
type BatchStockSource int

const (
	BatchSourceWatchlist BatchStockSource = iota // 从自选列表
	BatchSourcePortfolio                         // 从持股列表
	BatchSourceManual                            // 手动输入
)

// ============================================================================
// 辅助函数 - Alert Helper Functions
// ============================================================================

// AlertConditions 支持的警报条件列表
var AlertConditions = []string{">", "<", ">=", "<="}

// GetAlertTypeFromCursor 根据光标位置返回警报类型
func GetAlertTypeFromCursor(cursor int) AlertType {
	types := []AlertType{AlertTypePrice, AlertTypeRate, AlertTypeVolume}
	if cursor >= 0 && cursor < len(types) {
		return types[cursor]
	}
	return AlertTypePrice // 默认返回价格警报
}

// GetAlertConditionFromCursor 根据光标位置返回条件符号
func GetAlertConditionFromCursor(cursor int) string {
	if cursor >= 0 && cursor < len(AlertConditions) {
		return AlertConditions[cursor]
	}
	return ">" // 默认返回大于
}

// CheckNumericCondition 检查数值是否满足条件
// 参数:
//   - value: 当前值
//   - threshold: 阈值
//   - condition: 条件符号 (">", "<", ">=", "<=")
//
// 返回: 是否满足条件
func CheckNumericCondition(value, threshold float64, condition string) bool {
	switch condition {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	default:
		return false
	}
}
