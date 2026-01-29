package china

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"stock-monitor/internal/types"
)

// 东方财富板块API配置
const (
	eastMoneyPushAPI = "https://push2.eastmoney.com/api/qt/clist/get"

	// 行业板块筛选条件: m:90+t:2+f:!50
	// 概念板块筛选条件: m:90+t:3+f:!50
	industrySectorFilter = "m:90+t:2+f:!50"
	conceptSectorFilter  = "m:90+t:3+f:!50"
)

// FetchSectorList 获取板块列表
func FetchSectorList(sectorType types.SectorType) ([]types.Sector, error) {
	filter := industrySectorFilter
	if sectorType == types.SectorTypeConcept {
		filter = conceptSectorFilter
	}

	params := url.Values{}
	params.Set("pn", "1")
	params.Set("pz", "500") // 每页500条,足够覆盖所有板块
	params.Set("po", "1")
	params.Set("np", "1")
	params.Set("ut", "bd1d9ddb04089700cf9c27f6f7426281")
	params.Set("fltt", "2")
	params.Set("invt", "2")
	params.Set("fid", "f3") // 按涨跌幅排序
	params.Set("fs", filter)
	params.Set("fields", "f12,f14,f2,f3,f4,f5,f6,f8,f104,f105,f106,f128,f136,f140") // 字段列表
	// f12: 代码
	// f14: 名称
	// f2: 最新价
	// f3: 涨跌幅
	// f4: 涨跌额
	// f5: 成交量
	// f6: 成交额
	// f8: 换手率
	// f104: 上涨家数
	// f105: 下跌家数
	// f106: 平盘家数
	// f128: 领涨股名称
	// f136: 领涨股涨跌幅
	// f140: 领涨股代码

	reqURL := eastMoneyPushAPI + "?" + params.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("请求东方财富板块API失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Data struct {
			Diff []map[string]interface{} `json:"diff"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	sectors := make([]types.Sector, 0, len(result.Data.Diff))
	now := time.Now()

	for _, item := range result.Data.Diff {
		sector := types.Sector{
			Code:      getStringValue(item, "f12"),
			Name:      getStringValue(item, "f14"),
			Type:      sectorType,
			UpdatedAt: now,
		}

		// 数值字段
		sector.ChangePercent = getFloatValue(item, "f3")
		sector.Change = getFloatValue(item, "f4")
		sector.Turnover = getFloatValue(item, "f6")
		sector.TurnoverRate = getFloatValue(item, "f8") // 换手率
		sector.RiseCount = getIntValue(item, "f104")
		sector.FallCount = getIntValue(item, "f105")
		sector.LeaderName = getStringValue(item, "f128")  // f128 是领涨股名称
		sector.LeaderChange = getFloatValue(item, "f136") // f136 是领涨股涨跌幅
		sector.LeaderCode = getStringValue(item, "f140")  // f140 是领涨股代码

		sectors = append(sectors, sector)
	}

	return sectors, nil
}

// FetchSectorStocks 获取板块成分股
func FetchSectorStocks(sectorCode string) ([]types.SectorStock, error) {
	params := url.Values{}
	params.Set("pn", "1")
	params.Set("pz", "500") // 每页500条
	params.Set("po", "1")
	params.Set("np", "1")
	params.Set("ut", "bd1d9ddb04089700cf9c27f6f7426281")
	params.Set("fltt", "2")
	params.Set("invt", "2")
	params.Set("fid", "f3")                                 // 按涨跌幅排序
	params.Set("fs", fmt.Sprintf("b:%s+f:!50", sectorCode)) // b:板块代码
	params.Set("fields", "f12,f14,f2,f3,f4,f5,f6,f8")       // 字段列表
	// f12: 代码
	// f14: 名称
	// f2: 最新价
	// f3: 涨跌幅
	// f4: 涨跌额
	// f5: 成交量
	// f6: 成交额
	// f8: 换手率

	reqURL := eastMoneyPushAPI + "?" + params.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("请求东方财富成分股API失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Data struct {
			Diff []map[string]interface{} `json:"diff"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	stocks := make([]types.SectorStock, 0, len(result.Data.Diff))
	now := time.Now()

	for _, item := range result.Data.Diff {
		stock := types.SectorStock{
			Code:      normalizeStockCode(getStringValue(item, "f12")),
			Name:      getStringValue(item, "f14"),
			UpdatedAt: now,
		}

		// 数值字段
		stock.Price = getFloatValue(item, "f2")
		stock.ChangePercent = getFloatValue(item, "f3")
		stock.Change = getFloatValue(item, "f4")
		stock.Volume = int64(getFloatValue(item, "f5"))
		stock.Turnover = getFloatValue(item, "f6")
		stock.TurnoverRate = getFloatValue(item, "f8")

		stocks = append(stocks, stock)
	}

	return stocks, nil
}

// normalizeStockCode 规范化股票代码（添加市场前缀）
func normalizeStockCode(code string) string {
	if code == "" || code == "-" {
		return code
	}

	// 已经包含市场前缀
	if strings.Contains(code, ".") {
		return code
	}

	// 根据代码判断市场
	if strings.HasPrefix(code, "6") {
		return "SH" + code // 上海
	} else if strings.HasPrefix(code, "0") || strings.HasPrefix(code, "3") {
		return "SZ" + code // 深圳
	}

	return code
}

// getStringValue 从map中安全获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		switch val := v.(type) {
		case string:
			if val == "-" {
				return ""
			}
			return val
		case float64:
			return strconv.FormatFloat(val, 'f', -1, 64)
		case int:
			return strconv.Itoa(val)
		}
	}
	return ""
}

// getFloatValue 从map中安全获取浮点数值
func getFloatValue(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok && v != nil {
		switch val := v.(type) {
		case float64:
			return val
		case string:
			if val == "-" || val == "" {
				return 0
			}
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f
			}
		case int:
			return float64(val)
		}
	}
	return 0
}

// getIntValue 从map中安全获取整数值
func getIntValue(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok && v != nil {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case string:
			if val == "-" || val == "" {
				return 0
			}
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
	}
	return 0
}
