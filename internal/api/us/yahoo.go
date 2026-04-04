package us

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stock-monitor/internal/log"
	"stock-monitor/internal/types"
	"strings"
	"time"
)

// TryYahooFinanceAPI 使用Yahoo Finance API作为备用方案
func TryYahooFinanceAPI(symbol string) *types.StockData {
	convertedSymbol := strings.ToUpper(strings.TrimSpace(symbol))
	log.Debug("log.api.yahooSearch", convertedSymbol)

	// 使用Yahoo Finance的chart API接口，这个接口更稳定
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", convertedSymbol)
	log.Debug("log.api.yahooRequestUrl", url)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("log.api.yahooRequestFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	// 添加完整的浏览器请求头以避免被阻止
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		log.Error("log.api.yahooHttpFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		log.Debug("log.api.yahooRateLimit")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("log.api.yahooReadFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	log.Debug("log.api.yahooResponse", string(body))

	var yahooResp struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Symbol               string  `json:"symbol"`
					LongName             string  `json:"longName"`
					ShortName            string  `json:"shortName"`
					RegularMarketPrice   float64 `json:"regularMarketPrice"`
					ChartPreviousClose   float64 `json:"chartPreviousClose"`
					RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
					RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
					RegularMarketVolume  int64   `json:"regularMarketVolume"`
					// 盘前盘后数据（非交易时段才有值）
					PreMarketPrice          float64 `json:"preMarketPrice"`
					PreMarketChange         float64 `json:"preMarketChange"`
					PreMarketChangePercent  float64 `json:"preMarketChangePercent"`
					PostMarketPrice         float64 `json:"postMarketPrice"`
					PostMarketChange        float64 `json:"postMarketChange"`
					PostMarketChangePercent float64 `json:"postMarketChangePercent"`
				} `json:"meta"`
				Indicators struct {
					Quote []struct {
						Open   []float64 `json:"open"`
						High   []float64 `json:"high"`
						Low    []float64 `json:"low"`
						Close  []float64 `json:"close"`
						Volume []int64   `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error any `json:"error"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &yahooResp); err != nil {
		log.Error("log.api.yahooJsonFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	if yahooResp.Chart.Error != nil {
		log.Error("log.api.yahooError", yahooResp.Chart.Error)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	if len(yahooResp.Chart.Result) == 0 {
		log.Debug("log.api.yahooNoData")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	result := yahooResp.Chart.Result[0]
	meta := result.Meta

	if meta.RegularMarketPrice <= 0 {
		log.Debug("log.api.yahooPriceInvalid")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	// 获取开盘价、最高价、最低价
	var openPrice, highPrice, lowPrice float64
	var volume int64

	if len(result.Indicators.Quote) > 0 && len(result.Indicators.Quote[0].Open) > 0 {
		openPrice = result.Indicators.Quote[0].Open[0]
	}
	if len(result.Indicators.Quote) > 0 && len(result.Indicators.Quote[0].High) > 0 {
		highPrice = result.Indicators.Quote[0].High[0]
	}
	if len(result.Indicators.Quote) > 0 && len(result.Indicators.Quote[0].Low) > 0 {
		lowPrice = result.Indicators.Quote[0].Low[0]
	}
	if len(result.Indicators.Quote) > 0 && len(result.Indicators.Quote[0].Volume) > 0 {
		volume = result.Indicators.Quote[0].Volume[0]
	}

	// 如果没有从indicators获取到数据，使用meta中的数据
	if highPrice == 0 {
		highPrice = meta.RegularMarketDayHigh
	}
	if lowPrice == 0 {
		lowPrice = meta.RegularMarketDayLow
	}
	if volume == 0 {
		volume = meta.RegularMarketVolume
	}

	change := meta.RegularMarketPrice - meta.ChartPreviousClose
	changePercent := 0.0
	if meta.ChartPreviousClose > 0 {
		changePercent = (change / meta.ChartPreviousClose) * 100
	}

	name := meta.LongName
	if name == "" {
		name = meta.ShortName
	}
	if name == "" {
		name = symbol
	}

	log.Info("log.api.yahooSuccess", name, meta.RegularMarketPrice, change, changePercent, openPrice, highPrice, lowPrice, volume)

	return &types.StockData{
		Symbol:        symbol,
		Name:          name,
		Price:         meta.RegularMarketPrice,
		Change:        change,
		ChangePercent: changePercent,
		StartPrice:    openPrice,
		MaxPrice:      highPrice,
		MinPrice:      lowPrice,
		PrevClose:     meta.ChartPreviousClose,
		TurnoverRate:  0,
		Volume:        volume,
		// 盘前盘后数据（非交易时段才有值，交易时段为 0）
		PreMarketPrice:    meta.PreMarketPrice,
		PreMarketChange:   meta.PreMarketChange,
		PreMarketPercent:  meta.PreMarketChangePercent,
		PostMarketPrice:   meta.PostMarketPrice,
		PostMarketChange:  meta.PostMarketChange,
		PostMarketPercent: meta.PostMarketChangePercent,
	}
}
