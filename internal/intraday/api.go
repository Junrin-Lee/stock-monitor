package intraday

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stock-monitor/internal/api"
)

// FetchIntradayDataFromAPI tries all APIs in fallback order based on market type
func FetchIntradayDataFromAPI(stockCode string) ([]IntradayDataPoint, error) {
	var lastErr error
	market := string(api.GetMarketType(stockCode))

	// US stocks: Use Yahoo Finance API (best for US stocks)
	if market == "us" {
		logDebug("log.intraday.marketTypeUS", stockCode)

		data, err := tryGetIntradayFromYahoo(stockCode)
		if err == nil && len(data) > 0 {
			logDebug("log.intraday.yahooSuccess", stockCode, len(data))
			return data, nil
		}
		if err != nil {
			lastErr = err
			logDebug("log.intraday.yahooFail", stockCode, err)
		} else {
			logDebug("log.intraday.yahooNoData", stockCode)
		}

		return nil, fmt.Errorf("Yahoo Finance API失败: %w", lastErr)
	}

	// Hong Kong stocks: Try Tencent first, then Yahoo Finance as fallback
	if market == "hongkong" {
		logDebug("log.intraday.marketTypeHK", stockCode)

		// Try Tencent API (primary for HK stocks)
		data, err := tryGetIntradayFromTencent(stockCode)
		if err == nil && len(data) > 0 {
			logDebug("log.intraday.tencentSuccess", stockCode, len(data))
			return data, nil
		}
		if err != nil {
			lastErr = err
			logDebug("log.intraday.tencentFail", stockCode, err)
		} else {
			logDebug("log.intraday.tencentNoData", stockCode)
		}

		// Try Yahoo Finance API (fallback for HK stocks)
		data, err = tryGetIntradayFromYahoo(stockCode)
		if err == nil && len(data) > 0 {
			logDebug("log.intraday.yahooSuccess", stockCode, len(data))
			return data, nil
		}
		if err != nil {
			lastErr = err
			logDebug("log.intraday.yahooFail", stockCode, err)
		} else {
			logDebug("log.intraday.yahooNoData", stockCode)
		}

		// Try EastMoney API (secondary fallback)
		data, err = tryGetIntradayFromEastMoney(stockCode)
		if err == nil && len(data) > 0 {
			logDebug("log.intraday.eastMoneySuccess", stockCode, len(data))
			return data, nil
		}
		if err != nil {
			lastErr = err
			logDebug("log.intraday.eastMoneyFail", stockCode, err)
		} else {
			logDebug("log.intraday.eastMoneyNoData", stockCode)
		}

		return nil, fmt.Errorf("所有港股API失败, 最后错误: %w", lastErr)
	}

	// China A-shares: Use Chinese APIs (Tencent, EastMoney, Sina)
	logDebug("log.intraday.marketTypeChina", stockCode)

	// Try Tencent API (primary - most reliable for A-shares)
	data, err := tryGetIntradayFromTencent(stockCode)
	if err == nil && len(data) > 0 {
		logDebug("log.intraday.tencentSuccess", stockCode, len(data))
		return data, nil
	}
	if err != nil {
		lastErr = err
		logDebug("log.intraday.tencentFail", stockCode, err)
	} else {
		logDebug("log.intraday.tencentNoData", stockCode)
	}

	// Try EastMoney API (secondary)
	data, err = tryGetIntradayFromEastMoney(stockCode)
	if err == nil && len(data) > 0 {
		logDebug("log.intraday.eastMoneySuccess", stockCode, len(data))
		return data, nil
	}
	if err != nil {
		lastErr = err
		logDebug("log.intraday.eastMoneyFail", stockCode, err)
	} else {
		logDebug("log.intraday.eastMoneyNoData", stockCode)
	}

	// Try Sina Finance API (last fallback - K-line data, may not have today's data)
	data, err = tryGetIntradayFromSina(stockCode)
	if err == nil && len(data) > 0 {
		logDebug("log.intraday.sinaSuccess", stockCode, len(data))
		return data, nil
	}
	if err != nil {
		lastErr = err
		logDebug("log.intraday.sinaFail", stockCode, err)
	} else {
		logDebug("log.intraday.sinaNoData", stockCode)
	}

	return nil, fmt.Errorf("所有A股API失败, 最后错误: %w", lastErr)
}

// tryGetIntradayFromSina fetches intraday data from Sina Finance API
func tryGetIntradayFromSina(stockCode string) ([]IntradayDataPoint, error) {
	// Convert stock code for Sina API
	sinaCode := convertStockCodeForSina(stockCode)

	// Build URL
	url := fmt.Sprintf(
		"http://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData?symbol=%s&scale=1&datalen=250",
		sinaCode,
	)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://finance.sina.com.cn")

	// Send request with retry
	resp, err := fetchWithRetry(client, req, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse JSON response
	var sinaData []struct {
		Day    string `json:"day"`    // "2025-11-26 09:31:00"
		Open   string `json:"open"`   // "8.52"
		High   string `json:"high"`   // "8.53"
		Low    string `json:"low"`    // "8.51"
		Close  string `json:"close"`  // "8.52"
		Volume string `json:"volume"` // "12000"
	}

	if err := json.NewDecoder(resp.Body).Decode(&sinaData); err != nil {
		return nil, err
	}

	// Convert to IntradayDataPoint
	result := make([]IntradayDataPoint, 0, len(sinaData))
	for _, item := range sinaData {
		price, err := strconv.ParseFloat(item.Close, 64)
		if err != nil {
			continue
		}

		timeStr := formatIntradayTime(item.Day)
		if timeStr == "" {
			continue
		}

		// Sina API 在 JSON 字段中直接提供 volume
		vol, _ := strconv.ParseInt(item.Volume, 10, 64)

		result = append(result, IntradayDataPoint{
			Time:   timeStr,
			Price:  price,
			Volume: vol,
		})
	}

	return result, nil
}

// tryGetIntradayFromEastMoney fetches intraday data from EastMoney API
func tryGetIntradayFromEastMoney(stockCode string) ([]IntradayDataPoint, error) {
	// Convert stock code for EastMoney API
	emCode := ConvertStockCodeForEastMoney(stockCode)

	// Build URL
	url := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/stock/trends2/get?secid=%s&fields1=f1,f2,f3&fields2=f51,f52,f53,f54,f55&iscr=0",
		emCode,
	)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.eastmoney.com")

	resp, err := fetchWithRetry(client, req, 2) // Retry up to 2 times
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// Parse response
	var emData struct {
		Data struct {
			Trends []string `json:"trends"` // ["2025-11-26 09:31,8.52,12000,...]
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emData); err != nil {
		return nil, err
	}

	if emData.Data.Trends == nil {
		return nil, fmt.Errorf("no trends data")
	}

	// Parse each trend data
	result := make([]IntradayDataPoint, 0, len(emData.Data.Trends))
	for _, trend := range emData.Data.Trends {
		parts := strings.Split(trend, ",")
		if len(parts) < 2 {
			continue
		}

		timeStr := formatIntradayTime(parts[0])
		if timeStr == "" {
			continue
		}

		price, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		// 提取 Volume（parts[2]，东方财富 API 的格式为 "时间,价格,成交量,..."）
		var vol int64
		if len(parts) >= 3 {
			vol, _ = strconv.ParseInt(parts[2], 10, 64)
		}

		result = append(result, IntradayDataPoint{
			Time:   timeStr,
			Price:  price,
			Volume: vol,
		})
	}

	return result, nil
}

// tryGetIntradayFromTencent fetches intraday data from Tencent API (primary source)
func tryGetIntradayFromTencent(stockCode string) ([]IntradayDataPoint, error) {
	// Convert stock code for Tencent API
	tencentCode := ConvertStockCodeForTencent(stockCode)

	// Build URL - Tencent minute data API (JSONP format)
	// Response format: min_data_sh601138={"code":0,"data":{"sh601138":{"data":{"data":["0930 60.88 10989 66901032.00",...]}}}}
	url := fmt.Sprintf(
		"http://ifzq.gtimg.cn/appstock/app/minute/query?_var=min_data_%s&code=%s",
		tencentCode, tencentCode,
	)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://gu.qq.com")

	resp, err := fetchWithRetry(client, req, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSONP response - strip "min_data_XXX=" prefix
	bodyStr := string(body)
	eqIdx := strings.Index(bodyStr, "=")
	if eqIdx == -1 {
		return nil, fmt.Errorf("invalid JSONP response format")
	}
	jsonStr := bodyStr[eqIdx+1:]

	// Parse JSON response
	// Format: {"code":0,"data":{"sh601138":{"data":{"data":["0930 60.88 10989 66901032.00",...]}}}}
	var tencentResp struct {
		Code int `json:"code"`
		Data map[string]struct {
			Data struct {
				Data []string `json:"data"` // Array of "HHMM price volume amount"
			} `json:"data"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &tencentResp); err != nil {
		return nil, err
	}

	if tencentResp.Code != 0 {
		return nil, fmt.Errorf("API error code: %d", tencentResp.Code)
	}

	// Parse data points from all stocks in response
	result := make([]IntradayDataPoint, 0)

	for _, stockData := range tencentResp.Data {
		for _, dataStr := range stockData.Data.Data {
			// Format: "0930 60.88 10989 66901032.00" (time price volume amount)
			parts := strings.Split(dataStr, " ")
			if len(parts) < 2 {
				continue
			}

			// Parse time (format: "0930" -> "09:30")
			timeStr := parts[0]
			if len(timeStr) == 4 {
				timeStr = timeStr[:2] + ":" + timeStr[2:]
			}

			// Parse price
			price, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				continue
			}

			// 提取 Volume（parts[2]，部分 API 可能没有此字段）
			var vol int64
			if len(parts) >= 3 {
				vol, _ = strconv.ParseInt(parts[2], 10, 64)
			}

			result = append(result, IntradayDataPoint{
				Time:   timeStr,
				Price:  price,
				Volume: vol,
			})
		}
	}

	return result, nil
}

// tryGetIntradayFromYahoo fetches intraday data from Yahoo Finance API (for US and HK stocks)
// Yahoo Finance provides free, unlimited intraday data for global stocks
func tryGetIntradayFromYahoo(stockCode string) ([]IntradayDataPoint, error) {
	// Convert stock code for Yahoo Finance API
	yahooSymbol := ConvertStockCodeForYahoo(stockCode)

	// Build URL - Yahoo Finance chart API
	// interval=1m (1 minute), range=1d (1 day), includePrePost=true 包含盘前盘后数据
	url := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1m&range=1d&includePrePost=true",
		yahooSymbol,
	)

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to mimic browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := fetchWithRetry(client, req, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse JSON response
	var yahooResp struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Symbol string `json:"symbol"`
				} `json:"meta"`
				Timestamp  []int64 `json:"timestamp"` // Unix timestamps
				Indicators struct {
					Quote []struct {
						Close  []float64 `json:"close"`  // Closing prices
						Volume []int64   `json:"volume"` // 成交量（新增）
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&yahooResp); err != nil {
		return nil, err
	}

	// Check for API errors
	if yahooResp.Chart.Error != nil {
		return nil, fmt.Errorf("Yahoo API error: %s - %s",
			yahooResp.Chart.Error.Code,
			yahooResp.Chart.Error.Description)
	}

	// Check if we have data
	if len(yahooResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data in Yahoo response")
	}

	result := yahooResp.Chart.Result[0]
	timestamps := result.Timestamp
	quotes := result.Indicators.Quote

	if len(quotes) == 0 || len(quotes[0].Close) == 0 {
		return nil, fmt.Errorf("no price data in Yahoo response")
	}

	closePrices := quotes[0].Close

	// Convert timestamps and prices to IntradayDataPoint
	datapoints := make([]IntradayDataPoint, 0, len(timestamps))

	for i, timestamp := range timestamps {
		// Skip if we don't have a price for this timestamp
		if i >= len(closePrices) {
			break
		}

		price := closePrices[i]
		// Skip null/zero prices
		if price == 0 {
			continue
		}

		// Convert Unix timestamp to time
		t := time.Unix(timestamp, 0)

		// Format time as "HH:MM" in local market timezone
		// Yahoo returns timestamps in UTC, need to convert to market time
		market := string(api.GetMarketType(stockCode))
		var location *time.Location

		switch market {
		case "us":
			location, _ = time.LoadLocation("America/New_York")
		case "hongkong":
			location, _ = time.LoadLocation("Asia/Hong_Kong")
		default:
			location = time.Local
		}

		if location != nil {
			t = t.In(location)
		}

		timeStr := t.Format("15:04") // HH:MM format

		// 提取当前时间点的成交量
		var vol int64
		if len(quotes[0].Volume) > i {
			vol = quotes[0].Volume[i]
		}

		datapoints = append(datapoints, IntradayDataPoint{
			Time:   timeStr,
			Price:  price,
			Volume: vol,
		})
	}

	return datapoints, nil
}

// convertStockCodeForSina converts "SH600000" to "sh600000" for Sina API
// Also handles HK stocks: "HK2020" -> "hk02020" (pads to 5 digits)
func convertStockCodeForSina(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))

	if strings.HasPrefix(code, "SH") {
		return "sh" + strings.TrimPrefix(code, "SH")
	} else if strings.HasPrefix(code, "SZ") {
		return "sz" + strings.TrimPrefix(code, "SZ")
	} else if strings.HasPrefix(code, "HK") {
		// 港股格式: HK00700 -> hk00700, HK2020 -> hk02020
		// 港股代码需要补齐5位数字
		stockNum := strings.TrimPrefix(code, "HK")
		return "hk" + PadHKStockCodeIntraday(stockNum)
	} else if strings.HasSuffix(code, ".HK") {
		// 港股格式: 0700.HK -> hk00700, 2020.HK -> hk02020
		stockNum := strings.TrimSuffix(code, ".HK")
		return "hk" + PadHKStockCodeIntraday(stockNum)
	}

	// 检查是否为纯数字的6位A股代码
	if len(code) == 6 {
		// 根据首位数字判断市场
		if strings.HasPrefix(code, "6") {
			return "sh" + code // 上海
		} else if strings.HasPrefix(code, "0") || strings.HasPrefix(code, "3") {
			return "sz" + code // 深圳
		}
	}

	// 其他格式，直接转小写
	return strings.ToLower(code)
}

// convertStockCodeForEastMoney converts "SH600000" to "1.600000" for EastMoney API
// Also handles HK stocks: "HK00700" -> "116.00700" (Hong Kong market code is 116)
func ConvertStockCodeForEastMoney(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))

	if strings.HasPrefix(code, "SH") {
		// Shanghai market: "1." prefix
		return "1." + code[2:]
	} else if strings.HasPrefix(code, "SZ") {
		// Shenzhen market: "0." prefix
		return "0." + code[2:]
	} else if strings.HasPrefix(code, "HK") {
		// Hong Kong market: "116." prefix
		// 港股格式: HK00700 -> 116.00700, HK2020 -> 116.02020
		stockNum := strings.TrimPrefix(code, "HK")
		return "116." + PadHKStockCodeIntraday(stockNum)
	} else if strings.HasSuffix(code, ".HK") {
		// Hong Kong market: "116." prefix
		// 港股格式: 0700.HK -> 116.00700, 2020.HK -> 116.02020
		stockNum := strings.TrimSuffix(code, ".HK")
		return "116." + PadHKStockCodeIntraday(stockNum)
	}

	// 检查是否为纯数字的6位A股代码
	if len(code) == 6 {
		// 根据首位数字判断市场
		if strings.HasPrefix(code, "6") {
			return "1." + code // 上海
		} else if strings.HasPrefix(code, "0") || strings.HasPrefix(code, "3") {
			return "0." + code // 深圳
		}
	}

	// Default: 其他情况不处理（可能是美股等不支持的市场）
	return code
}

// ConvertStockCodeForTencent converts "SH600000" to "sh600000" for Tencent API
// Also handles HK stocks: "HK2020" -> "hk02020" (pads to 5 digits)
func ConvertStockCodeForTencent(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))

	if strings.HasPrefix(code, "SH") {
		return "sh" + strings.TrimPrefix(code, "SH")
	} else if strings.HasPrefix(code, "SZ") {
		return "sz" + strings.TrimPrefix(code, "SZ")
	} else if strings.HasPrefix(code, "HK") {
		// 港股格式: HK00700 -> hk00700, HK2020 -> hk02020
		// 港股代码需要补齐5位数字
		stockNum := strings.TrimPrefix(code, "HK")
		return "hk" + PadHKStockCodeIntraday(stockNum)
	} else if strings.HasSuffix(code, ".HK") {
		// 港股格式: 0700.HK -> hk00700, 2020.HK -> hk02020
		stockNum := strings.TrimSuffix(code, ".HK")
		return "hk" + PadHKStockCodeIntraday(stockNum)
	}

	// 美股或其他格式，直接转小写
	return strings.ToLower(code)
}

// convertStockCodeForYahoo converts stock code to Yahoo Finance format
// Examples:
//   - AAPL -> AAPL (US stocks keep as-is)
//   - HK00700 -> 0700.HK (Hong Kong stocks)
//   - HK2020 -> 2020.HK (Hong Kong stocks, no need to pad for Yahoo)
func ConvertStockCodeForYahoo(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))

	// Hong Kong stocks: HK00700 -> 0700.HK, HK2020 -> 2020.HK
	if strings.HasPrefix(code, "HK") {
		stockNum := strings.TrimPrefix(code, "HK")
		// Remove leading zeros for Yahoo format
		stockNum = strings.TrimLeft(stockNum, "0")
		if stockNum == "" {
			stockNum = "0"
		}
		return stockNum + ".HK"
	}

	// Already in .HK format
	if strings.HasSuffix(code, ".HK") {
		return code
	}

	// US stocks and others: return as-is
	return code
}

// padHKStockCodeIntraday 将港股代码补齐为5位数字
// 例如: "700" -> "00700", "2020" -> "02020", "00700" -> "00700"
func PadHKStockCodeIntraday(code string) string {
	code = strings.TrimSpace(code)
	if len(code) >= 5 {
		return code
	}
	// 补齐到5位
	return fmt.Sprintf("%05s", code)
}

// fetchWithRetry performs HTTP request with retry mechanism
func fetchWithRetry(client *http.Client, req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		// Clone the request for retry (request body can only be read once)
		reqClone := req.Clone(req.Context())
		resp, err := client.Do(reqClone)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("HTTP status %d", resp.StatusCode)
			resp.Body.Close()
		}
		// Wait before retry (exponential backoff: 500ms, 1000ms, ...)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}
	return nil, lastErr
}
