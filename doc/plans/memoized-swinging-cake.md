# Stock Monitor v9.0 开发计划 — Sparkline + 图表强化 + 盘前盘后/集合竞价

## Context

stock-monitor v8.1 已有完整的分时数据采集基础设施（Worker Pool、多市场、多 API 容灾），但图表展示仅有单日全屏分时线图。竞品 tickrs 已有 K线/多周期图表，ticker 支持盘前盘后，但**没有任何竞品实现表格内嵌 Sparkline**。

用户选定两个发展方向：
1. **Sparkline + 图表强化** — 内嵌迷你趋势图 + 多周期 + 成交量
2. **盘前盘后 + 集合竞价** — 美股扩展时段 + A股/港股竞价数据

---

## 一、公共能力提取（跨功能复用）

### 1.1 新建公共模块：`internal/ui/sparkline/sparkline.go`

**可复用场景**：投资组合表格、自选股表格、板块成分股表格、搜索结果表格

```go
package sparkline

// Generate 生成 Unicode 块字符迷你趋势图
// prices: 价格序列
// width: 目标字符宽度（推荐 12）
// upColor/downColor: 涨/跌颜色（由调用方传入，适配中英文色彩规则）
func Generate(prices []float64, width int, upColor, downColor, flatColor string) string

// GenerateWithDefaults 使用默认配置生成（便捷接口）
func GenerateWithDefaults(prices []float64, isAShare bool, lang string) string
```

**设计要点**：
- 使用 `▁▂▃▄▅▆▇█` 8级 Unicode 块字符，单行高度
- 降采样算法：`prices` 长度 > `width` 时，使用 LTTB（最大三角面积）算法或简单等距采样
- 颜色参数由调用方决定，而非内部判断 — 因为 A股（红涨绿跌）和美股（绿涨红跌）规则不同
- 复用 `internal/ui/format.go:Formatter` 的颜色逻辑判断涨跌方向

**为什么是公共能力**：板块成分股列表（`handlers_sector.go`）、搜索结果表格（`handlers_search.go`）未来也可展示趋势图，提取为独立包避免代码重复。

### 1.2 新建公共模块：`internal/intraday/loader.go`

**可复用场景**：Sparkline 数据加载、多周期图表数据加载、搜索模式图表

```go
package intraday

// LoadSingleDay 加载单日分时数据（现有 handlers_chart.go:96 loadIntradayDataForDate 的下沉）
func LoadSingleDay(code, date string) (*IntradayData, error)

// LoadMultiDay 加载最近 N 个交易日的分时数据（多周期图表用）
// 返回按日期排序的数据点切片，跨日数据已拼接
func LoadMultiDay(code string, days int) ([]IntradayDataPoint, []string, error)

// LoadLatestPrices 仅加载最新交易日的价格序列（Sparkline 专用，轻量级）
// 返回 []float64 价格数组，失败返回 nil
func LoadLatestPrices(code, market string) []float64

// Downsample 降采样算法（将 N 个点降到 targetWidth 个点）
func Downsample(prices []float64, targetWidth int) []float64
```

**设计要点**：
- `LoadSingleDay` 是从 `handlers_chart.go:96` 的 `loadIntradayDataForDate()` 提取而来
- `LoadLatestPrices` 是轻量级版本，不解析完整 `IntradayData`，只返回 `[]float64`
- `Downsample` 作为公共降采样工具，被 Sparkline 和多周期图表共用
- 复用现有 `storage.go` 的 `GetIntradayFilePath()` 做路径查找（含向后兼容）

**为什么是公共能力**：目前 `loadIntradayDataForDate()` 在 `handlers_chart.go` 中是 Model 方法，外部无法调用。搜索模式、Sparkline、多周期图表都需要独立于 Model 加载数据。

### 1.3 扩展公共格式化：`internal/ui/format.go` 添加盘前盘后格式化

```go
// FormatPrePostMarketPrice 格式化盘前/盘后价格（带涨跌颜色 + 时段标签）
func (f *Formatter) FormatPrePostMarketPrice(price, prevClose float64, sessionType string) string

// FormatPrePostChange 格式化盘前/盘后涨跌幅
func (f *Formatter) FormatPrePostChange(change, changePercent float64) string
```

**为什么添加到 Formatter**：`Formatter` 已封装了双语涨跌颜色逻辑（`FormatPriceWithColor`, `FormatProfitRateWithColor`），盘前盘后格式化是同一逻辑的扩展。

### 1.4 扩展公共数据结构

#### `internal/intraday/types.go` — IntradayDataPoint 添加 Volume

```go
type IntradayDataPoint struct {
    Time   string  `json:"time"`
    Price  float64 `json:"price"`
    Volume int64   `json:"volume,omitempty"`  // 新增
}
```

- `omitempty` 确保读取旧数据文件时不报错（向后兼容）
- 四个分时 API 都已返回 volume 数据，只需在解析时提取

#### `internal/types/stock.go` — StockData 添加盘前盘后字段

```go
type StockData struct {
    // ... 现有字段 ...
    PreMarketPrice    float64 `json:"pre_market_price,omitempty"`
    PreMarketChange   float64 `json:"pre_market_change,omitempty"`
    PreMarketPercent  float64 `json:"pre_market_percent,omitempty"`
    PostMarketPrice   float64 `json:"post_market_price,omitempty"`
    PostMarketChange  float64 `json:"post_market_change,omitempty"`
    PostMarketPercent float64 `json:"post_market_percent,omitempty"`
}
```

**根目录 `types.go` 同步**：根目录 `StockData`（约 L23-35）也需要添加相同字段（因遗留代码尚未迁移）。

---

## 二、功能 1：表格内嵌 Sparkline — 模块级开发细节

### 2.1 涉及模块与修改点

| 模块 | 文件 | 修改点 | 行号参考 |
|------|------|--------|----------|
| **新建** | `internal/ui/sparkline/sparkline.go` | Sparkline 生成公共能力 | 新文件 |
| **新建** | `internal/intraday/loader.go` | 数据加载公共能力 | 新文件 |
| 列注册 | `internal/ui/columns.go` | 添加 `ColTrend ColumnID = "trend"` 常量 (L14-34) + 注册到 portfolio 和 watchlist registry | L62-160, L163-247 |
| 行生成 | `main.go` | `GeneratePortfolioRow()` (L699-759) switch 中添加 `case ui.ColTrend:` | L699+ |
| 行生成 | `main.go` | `GenerateWatchlistRow()` (L785-860) switch 中添加 `case ui.ColTrend:` | L785+ |
| 缓存 | `main.go` 或 `types.go` | Model 结构体添加 `sparklineCache map[string]string` + `sparklineCacheTime time.Time` | types.go |
| i18n | `i18n/zh.json`, `i18n/en.json` | 添加 `"col.trend": "走势"` / `"Trend"` | - |
| 配置 | `cmd/conf/config_demo.yaml` | `portfolio_columns` 和 `watchlist_columns` 默认值添加 `trend` | - |

### 2.2 详细代码修改

#### A. `internal/ui/columns.go` — 注册 trend 列

**Step 1**: 在 L34 后添加常量：
```go
ColTrend ColumnID = "trend"  // 走势迷你图（Sparkline）
```

**Step 2**: 在 `makePortfolioColumnRegistry()` (L62) 的 return map 中添加：
```go
ColTrend: {
    ID:         ColTrend,
    I18nKey:    "col.trend",
    IsRequired: false,
    SortField:  nil, // 不可排序
},
```

**Step 3**: 在 `makeWatchlistColumnRegistry()` (L163) 的 return map 中添加相同条目。

**效果**：列配置管道自动生效 — 用户在 `config.yml` 的 `portfolio_columns` 中添加 `trend` 即可显示。

#### B. `main.go` — GeneratePortfolioRow() 添加 Sparkline 单元格

在 `GeneratePortfolioRow()` (L699) 的 switch 中添加：
```go
case ui.ColTrend:
    row[i] = m.getSparklineForStock(stock.Code)
```

新增 Model 方法：
```go
func (m *Model) getSparklineForStock(code string) string {
    // 1. 检查缓存（5秒 TTL）
    if cached, ok := m.sparklineCache[code]; ok {
        if time.Since(m.sparklineCacheTime) < 5*time.Second {
            return cached
        }
    }
    // 2. 加载数据（复用 intraday.LoadLatestPrices）
    market := string(api.GetMarketType(code))
    prices := intraday.LoadLatestPrices(code, market)
    if prices == nil || len(prices) < 3 {
        return "────────────" // 无数据占位符
    }
    // 3. 生成 Sparkline（复用 sparkline.Generate）
    isAShare := strings.HasPrefix(code, "SH") || strings.HasPrefix(code, "SZ")
    result := sparkline.GenerateWithDefaults(prices, isAShare, string(m.config.System.Language))
    // 4. 缓存
    if m.sparklineCache == nil {
        m.sparklineCache = make(map[string]string)
    }
    m.sparklineCache[code] = result
    m.sparklineCacheTime = time.Now()
    return result
}
```

**GenerateWatchlistRow() (L785) 同理**，添加相同的 `case ui.ColTrend:` 处理。

#### C. 缓存设计

Model 新增字段（`types.go` 根目录）：
```go
sparklineCache     map[string]string  // code -> sparkline string
sparklineCacheTime time.Time          // 上次全量刷新时间
```

缓存策略：TTL 与配置文件中的 `update.refresh_interval` 保持一致，**不硬编码**。

```go
func (m *Model) getSparklineForStock(code string) string {
    // TTL = m.config.Update.RefreshInterval 秒（与价格刷新周期对齐）
    ttl := time.Duration(m.config.Update.RefreshInterval) * time.Second
    if cached, ok := m.sparklineCache[code]; ok {
        if time.Since(m.sparklineCacheTime) < ttl {
            return cached
        }
    }
    // ... 其余逻辑不变
}
```

**设计理由**：Sparkline 反映的是分时价格趋势，数据新鲜度应与价格刷新保持一致。用户调整 `refresh_interval` 时，Sparkline 更新频率也自动跟随，无需单独配置。`m.config.Update.RefreshInterval` 的访问路径为 `internal/types/config.go:33` 的 `UpdateConfig.RefreshInterval`（yaml: `update.refresh_interval`）。

### 2.3 数据流路径

```
View() -> viewMonitoring() -> GeneratePortfolioRow()
    -> case ColTrend -> m.getSparklineForStock(code)
        -> 缓存命中? 返回缓存
        -> 缓存未命中:
            -> intraday.LoadLatestPrices(code, market)  // 新公共能力
                -> storage.GetIntradayFilePath(code, today)  // 复用现有路径查找
                -> json.Unmarshal -> 提取 prices []float64
            -> sparkline.Generate(prices, 12, upColor, downColor)  // 新公共能力
                -> Downsample(prices, 12)  // 降采样
                -> 映射 Unicode 块字符
                -> lipgloss 着色
            -> 缓存结果
            -> 返回 sparkline 字符串
```

---

## 三、功能 2：美股盘前盘后 — 模块级开发细节

### 3.1 涉及模块与修改点

| 模块 | 文件 | 修改点 | 行号参考 |
|------|------|--------|----------|
| 类型 | `internal/types/stock.go` | StockData 添加 6 个 Pre/PostMarket 字段 | L23-35 |
| 类型 | `types.go` (根目录) | 同步 StockData 字段（遗留代码） | 根目录 |
| API | `internal/api/us/yahoo.go` | Meta 结构体添加字段 + 解析逻辑 | L56-81, L150-163 |
| 列注册 | `internal/ui/columns.go` | 添加 `ColPreMarket`/`ColPostMarket` 列 | L14-34, L62+, L163+ |
| 行生成 | `main.go` | GeneratePortfolioRow/WatchlistRow 添加 case | L699+, L785+ |
| 格式化 | `internal/ui/format.go` | 添加 FormatPrePostMarketPrice | L80+ |
| 分时 API | `internal/intraday/api.go` | Yahoo URL 添加 `&includePrePost=true` | L372 |
| i18n | `i18n/zh.json`, `i18n/en.json` | 添加 "盘前"/"Pre" + "盘后"/"Post" | - |

### 3.2 详细代码修改

#### A. `internal/api/us/yahoo.go` — 解析盘前盘后（最低成本修改）

**Step 1**: 在 Meta 结构体 (L59-68) 中添加字段：
```go
Meta struct {
    // ... 现有字段 ...
    PreMarketPrice       float64 `json:"preMarketPrice"`
    PreMarketChange      float64 `json:"preMarketChange"`
    PreMarketChangePercent float64 `json:"preMarketChangePercent"`
    PostMarketPrice      float64 `json:"postMarketPrice"`
    PostMarketChange     float64 `json:"postMarketChange"`
    PostMarketChangePercent float64 `json:"postMarketChangePercent"`
} `json:"meta"`
```

**Step 2**: 在返回 StockData (L150-163) 时填充新字段：
```go
return &types.StockData{
    // ... 现有字段 ...
    PreMarketPrice:    meta.PreMarketPrice,
    PreMarketChange:   meta.PreMarketChange,
    PreMarketPercent:  meta.PreMarketChangePercent,
    PostMarketPrice:   meta.PostMarketPrice,
    PostMarketChange:  meta.PostMarketChange,
    PostMarketPercent: meta.PostMarketChangePercent,
}
```

**无需新增 API 调用**：Yahoo Finance 的现有 `/v8/finance/chart/` 端点已返回这些字段，仅需解析即可。

#### B. `internal/intraday/api.go` — Yahoo 分时请求添加扩展时段

修改 `tryGetIntradayFromYahoo()` (L372) 的 URL：
```go
// 现有:
url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1m&range=1d", yahooSymbol)
// 改为:
url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1m&range=1d&includePrePost=true", yahooSymbol)
```

**效果**：返回 4:00AM-8:00PM ET 的完整分钟数据（而非仅 9:30-16:00）。现有解析逻辑 (L444-482) 无需修改 — timestamp 转 HH:MM 格式自然包含盘前盘后时间点。

#### C. `internal/ui/columns.go` — 注册盘前盘后列

添加常量：
```go
ColPreMarket  ColumnID = "pre_market"
ColPostMarket ColumnID = "post_market"
```

在两个 registry 中注册（不可排序）：
```go
ColPreMarket: {ID: ColPreMarket, I18nKey: "col.pre_market", IsRequired: false, SortField: nil},
ColPostMarket: {ID: ColPostMarket, I18nKey: "col.post_market", IsRequired: false, SortField: nil},
```

#### D. `main.go` — 行生成

在 `GeneratePortfolioRow()` switch 中添加：
```go
case ui.ColPreMarket:
    if stock.PreMarketPrice > 0 {
        row[i] = m.formatPrePostWithColor(stock.PreMarketPrice, stock.PreMarketPercent, stock.PrevClose)
    } else {
        row[i] = "-"
    }
case ui.ColPostMarket:
    // 同理
```

**条件显示设计**：当 `PreMarketPrice == 0`（即 A股/港股或非盘前时段），显示 `-`。用户可通过 config 控制是否显示这些列。

### 3.3 数据流路径

```
定时刷新 Tick -> fetchStockPrices() -> api.FetchStockPrice(code)
    -> [美股] us.TryYahooFinanceAPI(symbol)
        -> JSON 解析 meta.preMarketPrice/postMarketPrice  // 新增解析
        -> 返回 StockData{..., PreMarketPrice: xxx}
    -> 缓存到 m.stockPriceCache
    -> View() -> GeneratePortfolioRow()
        -> case ColPreMarket -> 从 StockData 读取 PreMarketPrice
        -> Formatter.FormatPrePostMarketPrice() -> 格式化为带颜色字符串
```

---

## 四、功能 3：A股集合竞价 — 模块级开发细节

### 4.1 涉及模块与修改点

| 模块 | 文件 | 修改点 | 行号参考 |
|------|------|--------|----------|
| 分时 API | `internal/intraday/api.go` | EastMoney `iscr=0` **已是当前值** → 验证是否已包含竞价数据 | L202 |
| 交易时段 | `internal/intraday/trading.go` | `GetTradingState()` 添加竞价时段识别 | L15-61 |
| 数据完整度 | `internal/intraday/storage.go` | `GetExpectedDatapoints()` 更新预期数据点数 | L155-169 |
| 图表时间轴 | `handlers_chart.go` | `createFixedTimeRange()` 扩展竞价时间段 | L314-371 |
| 图表渲染 | `handlers_chart.go` | `createIntradayChart()` 竞价区域标记 | L378-569 |
| 配置 | `cmd/conf/config_demo.yaml` | A股交易时段新增竞价 session | - |
| Worker | `internal/intraday/worker.go` | 采集时间范围扩展到竞价时段 | 需确认 |

### 4.2 详细代码修改

#### A. `internal/intraday/api.go` — 验证东方财富已返回竞价数据

当前 `tryGetIntradayFromEastMoney()` (L196) 的 URL 已使用 `iscr=0`：
```
https://push2.eastmoney.com/api/qt/stock/trends2/get?secid=%s&fields1=f1,f2,f3&fields2=f51,f52,f53,f54,f55&iscr=0
```

`iscr=0` 意味着**已包含集合竞价数据**（0=包含，1=排除）。返回的 trends 中应已有 09:15-09:25 的数据点。**需验证**：是否因为 `formatIntradayTime()` 或其他过滤逻辑丢弃了这些数据点。

**关键检查**：`formatIntradayTime()` 函数（在 `api.go` 或 `helpers.go` 中）是否会过滤掉 09:15-09:25 范围的时间字符串。

#### B. `internal/intraday/trading.go` — 新增竞价状态

扩展 `GetTradingState()` (L15-61) 的 china 分支：
```go
case "china":
    auctionStart := time.Date(now.Year(), now.Month(), now.Day(), 9, 15, 0, 0, now.Location())
    morningStart := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
    // ... 现有定义 ...
    closeAuctionStart := time.Date(now.Year(), now.Month(), now.Day(), 14, 57, 0, 0, now.Location())

    // 新增竞价状态判断
    if now.After(auctionStart) && now.Before(morningStart) {
        return types.TradingStateAuction  // 新增状态
    }
```

**需新增 TradingState 常量**：在 `internal/types/intraday.go` (L37-45) 添加：
```go
TradingStateAuction  TradingState = iota + 5 // 集合竞价
```

**同时需修复两处 TODO**：
- L23: 集成假日检测（调用 `internal/market/holiday.go` 的 `IsTradingDay()`）
- L93: 同上

#### C. `cmd/conf/config_demo.yaml` — 交易时段扩展

```yaml
markets:
  china:
    trading_sessions:
      - start_time: "09:15"  # 开盘集合竞价（新增）
        end_time: "09:25"
        type: "auction"
      - start_time: "09:30"  # 上午连续交易
        end_time: "11:30"
        type: "regular"
      - start_time: "13:00"  # 下午连续交易
        end_time: "14:57"
        type: "regular"
      - start_time: "14:57"  # 收盘集合竞价（新增）
        end_time: "15:00"
        type: "auction"
```

#### D. `handlers_chart.go` — 竞价区域标记

`createFixedTimeRange()` (L314-371) **无需修改核心逻辑** — 它已遍历 config 中所有 `TradingSessions` 生成时间点。添加竞价 session 后，时间轴自动扩展。

但需修改 `createIntradayChart()` (L378) 中的 **X轴标签格式化** (L502-544)。当前硬编码了 4 个时间标签（09:30, 11:30, 13:00, 15:00），需要新增：
```go
// 新增竞价时间标签
{hour: 9, minute: 15, label: "09:15"},  // 开盘竞价开始
{hour: 14, minute: 57, label: "14:57"}, // 收盘竞价开始
```

**竞价区域视觉区分**：在 `createIntradayChart()` 中，对竞价时段的数据点使用虚线样式或不同颜色绘制。通过判断 TimePoint 时间是否在竞价范围内来切换样式。

#### E. `internal/intraday/storage.go` — 更新预期数据点

`GetExpectedDatapoints()` (L155-169) 更新 A 股预期值：
```go
case "china":
    // 原: 240 (09:30-11:30 120分 + 13:00-15:00 120分)
    // 新: 240 + 10 (竞价) + 3 (收盘竞价) = 253
    return 253
```

### 4.3 Volume 数据采集（Sparkline + 成交量图表的前置条件）

需修改 4 个分时 API 函数以提取 volume：

**a. `tryGetIntradayFromTencent()` (L270-362)** — 修改 L354：
```go
// 现有: result = append(result, IntradayDataPoint{Time: timeStr, Price: price})
// 改为:
var vol int64
if len(parts) >= 3 {
    vol, _ = strconv.ParseInt(parts[2], 10, 64)
}
result = append(result, IntradayDataPoint{Time: timeStr, Price: price, Volume: vol})
```

**b. `tryGetIntradayFromEastMoney()` (L196-267)** — 修改 L260：
```go
var vol int64
if len(parts) >= 3 {
    vol, _ = strconv.ParseInt(parts[2], 10, 64)
}
result = append(result, IntradayDataPoint{Time: timeStr, Price: price, Volume: vol})
```

**c. `tryGetIntradayFromSina()` (L131-193)** — 修改 L186：
```go
vol, _ := strconv.ParseInt(item.Volume, 10, 64)
result = append(result, IntradayDataPoint{Time: timeStr, Price: price, Volume: vol})
```

**d. `tryGetIntradayFromYahoo()` (L366-486)** — 修改结构体 + L479：
```go
// 结构体 Quote 中添加 Volume 解析
Quote []struct {
    Close  []float64 `json:"close"`
    Volume []int64   `json:"volume"`  // 新增
} `json:"quote"`

// 数据提取时添加 volume
var vol int64
if len(quotes[0].Volume) > i {
    vol = quotes[0].Volume[i]
}
datapoints = append(datapoints, IntradayDataPoint{Time: timeStr, Price: price, Volume: vol})
```

---

## 五、功能 4：港股竞价时段 — 模块级开发细节

与 A 股方案一致，修改：
- `internal/intraday/trading.go` — `GetTradingState()` 的 hongkong 分支添加 9:00-9:30 竞价识别
- `cmd/conf/config_demo.yaml` — 港股添加 `{start_time: "09:00", end_time: "09:30", type: "auction"}` session
- X 轴标签新增 "09:00" 时间标签

---

## 六、功能 5：多周期图表 — 模块级开发细节

### 6.1 涉及模块

| 模块 | 文件 | 修改点 |
|------|------|--------|
| 公共加载 | `internal/intraday/loader.go` | `LoadMultiDay()` + `Downsample()` |
| 图表 | `handlers_chart.go` | `handleIntradayChartViewing()` 添加周期切换键 (L601+) |
| 图表 | `handlers_chart.go` | 新增 `createMultiDayChart()` 函数 |
| Model | `types.go` (根目录) | 添加 `chartPeriod string` 字段 |
| i18n | `i18n/*.json` | 周期名称翻译 |

### 6.2 键绑定

在 `handleIntradayChartViewing()` (L601) 的 switch 中添加：
```go
case "1": m.chartPeriod = "1D"  // 重新加载单日
case "5": m.chartPeriod = "5D"  // 加载5日
case "m": m.chartPeriod = "1M"
case "3": m.chartPeriod = "3M"
case "y": m.chartPeriod = "1Y"
```

切换周期时调用 `createMultiDayChart()` 替代 `createIntradayChart()`，该函数使用 `intraday.LoadMultiDay()` 加载多日数据并降采样。

### 6.3 图表标题

多周期模式下标题需显示实际数据覆盖范围：
```
📈 AAPL - Apple Inc. | 5D | 2026-02-04 ~ 2026-02-08 (实际4天数据)
```

---

## 七、实施顺序与依赖关系

```
Step 0: 公共基础设施
  ├── 0a. IntradayDataPoint 添加 Volume 字段
  ├── 0b. StockData 添加 Pre/PostMarket 字段
  ├── 0c. 新建 internal/ui/sparkline/sparkline.go
  ├── 0d. 新建 internal/intraday/loader.go (提取 LoadSingleDay/LoadLatestPrices)
  └── 0e. TradingState 添加 Auction 状态

Step 1: Sparkline (依赖 0a, 0c, 0d)
  ├── 1a. columns.go 注册 ColTrend
  ├── 1b. main.go GeneratePortfolioRow/WatchlistRow 添加 case
  ├── 1c. Model 添加 sparkline 缓存字段
  ├── 1d. i18n 添加翻译键
  └── 1e. config_demo.yaml 默认列添加 trend

Step 2: 美股盘前盘后 (依赖 0b)
  ├── 2a. yahoo.go Meta 结构体扩展 + 解析
  ├── 2b. columns.go 注册 ColPreMarket/ColPostMarket
  ├── 2c. main.go GenerateRow 添加 case
  ├── 2d. format.go 添加 FormatPrePostMarketPrice
  ├── 2e. intraday/api.go Yahoo URL 添加 includePrePost
  └── 2f. i18n 添加翻译键

Step 3: 分时 API Volume 采集 (依赖 0a)
  ├── 3a. Tencent API 提取 parts[2]
  ├── 3b. EastMoney API 提取 parts[2]
  ├── 3c. Sina API 提取 item.Volume
  └── 3d. Yahoo API 提取 quotes[0].Volume

Step 4: A股集合竞价 (依赖 0e, Step 3)
  ├── 4a. trading.go GetTradingState() 扩展竞价识别
  ├── 4b. config_demo.yaml 竞价 session 配置
  ├── 4c. handlers_chart.go X轴标签扩展
  ├── 4d. storage.go 更新预期数据点数
  └── 4e. 假日检测 TODO 修复

Step 5: 港股竞价 (依赖 0e)
  └── 同 Step 4 的港股版本

Step 6: 多周期图表 (依赖 0d)
  ├── 6a. loader.go LoadMultiDay + Downsample
  ├── 6b. handlers_chart.go 键绑定 + createMultiDayChart
  └── 6c. 图表标题显示数据范围
```

---

## 八、关键文件修改清单（精确到函数级）

| 文件 | 函数/位置 | 修改类型 | 描述 |
|------|----------|---------|------|
| `internal/intraday/types.go:4` | `IntradayDataPoint` | 扩展 | 添加 `Volume int64` 字段 |
| `internal/types/stock.go:23` | `StockData` | 扩展 | 添加 6 个 Pre/PostMarket 字段 |
| `types.go` (根目录) | `StockData` | 扩展 | 同步添加 Pre/PostMarket 字段 |
| `internal/types/intraday.go:37` | `TradingState` | 扩展 | 添加 `TradingStateAuction` |
| `internal/ui/columns.go:14` | 常量区 | 添加 | `ColTrend`, `ColPreMarket`, `ColPostMarket` |
| `internal/ui/columns.go:62` | `makePortfolioColumnRegistry()` | 扩展 | 注册 3 个新列 |
| `internal/ui/columns.go:163` | `makeWatchlistColumnRegistry()` | 扩展 | 注册 3 个新列 |
| `internal/ui/format.go:80` | (新函数) | 新增 | `FormatPrePostMarketPrice()` |
| `internal/api/us/yahoo.go:59` | Meta 结构体 | 扩展 | 添加 PreMarket/PostMarket 字段 |
| `internal/api/us/yahoo.go:150` | return 语句 | 扩展 | 填充 Pre/PostMarket 字段 |
| `internal/intraday/api.go:354` | `tryGetIntradayFromTencent()` | 修改 | 提取 parts[2] 作为 Volume |
| `internal/intraday/api.go:260` | `tryGetIntradayFromEastMoney()` | 修改 | 提取 parts[2] 作为 Volume |
| `internal/intraday/api.go:186` | `tryGetIntradayFromSina()` | 修改 | 提取 item.Volume |
| `internal/intraday/api.go:479` | `tryGetIntradayFromYahoo()` | 修改 | 提取 quotes[0].Volume[i] |
| `internal/intraday/api.go:372` | `tryGetIntradayFromYahoo()` URL | 修改 | 添加 `&includePrePost=true` |
| `internal/intraday/trading.go:15` | `GetTradingState()` | 扩展 | A股/港股竞价时段识别 |
| `internal/intraday/trading.go:23` | TODO | 修复 | 假日检测集成 |
| `internal/intraday/trading.go:93` | TODO | 修复 | 假日检测集成 |
| `internal/intraday/storage.go:155` | `GetExpectedDatapoints()` | 修改 | A股 240→253 |
| `handlers_chart.go:314` | `createFixedTimeRange()` | 无需修改 | config 驱动，自动适配新 session |
| `handlers_chart.go:502` | X轴标签格式化 | 修改 | 添加竞价时间标签 |
| `handlers_chart.go:601` | `handleIntradayChartViewing()` | 扩展 | 添加周期切换键 1/5/M/3/Y |
| `main.go:699` | `GeneratePortfolioRow()` | 扩展 | 添加 ColTrend/ColPreMarket/ColPostMarket case |
| `main.go:785` | `GenerateWatchlistRow()` | 扩展 | 同上 |
| **新文件** | `internal/ui/sparkline/sparkline.go` | 新建 | Sparkline 生成公共能力 |
| **新文件** | `internal/intraday/loader.go` | 新建 | 数据加载公共能力 |

---

## 九、验证方案

### Step 0 验证
```bash
go build -o cmd/stock-monitor  # 确认编译通过
go test -v ./internal/intraday/...  # 确认类型扩展不破坏现有测试
```

### Step 1 (Sparkline) 验证
1. 运行程序进入投资组合，确认每行末尾有 `▁▂▃▄▅▆▇` 形式的迷你趋势图
2. 无分时数据的股票显示 `────────────`
3. A股红涨绿跌，美股绿涨红跌
4. 修改 config.yml 移除 `trend` 列，确认列消失
5. 自选股视图同样有 trend 列

### Step 2 (盘前盘后) 验证
1. 在美股盘前/盘后时段运行，确认 PreMarket/PostMarket 列有数据
2. A股股票的盘前盘后列显示 `-`
3. `go test -v ./internal/api/us/...` 确认 Yahoo 解析正确

### Step 3 (Volume) 验证
1. 运行程序采集分时数据，检查 `data/intraday/CN/SHxxxxxx/` 下 JSON 文件
2. 确认每个 datapoint 包含 `volume` 字段

### Step 4 (集合竞价) 验证
1. 在 9:15-9:25 运行，确认分时数据包含竞价时段时间点
2. 分时图 X 轴显示 "09:15" 标签
3. `GetExpectedDatapoints("china", false)` 返回 253

### Step 6 (多周期) 验证
1. 图表视图按 `5` 键，确认显示 5 日拼接数据
2. 标题显示实际数据范围
3. 按 `1` 切回单日

---

## 十、结论

本计划提取了 2 个新公共模块（`sparkline/`、`loader.go`）和 1 个格式化扩展，确保新功能可被投资组合、自选股、板块、搜索等多个模块复用。所有修改通过 `omitempty` JSON tag 确保向后兼容。

**最高 ROI 路径**：Step 0 + Step 1 (Sparkline) + Step 2 (盘前盘后) 可在同一迭代完成，这两个功能共享公共基础设施但互不依赖，可并行开发。
