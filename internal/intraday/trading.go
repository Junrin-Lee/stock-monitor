package intraday

import (
	"fmt"
	"time"

	"stock-monitor/internal/api"
	"stock-monitor/internal/types"
)

// GetTradingState 判断市场当前的交易状态
// now: 当前时间（已转换为市场时区）
// marketType: 市场类型
// 返回：TradingState 表示市场状态
func GetTradingState(now time.Time, marketType string) types.TradingState {
	weekday := now.Weekday()

	// 检查周末
	if weekday == time.Saturday || weekday == time.Sunday {
		return types.TradingStateWeekend
	}

	// TODO: 假日检测 (v2 enhancement)
	// 目前假设工作日都是交易日

	// 获取市场特定的交易时段
	var morningStart, morningEnd, afternoonStart, afternoonEnd time.Time

	switch marketType {
	case "china": // A股: 09:30-11:30, 13:00-15:00 CST
		morningStart = time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
		morningEnd = time.Date(now.Year(), now.Month(), now.Day(), 11, 30, 0, 0, now.Location())
		afternoonStart = time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, now.Location())
		afternoonEnd = time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location())

	case "us": // 美股: 09:30-16:00 EST/EDT (无午休)
		morningStart = time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
		afternoonEnd = time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
		morningEnd = afternoonEnd     // 无午休
		afternoonStart = morningStart // 无午休

	case "hongkong": // 港股: 09:30-12:00, 13:00-16:00 HKT
		morningStart = time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
		morningEnd = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
		afternoonStart = time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, now.Location())
		afternoonEnd = time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())

	default:
		return types.TradingStatePostMarket
	}

	// 判断当前状态
	if now.Before(morningStart) {
		return types.TradingStatePreMarket
	} else if (now.After(morningStart) && now.Before(morningEnd)) ||
		(now.After(afternoonStart) && now.Before(afternoonEnd)) {
		return types.TradingStateLive
	} else {
		return types.TradingStatePostMarket
	}
}

// FindPreviousTradingDay 查找上一个交易日（支持多市场时区）
// stockCode: 股票代码（用于检测市场类型）
// currentDate: 当前日期 (YYYYMMDD)
// m: Model 引用
// 返回: 上一个交易日 (YYYYMMDD)
func FindPreviousTradingDay(stockCode string, currentDate string, m ModelInterface) string {
	// 获取市场类型和时区
	marketType := string(api.GetMarketType(stockCode))
	location, err := GetMarketLocation(marketType)
	if err != nil {
		// 降级到本地时区
		location = time.Local
	}

	// 解析当前日期
	date, err := time.ParseInLocation("20060102", currentDate, location)
	if err != nil {
		return currentDate // 解析失败，返回原日期
	}

	// 最多回溯 7 天，找到上一个交易日
	for i := 1; i <= 7; i++ {
		prevDate := date.AddDate(0, 0, -i)
		weekday := prevDate.Weekday()

		// 跳过周末（周六、周日）
		if weekday == time.Saturday || weekday == time.Sunday {
			continue
		}

		// TODO: 检查假日日历 (v2 enhancement)
		// 目前假设非周末的工作日都是交易日

		return prevDate.Format("20060102")
	}

	// 降级：如果 7 天内找不到，直接返回前一天
	return date.AddDate(0, 0, -1).Format("20060102")
}

// GetTradingDayForCollection 决定应该采集哪天的数据以及采集模式
// stockCode: 股票代码
// m: Model 引用
// 返回: (targetDate, mode, error)
//   - targetDate: 目标日期 (YYYYMMDD)
//   - mode: 采集模式 (Historical/Live/Complete)
//   - error: 错误（如果有）
func GetTradingDayForCollection(stockCode string, m ModelInterface) (string, types.CollectionMode, error) {
	// 步骤 1: 检测市场类型
	marketType := string(api.GetMarketType(stockCode))
	if marketType == "" {
		return "", types.CollectionModeComplete, fmt.Errorf("unable to detect market for stock: %s", stockCode)
	}

	// 步骤 2: 获取市场时区和当前时间
	location, err := GetMarketLocation(marketType)
	if err != nil {
		return "", types.CollectionModeComplete, err
	}
	now := time.Now().In(location)

	// 步骤 3: 判断交易状态
	tradingState := GetTradingState(now, marketType)

	switch tradingState {
	case types.TradingStatePreMarket:
		// 盘前 -> 获取上一个交易日的数据
		prevDate := FindPreviousTradingDay(stockCode, now.Format("20060102"), m)
		return prevDate, types.CollectionModeHistorical, nil

	case types.TradingStateLive:
		// 交易中 -> 获取当日实时数据
		return now.Format("20060102"), types.CollectionModeLive, nil

	case types.TradingStatePostMarket:
		// 盘后 -> 获取今天的完整数据
		todayDate := now.Format("20060102")

		// 检查今天的数据是否已经完整
		complete, _ := IsDataComplete(stockCode, todayDate, marketType, false)
		if complete {
			return todayDate, types.CollectionModeComplete, nil
		}
		return todayDate, types.CollectionModeHistorical, nil

	case types.TradingStateWeekend, types.TradingStateHoliday:
		// 周末/假日 -> 获取上一个交易日的数据
		prevDate := FindPreviousTradingDay(stockCode, now.Format("20060102"), m)
		return prevDate, types.CollectionModeHistorical, nil

	default:
		return "", types.CollectionModeComplete, fmt.Errorf("unknown trading state")
	}
}

// IsMarketOpen 检查当前是否在交易时间内（支持多市场）
func IsMarketOpen(stockCode string, m ModelInterface) bool {
	market := string(api.GetMarketType(stockCode))

	// Get market config
	config := m.GetConfig()

	// Type assertion to access market configs
	type MarketGetter interface {
		GetMarkets() interface {
			GetChina() interface{ GetTradingSessions() []interface{} }
			GetUS() interface{ GetTradingSessions() []interface{} }
			GetHongKong() interface{ GetTradingSessions() []interface{} }
		}
	}

	var tradingSessions []interface{}
	if cfg, ok := config.(MarketGetter); ok {
		switch market {
		case "china":
			tradingSessions = cfg.GetMarkets().GetChina().GetTradingSessions()
		case "us":
			tradingSessions = cfg.GetMarkets().GetUS().GetTradingSessions()
		case "hongkong":
			tradingSessions = cfg.GetMarkets().GetHongKong().GetTradingSessions()
		default:
			logDebug("log.market.unknownType", stockCode, market)
			return false
		}
	}

	// 检查配置是否有效（向后兼容降级）
	if len(tradingSessions) == 0 {
		if market == "china" {
			return isMarketOpenHardcoded() // 保留老版本硬编码逻辑
		}
		return false
	}

	now := time.Now()
	return isMarketOpenForConfig(now, tradingSessions)
}

// isMarketOpenHardcoded 硬编码的A股交易时间判断（降级方案）
func isMarketOpenHardcoded() bool {
	now := time.Now()

	// Check if weekday
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	// Market hours: 09:30 - 11:30, 13:00 - 15:00 (China timezone)
	hour := now.Hour()
	minute := now.Minute()
	currentTime := hour*100 + minute

	// Morning session: 09:30 - 11:30
	if currentTime >= 930 && currentTime <= 1130 {
		return true
	}

	// Afternoon session: 13:00 - 15:00
	if currentTime >= 1300 && currentTime <= 1500 {
		return true
	}

	return false
}

// isMarketOpenForConfig checks if market is open based on trading sessions config
func isMarketOpenForConfig(now time.Time, sessions []interface{}) bool {
	// Check if weekday
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	// Check each trading session
	for _, session := range sessions {
		// Type assertion for session
		type SessionGetter interface {
			GetStartTime() string
			GetEndTime() string
		}

		if s, ok := session.(SessionGetter); ok {
			// Parse time strings (format: "09:30", "15:00")
			startParts := parseTimeString(s.GetStartTime())
			endParts := parseTimeString(s.GetEndTime())

			if len(startParts) == 2 && len(endParts) == 2 {
				startTime := time.Date(now.Year(), now.Month(), now.Day(), startParts[0], startParts[1], 0, 0, now.Location())
				endTime := time.Date(now.Year(), now.Month(), now.Day(), endParts[0], endParts[1], 0, 0, now.Location())

				if now.After(startTime) && now.Before(endTime) {
					return true
				}
			}
		}
	}

	return false
}

// parseTimeString parses "09:30" format to [hour, minute]
func parseTimeString(timeStr string) []int {
	var hour, minute int
	if _, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute); err == nil {
		return []int{hour, minute}
	}
	return []int{}
}

// GetMarketLocation returns the timezone location for a market type
func GetMarketLocation(marketType string) (*time.Location, error) {
	switch marketType {
	case "china":
		return time.LoadLocation("Asia/Shanghai")
	case "us":
		return time.LoadLocation("America/New_York")
	case "hongkong":
		return time.LoadLocation("Asia/Hong_Kong")
	default:
		return nil, fmt.Errorf("unknown market type: %s", marketType)
	}
}

// getCurrentDateForMarket returns the current date for a market (fallback for compatibility)
func getCurrentDateForMarket(market string, m ModelInterface) string {
	location, err := GetMarketLocation(market)
	if err != nil {
		return time.Now().Format("20060102")
	}
	return time.Now().In(location).Format("20060102")
}
