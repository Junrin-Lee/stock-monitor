package intraday

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"stock-monitor/internal/api"
)

// LoadSingleDay 从磁盘加载特定股票和日期的分时数据
// 这是 handlers_chart.go:loadIntradayDataForDate() 的独立公共版本，无需 Model 实例
func LoadSingleDay(code, date string) (*IntradayData, error) {
	filePath := GetIntradayFilePath(code, date)

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	var data IntradayData
	if err := json.Unmarshal(fileData, &data); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// 向后兼容：如果 Market 为空，自动识别市场
	if data.Market == "" {
		data.Market = string(api.GetMarketType(code))
	}

	return &data, nil
}

// LoadLatestPrices 加载最新交易日的价格序列（Sparkline 专用轻量级接口）
// 返回 []float64 价格数组；无数据或出错时返回 nil
func LoadLatestPrices(code string) []float64 {
	date := findLatestIntradayDate(code)
	if date == "" {
		return nil
	}

	data, err := LoadSingleDay(code, date)
	if err != nil || len(data.Datapoints) == 0 {
		return nil
	}

	prices := make([]float64, 0, len(data.Datapoints))
	for _, dp := range data.Datapoints {
		if dp.Price > 0 {
			prices = append(prices, dp.Price)
		}
	}
	return prices
}

// LoadMultiDay 加载最近 N 个交易日的分时数据并按时间顺序拼接
// 返回：(拼接后的数据点切片, 实际加载的日期列表, 每日数据点数, 错误)
// 实际加载天数可能少于 days（本地数据不足时）
func LoadMultiDay(code string, days int) ([]IntradayDataPoint, []string, []int, error) {
	dates := findRecentIntradayDates(code, days)
	if len(dates) == 0 {
		return nil, nil, nil, fmt.Errorf("no intraday data found for %s", code)
	}

	var allPoints []IntradayDataPoint
	var loadedDates []string
	var pointsPerDay []int

	// 倒序遍历：dates 是降序（最新在前），倒序遍历使结果为升序（旧→新）
	for i := len(dates) - 1; i >= 0; i-- {
		data, err := LoadSingleDay(code, dates[i])
		if err != nil {
			continue
		}
		allPoints = append(allPoints, data.Datapoints...)
		loadedDates = append(loadedDates, dates[i])
		pointsPerDay = append(pointsPerDay, len(data.Datapoints))
	}

	if len(allPoints) == 0 {
		return nil, nil, nil, fmt.Errorf("no datapoints loaded for %s", code)
	}

	return allPoints, loadedDates, pointsPerDay, nil
}

// Downsample 将 prices 降采样到 targetWidth 个点（等距采样）
// 供 loader 外部直接使用（与 sparkline 包中的 Downsample 同逻辑）
func Downsample(prices []float64, targetWidth int) []float64 {
	if len(prices) <= targetWidth {
		return prices
	}
	result := make([]float64, targetWidth)
	step := float64(len(prices)-1) / float64(targetWidth-1)
	for i := 0; i < targetWidth; i++ {
		idx := int(float64(i) * step)
		if idx >= len(prices) {
			idx = len(prices) - 1
		}
		result[i] = prices[idx]
	}
	return result
}

// LoadDailyClosingPrices 从本地分时数据文件中提取每日收盘价
// 返回升序排列（旧→新）的 DailyClosePoint 切片
// maxDays: 最多加载的交易日数（受限于本地已有数据量）
func LoadDailyClosingPrices(code string, maxDays int) ([]DailyClosePoint, error) {
	dates := findRecentIntradayDates(code, maxDays) // 降序（最新在前）
	if len(dates) == 0 {
		return nil, fmt.Errorf("no intraday data found for %s", code)
	}

	// 反转为升序（旧→新）
	result := make([]DailyClosePoint, 0, len(dates))
	for i := len(dates) - 1; i >= 0; i-- {
		data, err := LoadSingleDay(code, dates[i])
		if err != nil || len(data.Datapoints) == 0 {
			continue
		}
		lastPrice := data.Datapoints[len(data.Datapoints)-1].Price
		if lastPrice > 0 {
			result = append(result, DailyClosePoint{Date: dates[i], Close: lastPrice})
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid closing prices for %s", code)
	}
	return result, nil
}

// AggregateWeekly 将每日收盘价按每5个交易日分组聚合为周线数据
// 每组取最后一天的收盘价作为该周期的收盘价
func AggregateWeekly(dailyCloses []DailyClosePoint) []AggregatedPoint {
	if len(dailyCloses) == 0 {
		return nil
	}
	var result []AggregatedPoint
	for i := 0; i < len(dailyCloses); i += 5 {
		end := i + 5
		if end > len(dailyCloses) {
			end = len(dailyCloses)
		}
		last := dailyCloses[end-1]
		label := last.Date[4:6] + "/" + last.Date[6:8] // MM/DD
		result = append(result, AggregatedPoint{Label: label, Close: last.Close, Date: last.Date})
	}
	return result
}

// AggregateMonthly 将每日收盘价按自然月分组聚合为月线数据
// 每月取最后一个交易日的收盘价
func AggregateMonthly(dailyCloses []DailyClosePoint) []AggregatedPoint {
	if len(dailyCloses) == 0 {
		return nil
	}
	var result []AggregatedPoint
	currentMonth := dailyCloses[0].Date[:6] // YYYYMM
	var lastInMonth DailyClosePoint

	for _, dc := range dailyCloses {
		month := dc.Date[:6]
		if month != currentMonth {
			// 新月份开始，保存上个月的数据
			label := currentMonth[:4] + "/" + currentMonth[4:6] // YYYY/MM
			result = append(result, AggregatedPoint{Label: label, Close: lastInMonth.Close, Date: lastInMonth.Date})
			currentMonth = month
		}
		lastInMonth = dc
	}
	// 保存最后一个月
	label := currentMonth[:4] + "/" + currentMonth[4:6]
	result = append(result, AggregatedPoint{Label: label, Close: lastInMonth.Close, Date: lastInMonth.Date})

	return result
}

// AggregateQuarterly 将每日收盘价按自然季度分组聚合为季线数据
// Q1=1-3月, Q2=4-6月, Q3=7-9月, Q4=10-12月
func AggregateQuarterly(dailyCloses []DailyClosePoint) []AggregatedPoint {
	if len(dailyCloses) == 0 {
		return nil
	}
	var result []AggregatedPoint
	currentQKey := quarterKey(dailyCloses[0].Date)
	var lastInQuarter DailyClosePoint

	for _, dc := range dailyCloses {
		qk := quarterKey(dc.Date)
		if qk != currentQKey {
			result = append(result, AggregatedPoint{Label: currentQKey, Close: lastInQuarter.Close, Date: lastInQuarter.Date})
			currentQKey = qk
		}
		lastInQuarter = dc
	}
	result = append(result, AggregatedPoint{Label: currentQKey, Close: lastInQuarter.Close, Date: lastInQuarter.Date})

	return result
}

// AggregateYearly 将每日收盘价按自然年分组聚合为年线数据
func AggregateYearly(dailyCloses []DailyClosePoint) []AggregatedPoint {
	if len(dailyCloses) == 0 {
		return nil
	}
	var result []AggregatedPoint
	currentYear := dailyCloses[0].Date[:4]
	var lastInYear DailyClosePoint

	for _, dc := range dailyCloses {
		year := dc.Date[:4]
		if year != currentYear {
			result = append(result, AggregatedPoint{Label: currentYear, Close: lastInYear.Close, Date: lastInYear.Date})
			currentYear = year
		}
		lastInYear = dc
	}
	result = append(result, AggregatedPoint{Label: currentYear, Close: lastInYear.Close, Date: lastInYear.Date})

	return result
}

// quarterKey 返回日期对应的季度标识，格式 "YY Q1"
func quarterKey(date string) string {
	if len(date) < 6 {
		return "??"
	}
	year := date[2:4] // 取后两位年份
	monthStr := date[4:6]
	month := 0
	for _, c := range monthStr {
		month = month*10 + int(c-'0')
	}
	q := (month-1)/3 + 1
	return fmt.Sprintf("%sQ%d", year, q)
}

// findLatestIntradayDate 查找指定股票最新的分时数据日期（YYYYMMDD 格式）
func findLatestIntradayDate(code string) string {
	dates := findRecentIntradayDates(code, 1)
	if len(dates) == 0 {
		return ""
	}
	return dates[0]
}

// findRecentIntradayDates 查找指定股票最近 n 个有数据的交易日期，降序排列（最新在前）
func findRecentIntradayDates(code string, n int) []string {
	marketDir := getMarketDirectory(code)
	dirPath := filepath.Join("data", "intraday", marketDir, code)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		// 尝试旧目录结构（向后兼容）
		dirPath = filepath.Join("data", "intraday", code)
		entries, err = os.ReadDir(dirPath)
		if err != nil {
			return nil
		}
	}

	var dates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".tmp") {
			date := strings.TrimSuffix(name, ".json")
			if len(date) == 8 { // YYYYMMDD
				dates = append(dates, date)
			}
		}
	}

	// 降序排列（最新日期在前）
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	if len(dates) > n {
		return dates[:n]
	}
	return dates
}
