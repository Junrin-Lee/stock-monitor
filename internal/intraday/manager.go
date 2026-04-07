package intraday

import (
	"fmt"
	"sync"
	"time"

	"stock-monitor/internal/types"
)

// IntradayManager manages background fetching of intraday data
type IntradayManager struct {
	activeStocks   map[string]bool                   // Currently tracking stocks
	workerPool     chan struct{}                     // Semaphore for max 10 concurrent workers
	CancelChan     chan struct{}                     // Channel to stop all workers (exported for intraday_chart.go)
	mu             sync.RWMutex                      // Protects activeStocks
	lastFetchTime  map[string]time.Time              // Track last fetch per stock
	fetchInterval  time.Duration                     // 1 minute
	workerMetadata map[string]*types.WorkerMetadata  // Track each worker's state
	metadataMutex  sync.RWMutex                      // Protects workerMetadata
	model          ModelInterface                    // Reference to main model
}

// ModelInterface defines the interface for model access
type ModelInterface interface {
	GetConfig() interface{}
	GetStockPriceCache() map[string]interface{}
	GetStockPriceMutex() *sync.RWMutex
}

// NewIntradayManager creates and initializes an IntradayManager
func NewIntradayManager(model ModelInterface) *IntradayManager {
	return &IntradayManager{
		activeStocks:   make(map[string]bool),
		workerPool:     make(chan struct{}, 10), // Max 10 concurrent workers
		CancelChan:     make(chan struct{}),
		lastFetchTime:  make(map[string]time.Time),
		fetchInterval:  1 * time.Minute,
		workerMetadata: make(map[string]*types.WorkerMetadata),
		model:          model,
	}
}

// StartCollection 开始智能数据采集（入口点）
// 根据市场状态和数据完整性自动决定采集策略
// stockCode: 股票代码
// stockName: 股票名称
// 返回: error（如果启动失败）
func (im *IntradayManager) StartCollection(stockCode, stockName string) error {
	// 步骤 1: 决定应该采集什么数据
	targetDate, mode, err := GetTradingDayForCollection(stockCode, im.model)
	if err != nil {
		return fmt.Errorf("failed to determine collection params for %s: %w", stockCode, err)
	}

	// 步骤 2: 如果数据已完整，跳过启动 worker
	if mode == types.CollectionModeComplete {
		logInfoDirect("[Intraday] Skipping %s: data already complete for %s",
			stockCode, targetDate)
		return nil
	}

	// 步骤 3: 检查是否已有 worker 在运行，同时标记为活动（原子操作防止 TOCTOU）
	im.mu.Lock()
	if im.activeStocks[stockCode] {
		im.mu.Unlock()
		logInfoDirect("[Intraday] Worker already running for %s", stockCode)
		return nil
	}
	im.activeStocks[stockCode] = true
	im.mu.Unlock()

	// 步骤 4: 初始化 worker 元数据
	im.metadataMutex.Lock()
	im.workerMetadata[stockCode] = &types.WorkerMetadata{
		StockCode:         stockCode,
		TargetDate:        targetDate,
		Mode:              mode,
		StartTime:         time.Now(),
		LastUpdateTime:    time.Now(),
		DatapointCount:    0,
		ConsecutiveErrors: 0,
		ConsecutiveSkips:  0, // 初始化连续跳过计数
		IsRunning:         true,
	}
	im.metadataMutex.Unlock()

	// 步骤 5: 启动智能 worker
	logInfoDirect("[Intraday] Starting %s collection for %s (target: %s)",
		mode.String(), stockCode, targetDate)

	go im.startSmartWorker(stockCode, stockName, targetDate, mode)

	return nil
}

// startWorker launches a background goroutine to fetch data for one stock
func (im *IntradayManager) startWorker(stockCode, stockName string, m ModelInterface) {
	// Prevent duplicate workers
	im.mu.Lock()
	if im.activeStocks[stockCode] {
		im.mu.Unlock()
		return
	}
	im.activeStocks[stockCode] = true
	im.mu.Unlock()

	go func() {
		// Cleanup function
		defer func() {
			im.mu.Lock()
			delete(im.activeStocks, stockCode)
			im.mu.Unlock()
			logDebug("log.intraday.workerStop", stockCode)
		}()

		logDebug("log.intraday.workerStart", stockCode, stockName)

		// Create ticker
		ticker := time.NewTicker(im.fetchInterval)
		defer ticker.Stop()

		// Initial fetch (skip market hours check to get today's data even after market close)
		// 使用空字符串作为 targetDate，让函数自动根据交易状态计算日期
		im.fetchAndSaveIntradayData(stockCode, stockName, m, false, "")

		// Periodic loop
		for {
			select {
			case <-ticker.C:
				// Check market hours for periodic updates
				if !IsMarketOpen(stockCode, m) {
					continue
				}

				// Acquire worker slot (blocks if all 10 slots are busy)
				im.workerPool <- struct{}{}

				// Fetch with timeout
				go func() {
					defer func() {
						<-im.workerPool // Release slot
					}()
					// 使用空字符串作为 targetDate，让函数自动根据交易状态计算日期
					im.fetchAndSaveIntradayData(stockCode, stockName, m, true, "")
				}()

			case <-im.CancelChan:
				return // Graceful exit
			}
		}
	}()
}

// logInfoDirect is a placeholder for direct logging (to be implemented by caller)
var logInfoDirect = func(format string, args ...interface{}) {
	// Default no-op, should be set by caller
}

// logDebug is a placeholder for debug logging (to be implemented by caller)
var logDebug = func(key string, args ...interface{}) {
	// Default no-op, should be set by caller
}

// SetLoggers sets the logging functions for the intraday package
func SetLoggers(infoLogger func(string, ...interface{}), debugLogger func(string, ...interface{})) {
	logInfoDirect = infoLogger
	logDebug = debugLogger
}
