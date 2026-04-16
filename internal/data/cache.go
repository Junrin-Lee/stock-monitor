package data

import (
	"sync"
	"time"

	"stock-monitor/internal/types"
)

// StockPriceCache 股价缓存管理器
type StockPriceCache struct {
	cache      map[string]*types.StockPriceCacheEntry
	mutex      sync.RWMutex
	updateTime time.Time
	ttl        time.Duration // 缓存过期时间
}

// NewStockPriceCache 创建股价缓存管理器
func NewStockPriceCache(ttl time.Duration) *StockPriceCache {
	if ttl <= 0 {
		ttl = 30 * time.Second // 默认30秒TTL
	}
	return &StockPriceCache{
		cache: make(map[string]*types.StockPriceCacheEntry),
		ttl:   ttl,
	}
}

// Get 从缓存获取股价数据（非阻塞）
func (c *StockPriceCache) Get(symbol string) *types.StockData {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if entry, exists := c.cache[symbol]; exists {
		// 检查缓存是否过期
		if time.Since(entry.UpdateTime) < c.ttl {
			return entry.Data
		}
	}
	// 如果缓存中没有数据或已过期，返回nil
	return nil
}

// Set 设置股价数据到缓存
func (c *StockPriceCache) Set(symbol string, data *types.StockData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[symbol] = &types.StockPriceCacheEntry{
		Data:       data,
		UpdateTime: time.Now(),
		IsUpdating: false,
	}
}

// SetUpdating 标记股票正在更新
func (c *StockPriceCache) SetUpdating(symbol string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, exists := c.cache[symbol]; exists {
		entry.IsUpdating = true
	} else {
		c.cache[symbol] = &types.StockPriceCacheEntry{
			Data:       nil,
			UpdateTime: time.Time{},
			IsUpdating: true,
		}
	}
}

// TrySetUpdating atomically checks if the stock is already updating,
// and if not, marks it as updating. Returns true if the flag was set
// (caller should proceed with the fetch), false if already updating
// (caller should skip). This prevents the TOCTOU race of separate
// IsUpdating + SetUpdating calls.
func (c *StockPriceCache) TrySetUpdating(symbol string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, exists := c.cache[symbol]; exists {
		if entry.IsUpdating {
			return false
		}
		entry.IsUpdating = true
		return true
	}
	c.cache[symbol] = &types.StockPriceCacheEntry{
		Data:       nil,
		UpdateTime: time.Time{},
		IsUpdating: true,
	}
	return true
}

// IsUpdating 检查股票是否正在更新
func (c *StockPriceCache) IsUpdating(symbol string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if entry, exists := c.cache[symbol]; exists {
		return entry.IsUpdating
	}
	return false
}

// IsExpired 检查缓存是否过期
func (c *StockPriceCache) IsExpired(symbol string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if entry, exists := c.cache[symbol]; exists {
		return time.Since(entry.UpdateTime) >= c.ttl
	}
	return true
}

// GetEntry 获取完整的缓存条目
func (c *StockPriceCache) GetEntry(symbol string) *types.StockPriceCacheEntry {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if entry, exists := c.cache[symbol]; exists {
		// 返回副本以避免并发修改
		return &types.StockPriceCacheEntry{
			Data:       entry.Data,
			UpdateTime: entry.UpdateTime,
			IsUpdating: entry.IsUpdating,
		}
	}
	return nil
}

// Clear 清空缓存
func (c *StockPriceCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*types.StockPriceCacheEntry)
}

// Delete 删除指定股票的缓存
func (c *StockPriceCache) Delete(symbol string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, symbol)
}

// GetUpdateTime 获取上次更新时间
func (c *StockPriceCache) GetUpdateTime() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.updateTime
}

// SetUpdateTime 设置上次更新时间
func (c *StockPriceCache) SetUpdateTime(t time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.updateTime = t
}

// ShouldUpdate 检查是否应该开始新的更新周期
func (c *StockPriceCache) ShouldUpdate(interval time.Duration) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return time.Since(c.updateTime) >= interval
}

// GetAllSymbols 获取所有缓存的股票代码
func (c *StockPriceCache) GetAllSymbols() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	symbols := make([]string, 0, len(c.cache))
	for symbol := range c.cache {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// Size 获取缓存大小
func (c *StockPriceCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.cache)
}

// EvictExpired 删除所有过期的缓存条目，防止内存无限增长
func (c *StockPriceCache) EvictExpired() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	evicted := 0
	for symbol, entry := range c.cache {
		if time.Since(entry.UpdateTime) >= c.ttl && !entry.IsUpdating {
			delete(c.cache, symbol)
			evicted++
		}
	}
	return evicted
}

// GetValidCount 获取有效（未过期）的缓存条目数量
func (c *StockPriceCache) GetValidCount() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	count := 0
	for _, entry := range c.cache {
		if time.Since(entry.UpdateTime) < c.ttl {
			count++
		}
	}
	return count
}
