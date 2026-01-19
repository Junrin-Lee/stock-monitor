package china

import (
	"fmt"
	"io"
	"net/http"
	"stock-monitor/internal/api/common"
	"stock-monitor/internal/log"
	"stock-monitor/internal/types"
	"strings"
	"time"
)

// SearchStockBySinaAPI 使用新浪财经搜索API查找股票
func SearchStockBySinaAPI(keyword string) *types.StockData {
	log.Debug("log.api.sinaSearchStart", keyword)

	// 新浪财经搜索API URL
	url := fmt.Sprintf("https://suggest3.sinajs.cn/suggest/type=11,12,13,14,15&key=%s", keyword)
	log.Debug("log.api.sinaRequestUrl", url)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Error("log.api.sinaHttpFail", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("log.api.sinaReadFail", err)
		return nil
	}

	content := string(body)
	log.Debug("log.api.sinaResponse", content)

	// 解析新浪搜索结果
	return parseSinaSearchResults(content, keyword)
}

// parseSinaSearchResults 解析新浪搜索结果
func parseSinaSearchResults(content, keyword string) *types.StockData {
	// 新浪返回格式类似: var suggestvalue="sz000858,五粮液;sh600519,贵州茅台;";
	lines := strings.Split(content, ";")

	for _, line := range lines {
		if !strings.Contains(line, ",") {
			continue
		}

		// 提取股票信息
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		code := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])

		// 清理代码和名称中的特殊字符
		code = strings.Trim(code, "\"'")
		name = strings.Trim(name, "\"'")

		if code == "" || name == "" {
			continue
		}

		// 检查名称是否匹配关键词
		if strings.Contains(name, keyword) {
			log.Debug("log.api.sinaSearchFound", name, code)

			// 转换为标准格式
			standardCode := common.ConvertSinaCodeToStandard(code)

			// 获取详细信息
			stockData := TryTencentAPI(standardCode)
			if stockData != nil && stockData.Price > 0 {
				stockData.Symbol = standardCode
				stockData.Name = name
				return stockData
			}
		}
	}

	return nil
}
