package intraday

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"stock-monitor/internal/api"
)

// validateStockCode checks that a stock code does not contain path traversal sequences.
func validateStockCode(code string) error {
	if strings.Contains(code, "..") || strings.Contains(code, "/") || strings.Contains(code, "\\") || strings.Contains(code, string(os.PathSeparator)) {
		return fmt.Errorf("invalid stock code: %q", code)
	}
	return nil
}

// File locks for thread-safe file operations
var intradayFileLocks sync.Map // map[string]*sync.Mutex

// SaveIntradayData writes IntradayData to JSON file with thread-safe locking
func SaveIntradayData(filePath string, data *IntradayData) error {
	lock := getFileLock(filePath)
	lock.Lock()
	defer lock.Unlock()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: temp → sync → close → rename (random suffix avoids multi-instance collision)
	dir := filepath.Dir(filePath)
	f, err := os.CreateTemp(dir, filepath.Base(filePath)+".tmp.*")
	if err != nil {
		return err
	}
	tempPath := f.Name()
	if _, err := f.Write(jsonData); err != nil {
		f.Close()
		os.Remove(tempPath)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tempPath)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	return os.Rename(tempPath, filePath)
}

// getFileLock returns a mutex for the given file path
func getFileLock(filePath string) *sync.Mutex {
	lock, _ := intradayFileLocks.LoadOrStore(filePath, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// ensureIntradayDirectory creates the directory structure for a stock if needed
func ensureIntradayDirectory(stockCode string) error {
	if err := validateStockCode(stockCode); err != nil {
		return err
	}
	dirPath := filepath.Join("data", "intraday", stockCode)
	return os.MkdirAll(dirPath, 0755)
}

// getMarketDirectory returns market subdirectory (CN/HK/US) based on stock code
func getMarketDirectory(code string) string {
	market := string(api.GetMarketType(code))
	switch market {
	case "china":
		return "CN"
	case "hongkong":
		return "HK"
	case "us":
		return "US"
	default:
		return "US"
	}
}

// GetIntradayFilePath returns file path with backward compatibility fallback
// Priority: new market-based structure (data/intraday/CN/SH600058/20251211.json)
//
//	→ old flat structure (data/intraday/SH600058/20251211.json)
func GetIntradayFilePath(stockCode, date string) string {
	if err := validateStockCode(stockCode); err != nil {
		return ""
	}
	// Try new market-based structure first
	marketDir := getMarketDirectory(stockCode)
	newPath := filepath.Join("data", "intraday", marketDir, stockCode, date+".json")
	if fileExists(newPath) {
		return newPath
	}

	// Fallback to old flat structure for backward compatibility
	return filepath.Join("data", "intraday", stockCode, date+".json")
}

// ensureIntradayDirectoryWithMarket creates market-based directory structure
// New implementation that organizes stocks by market (CN/HK/US)
func ensureIntradayDirectoryWithMarket(stockCode string) error {
	if err := validateStockCode(stockCode); err != nil {
		return err
	}
	marketDir := getMarketDirectory(stockCode)
	dirPath := filepath.Join("data", "intraday", marketDir, stockCode)
	return os.MkdirAll(dirPath, 0755)
}

// fileExists checks if a file path exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDataComplete 检查本地分时数据是否完整
// stockCode: 股票代码
// date: 目标日期 (YYYYMMDD)
// marketType: 市场类型
// isLiveMode: 是否为实时模式（实时模式使用较低的完整性阈值）
// 返回: (是否完整, 错误)
func IsDataComplete(stockCode string, date string, marketType string, isLiveMode bool) (bool, error) {
	// 使用 GetIntradayFilePath 兼容新旧目录结构
	filePath := GetIntradayFilePath(stockCode, date)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, nil // 文件不存在 -> 不完整（不是错误）
	}

	// 加载并解析文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	var intradayData IntradayData
	if err := json.Unmarshal(data, &intradayData); err != nil {
		return false, err
	}

	// 统计数据点数量
	actualDatapoints := len(intradayData.Datapoints)
	expectedDatapoints := GetExpectedDatapoints(marketType, isLiveMode)

	// 定义完整性标准
	minDatapoints := 20           // 绝对最小数据点（防止误判）
	completenessThreshold := 90.0 // 完整性阈值（百分比）

	// 实时模式使用较低的阈值
	if isLiveMode {
		completenessThreshold = 50.0 // 实时模式只需 50%
	}

	// 计算实际完整度
	completenessPercent := (float64(actualDatapoints) / float64(expectedDatapoints)) * 100.0

	// 检查是否满足标准
	if actualDatapoints < minDatapoints {
		return false, nil
	}

	if completenessPercent < completenessThreshold {
		return false, nil
	}

	return true, nil
}

// GetExpectedDatapoints 计算完整交易日的预期数据点数量
// marketType: 市场类型
// isLiveMode: 是否为实时模式（如果是，返回较宽松的预期）
// 返回: 预期的数据点数量
//
// A股: 09:30-11:30 (120分钟) + 13:00-15:00 (120分钟) = 240数据点
// 美股: 09:30-16:00 (390分钟) = 390数据点
// 港股: 09:30-12:00 (150分钟) + 13:00-16:00 (180分钟) = 330数据点
func GetExpectedDatapoints(marketType string, isLiveMode bool) int {
	switch marketType {
	case "china":
		// 09:15-09:25 开盘竞价 (10分) + 09:30-11:30 上午 (120分) + 13:00-14:57 下午 (117分) + 14:57-15:00 收盘竞价 (3分) = 250
		// 注：收盘时间 15:00 本身含在下午交易，故总计 240 + 10 = 250
		return 250

	case "us":
		return 390 // 6.5小时 × 60分钟

	case "hongkong":
		return 340 // 5.5小时 × 60分钟 + CAS 10分钟

	default:
		return 240 // 默认使用 A股 标准
	}
}
