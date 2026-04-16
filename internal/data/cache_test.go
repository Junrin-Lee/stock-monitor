package data

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"stock-monitor/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewStockPriceCache 测试创建缓存实例
func TestNewStockPriceCache(t *testing.T) {
	tests := []struct {
		name        string
		ttl         time.Duration
		expectedTTL time.Duration
		desc        string
	}{
		{
			name:        "正常TTL",
			ttl:         30 * time.Second,
			expectedTTL: 30 * time.Second,
			desc:        "使用指定的TTL",
		},
		{
			name:        "零TTL-使用默认值",
			ttl:         0,
			expectedTTL: 30 * time.Second,
			desc:        "零值使用默认30秒",
		},
		{
			name:        "负数TTL-使用默认值",
			ttl:         -5 * time.Second,
			expectedTTL: 30 * time.Second,
			desc:        "负数使用默认30秒",
		},
		{
			name:        "自定义TTL-1分钟",
			ttl:         60 * time.Second,
			expectedTTL: 60 * time.Second,
			desc:        "自定义1分钟TTL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewStockPriceCache(tt.ttl)

			assert.NotNil(t, cache, "缓存实例不应为空")
			assert.NotNil(t, cache.cache, "内部map不应为空")
			assert.Equal(t, tt.expectedTTL, cache.ttl, "%s: TTL应为 %v", tt.desc, tt.expectedTTL)
			assert.Empty(t, cache.cache, "新缓存应为空")
		})
	}
}

// TestCacheGetSet 测试基本的Get/Set操作
func TestCacheGetSet(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)
	testData := testutil.NewTestStockData("SH601138", "工业富联", 10.5)

	// 初始状态-Get应返回nil
	result := cache.Get("SH601138")
	assert.Nil(t, result, "未设置的股票应返回nil")

	// Set数据
	cache.Set("SH601138", testData)

	// Get应返回数据
	result = cache.Get("SH601138")
	require.NotNil(t, result, "已设置的股票应返回数据")
	assert.Equal(t, testData.Symbol, result.Symbol)
	assert.Equal(t, testData.Name, result.Name)
	assert.Equal(t, testData.Price, result.Price)
}

// TestCacheExpiry 测试缓存过期机制
func TestCacheExpiry(t *testing.T) {
	// 使用很短的TTL便于测试
	cache := NewStockPriceCache(100 * time.Millisecond)
	testData := testutil.NewTestStockData("TEST", "测试股票", 10.0)

	cache.Set("TEST", testData)

	// 立即获取-应该存在
	result := cache.Get("TEST")
	assert.NotNil(t, result, "刚设置的数据应该存在")

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 再次获取-应该为nil
	result = cache.Get("TEST")
	assert.Nil(t, result, "过期的数据应返回nil")
}

// TestIsExpired 测试过期检查
func TestIsExpired(t *testing.T) {
	cache := NewStockPriceCache(200 * time.Millisecond)
	testData := testutil.NewTestStockData("TEST", "测试", 10.0)

	tests := []struct {
		name     string
		symbol   string
		wait     time.Duration
		expected bool
		desc     string
	}{
		{
			name:     "立即检查-未过期",
			symbol:   "TEST",
			wait:     0,
			expected: false,
			desc:     "刚设置的数据未过期",
		},
		{
			name:     "100ms后-未过期",
			symbol:   "TEST",
			wait:     100 * time.Millisecond,
			expected: false,
			desc:     "TTL内的数据未过期",
		},
		{
			name:     "250ms后-已过期",
			symbol:   "TEST",
			wait:     250 * time.Millisecond,
			expected: true,
			desc:     "超过TTL的数据已过期",
		},
		{
			name:     "不存在的股票-视为过期",
			symbol:   "NOTEXIST",
			wait:     0,
			expected: true,
			desc:     "不存在的股票视为过期",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重新设置缓存
			cache.Set("TEST", testData)

			// 等待指定时间
			if tt.wait > 0 {
				time.Sleep(tt.wait)
			}

			// 检查是否过期
			result := cache.IsExpired(tt.symbol)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

// TestSetUpdating 测试更新状态标记
func TestSetUpdating(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)

	// 场景1: 标记不存在的股票为更新中
	cache.SetUpdating("NEW")
	assert.True(t, cache.IsUpdating("NEW"), "新股票应标记为更新中")

	// 场景2: 标记已存在的股票为更新中
	testData := testutil.NewTestStockData("EXIST", "已存在", 10.0)
	cache.Set("EXIST", testData)
	cache.SetUpdating("EXIST")
	assert.True(t, cache.IsUpdating("EXIST"), "已存在股票应标记为更新中")

	// 场景3: 未标记的股票应返回false
	cache.Set("NOT_UPDATING", testData)
	assert.False(t, cache.IsUpdating("NOT_UPDATING"), "未标记的股票应返回false")
}

// TestIsUpdating 测试更新状态查询
func TestIsUpdating(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)

	tests := []struct {
		name     string
		symbol   string
		setup    func()
		expected bool
	}{
		{
			name:   "更新中的股票",
			symbol: "UPDATING",
			setup: func() {
				cache.SetUpdating("UPDATING")
			},
			expected: true,
		},
		{
			name:   "未更新的股票",
			symbol: "NOT_UPDATING",
			setup: func() {
				testData := testutil.NewTestStockData("NOT_UPDATING", "不更新", 10.0)
				cache.Set("NOT_UPDATING", testData)
			},
			expected: false,
		},
		{
			name:     "不存在的股票",
			symbol:   "NOTEXIST",
			setup:    func() {},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.Clear() // 清空缓存
			tt.setup()
			result := cache.IsUpdating(tt.symbol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetEntry 测试获取完整缓存条目
func TestGetEntry(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)
	testData := testutil.NewTestStockData("TEST", "测试", 10.0)

	// 不存在的条目
	entry := cache.GetEntry("NOTEXIST")
	assert.Nil(t, entry, "不存在的条目应返回nil")

	// 存在的条目
	cache.Set("TEST", testData)
	entry = cache.GetEntry("TEST")
	require.NotNil(t, entry, "存在的条目应返回数据")
	assert.Equal(t, testData, entry.Data)
	assert.False(t, entry.IsUpdating, "默认不应处于更新中状态")

	// 修改返回的副本不应影响原始数据
	entry.IsUpdating = true
	originalEntry := cache.GetEntry("TEST")
	assert.False(t, originalEntry.IsUpdating, "修改副本不应影响原始数据")
}

// TestClear 测试清空缓存
func TestClear(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)

	// 添加多个数据
	cache.Set("STOCK1", testutil.NewTestStockData("STOCK1", "股票1", 10.0))
	cache.Set("STOCK2", testutil.NewTestStockData("STOCK2", "股票2", 20.0))
	cache.Set("STOCK3", testutil.NewTestStockData("STOCK3", "股票3", 30.0))

	assert.Equal(t, 3, cache.Size(), "应有3个缓存条目")

	// 清空缓存
	cache.Clear()

	assert.Equal(t, 0, cache.Size(), "清空后应为0")
	assert.Nil(t, cache.Get("STOCK1"), "清空后无法获取数据")
	assert.Nil(t, cache.Get("STOCK2"), "清空后无法获取数据")
	assert.Nil(t, cache.Get("STOCK3"), "清空后无法获取数据")
}

// TestDelete 测试删除单个缓存条目
func TestDelete(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)

	// 添加数据
	cache.Set("DELETE_ME", testutil.NewTestStockData("DELETE_ME", "删除我", 10.0))
	cache.Set("KEEP_ME", testutil.NewTestStockData("KEEP_ME", "保留我", 20.0))

	assert.NotNil(t, cache.Get("DELETE_ME"), "删除前应存在")
	assert.Equal(t, 2, cache.Size(), "应有2个条目")

	// 删除指定条目
	cache.Delete("DELETE_ME")

	assert.Nil(t, cache.Get("DELETE_ME"), "删除后应不存在")
	assert.NotNil(t, cache.Get("KEEP_ME"), "其他条目应保留")
	assert.Equal(t, 1, cache.Size(), "应剩余1个条目")

	// 删除不存在的条目-不应报错
	cache.Delete("NOTEXIST")
	assert.Equal(t, 1, cache.Size(), "大小不应变化")
}

// TestGetAllSymbols 测试获取所有股票代码
func TestGetAllSymbols(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)

	// 空缓存
	symbols := cache.GetAllSymbols()
	assert.Empty(t, symbols, "空缓存应返回空切片")

	// 添加多个股票
	cache.Set("SH601138", testutil.NewTestStockData("SH601138", "工业富联", 10.0))
	cache.Set("AAPL", testutil.NewTestStockData("AAPL", "Apple", 150.0))
	cache.Set("HK00700", testutil.NewTestStockData("HK00700", "腾讯", 380.0))

	symbols = cache.GetAllSymbols()
	assert.Len(t, symbols, 3, "应返回3个股票代码")
	assert.Contains(t, symbols, "SH601138")
	assert.Contains(t, symbols, "AAPL")
	assert.Contains(t, symbols, "HK00700")
}

// TestSize 测试缓存大小统计
func TestSize(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)

	tests := []struct {
		name     string
		action   func()
		expected int
	}{
		{
			name:     "初始大小为0",
			action:   func() {},
			expected: 0,
		},
		{
			name: "添加1个",
			action: func() {
				cache.Set("S1", testutil.NewTestStockData("S1", "股票1", 10.0))
			},
			expected: 1,
		},
		{
			name: "添加3个",
			action: func() {
				cache.Set("S2", testutil.NewTestStockData("S2", "股票2", 20.0))
				cache.Set("S3", testutil.NewTestStockData("S3", "股票3", 30.0))
			},
			expected: 3,
		},
		{
			name: "删除1个",
			action: func() {
				cache.Delete("S1")
			},
			expected: 2,
		},
		{
			name: "清空",
			action: func() {
				cache.Clear()
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.action()
			assert.Equal(t, tt.expected, cache.Size())
		})
	}
}

// TestGetValidCount 测试有效缓存计数
func TestGetValidCount(t *testing.T) {
	cache := NewStockPriceCache(100 * time.Millisecond)

	// 添加3个股票
	cache.Set("S1", testutil.NewTestStockData("S1", "股票1", 10.0))
	cache.Set("S2", testutil.NewTestStockData("S2", "股票2", 20.0))
	cache.Set("S3", testutil.NewTestStockData("S3", "股票3", 30.0))

	// 立即检查-全部有效
	assert.Equal(t, 3, cache.GetValidCount(), "立即检查应全部有效")

	// 等待部分过期
	time.Sleep(60 * time.Millisecond)

	// 添加1个新的
	cache.Set("S4", testutil.NewTestStockData("S4", "股票4", 40.0))

	// S1/S2/S3未完全过期 + S4新鲜
	assert.Equal(t, 4, cache.GetValidCount(), "应有4个有效")

	// 等待S1/S2/S3过期
	time.Sleep(60 * time.Millisecond)

	// 只有S4未过期
	assert.Equal(t, 1, cache.GetValidCount(), "应只剩1个有效")

	// 等待全部过期
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, cache.GetValidCount(), "应全部过期")
}

// TestShouldUpdate 测试更新周期判断
func TestShouldUpdate(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)
	interval := 200 * time.Millisecond

	// 未设置过更新时间-应该更新
	assert.True(t, cache.ShouldUpdate(interval), "初始状态应该更新")

	// 设置更新时间
	cache.SetUpdateTime(time.Now())

	// 立即检查-不应该更新
	assert.False(t, cache.ShouldUpdate(interval), "刚更新不应再次更新")

	// 等待间隔时间
	time.Sleep(250 * time.Millisecond)

	// 再次检查-应该更新
	assert.True(t, cache.ShouldUpdate(interval), "超过间隔应该更新")
}

// TestUpdateTime 测试更新时间的Get/Set
func TestUpdateTime(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)

	// 初始时间应为零值
	initialTime := cache.GetUpdateTime()
	assert.True(t, initialTime.IsZero(), "初始时间应为零值")

	// 设置更新时间
	now := time.Now()
	cache.SetUpdateTime(now)

	// 获取更新时间
	updateTime := cache.GetUpdateTime()
	assert.True(t, updateTime.Equal(now), "更新时间应一致")
}

// TestConcurrency 测试并发安全性
func TestConcurrency(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 10

	// 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				symbol := fmt.Sprintf("STOCK%d", index)
				data := testutil.NewTestStockData(symbol, fmt.Sprintf("股票%d", index), float64(index))
				cache.Set(symbol, data)
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				symbol := fmt.Sprintf("STOCK%d", index)
				cache.Get(symbol)
				cache.IsExpired(symbol)
				cache.IsUpdating(symbol)
			}
		}(i)
	}

	// 并发删除
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				symbol := fmt.Sprintf("STOCK%d", index)
				cache.Delete(symbol)
			}
		}(i)
	}

	// 并发状态标记
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				symbol := fmt.Sprintf("STOCK%d", index)
				cache.SetUpdating(symbol)
			}
		}(i)
	}

	wg.Wait()

	// 验证没有 race condition
	// 如果有并发问题,go test -race 会检测到
	size := cache.Size()
	assert.True(t, size >= 0 && size <= numGoroutines,
		"并发操作后缓存大小应在合理范围: %d", size)
}

// TestTrySetUpdating 测试原子的check-and-set更新标记
func TestTrySetUpdating(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)

	// 场景1: 不存在的股票 - 应成功标记
	assert.True(t, cache.TrySetUpdating("NEW"), "不存在的股票应成功标记")
	assert.True(t, cache.IsUpdating("NEW"), "标记后应处于更新中状态")

	// 场景2: 已标记为更新中的股票 - 应返回false
	assert.False(t, cache.TrySetUpdating("NEW"), "已更新中的股票应返回false")

	// 场景3: 存在但未更新中的股票 - 应成功标记
	testData := testutil.NewTestStockData("EXIST", "已存在", 10.0)
	cache.Set("EXIST", testData)
	assert.True(t, cache.TrySetUpdating("EXIST"), "未更新中的股票应成功标记")
	assert.True(t, cache.IsUpdating("EXIST"), "标记后应处于更新中状态")
}

// TestTrySetUpdatingConcurrent 测试TrySetUpdating在并发下只有一个goroutine成功
func TestTrySetUpdatingConcurrent(t *testing.T) {
	cache := NewStockPriceCache(30 * time.Second)
	numGoroutines := 100
	var successCount int64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if cache.TrySetUpdating("RACE") {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(1), successCount,
		"并发TrySetUpdating应恰好有1个goroutine成功, 实际 %d 个", successCount)
}

// BenchmarkCacheSet 性能基准测试-Set操作
func BenchmarkCacheSet(b *testing.B) {
	cache := NewStockPriceCache(30 * time.Second)
	testData := testutil.NewTestStockData("BENCH", "基准测试", 10.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("BENCH", testData)
	}
}

// BenchmarkCacheGet 性能基准测试-Get操作
func BenchmarkCacheGet(b *testing.B) {
	cache := NewStockPriceCache(30 * time.Second)
	testData := testutil.NewTestStockData("BENCH", "基准测试", 10.0)
	cache.Set("BENCH", testData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("BENCH")
	}
}

// BenchmarkCacheConcurrent 并发性能测试
func BenchmarkCacheConcurrent(b *testing.B) {
	cache := NewStockPriceCache(30 * time.Second)
	testData := testutil.NewTestStockData("BENCH", "基准测试", 10.0)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Set("BENCH", testData)
			cache.Get("BENCH")
		}
	})
}
