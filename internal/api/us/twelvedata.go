package us

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stock-monitor/internal/log"
	"stock-monitor/internal/types"
	"strconv"
	"strings"
	"time"
)

// SearchStockByTwelveDataAPI 使用TwelveData搜索API查找股票
func SearchStockByTwelveDataAPI(keyword string) *types.StockData {
	log.Debug("log.api.twelveDataSearchStart", keyword)

	// 先尝试符号搜索
	searchUrl := fmt.Sprintf("https://api.twelvedata.com/symbol_search?symbol=%s&apikey=demo", keyword)
	log.Debug("log.api.twelveDataSearchUrl", searchUrl)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(searchUrl)
	if err != nil {
		log.Error("log.api.twelveDataSearchHttpFail", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body) // drain body to allow connection reuse
		log.Error("log.api.twelveDataSearchStatusFail", resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("log.api.twelveDataSearchReadFail", err)
		return nil
	}

	log.Debug("log.api.twelveDataSearchResponse", string(body))

	var searchResult struct {
		Data []struct {
			Symbol         string `json:"symbol"`
			InstrumentName string `json:"instrument_name"`
			Exchange       string `json:"exchange"`
			Country        string `json:"country"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &searchResult); err != nil {
		log.Error("log.api.twelveDataSearchJsonFail", err)
		return nil
	}

	if len(searchResult.Data) == 0 {
		log.Debug("log.api.twelveDataSearchNoMatch")
		return nil
	}

	// 选择第一个匹配的结果，优先选择美国市场的股票
	var selectedSymbol, selectedName string
	for _, item := range searchResult.Data {
		if item.Country == "United States" && item.Exchange == "NASDAQ" {
			selectedSymbol = item.Symbol
			selectedName = item.InstrumentName
			break
		}
	}

	// 如果没有找到美国NASDAQ的，就用第一个结果
	if selectedSymbol == "" {
		selectedSymbol = searchResult.Data[0].Symbol
		selectedName = searchResult.Data[0].InstrumentName
	}

	log.Debug("log.api.twelveDataSearchSelect", selectedName, selectedSymbol)

	// 获取股票报价
	return TryTwelveDataAPI(selectedSymbol)
}

// TryTwelveDataAPI 使用TwelveData API获取股票报价
func TryTwelveDataAPI(symbol string) *types.StockData {
	convertedSymbol := strings.ToUpper(strings.TrimSpace(symbol))
	log.Debug("log.api.twelveDataConvert", symbol, convertedSymbol)

	// 使用TwelveData API获取股票报价
	url := fmt.Sprintf("https://api.twelvedata.com/quote?symbol=%s&apikey=demo", convertedSymbol)
	log.Debug("log.api.twelveDataUrl", url)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Error("log.api.twelveDataHttpFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body) // drain body to allow connection reuse
		log.Error("log.api.twelveDataStatusFail", resp.StatusCode)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("log.api.twelveDataReadFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	log.Debug("log.api.twelveDataResponse", string(body))

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		log.Error("log.api.twelveDataJsonFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	// 检查是否有错误信息
	if errMsg, hasErr := result["message"]; hasErr {
		log.Error("log.api.twelveDataApiError", errMsg)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	// 解析股票数据
	name, _ := result["name"].(string)
	if name == "" {
		name = symbol
	}

	closeStr, closeOk := result["close"].(string)
	prevCloseStr, prevOk := result["previous_close"].(string)

	if !closeOk || !prevOk {
		log.Debug("log.api.twelveDataInvalidData")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	current, err := strconv.ParseFloat(closeStr, 64)
	if err != nil {
		log.Error("log.api.twelveDataPriceFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	previous, err := strconv.ParseFloat(prevCloseStr, 64)
	if err != nil {
		log.Error("log.api.twelveDataPrevCloseFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	if current <= 0 {
		log.Debug("log.api.twelveDataInvalidPrice")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	// 解析开盘价、最高价、最低价、成交量
	var openPrice, maxPrice, minPrice float64
	var volume int64

	if openStr, ok := result["open"].(string); ok {
		openPrice, _ = strconv.ParseFloat(openStr, 64)
	}
	if highStr, ok := result["high"].(string); ok {
		maxPrice, _ = strconv.ParseFloat(highStr, 64)
	}
	if lowStr, ok := result["low"].(string); ok {
		minPrice, _ = strconv.ParseFloat(lowStr, 64)
	}
	if volumeStr, ok := result["volume"].(string); ok {
		volume, _ = strconv.ParseInt(volumeStr, 10, 64)
	}

	change := current - previous
	changePercent := 0.0
	if previous > 0 {
		changePercent = (change / previous) * 100
	}

	log.Info("log.api.twelveDataGetSuccess",
		name, current, change, changePercent, openPrice, maxPrice, minPrice, volume)

	return &types.StockData{
		Symbol:        symbol,
		Name:          name,
		Price:         current,
		Change:        change,
		ChangePercent: changePercent,
		StartPrice:    openPrice,
		MaxPrice:      maxPrice,
		MinPrice:      minPrice,
		PrevClose:     previous,
		TurnoverRate:  0, // TwelveData不提供换手率
		Volume:        volume,
	}
}
