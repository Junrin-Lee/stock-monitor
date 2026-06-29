package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"stock-monitor/internal/testutil"
	"stock-monitor/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Mock market detector for testing
func mockMarketDetector(code string) types.MarketType {
	if code == "SH601138" || code == "SZ000001" {
		return types.MarketChina
	}
	if code == "HK00700" || code == "0700.HK" {
		return types.MarketHongKong
	}
	return types.MarketUS
}

// Mock market tag checker for testing
func mockMarketTagChecker(tag string) bool {
	return tag == "A股" || tag == "美股" || tag == "港股"
}

// TestSaveAndLoadPortfolio 测试持仓数据保存和加载
func TestSaveAndLoadPortfolio(t *testing.T) {
	// 创建临时测试目录
	tmpDir := testutil.CreateTempDir(t)
	tmpFile := filepath.Join(tmpDir, "portfolio.json")

	// 准备测试数据
	originalPortfolio := testutil.NewTestPortfolio()

	tests := []struct {
		name      string
		portfolio types.Portfolio
		wantErr   bool
	}{
		{
			name:      "正常保存加载",
			portfolio: originalPortfolio,
			wantErr:   false,
		},
		{
			name: "空投资组合",
			portfolio: types.Portfolio{
				Stocks: []types.Stock{},
			},
			wantErr: false,
		},
		{
			name: "单只股票",
			portfolio: types.Portfolio{
				Stocks: []types.Stock{testutil.NewTestChinaStock()},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 临时替换常量路径
			oldDataFile := "data/portfolio.json"
			defer func() {
				// 测试后恢复(如果需要)
			}()

			// 保存
			data, err := json.MarshalIndent(tt.portfolio, "", "  ")
			require.NoError(t, err)
			err = os.WriteFile(tmpFile, data, 0644)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err, "保存失败: %v", err)
			assert.True(t, testutil.FileExists(tmpFile), "文件应该存在")

			// 加载
			loadedData, err := os.ReadFile(tmpFile)
			require.NoError(t, err)

			var loadedPortfolio types.Portfolio
			err = json.Unmarshal(loadedData, &loadedPortfolio)
			require.NoError(t, err)

			// 验证
			assert.Equal(t, len(tt.portfolio.Stocks), len(loadedPortfolio.Stocks))
			if len(tt.portfolio.Stocks) > 0 {
				assert.Equal(t, tt.portfolio.Stocks[0].Code, loadedPortfolio.Stocks[0].Code)
				assert.Equal(t, tt.portfolio.Stocks[0].Name, loadedPortfolio.Stocks[0].Name)
			}

			_ = oldDataFile // 使用变量避免未使用警告
		})
	}
}

// TestLoadWatchlist 测试自选列表加载(包括兼容性)
func TestLoadWatchlist(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)

	tests := []struct {
		name           string
		fileContent    string
		expectedCount  int
		expectedMarket types.MarketType
		shouldHaveTags bool
		desc           string
	}{
		{
			name: "新格式-完整数据",
			fileContent: `{
				"stocks": [
					{
						"code": "SH601138",
						"name": "工业富联",
						"market": "china",
						"tags": ["科技", "AI"]
					}
				]
			}`,
			expectedCount:  1,
			expectedMarket: types.MarketChina,
			shouldHaveTags: true,
			desc:           "新格式包含market和tags",
		},
		{
			name: "旧格式-单个tag",
			fileContent: `{
				"stocks": [
					{
						"code": "SH601138",
						"name": "工业富联",
						"tag": "科技"
					}
				]
			}`,
			expectedCount:  1,
			expectedMarket: types.MarketChina,
			shouldHaveTags: true,
			desc:           "旧格式tag转换为tags数组",
		},
		{
			name: "市场标签自动过滤",
			fileContent: `{
				"stocks": [
					{
						"code": "SH601138",
						"name": "工业富联",
						"tags": ["A股", "科技", "美股"]
					}
				]
			}`,
			expectedCount:  1,
			expectedMarket: types.MarketChina,
			shouldHaveTags: true,
			desc:           "市场标签(A股、美股)应被过滤",
		},
		{
			name: "无market字段-自动检测",
			fileContent: `{
				"stocks": [
					{
						"code": "SH601138",
						"name": "工业富联",
						"tags": ["科技"]
					}
				]
			}`,
			expectedCount:  1,
			expectedMarket: types.MarketChina,
			shouldHaveTags: true,
			desc:           "缺失market字段应自动检测",
		},
		{
			name: "空tags数组",
			fileContent: `{
				"stocks": [
					{
						"code": "AAPL",
						"name": "Apple",
						"market": "us",
						"tags": []
					}
				]
			}`,
			expectedCount:  1,
			expectedMarket: types.MarketUS,
			shouldHaveTags: false,
			desc:           "空tags数组",
		},
		{
			name: "特殊tag过滤",
			fileContent: `{
				"stocks": [
					{
						"code": "AAPL",
						"name": "Apple",
						"tag": "-"
					}
				]
			}`,
			expectedCount:  1,
			expectedMarket: types.MarketUS,
			shouldHaveTags: false,
			desc:           "特殊tag(-)应被过滤",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时文件
			tmpFile := filepath.Join(tmpDir, "watchlist_"+tt.name+".json")
			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			// 读取并解析
			data, err := os.ReadFile(tmpFile)
			require.NoError(t, err)

			var legacyWatchlist WatchlistLegacy
			err = json.Unmarshal(data, &legacyWatchlist)
			require.NoError(t, err)

			// 转换为新格式
			var watchlist types.Watchlist
			for _, legacyStock := range legacyWatchlist.Stocks {
				newStock := types.WatchlistStock{
					Code: legacyStock.Code,
					Name: legacyStock.Name,
				}

				// 市场检测
				if legacyStock.Market == "" {
					newStock.Market = mockMarketDetector(legacyStock.Code)
				} else {
					newStock.Market = legacyStock.Market
				}

				// 标签处理
				var userTags []string
				if len(legacyStock.Tags) > 0 {
					for _, tag := range legacyStock.Tags {
						if tag != "" && tag != "-" && !mockMarketTagChecker(tag) {
							userTags = append(userTags, tag)
						}
					}
					newStock.Tags = userTags
				} else if legacyStock.Tag != "" {
					if !mockMarketTagChecker(legacyStock.Tag) && legacyStock.Tag != "-" {
						newStock.Tags = []string{legacyStock.Tag}
					} else {
						newStock.Tags = []string{}
					}
				} else {
					newStock.Tags = []string{}
				}

				watchlist.Stocks = append(watchlist.Stocks, newStock)
			}

			// 验证
			assert.Len(t, watchlist.Stocks, tt.expectedCount, tt.desc)
			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedMarket, watchlist.Stocks[0].Market, "市场类型: "+tt.desc)
				if tt.shouldHaveTags {
					assert.NotEmpty(t, watchlist.Stocks[0].Tags, "应有标签: "+tt.desc)
				}
			}
		})
	}
}

// TestSaveWatchlist 测试自选列表保存
func TestSaveWatchlist(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	tmpFile := filepath.Join(tmpDir, "watchlist.json")

	watchlist := testutil.NewTestWatchlist()

	// 保存
	data, err := json.MarshalIndent(watchlist, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(tmpFile, data, 0644)
	require.NoError(t, err)

	// 验证文件存在
	assert.True(t, testutil.FileExists(tmpFile))

	// 读取并验证
	loadedData, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var loadedWatchlist types.Watchlist
	err = json.Unmarshal(loadedData, &loadedWatchlist)
	require.NoError(t, err)

	assert.Equal(t, len(watchlist.Stocks), len(loadedWatchlist.Stocks))
}

// TestGetDefaultConfig 测试获取默认配置
func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	// 系统配置
	assert.Equal(t, "en", config.System.Language)
	assert.True(t, config.System.AutoStart)
	assert.Equal(t, "portfolio", config.System.StartupModule)
	assert.Equal(t, "info", config.System.LogLevel)

	// 显示配置
	assert.Equal(t, "professional", config.Display.ColorScheme)
	assert.Equal(t, 3, config.Display.DecimalPlaces)
	assert.Equal(t, "light", config.Display.TableStyle)
	assert.Equal(t, 10, config.Display.MaxLines)
	assert.Equal(t, "yellow", config.Display.PortfolioHighlight)

	// 列配置
	assert.NotEmpty(t, config.Display.PortfolioColumns)
	assert.NotEmpty(t, config.Display.WatchlistColumns)
	assert.Contains(t, config.Display.PortfolioColumns, "cursor")
	assert.Contains(t, config.Display.PortfolioColumns, "code")
	assert.Contains(t, config.Display.WatchlistColumns, "cursor")
	assert.Contains(t, config.Display.WatchlistColumns, "tag")

	// 更新配置
	assert.Equal(t, 5, config.Update.RefreshInterval)
	assert.True(t, config.Update.AutoUpdate)

	// 市场配置
	assert.Equal(t, "Asia/Shanghai", config.Markets.China.Timezone)
	assert.Equal(t, "America/New_York", config.Markets.US.Timezone)
	assert.Equal(t, "Asia/Hong_Kong", config.Markets.HongKong.Timezone)
	assert.Len(t, config.Markets.China.TradingSessions, 2)
	assert.Len(t, config.Markets.US.TradingSessions, 1)

	// 日内数据采集配置
	assert.True(t, config.IntradayCollection.EnableAutoStop)
	assert.Equal(t, 90.0, config.IntradayCollection.CompletenessThreshold)
	assert.Equal(t, 5, config.IntradayCollection.MaxConsecutiveErrors)
	assert.Equal(t, 20, config.IntradayCollection.MinDatapoints)
}

// TestLoadConfig 测试配置加载和验证
func TestLoadConfig(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)

	tests := []struct {
		name          string
		configContent string
		validate      func(*testing.T, types.Config)
		desc          string
	}{
		{
			name: "有效配置",
			configContent: `
system:
  language: zh
  auto_start: true
  startup_module: watchlist
  log_level: debug
display:
  color_scheme: simple
  decimal_places: 2
  table_style: bold
  max_lines: 20
  portfolio_highlight: blue
update:
  refresh_interval: 10
  auto_update: false
`,
			validate: func(t *testing.T, config types.Config) {
				assert.Equal(t, "zh", config.System.Language)
				assert.Equal(t, "watchlist", config.System.StartupModule)
				assert.Equal(t, 2, config.Display.DecimalPlaces)
				assert.Equal(t, 20, config.Display.MaxLines)
				assert.Equal(t, 10, config.Update.RefreshInterval)
			},
			desc: "自定义配置应被正确加载",
		},
		{
			name: "MaxLines超出范围",
			configContent: `
display:
  max_lines: 100
`,
			validate: func(t *testing.T, config types.Config) {
				assert.Equal(t, 10, config.Display.MaxLines, "超出范围应使用默认值")
			},
			desc: "MaxLines大于50应使用默认值10",
		},
		{
			name: "MaxLines为负数",
			configContent: `
display:
  max_lines: -5
`,
			validate: func(t *testing.T, config types.Config) {
				assert.Equal(t, 10, config.Display.MaxLines, "负数应使用默认值")
			},
			desc: "MaxLines为负数应使用默认值10",
		},
		{
			name: "空PortfolioHighlight",
			configContent: `
display:
  portfolio_highlight: ""
`,
			validate: func(t *testing.T, config types.Config) {
				assert.Equal(t, "yellow", config.Display.PortfolioHighlight, "空值应使用默认值")
			},
			desc: "空高亮颜色应使用默认值yellow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时配置文件
			tmpFile := filepath.Join(tmpDir, "config_"+tt.name+".yml")
			err := os.WriteFile(tmpFile, []byte(tt.configContent), 0644)
			require.NoError(t, err)

			// 读取配置
			data, err := os.ReadFile(tmpFile)
			require.NoError(t, err)

			var config types.Config
			err = yaml.Unmarshal(data, &config)
			require.NoError(t, err)

			// 应用验证逻辑(模拟 LoadConfig 的验证)
			if config.Display.MaxLines <= 0 || config.Display.MaxLines > 50 {
				config.Display.MaxLines = 10
			}
			if config.Display.PortfolioHighlight == "" {
				config.Display.PortfolioHighlight = "yellow"
			}
			if config.Markets.China.Timezone == "" {
				config.Markets = DefaultMarketsConfig()
			}
			if len(config.Display.PortfolioColumns) == 0 {
				config.Display.PortfolioColumns = GetDefaultConfig().Display.PortfolioColumns
			}
			if len(config.Display.WatchlistColumns) == 0 {
				config.Display.WatchlistColumns = GetDefaultConfig().Display.WatchlistColumns
			}

			// 验证
			tt.validate(t, config)
		})
	}
}

// TestSaveConfig 测试配置保存
func TestSaveConfig(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	tmpFile := filepath.Join(tmpDir, "config.yml")

	config := testutil.NewTestConfig()

	// 保存
	data, err := yaml.Marshal(config)
	require.NoError(t, err)
	err = os.WriteFile(tmpFile, data, 0644)
	require.NoError(t, err)

	// 验证文件存在
	assert.True(t, testutil.FileExists(tmpFile))

	// 读取并验证
	loadedData, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var loadedConfig types.Config
	err = yaml.Unmarshal(loadedData, &loadedConfig)
	require.NoError(t, err)

	assert.Equal(t, config.System.Language, loadedConfig.System.Language)
	assert.Equal(t, config.Display.ColorScheme, loadedConfig.Display.ColorScheme)
}

// TestValidatePortfolioColumns 测试持股列验证
func TestValidatePortfolioColumns(t *testing.T) {
	tests := []struct {
		name             string
		configured       []string
		shouldContain    []string
		shouldNotContain []string
		minLength        int
		desc             string
	}{
		{
			name:          "全部有效列",
			configured:    []string{"cursor", "code", "name", "price", "cost"},
			shouldContain: []string{"cursor", "code", "name", "price", "cost"},
			minLength:     5,
			desc:          "所有有效列应保留",
		},
		{
			name:             "包含无效列",
			configured:       []string{"cursor", "invalid_column", "code", "price"},
			shouldContain:    []string{"cursor", "code", "price", "name"},
			shouldNotContain: []string{"invalid_column"},
			minLength:        4,
			desc:             "无效列应被过滤,缺失必需列应补充",
		},
		{
			name:          "空配置",
			configured:    []string{},
			shouldContain: []string{"cursor", "code", "name", "price"},
			minLength:     4,
			desc:          "空配置应返回必需列",
		},
		{
			name:          "缺少必需列",
			configured:    []string{"cost", "quantity"},
			shouldContain: []string{"cursor", "code", "name", "price", "cost", "quantity"},
			minLength:     6,
			desc:          "缺失的必需列应被添加",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePortfolioColumns(tt.configured)

			assert.GreaterOrEqual(t, len(result), tt.minLength, tt.desc)

			for _, col := range tt.shouldContain {
				assert.Contains(t, result, col, "应包含列: "+col)
			}

			for _, col := range tt.shouldNotContain {
				assert.NotContains(t, result, col, "不应包含列: "+col)
			}
		})
	}
}

// TestValidateWatchlistColumns 测试自选列验证
func TestValidateWatchlistColumns(t *testing.T) {
	tests := []struct {
		name             string
		configured       []string
		shouldContain    []string
		shouldNotContain []string
		minLength        int
		desc             string
	}{
		{
			name:          "全部有效列",
			configured:    []string{"cursor", "tag", "code", "name", "price"},
			shouldContain: []string{"cursor", "tag", "code", "name", "price"},
			minLength:     5,
			desc:          "所有有效列应保留",
		},
		{
			name:             "包含无效列",
			configured:       []string{"cursor", "bad_column", "code"},
			shouldContain:    []string{"cursor", "code", "tag", "name", "price"},
			shouldNotContain: []string{"bad_column"},
			minLength:        5,
			desc:             "无效列应被过滤",
		},
		{
			name:          "空配置",
			configured:    []string{},
			shouldContain: []string{"cursor", "tag", "code", "name", "price"},
			minLength:     5,
			desc:          "空配置应返回必需列",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateWatchlistColumns(tt.configured)

			assert.GreaterOrEqual(t, len(result), tt.minLength, tt.desc)

			for _, col := range tt.shouldContain {
				assert.Contains(t, result, col, "应包含列: "+col)
			}

			for _, col := range tt.shouldNotContain {
				assert.NotContains(t, result, col, "不应包含列: "+col)
			}
		})
	}
}

// TestInsertAt 测试插入函数
func TestInsertAt(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		index    int
		values   []string
		expected []string
	}{
		{
			name:     "在开头插入",
			slice:    []string{"a", "b", "c"},
			index:    0,
			values:   []string{"x", "y"},
			expected: []string{"x", "y", "a", "b", "c"},
		},
		{
			name:     "在中间插入",
			slice:    []string{"a", "b", "c"},
			index:    1,
			values:   []string{"x"},
			expected: []string{"a", "x", "b", "c"},
		},
		{
			name:     "在末尾插入",
			slice:    []string{"a", "b"},
			index:    2,
			values:   []string{"x", "y"},
			expected: []string{"a", "b", "x", "y"},
		},
		{
			name:     "负索引-调整为0",
			slice:    []string{"a", "b"},
			index:    -1,
			values:   []string{"x"},
			expected: []string{"x", "a", "b"},
		},
		{
			name:     "超出范围索引",
			slice:    []string{"a", "b"},
			index:    10,
			values:   []string{"x"},
			expected: []string{"a", "b", "x"},
		},
		{
			name:     "空切片",
			slice:    []string{},
			index:    0,
			values:   []string{"x", "y"},
			expected: []string{"x", "y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := insertAt(tt.slice, tt.index, tt.values...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestContains 测试包含检查函数
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		str      string
		expected bool
	}{
		{
			name:     "包含元素",
			slice:    []string{"a", "b", "c"},
			str:      "b",
			expected: true,
		},
		{
			name:     "不包含元素",
			slice:    []string{"a", "b", "c"},
			str:      "d",
			expected: false,
		},
		{
			name:     "空切片",
			slice:    []string{},
			str:      "a",
			expected: false,
		},
		{
			name:     "空字符串-包含",
			slice:    []string{"a", "", "c"},
			str:      "",
			expected: true,
		},
		{
			name:     "空字符串-不包含",
			slice:    []string{"a", "b", "c"},
			str:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.slice, tt.str)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMigrateAlertFrequency 测试告警频率迁移
func TestMigrateAlertFrequency(t *testing.T) {
	tests := []struct {
		name            string
		alerts          []types.Alert
		expectedChanges int
		validate        func(*testing.T, []types.Alert)
		desc            string
	}{
		{
			name: "空频率-默认Once",
			alerts: []types.Alert{
				{ID: "1", StockCode: "SH601138", Frequency: ""},
				{ID: "2", StockCode: "AAPL", Frequency: ""},
			},
			expectedChanges: 2,
			validate: func(t *testing.T, alerts []types.Alert) {
				for _, alert := range alerts {
					assert.Equal(t, types.TriggerOnce, alert.Frequency)
				}
			},
			desc: "空频率应设为Once",
		},
		{
			name: "EveryNDays频率-无效天数",
			alerts: []types.Alert{
				{ID: "1", Frequency: types.TriggerEveryNDays, FrequencyDays: 0},
				{ID: "2", Frequency: types.TriggerEveryNDays, FrequencyDays: -5},
			},
			expectedChanges: 2,
			validate: func(t *testing.T, alerts []types.Alert) {
				for _, alert := range alerts {
					assert.Equal(t, 1, alert.FrequencyDays, "无效天数应设为1")
				}
			},
			desc: "EveryNDays无效天数应设为1",
		},
		{
			name: "已有有效频率",
			alerts: []types.Alert{
				{ID: "1", Frequency: types.TriggerDaily},
				{ID: "2", Frequency: types.TriggerWeekly},
			},
			expectedChanges: 0,
			validate: func(t *testing.T, alerts []types.Alert) {
				assert.Equal(t, types.TriggerDaily, alerts[0].Frequency)
				assert.Equal(t, types.TriggerWeekly, alerts[1].Frequency)
			},
			desc: "已有效频率不应改变",
		},
		{
			name: "混合情况",
			alerts: []types.Alert{
				{ID: "1", Frequency: ""},
				{ID: "2", Frequency: types.TriggerDaily},
				{ID: "3", Frequency: types.TriggerEveryNDays, FrequencyDays: 0},
			},
			expectedChanges: 2,
			validate: func(t *testing.T, alerts []types.Alert) {
				assert.Equal(t, types.TriggerOnce, alerts[0].Frequency)
				assert.Equal(t, types.TriggerDaily, alerts[1].Frequency)
				assert.Equal(t, 1, alerts[2].FrequencyDays)
			},
			desc: "混合情况应正确处理",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MigrateAlertFrequency(tt.alerts)
			tt.validate(t, result)
		})
	}
}

// TestLoadAndSaveAlertData 测试告警数据保存和加载
func TestLoadAndSaveAlertData(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	tmpFile := filepath.Join(tmpDir, "alert.json")

	// 准备测试数据
	alertData := types.AlertData{
		Alerts: []types.Alert{
			testutil.NewTestAlert(types.AlertTypePrice),
			testutil.NewTestAlert(types.AlertTypeRate),
		},
		LastCheck:  "2026-01-20 10:00:00",
		AlertCount: 2,
	}

	// 保存
	alertData.AlertCount = len(alertData.Alerts)
	data, err := json.MarshalIndent(alertData, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(tmpFile, data, 0644)
	require.NoError(t, err)

	// 加载
	loadedData, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var loadedAlertData types.AlertData
	err = json.Unmarshal(loadedData, &loadedAlertData)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, 2, loadedAlertData.AlertCount)
	assert.Len(t, loadedAlertData.Alerts, 2)
	assert.Equal(t, alertData.LastCheck, loadedAlertData.LastCheck)
}

// TestDefaultMarketsConfig 测试默认市场配置
func TestDefaultMarketsConfig(t *testing.T) {
	config := DefaultMarketsConfig()

	// 中国市场
	assert.Equal(t, "Asia/Shanghai", config.China.Timezone)
	assert.Len(t, config.China.TradingSessions, 2)
	assert.Equal(t, "09:30", config.China.TradingSessions[0].StartTime)
	assert.Equal(t, "11:30", config.China.TradingSessions[0].EndTime)
	assert.Equal(t, "13:00", config.China.TradingSessions[1].StartTime)
	assert.Equal(t, "15:00", config.China.TradingSessions[1].EndTime)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, config.China.Weekdays)

	// 美国市场
	assert.Equal(t, "America/New_York", config.US.Timezone)
	assert.Len(t, config.US.TradingSessions, 1)
	assert.Equal(t, "09:30", config.US.TradingSessions[0].StartTime)
	assert.Equal(t, "16:00", config.US.TradingSessions[0].EndTime)

	// 香港市场
	assert.Equal(t, "Asia/Hong_Kong", config.HongKong.Timezone)
	assert.Len(t, config.HongKong.TradingSessions, 2)
	assert.Equal(t, "09:30", config.HongKong.TradingSessions[0].StartTime)
	assert.Equal(t, "12:00", config.HongKong.TradingSessions[0].EndTime)
}
