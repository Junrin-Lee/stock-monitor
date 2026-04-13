package china

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stock-monitor/internal/api/common"
	"stock-monitor/internal/log"
	"stock-monitor/internal/types"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// SearchStockByTencentAPI 使用腾讯搜索API查找股票
func SearchStockByTencentAPI(keyword string) *types.StockData {
	log.Debug("log.api.tencentSearchStart", keyword)

	// 腾讯股票搜索API URL - 使用t=all支持A股、港股、美股搜索
	url := fmt.Sprintf("https://smartbox.gtimg.cn/s3/?q=%s&t=all", keyword)
	log.Debug("log.api.tencentSearchUrl", url)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("log.api.tencentSearchReqFail", err)
		return nil
	}

	// 添加必要的请求头，提高成功率
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://stockapp.finance.qq.com/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		log.Error("log.api.tencentSearchHttpFail", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Error("log.api.tencentSearchStatusFail", resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("log.api.tencentSearchReadFail", err)
		return nil
	}

	content, err := gbkToUtf8(body)
	if err != nil {
		log.Error("log.api.tencentSearchEncodeFail", err)
		return nil
	}
	log.Debug("log.api.tencentSearchResponse", content[:common.Min(300, len(content))])

	// 解析搜索结果
	return ParseSearchResults(content, keyword)
}

// TryTencentAPI 使用腾讯API获取股票价格
func TryTencentAPI(symbol string) *types.StockData {
	tencentSymbol := ConvertStockSymbolForTencent(symbol)
	log.Debug("log.api.tencentConvert", symbol, tencentSymbol)

	url := fmt.Sprintf("https://qt.gtimg.cn/q=%s", tencentSymbol)
	log.Debug("log.api.tencentUrl", url)

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("log.api.tencentReqFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	// 添加必要的请求头，与搜索API保持一致
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://stockapp.finance.qq.com/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		log.Error("log.api.tencentHttpFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("log.api.tencentReadFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	content, err := gbkToUtf8(body)
	if err != nil {
		log.Error("log.api.tencentEncodeFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}
	log.Debug("log.api.tencentResponse", content[:common.Min(100, len(content))])

	if !strings.Contains(content, "~") {
		log.Error("log.api.tencentFormatError")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	fields := strings.Split(content, "~")
	if len(fields) < 5 {
		log.Error("log.api.tencentFieldsError")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	stockName := fields[1]

	price, err := strconv.ParseFloat(fields[3], 64)
	if err != nil || price <= 0 {
		log.Error("log.api.tencentPriceError", fields[3])
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	previousClose, err := strconv.ParseFloat(fields[4], 64)
	if err != nil || previousClose <= 0 {
		log.Error("log.api.tencentPrevCloseError", fields[4])
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	// 解析开盘价、最高价、最低价、换手率、成交量
	var openPrice, maxPrice, minPrice, turnoverRate float64
	var volume int64

	// 腾讯API字段位置：fields[5]=开盘价, fields[33]=最高价, fields[34]=最低价, fields[38]=换手率, fields[36]=成交量
	if len(fields) > 5 {
		openPrice, _ = strconv.ParseFloat(fields[5], 64)
	}
	if len(fields) > 33 {
		maxPrice, _ = strconv.ParseFloat(fields[33], 64)
	}
	if len(fields) > 34 {
		minPrice, _ = strconv.ParseFloat(fields[34], 64)
	}
	if len(fields) > 38 {
		turnoverRate, _ = strconv.ParseFloat(fields[38], 64)
	}
	if len(fields) > 36 {
		volume, _ = strconv.ParseInt(fields[36], 10, 64)
	}

	change := price - previousClose
	changePercent := (change / previousClose) * 100

	log.Info("log.api.tencentSuccess", stockName, price, change, changePercent, openPrice, maxPrice, minPrice, turnoverRate, volume)

	return &types.StockData{
		Symbol:        symbol,
		Name:          stockName,
		Price:         price,
		Change:        change,
		ChangePercent: changePercent,
		StartPrice:    openPrice,
		MaxPrice:      maxPrice,
		MinPrice:      minPrice,
		PrevClose:     previousClose,
		TurnoverRate:  turnoverRate,
		Volume:        volume,
	}
}

// ConvertStockSymbolForTencent 转换股票代码为腾讯API格式
func ConvertStockSymbolForTencent(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if strings.HasPrefix(symbol, "SH") {
		return "sh" + strings.TrimPrefix(symbol, "SH")
	} else if strings.HasPrefix(symbol, "SZ") {
		return "sz" + strings.TrimPrefix(symbol, "SZ")
	} else if strings.HasPrefix(symbol, "HK") {
		// 港股格式: HK00700 -> hk00700, HK2020 -> hk02020
		// 港股代码需要补齐5位数字
		code := strings.TrimPrefix(symbol, "HK")
		return "hk" + common.PadHKStockCode(code)
	} else if strings.HasSuffix(symbol, ".HK") {
		// 港股格式: 0700.HK -> hk00700, 2020.HK -> hk02020
		code := strings.TrimSuffix(symbol, ".HK")
		return "hk" + common.PadHKStockCode(code)
	}

	if len(symbol) == 6 && strings.HasPrefix(symbol, "6") {
		return "sh" + symbol
	} else if len(symbol) == 6 && (strings.HasPrefix(symbol, "0") || strings.HasPrefix(symbol, "3")) {
		return "sz" + symbol
	}

	return symbol
}

// gbkToUtf8 converts GBK encoding to UTF-8
func gbkToUtf8(gbkData []byte) (string, error) {
	decoder := simplifiedchinese.GBK.NewDecoder()
	utf8Data, err := decoder.Bytes(gbkData)
	if err != nil {
		return "", err
	}
	return string(utf8Data), nil
}

// ParseSearchResults 解析腾讯搜索结果
func ParseSearchResults(content, keyword string) *types.StockData {
	log.Debug("log.api.parseSearchStart")

	// 尝试解析新的腾讯格式 (v_hint=)
	result := parseTencentHintFormat(content, keyword)
	if result != nil {
		return result
	}

	// 尝试解析JSON格式的响应
	result = parseJSONSearchResults(content, keyword)
	if result != nil {
		return result
	}

	// 如果JSON解析失败，尝试解析旧格式
	return parseLegacySearchResults(content, keyword)
}

// parseTencentHintFormat 解析腾讯Hint格式的搜索结果
func parseTencentHintFormat(content, keyword string) *types.StockData {
	// 格式: v_hint="sz~000880~潍柴重机~wczj~GP-A"
	log.Debug("log.api.tryTencentHint")

	// 查找v_hint=
	if !strings.Contains(content, "v_hint=") {
		return nil
	}

	// 提取引号内的内容
	startPos := strings.Index(content, "v_hint=\"")
	if startPos == -1 {
		return nil
	}
	startPos += len("v_hint=\"")

	endPos := strings.Index(content[startPos:], "\"")
	if endPos == -1 {
		return nil
	}

	data := content[startPos : startPos+endPos]
	log.Debug("log.api.extractedData", data)

	// 按^分割多个结果，取第一个
	results := strings.Split(data, "^")
	if len(results) == 0 {
		log.Debug("log.api.noSearchResult")
		return nil
	}

	// 处理第一个结果
	firstResult := results[0]
	fields := strings.Split(firstResult, "~")
	if len(fields) < 3 {
		log.Debug("log.api.fieldsNotEnough", len(fields))
		return nil
	}

	market := fields[0] // sz, sh, hk
	code := fields[1]   // 000880
	name := fields[2]   // 潍柴重机（可能是Unicode编码）

	// 尝试解码Unicode字符串
	decodedName, err := strconv.Unquote(`"` + name + `"`)
	if err == nil {
		name = decodedName
	}

	log.Debug("log.api.parseResultDetail", market, code, name)

	// 转换为标准格式
	standardCode := strings.ToUpper(market) + code
	log.Debug("log.api.tencentHintFound", name, standardCode)

	// 获取详细信息
	stockData := TryTencentAPI(standardCode)
	if stockData != nil && stockData.Price > 0 {
		stockData.Symbol = standardCode
		stockData.Name = name
		return stockData
	}

	return nil
}

// parseJSONSearchResults 解析JSON格式的搜索结果
func parseJSONSearchResults(content, keyword string) *types.StockData {
	// 尝试解析为JSON
	var searchResult map[string]interface{}
	if err := json.Unmarshal([]byte(content), &searchResult); err != nil {
		log.Error("log.api.jsonParseFail", err)
		return nil
	}

	// 查找数据字段
	data, ok := searchResult["data"]
	if !ok {
		log.Debug("log.api.noDataField")
		return nil
	}

	dataArray, ok := data.([]interface{})
	if !ok {
		log.Debug("log.api.dataNotArray")
		return nil
	}

	for _, item := range dataArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// 提取股票信息
		code, _ := itemMap["code"].(string)
		name, _ := itemMap["name"].(string)

		if code == "" || name == "" {
			continue
		}

		// 检查名称是否匹配关键词
		if strings.Contains(name, keyword) {
			log.Debug("log.api.jsonFormatFound", name, code)

			// 转换为标准格式
			standardCode := common.ConvertJSONCodeToStandard(code)

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

// parseLegacySearchResults 解析旧格式的搜索结果
func parseLegacySearchResults(content, keyword string) *types.StockData {
	log.Debug("log.api.useLegacyFormat")
	// 腾讯搜索结果格式分析
	// 格式类似: v_s_关键词="sz002415~海康威视~002415~7.450~-0.160~-2.105~15270~7705~7565~7.610"
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if !strings.Contains(line, "~") {
			continue
		}

		// 找到符号="的位置
		startPos := strings.Index(line, "\"")
		endPos := strings.LastIndex(line, "\"")
		if startPos == -1 || endPos == -1 || startPos >= endPos {
			continue
		}

		// 提取数据部分
		data := line[startPos+1 : endPos]
		fields := strings.Split(data, "~")

		if len(fields) < 4 {
			continue
		}

		// 解析字段
		code := fields[0]
		name := fields[1]
		shortCode := fields[2]

		// 检查名称是否匹配关键词
		if strings.Contains(name, keyword) {
			log.Debug("log.api.legacyFormatFound", name, code)

			// 转换为标准格式
			standardCode := common.ConvertToStandardCode(code, shortCode)

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
