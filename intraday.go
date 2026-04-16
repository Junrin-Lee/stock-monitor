package main

import (
	"sync"

	"stock-monitor/internal/intraday"
)

// Re-export types from internal/intraday for backward compatibility
type (
	IntradayDataPoint    = intraday.IntradayDataPoint
	IntradayData         = intraday.IntradayData
	DatapointDiffResult  = intraday.DatapointDiffResult
	SaveDecision         = intraday.SaveDecision
)

// Re-export constants
const (
	SaveDecisionSkip   = intraday.SaveDecisionSkip
	SaveDecisionAppend = intraday.SaveDecisionAppend
	SaveDecisionUpdate = intraday.SaveDecisionUpdate
)

// IntradayManager is an alias for the internal implementation
type IntradayManager = intraday.IntradayManager

// newIntradayManager creates and initializes an IntradayManager
func newIntradayManager(model *Model) *IntradayManager {
	// Set up logging functions for the intraday package
	intraday.SetLoggers(logInfoDirect, logDebug)

	// Create adapter for model
	adapter := &modelAdapter{model: model}
	return intraday.NewIntradayManager(adapter)
}

// modelAdapter implements the ModelInterface for the intraday package
type modelAdapter struct {
	model *Model
}

func (a *modelAdapter) GetConfig() interface{} {
	return &configAdapter{config: &a.model.config}
}

func (a *modelAdapter) GetStockPriceCache() map[string]interface{} {
	// 持锁遍历 map，防止 concurrent map iteration and map write 崩溃
	a.model.stockPriceMutex.RLock()
	defer a.model.stockPriceMutex.RUnlock()

	result := make(map[string]interface{}, len(a.model.stockPriceCache))
	for k, v := range a.model.stockPriceCache {
		result[k] = &cacheEntryAdapter{entry: v}
	}
	return result
}

func (a *modelAdapter) GetStockPriceMutex() *sync.RWMutex {
	return &a.model.stockPriceMutex
}

func (a *modelAdapter) GetStockPriceCacheEntry(code string) interface{} {
	a.model.stockPriceMutex.RLock()
	defer a.model.stockPriceMutex.RUnlock()
	if entry, exists := a.model.stockPriceCache[code]; exists {
		return &cacheEntryAdapter{entry: entry}
	}
	return nil
}

// configAdapter adapts Config to the interface expected by intraday package
type configAdapter struct {
	config *Config
}

func (c *configAdapter) GetIntradayCollection() interface {
	GetMaxConsecutiveErrors() int
} {
	return &intradayCollectionAdapter{ic: &c.config.IntradayCollection}
}

func (c *configAdapter) GetMarkets() interface {
	GetChina() interface{ GetTradingSessions() []interface{} }
	GetUS() interface{ GetTradingSessions() []interface{} }
	GetHongKong() interface{ GetTradingSessions() []interface{} }
} {
	return &marketsAdapter{markets: &c.config.Markets}
}

type intradayCollectionAdapter struct {
	ic *IntradayCollectionConfig
}

func (i *intradayCollectionAdapter) GetMaxConsecutiveErrors() int {
	return i.ic.MaxConsecutiveErrors
}

type marketsAdapter struct {
	markets *MarketsConfig
}

func (m *marketsAdapter) GetChina() interface{ GetTradingSessions() []interface{} } {
	return &marketConfigAdapter{mc: &m.markets.China}
}

func (m *marketsAdapter) GetUS() interface{ GetTradingSessions() []interface{} } {
	return &marketConfigAdapter{mc: &m.markets.US}
}

func (m *marketsAdapter) GetHongKong() interface{ GetTradingSessions() []interface{} } {
	return &marketConfigAdapter{mc: &m.markets.HongKong}
}

type marketConfigAdapter struct {
	mc *MarketConfig
}

func (m *marketConfigAdapter) GetTradingSessions() []interface{} {
	result := make([]interface{}, len(m.mc.TradingSessions))
	for i, session := range m.mc.TradingSessions {
		result[i] = &tradingSessionAdapter{ts: &session}
	}
	return result
}

type tradingSessionAdapter struct {
	ts *TradingSession
}

func (t *tradingSessionAdapter) GetStartTime() string {
	return t.ts.StartTime
}

func (t *tradingSessionAdapter) GetEndTime() string {
	return t.ts.EndTime
}

type cacheEntryAdapter struct {
	entry *StockPriceCacheEntry
}

func (c *cacheEntryAdapter) GetPrevClose() float64 {
	if c.entry != nil && c.entry.Data != nil {
		return c.entry.Data.PrevClose
	}
	return 0
}

// Re-export helper functions
var (
	GetTradingDayForCollection = intraday.GetTradingDayForCollection
)
