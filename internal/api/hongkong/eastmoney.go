package hongkong

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stock-monitor/internal/api/common"
	"stock-monitor/internal/log"
	"strings"
	"time"
)

// TryEastMoneyHKTurnover 从东方财富获取港股换手率
// 仅用于补充腾讯API缺失的港股换手率数据
func TryEastMoneyHKTurnover(symbol string) (float64, int64, error) {
	// 转换股票代码为东方财富格式 (HK00700 → 116.00700)
	emCode := ConvertStockCodeForEastMoneyAPI(symbol)
	if emCode == "" {
		return 0, 0, fmt.Errorf("invalid HK stock code: %s", symbol)
	}

	// 构建API URL (只请求必要字段以减少流量)
	url := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/stock/get?secid=%s&fields=f47,f168",
		emCode,
	)
	log.Debug("log.api.eastmoneyTurnoverUrl", url)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Error("log.api.eastmoneyTurnoverHttpFail", err)
		return 0, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("log.api.eastmoneyTurnoverReadFail", err)
		return 0, 0, err
	}

	var result struct {
		Data struct {
			F47  int64 `json:"f47"`  // 成交量
			F168 int   `json:"f168"` // 换手率 (需除以100)
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Error("log.api.eastmoneyTurnoverJsonFail", err)
		return 0, 0, err
	}

	// 换手率需要除以100 (19 → 0.19%)
	turnover := float64(result.Data.F168) / 100.0
	volume := result.Data.F47

	log.Info("log.api.eastmoneyTurnoverSuccess", symbol, turnover, volume)
	return turnover, volume, nil
}

// ConvertStockCodeForEastMoneyAPI 转换股票代码为东方财富API格式
func ConvertStockCodeForEastMoneyAPI(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if strings.HasPrefix(symbol, "HK") {
		// HK00700 → 116.00700, HK9626 → 116.09626
		code := strings.TrimPrefix(symbol, "HK")
		return "116." + common.PadHKStockCode(code)
	} else if strings.HasSuffix(symbol, ".HK") {
		// 0700.HK → 116.00700
		code := strings.TrimSuffix(symbol, ".HK")
		return "116." + common.PadHKStockCode(code)
	}

	return "" // 非港股返回空字符串
}
