package main

import (
	"fmt"
	"stock-monitor/internal/api"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// 股价缓存管理
// ============================================================================

// getStockPriceFromCache 从缓存获取股价数据（非阻塞）
func (m *Model) getStockPriceFromCache(symbol string) *StockData {
	m.stockPriceMutex.RLock()
	defer m.stockPriceMutex.RUnlock()
	if entry, exists := m.stockPriceCache[symbol]; exists {
		// 检查缓存是否过期（超过30秒）
		if time.Since(entry.UpdateTime) < 30*time.Second {
			return entry.Data
		}
	}
	// 如果缓存中没有数据或已过期，返回nil，触发异步更新
	return nil
}

// ============================================================================
// 股价异步更新
// ============================================================================

// startStockPriceUpdates 启动股价异步更新
func (m *Model) startStockPriceUpdates() tea.Cmd {
	// 检查是否需要开始新的更新周期
	if time.Since(m.stockPriceUpdateTime) < 5*time.Second {
		logDebug("log.cache.skipUpdate", time.Since(m.stockPriceUpdateTime))
		return nil // 还未到更新时间
	}

	// 收集所有需要更新的股票代码
	stockCodes := make([]string, 0)

	// 添加自选列表中的股票 - 注意：这里应该获取所有自选股票，而不是过滤后的
	for _, stock := range m.watchlist.Stocks {
		stockCodes = append(stockCodes, stock.Code)
	}

	// 添加持股列表中的股票
	for _, stock := range m.portfolio.Stocks {
		stockCodes = append(stockCodes, stock.Code)
	}

	if len(stockCodes) == 0 {
		logDebug("log.cache.noStocks")
		return nil
	}

	// 去重股票代码
	uniqueCodes := make(map[string]bool)
	var uniqueStockCodes []string
	for _, code := range stockCodes {
		if !uniqueCodes[code] {
			uniqueCodes[code] = true
			uniqueStockCodes = append(uniqueStockCodes, code)
		}
	}

	// 更新开始时间
	m.stockPriceUpdateTime = time.Now()

	logDebug("log.cache.startAsync", len(uniqueStockCodes))

	// 逐个发起异步获取请求（跳过已在更新中的股票，防止请求堆积）
	// 使用 atomic check-and-set 模式，在单次 Lock 内完成检查+标记，避免 TOCTOU 竞态
	var cmds []tea.Cmd
	for _, code := range uniqueStockCodes {
		if !m.trySetStockUpdating(code) {
			continue // 已在更新中，跳过
		}

		// 为每个股票添加一个延迟，避免同时请求太多
		delay := time.Duration(len(cmds)) * 100 * time.Millisecond
		// 修复闭包问题：将code变量复制到局部变量
		stockCode := code

		// 创建异步命令：延迟后执行 fetchStockPriceCmd
		cmds = append(cmds, tea.Tick(delay, func(t time.Time) tea.Msg {
			// 返回一个消息，触发实际的获取命令
			return fetchStockPriceTriggerMsg{symbol: stockCode}
		}))
	}

	return tea.Batch(cmds...)
}

// evictStaleStockPriceCache removes cache entries for stocks no longer in portfolio or watchlist,
// preventing unbounded memory growth during long-running sessions.
func (m *Model) evictStaleStockPriceCache() {
	// Build set of active codes
	activeCodes := make(map[string]bool)
	for _, stock := range m.portfolio.Stocks {
		activeCodes[stock.Code] = true
	}
	for _, stock := range m.watchlist.Stocks {
		activeCodes[stock.Code] = true
	}

	m.stockPriceMutex.Lock()
	defer m.stockPriceMutex.Unlock()

	for code, entry := range m.stockPriceCache {
		if !activeCodes[code] && !entry.IsUpdating && time.Since(entry.UpdateTime) >= 30*time.Second {
			delete(m.stockPriceCache, code)
		}
	}
}

// trySetStockUpdating atomically checks and sets the IsUpdating flag for a stock.
// Returns true if the caller should proceed with the fetch (flag was set),
// false if another fetch is already in progress (caller should skip).
// This prevents the TOCTOU race of separate check + set with unlock in between.
func (m *Model) trySetStockUpdating(code string) bool {
	m.stockPriceMutex.Lock()
	defer m.stockPriceMutex.Unlock()

	if entry, exists := m.stockPriceCache[code]; exists {
		if entry.IsUpdating {
			return false
		}
		entry.IsUpdating = true
		return true
	}
	m.stockPriceCache[code] = &StockPriceCacheEntry{
		Data:       nil,
		UpdateTime: time.Time{},
		IsUpdating: true,
	}
	return true
}

// fetchStockPriceTriggerMsg 触发股价获取的消息
type fetchStockPriceTriggerMsg struct {
	symbol string
}

// fetchStockPriceCmd 异步获取单个股票价格（正确的 Bubble Tea 模式）
func fetchStockPriceCmd(symbol string) tea.Cmd {
	return func() tea.Msg {
		// 在后台 goroutine 中执行 API 调用
		data := api.GetStockPrice(symbol)

		var err error
		if data == nil || data.Price <= 0 {
			err = fmt.Errorf("failed to get stock price for %s", symbol)
		}

		// 返回消息，由 Update() 方法处理
		return stockPriceUpdateMsg{
			Symbol: symbol,
			Data:   data,
			Error:  err,
		}
	}
}
