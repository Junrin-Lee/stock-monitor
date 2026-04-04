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
// 返回：(拼接后的数据点切片, 实际加载的日期列表, 错误)
// 实际加载天数可能少于 days（本地数据不足时）
func LoadMultiDay(code string, days int) ([]IntradayDataPoint, []string, error) {
	dates := findRecentIntradayDates(code, days)
	if len(dates) == 0 {
		return nil, nil, fmt.Errorf("no intraday data found for %s", code)
	}

	var allPoints []IntradayDataPoint
	var loadedDates []string

	for _, date := range dates {
		data, err := LoadSingleDay(code, date)
		if err != nil {
			continue
		}
		allPoints = append(allPoints, data.Datapoints...)
		loadedDates = append(loadedDates, date)
	}

	if len(allPoints) == 0 {
		return nil, nil, fmt.Errorf("no datapoints loaded for %s", code)
	}

	return allPoints, loadedDates, nil
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
