package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"stock-monitor/internal/consts"
	"stock-monitor/internal/types"
)

// atomicWriteFile 原子写文件：先写临时文件再 rename，防止中途崩溃导致数据截断
func atomicWriteFile(filePath string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, filepath.Base(filePath)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp to target: %w", err)
	}
	return nil
}

// backupCorruptedFile saves a corrupted file with a .corrupt suffix to prevent data loss.
func backupCorruptedFile(filePath string, data []byte) {
	backupPath := filePath + ".corrupt"
	_ = os.WriteFile(backupPath, data, 0644)
	fmt.Fprintf(os.Stderr, "Warning: corrupted file backed up to %s\n", backupPath)
}

// ============================================================================
// Portfolio 持仓数据持久化
// ============================================================================

// SavePortfolio 保存持仓数据到文件（原子写）
func SavePortfolio(portfolio types.Portfolio) error {
	data, err := json.MarshalIndent(portfolio, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(consts.DataFile, data, 0644)
}

// LoadPortfolio 从文件加载持仓数据
func LoadPortfolio() types.Portfolio {
	data, err := os.ReadFile(consts.DataFile)
	if err != nil {
		return types.Portfolio{Stocks: []types.Stock{}}
	}

	var portfolio types.Portfolio
	err = json.Unmarshal(data, &portfolio)
	if err != nil {
		backupCorruptedFile(consts.DataFile, data)
		return types.Portfolio{Stocks: []types.Stock{}}
	}
	return portfolio
}

// ============================================================================
// Watchlist 自选股数据持久化
// ============================================================================

// WatchlistStockLegacy 旧版自选股数据结构（用于迁移兼容）
type WatchlistStockLegacy struct {
	Code   string           `json:"code"`
	Name   string           `json:"name"`
	Tag    string           `json:"tag,omitempty"`    // 旧格式的单个标签
	Tags   []string         `json:"tags,omitempty"`   // 新格式的多个标签
	Market types.MarketType `json:"market,omitempty"` // 市场类型
}

// WatchlistLegacy 旧版自选股列表（用于迁移兼容）
type WatchlistLegacy struct {
	Stocks []WatchlistStockLegacy `json:"stocks"`
}

// MarketDetector 市场类型检测函数类型
type MarketDetector func(code string) types.MarketType

// MarketTagChecker 市场标签检查函数类型
type MarketTagChecker func(tag string) bool

// LoadWatchlist 加载自选股票列表（支持旧格式迁移）
// marketDetector: 用于检测股票市场类型的函数
// marketTagChecker: 用于检查是否为市场标签的函数
func LoadWatchlist(marketDetector MarketDetector, marketTagChecker MarketTagChecker) types.Watchlist {
	data, err := os.ReadFile(consts.WatchlistFile)
	if err != nil {
		return types.Watchlist{Stocks: []types.WatchlistStock{}}
	}

	// 先尝试用兼容性结构体加载数据
	var legacyWatchlist WatchlistLegacy
	err = json.Unmarshal(data, &legacyWatchlist)
	if err != nil {
		backupCorruptedFile(consts.WatchlistFile, data)
		return types.Watchlist{Stocks: []types.WatchlistStock{}}
	}

	// 转换为新格式
	var watchlist types.Watchlist
	for _, legacyStock := range legacyWatchlist.Stocks {
		newStock := types.WatchlistStock{
			Code: legacyStock.Code,
			Name: legacyStock.Name,
		}

		// 处理市场字段的兼容性
		if legacyStock.Market == "" {
			// 自动识别市场类型
			newStock.Market = marketDetector(legacyStock.Code)
		} else {
			newStock.Market = legacyStock.Market
		}

		// 处理标签字段的兼容性并清理市场标签
		var userTags []string
		if len(legacyStock.Tags) > 0 {
			// 新格式：过滤掉市场标签，只保留用户自定义标签
			for _, tag := range legacyStock.Tags {
				if tag != "" && tag != "-" && !marketTagChecker(tag) {
					userTags = append(userTags, tag)
				}
			}
			newStock.Tags = userTags
		} else if legacyStock.Tag != "" {
			// 旧格式：将单个 Tag 转换为 Tags 数组（如果不是市场标签）
			if !marketTagChecker(legacyStock.Tag) && legacyStock.Tag != "-" {
				newStock.Tags = []string{legacyStock.Tag}
			} else {
				newStock.Tags = []string{}
			}
		} else {
			// 没有标签：使用空数组
			newStock.Tags = []string{}
		}

		watchlist.Stocks = append(watchlist.Stocks, newStock)
	}

	return watchlist
}

// SaveWatchlist 保存自选股票列表（原子写）
func SaveWatchlist(watchlist types.Watchlist) error {
	data, err := json.MarshalIndent(watchlist, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(consts.WatchlistFile, data, 0644)
}

// ============================================================================
// Config 配置文件持久化
// ============================================================================

// DefaultMarketsConfig 获取默认的市场配置
func DefaultMarketsConfig() types.MarketsConfig {
	return types.MarketsConfig{
		China: types.MarketConfig{
			Timezone: "Asia/Shanghai",
			TradingSessions: []types.TradingSession{
				{StartTime: "09:30", EndTime: "11:30"},
				{StartTime: "13:00", EndTime: "15:00"},
			},
			Weekdays: []int{1, 2, 3, 4, 5},
		},
		US: types.MarketConfig{
			Timezone: "America/New_York",
			TradingSessions: []types.TradingSession{
				{StartTime: "09:30", EndTime: "16:00"},
			},
			Weekdays: []int{1, 2, 3, 4, 5},
		},
		HongKong: types.MarketConfig{
			Timezone: "Asia/Hong_Kong",
			TradingSessions: []types.TradingSession{
				{StartTime: "09:30", EndTime: "12:00"},
				{StartTime: "13:00", EndTime: "16:00"},
			},
			Weekdays: []int{1, 2, 3, 4, 5},
		},
	}
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() types.Config {
	return types.Config{
		System: types.SystemConfig{
			Language:      "en",        // 默认英文
			AutoStart:     true,        // 有数据时自动进入监控模式
			StartupModule: "portfolio", // 默认启动持股模块
			LogLevel:      "info",      // 默认日志级别
		},
		Display: types.DisplayConfig{
			ColorScheme:        "professional", // 专业配色方案
			DecimalPlaces:      3,              // 3位小数
			TableStyle:         "light",        // 轻量表格样式
			MaxLines:           10,             // 默认每页显示10行
			PortfolioHighlight: "yellow",       // 默认黄色
			// 持股列表默认显示所有列（按当前顺序）
			PortfolioColumns: []string{
				"cursor", "code", "name", "prev_close", "open", "high",
				"low", "price", "cost", "quantity", "today_change",
				"position_profit", "profit_rate", "market_value",
			},
			// 自选列表默认显示所有列（按当前顺序）
			WatchlistColumns: []string{
				"cursor", "tag", "code", "name", "price", "prev_close",
				"open", "high", "low", "today_change", "turnover", "volume",
			},
		},
		Update: types.UpdateConfig{
			RefreshInterval: 5,    // 5秒刷新间隔
			AutoUpdate:      true, // 自动更新开启
		},
		Markets: DefaultMarketsConfig(), // 市场配置
		IntradayCollection: types.IntradayCollectionConfig{
			EnableAutoStop:        true, // 启用自动停止
			CompletenessThreshold: 90.0, // 90% 完整性阈值
			MaxConsecutiveErrors:  5,    // 最大连续错误5次
			MinDatapoints:         20,   // 最小数据点20个
		},
	}
}

// LoadConfig 加载配置文件
func LoadConfig() types.Config {
	data, err := os.ReadFile(consts.ConfigFile)
	if err != nil {
		// 如果配置文件不存在，创建默认配置文件
		config := GetDefaultConfig()
		if saveErr := SaveConfig(config); saveErr != nil {
			// 首次运行配置创建失败仅记日志，不阻塞启动
			fmt.Fprintf(os.Stderr, "Warning: failed to save default config: %v\n", saveErr)
		}
		return config
	}

	var config types.Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		backupCorruptedFile(consts.ConfigFile, data)
		return GetDefaultConfig()
	}

	// 验证刷新间隔
	if config.Update.RefreshInterval < 1 {
		config.Update.RefreshInterval = 5
	}

	// 验证配置的合理性
	if config.Display.MaxLines <= 0 || config.Display.MaxLines > 50 {
		config.Display.MaxLines = 10 // 默认值
	}

	// 验证并设置高亮颜色的默认值
	if config.Display.PortfolioHighlight == "" {
		config.Display.PortfolioHighlight = "yellow" // 默认黄色背景
	}

	// 如果 Markets 为空，填充默认值（向后兼容）
	if config.Markets.China.Timezone == "" {
		config.Markets = DefaultMarketsConfig()
	}

	// 向后兼容：如果列配置为空，使用默认值
	if len(config.Display.PortfolioColumns) == 0 {
		config.Display.PortfolioColumns = GetDefaultConfig().Display.PortfolioColumns
	}
	if len(config.Display.WatchlistColumns) == 0 {
		config.Display.WatchlistColumns = GetDefaultConfig().Display.WatchlistColumns
	}

	// 验证列配置
	config.Display.PortfolioColumns = ValidatePortfolioColumns(config.Display.PortfolioColumns)
	config.Display.WatchlistColumns = ValidateWatchlistColumns(config.Display.WatchlistColumns)

	return config
}

// SaveConfig 保存配置文件（原子写）
func SaveConfig(config types.Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return atomicWriteFile(consts.ConfigFile, data, 0644)
}

// ValidatePortfolioColumns 验证Portfolio列配置
func ValidatePortfolioColumns(configured []string) []string {
	required := []string{"cursor", "code", "name", "price"}
	valid := map[string]bool{
		"cursor": true, "code": true, "name": true, "prev_close": true,
		"open": true, "high": true, "low": true, "price": true,
		"cost": true, "quantity": true, "today_change": true,
		"position_profit": true, "profit_rate": true, "market_value": true,
		"trend": true, "pre_market": true, "post_market": true,
	}

	return smartMergeRequiredColumns(configured, required, valid)
}

// ValidateWatchlistColumns 验证Watchlist列配置
func ValidateWatchlistColumns(configured []string) []string {
	required := []string{"cursor", "tag", "code", "name", "price"}
	valid := map[string]bool{
		"cursor": true, "tag": true, "code": true, "name": true,
		"price": true, "prev_close": true, "open": true, "high": true,
		"low": true, "today_change": true, "turnover": true, "volume": true,
		"trend": true, "pre_market": true, "post_market": true,
	}

	return smartMergeRequiredColumns(configured, required, valid)
}

// smartMergeRequiredColumns 智能合并必须列
// 算法：在保留用户配置顺序的同时，智能插入缺失的必须列
func smartMergeRequiredColumns(userConfig []string, required []string, valid map[string]bool) []string {
	result := []string{}
	inserted := make(map[string]bool)

	// 第一步：添加用户配置的有效列
	for _, col := range userConfig {
		if valid[col] {
			result = append(result, col)
			inserted[col] = true
		}
	}

	// 第二步：收集缺失的必须列
	missingRequired := []string{}
	for _, req := range required {
		if !inserted[req] {
			missingRequired = append(missingRequired, req)
		}
	}

	// 第三步：智能插入缺失的必须列
	// 策略：在第一个用户配置列之后插入，如果用户配置为空则放在最前面
	if len(missingRequired) > 0 {
		insertPosition := 0
		if len(result) > 0 {
			insertPosition = 1 // 在第一列之后插入
		}
		result = insertAt(result, insertPosition, missingRequired...)
	}

	// 如果结果为空（用户配置为空且无必须列），返回必须列
	if len(result) == 0 {
		return required
	}

	return result
}

// insertAt 在指定位置插入元素
func insertAt(slice []string, index int, values ...string) []string {
	// 确保index在有效范围内
	if index < 0 {
		index = 0
	}
	if index > len(slice) {
		index = len(slice)
	}

	// 创建新切片
	result := make([]string, 0, len(slice)+len(values))
	result = append(result, slice[:index]...)
	result = append(result, values...)
	result = append(result, slice[index:]...)
	return result
}

// Contains 检查字符串切片是否包含指定字符串
func Contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// ============================================================================
// Alert 告警数据持久化
// ============================================================================

// MigrateAlertFrequency 迁移告警频率数据（向后兼容）
// 确保旧数据的 Frequency 字段有有效值
func MigrateAlertFrequency(alerts []types.Alert) []types.Alert {
	for i, alert := range alerts {
		// 如果 Frequency 未设置，默认为一次性告警
		if alert.Frequency == "" {
			alerts[i].Frequency = types.TriggerOnce
		}
		// 确保 FrequencyDays 有有效默认值
		if alert.Frequency == types.TriggerEveryNDays && alert.FrequencyDays <= 0 {
			alerts[i].FrequencyDays = 1
		}
	}
	return alerts
}

// LoadAlertData 从文件加载告警数据
func LoadAlertData() types.AlertData {
	data, err := os.ReadFile(consts.AlertFile)
	if err != nil {
		return types.AlertData{
			Alerts:     []types.Alert{},
			LastCheck:  "",
			AlertCount: 0,
		}
	}

	var alertData types.AlertData
	err = json.Unmarshal(data, &alertData)
	if err != nil {
		backupCorruptedFile(consts.AlertFile, data)
		return types.AlertData{
			Alerts:     []types.Alert{},
			LastCheck:  "",
			AlertCount: 0,
		}
	}

	// 迁移旧数据到新的频率格式
	alertData.Alerts = MigrateAlertFrequency(alertData.Alerts)

	return alertData
}

// SaveAlertData 保存告警数据到文件（原子写）
func SaveAlertData(alertData types.AlertData) error {
	alertData.AlertCount = len(alertData.Alerts)
	data, err := json.MarshalIndent(alertData, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(consts.AlertFile, data, 0644)
}
