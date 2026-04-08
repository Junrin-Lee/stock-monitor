package intraday

import (
	"fmt"
	"time"

	"stock-monitor/internal/api"
	"stock-monitor/internal/market"
	"stock-monitor/internal/types"
)

// GetTradingState 判断市场当前的交易状态
// now: 当前时间（已转换为市场时区）
// marketType: 市场类型
// 返回：TradingState 表示市场状态
func GetTradingState(now time.Time, marketType string) types.TradingState {
	weekday := now.Weekday()

	// 检查周末（补班日除外）
	date := now.Format("20060102")
	year := now.Year()
	if weekday == time.Saturday || weekday == time.Sunday {
		if marketType == "china" && market.IsCompDay(date, year) {
			// 补班日按正常交易日处理，继续往下检查交易时段
		} else {
			return types.TradingStateWeekend
		}
	}

	// 节假日检测（仅 A 股有日历数据）
	if marketType == "china" && market.IsHoliday(date, year) {
		return types.TradingStateHoliday
	}

	// 获取市场特定的交易时段
	var morningStart, morningEnd, afternoonStart, afternoonEnd time.Time

	switch marketType {
	case "china": // A股: 09:30-11:30, 13:00-15:00 CST（含集合竞价 09:15-09:25 开盘，14:57-15:00 收盘）
		auctionStart := time.Date(now.Year(), now.Month(), now.Day(), 9, 15, 0, 0, now.Location())
		auctionEnd := time.Date(now.Year(), now.Month(), now.Day(), 9, 25, 0, 0, now.Location()) // 09:25后为静默期
		morningStart = time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
		morningEnd = time.Date(now.Year(), now.Month(), now.Day(), 11, 30, 0, 0, now.Location())
		afternoonStart = time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, now.Location())
		afternoonEnd = time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location())
		// 竞价时段判断：09:15-09:25（含整点，09:25-09:30为静默期归入盘前）
		if !now.Before(auctionStart) && now.Before(auctionEnd) {
			return types.TradingStateAuction
		}

	case "us": // 美股: 09:30-16:00 EST/EDT (无午休)
		morningStart = time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
		afternoonEnd = time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
		morningEnd = afternoonEnd     // 无午休
		afternoonStart = morningStart // 无午休

	case "hongkong": // 港股: 09:30-12:00, 13:00-16:00 HKT（含集合竞价 09:00-09:30, 收盘竞价 16:00-16:10）
		auctionStart := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
		morningStart = time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
		morningEnd = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
		afternoonStart = time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, now.Location())
		afternoonEnd = time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
		casEnd := time.Date(now.Year(), now.Month(), now.Day(), 16, 10, 0, 0, now.Location())
		// 开盘竞价时段（09:00-09:30）
		if !now.Before(auctionStart) && now.Before(morningStart) {
			return types.TradingStateAuction
		}
		// 收盘竞价时段 CAS（16:00-16:10）
		if !now.Before(afternoonEnd) && now.Before(casEnd) {
			return types.TradingStateAuction
		}

	default:
		return types.TradingStatePostMarket
	}

	// 判断当前状态
	if now.Before(morningStart) {
		return types.TradingStatePreMarket
	} else if (!now.Before(morningStart) && now.Before(morningEnd)) ||
		(!now.Before(afternoonStart) && now.Before(afternoonEnd)) {
		return types.TradingStateLive
	} else if !now.Before(morningEnd) && now.Before(afternoonStart) {
		return types.TradingStateLunchBreak // 午休时段
	} else if !now.Before(afternoonEnd) {
		return types.TradingStatePostMarket
	} else {
		return types.TradingStatePostMarket // 安全兜底
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

	// 最多回溯 15 天（覆盖春节、国庆等长假+调休）
	for i := 1; i <= 15; i++ {
		prevDate := date.AddDate(0, 0, -i)
		weekday := prevDate.Weekday()
		dateStr := prevDate.Format("20060102")

		if weekday == time.Saturday || weekday == time.Sunday {
			// A 股补班日是交易日
			if marketType == "china" && market.IsCompDay(dateStr, prevDate.Year()) {
				return dateStr
			}
			continue
		}

		// A 股节假日跳过
		if marketType == "china" && market.IsHoliday(dateStr, prevDate.Year()) {
			continue
		}

		return dateStr
	}

	// 降级：如果 15 天内找不到，直接返回前一天
	return date.AddDate(0, 0, -1).Format("20060102")
}

// FindNextTradingDay 查找下一个交易日（支持多市场时区+节假日，与 FindPreviousTradingDay 对称）
// stockCode: 股票代码（用于检测市场类型）
// currentDate: 当前日期 (YYYYMMDD)
// maxDate: 不超过此日期（通常是市场时区的今天）
// 返回: (下一个交易日 YYYYMMDD, error)
func FindNextTradingDay(stockCode string, currentDate string, maxDate time.Time) (string, error) {
	marketType := string(api.GetMarketType(stockCode))
	location, err := GetMarketLocation(marketType)
	if err != nil {
		location = time.Local
	}

	date, err := time.ParseInLocation("20060102", currentDate, location)
	if err != nil {
		return "", err
	}

	// 最多往后查找 15 天（覆盖长假）
	for i := 1; i <= 15; i++ {
		nextDate := date.AddDate(0, 0, i)

		// 不能超过最大日期
		if nextDate.After(maxDate) {
			return "", fmt.Errorf("已到达最新日期")
		}

		weekday := nextDate.Weekday()
		dateStr := nextDate.Format("20060102")

		if weekday == time.Saturday || weekday == time.Sunday {
			// A 股补班日是交易日
			if marketType == "china" && market.IsCompDay(dateStr, nextDate.Year()) {
				return dateStr, nil
			}
			continue
		}

		// A 股节假日跳过
		if marketType == "china" && market.IsHoliday(dateStr, nextDate.Year()) {
			continue
		}

		return dateStr, nil
	}

	return "", fmt.Errorf("无法找到下一个交易日")
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
		// 盘前（含 A 股静默期 09:25-09:30）-> 获取上一个交易日的数据
		prevDate := FindPreviousTradingDay(stockCode, now.Format("20060102"), m)
		return prevDate, types.CollectionModeHistorical, nil

	case types.TradingStateAuction:
		// 集合竞价 -> 采集当日实时数据（竞价阶段已产生当日行情）
		return now.Format("20060102"), types.CollectionModeLive, nil

	case types.TradingStateLive:
		// 交易中 -> 获取当日实时数据
		return now.Format("20060102"), types.CollectionModeLive, nil

	case types.TradingStateLunchBreak:
		// 午休期间按 Live 模式处理当日数据
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

// IsMarketOpen 检查当前是否在交易时间内（支持多市场时区+节假日）
// 基于 GetTradingState 统一判断，确保时区转换和节假日检测的一致性
// 注意：午休时段不算"开市"，worker 会暂停采集避免连续 skip 触发 auto-stop
func IsMarketOpen(stockCode string, _ ModelInterface) bool {
	marketType := string(api.GetMarketType(stockCode))
	location, err := GetMarketLocation(marketType)
	if err != nil {
		return false
	}
	now := time.Now().In(location)
	state := GetTradingState(now, marketType)
	return state == types.TradingStateLive ||
		state == types.TradingStateAuction
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
