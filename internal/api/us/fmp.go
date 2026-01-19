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

// TryFMPFreeAPI 使用免费的Financial Modeling Prep API (不需要API key的基础功能)
func TryFMPFreeAPI(symbol string) *types.StockData {
	convertedSymbol := strings.ToUpper(strings.TrimSpace(symbol))
	log.Debug("log.api.fmpSearch", convertedSymbol)

	// 尝试使用免费的实时报价接口
	url := fmt.Sprintf("https://financialmodelingprep.com/api/v3/quote/%s", convertedSymbol)
	log.Debug("log.api.fmpRequestUrl", url)

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("log.api.fmpRequestFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	// 添加用户代理避免被阻止
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; StockMonitor/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		log.Error("log.api.fmpHttpFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("log.api.fmpReadFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	log.Debug("log.api.fmpResponse", string(body))

	// 检查是否是错误响应
	if strings.Contains(string(body), "Error Message") {
		log.Error("log.api.fmpError")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	var results []map[string]any
	if err := json.Unmarshal(body, &results); err != nil {
		log.Error("log.api.fmpJsonFail", err)
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	if len(results) == 0 {
		log.Debug("log.api.fmpNoData")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	result := results[0]

	// 解析价格数据
	var price, previousClose, dayLow, dayHigh, open float64
	var volume int64
	var name string

	if p, ok := result["price"].(float64); ok {
		price = p
	}
	if pc, ok := result["previousClose"].(float64); ok {
		previousClose = pc
	}
	if low, ok := result["dayLow"].(float64); ok {
		dayLow = low
	}
	if high, ok := result["dayHigh"].(float64); ok {
		dayHigh = high
	}
	if o, ok := result["open"].(float64); ok {
		open = o
	}
	if vol, ok := result["volume"].(float64); ok {
		volume = int64(vol)
	}
	if n, ok := result["name"].(string); ok {
		name = n
	}

	if name == "" {
		name = symbol
	}

	if price <= 0 {
		log.Debug("log.api.fmpPriceInvalid")
		return &types.StockData{Symbol: symbol, Price: 0}
	}

	change := price - previousClose
	changePercent := 0.0
	if previousClose > 0 {
		changePercent = (change / previousClose) * 100
	}

	log.Info("log.api.fmpSuccess", name, price, change, changePercent)

	return &types.StockData{
		Symbol:        symbol,
		Name:          name,
		Price:         price,
		Change:        change,
		ChangePercent: changePercent,
		StartPrice:    open,
		MaxPrice:      dayHigh,
		MinPrice:      dayLow,
		PrevClose:     previousClose,
		TurnoverRate:  0,
		Volume:        volume,
	}
}
