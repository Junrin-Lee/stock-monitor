package intraday

import (
	"time"

	"stock-monitor/internal/api"
	"stock-monitor/internal/types"
)

// startSmartWorker 启动带有自动停止逻辑的智能 worker
// stockCode: 股票代码
// stockName: 股票名称
// targetDate: 目标日期 (YYYYMMDD)
// mode: 采集模式 (Historical/Live)
func (im *IntradayManager) startSmartWorker(stockCode, stockName, targetDate string, mode types.CollectionMode) {
	// 注：activeStocks[stockCode] = true 已在 StartCollection 中设置，此处无需重复

	// 清理函数
	defer func() {
		im.mu.Lock()
		delete(im.activeStocks, stockCode)
		im.mu.Unlock()

		im.metadataMutex.Lock()
		delete(im.workerMetadata, stockCode)
		im.metadataMutex.Unlock()

		logInfoDirect("[Intraday] Worker stopped for %s", stockCode)
	}()

	// 创建定时器（1分钟间隔）
	ticker := time.NewTicker(im.fetchInterval)
	defer ticker.Stop()

	// 获取配置（默认值）
	maxConsecutiveErrors := 5
	config := im.model.GetConfig()
	if cfg, ok := config.(interface {
		GetIntradayCollection() interface {
			GetMaxConsecutiveErrors() int
		}
	}); ok {
		if cfg.GetIntradayCollection().GetMaxConsecutiveErrors() > 0 {
			maxConsecutiveErrors = cfg.GetIntradayCollection().GetMaxConsecutiveErrors()
		}
	}

	// 初始立即获取一次（跳过市场时间检查，使用 targetDate）
	im.fetchAndSaveIntradayData(stockCode, stockName, im.model, false, targetDate)

	// 主循环
	for {
		select {
		case <-ticker.C:
			// === 步骤 1: 检查自动停止条件（在 fetch 前检查，避免读取旧数据） ===
			// 在锁内复制需要的值，防止与写入 goroutine 竞态
			im.metadataMutex.RLock()
			meta, exists := im.workerMetadata[stockCode]
			var consecutiveErrors, consecutiveSkips int
			if exists {
				consecutiveErrors = meta.ConsecutiveErrors
				consecutiveSkips = meta.ConsecutiveSkips
			}
			im.metadataMutex.RUnlock()

			if !exists {
				return // 元数据丢失，停止
			}

			// 条件 1: 连续错误过多
			if consecutiveErrors >= maxConsecutiveErrors {
				logInfoDirect("[Intraday] Worker for %s stopped: %d consecutive errors",
					stockCode, consecutiveErrors)
				return
			}

			// 条件 2: 连续 Skip 次数过多（数据完全一致）
			maxConsecutiveSkips := 3 // 连续 3 次数据完全一致即停止
			if consecutiveSkips >= maxConsecutiveSkips {
				logDebug("log.intraday.consecutiveSkips", stockCode, consecutiveSkips)
				logDebug("log.intraday.stopDataStable", stockCode)
				return
			}

			// 条件 3: Historical 模式 + 数据完整
			if mode == types.CollectionModeHistorical {
				marketType := string(api.GetMarketType(stockCode))
				complete, err := IsDataComplete(stockCode, targetDate, marketType, false)
				if err == nil && complete {
					logInfoDirect("[Intraday] Worker for %s stopped: historical data complete for %s",
						stockCode, targetDate)
					return
				}
			}

			// 条件 4: Live 模式 + 市场关闭 + 数据完整
			if mode == types.CollectionModeLive {
				marketType := string(api.GetMarketType(stockCode))
				location, _ := GetMarketLocation(marketType)
				if location != nil {
					now := time.Now().In(location)
					tradingState := GetTradingState(now, marketType)

					if tradingState == types.TradingStatePostMarket {
						complete, err := IsDataComplete(stockCode, targetDate, marketType, false)
						if err == nil && complete {
							logDebug("log.intraday.stopPostMarketComplete", stockCode, targetDate)
							return
						}
					}
				}
			}

			// === 步骤 2: 获取并保存数据 ===
			// 对于 Live 模式，检查市场是否开市（或处于集合竞价时段）
			checkMarketHours := true
			if mode == types.CollectionModeLive {
				if !IsMarketOpen(stockCode, im.model) {
					// IsMarketOpen 不覆盖集合竞价窗口，需单独检查交易状态
					marketType := string(api.GetMarketType(stockCode))
					location, _ := GetMarketLocation(marketType)
					allowCollection := false
					if location != nil {
						now := time.Now().In(location)
						switch GetTradingState(now, marketType) {
						case types.TradingStateAuction:
							// 竞价时段：允许采集，且跳过 fetchAndSave 内部的市场检查
							// 重置计数器防止竞价期间 API 空数据/重复快照触发 smart-stop
							allowCollection = true
							checkMarketHours = false
							im.metadataMutex.Lock()
							if meta, exists := im.workerMetadata[stockCode]; exists {
								meta.ConsecutiveErrors = 0
								meta.ConsecutiveSkips = 0
							}
							im.metadataMutex.Unlock()
						case types.TradingStateLunchBreak:
							// 午休期间重置连续跳过计数，防止下午开盘后误触自动停止
							im.metadataMutex.Lock()
							if meta, exists := im.workerMetadata[stockCode]; exists {
								meta.ConsecutiveSkips = 0
							}
							im.metadataMutex.Unlock()
						}
					}
					if !allowCollection {
						continue // 市场未开且非竞价，跳过本次采集
					}
				}
			}

			//获取 worker 槽位（限制并发数），支持取消
			checkHours := checkMarketHours // 捕获到 goroutine 闭包
			select {
			case im.workerPool <- struct{}{}:
			case <-im.CancelChan:
				return
			}
			go func() {
				defer func() { <-im.workerPool }()
				decision, err := im.fetchAndSaveIntradayData(stockCode, stockName, im.model, checkHours, targetDate)

				// 更新元数据
				im.metadataMutex.Lock()
				if meta, exists := im.workerMetadata[stockCode]; exists {
					meta.LastUpdateTime = time.Now()
					if err != nil {
						meta.ConsecutiveErrors++
						meta.ConsecutiveSkips = 0 // 重置连续跳过计数
					} else {
						meta.ConsecutiveErrors = 0
						// 根据 SaveDecision 更新计数器
						if decision == SaveDecisionSkip {
							meta.ConsecutiveSkips++
						} else {
							meta.ConsecutiveSkips = 0 // 有数据变化，重置连续跳过计数
						}
					}
				}
				im.metadataMutex.Unlock()
			}()

		case <-im.CancelChan:
			// 全局取消信号
			return
		}
	}
}
