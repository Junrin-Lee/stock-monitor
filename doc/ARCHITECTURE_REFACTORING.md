# Stock Monitor - 架构模块划分文档

## Round 3-5 重构总结

本文档定义了 stock-monitor 项目的模块化架构边界和未来重构路线图。

当前状态（Round 1-2 完成）：
- ✅ 基础类型系统已模块化 (`internal/types`)
- ✅ UI 工具函数已模块化 (`internal/ui`)
- ✅ 数据持久化已模块化 (`internal/data`)
- ✅ 市场逻辑已模块化 (`internal/market`)
- ✅ 排序逻辑已模块化 (`internal/sort`)
- ✅ 项目可编译，所有测试通过

未来重构目标（Round 3-5 蓝图）：
- 📋 API 模块划分
- 📋 Intraday 模块划分
- 📋 Alert 模块划分
- 📋 状态处理器模块划分
- 📋 main.go 精简

---

## 模块 1: API 模块 (`api.go` 1,355行)

### 当前结构
- 单一文件包含所有 API 调用逻辑
- 多个 API 提供商（腾讯、新浪、TwelveData、Yahoo、East Money）
- 搜索功能和价格获取混合

### 目标结构

```
internal/api/
├── api.go              # 公共入口和接口
├── types.go            # API 数据类型
├── helpers.go          # 代码转换、编码处理等工具
├── search.go           # 搜索功能聚合
├── china/              # A股 API
│   ├── tencent.go      # 腾讯 API
│   ├── sina.go         # 新浪 API
│   └── eastmoney.go    # 东方财富 API
├── us/                 # 美股 API
│   ├── twelvedata.go   # TwelveData API
│   ├── fmp.go          # Financial Modeling Prep API
│   └── yahoo.go        # Yahoo Finance API
└── hongkong/           # 港股 API
    ├── tencent.go      # 腾讯港股 API
    └── eastmoney.go    # 东财港股换手率补充

```

### 函数职责分配

**api.go - 主入口 (~50行)**
- `GetStockInfo(symbol)` - 股票信息获取入口
- `GetStockPrice(symbol)` - 价格获取带降级策略

**helpers.go (~200行)**
- 代码判断：`IsChinaStock()`, `IsHKStock()`, `GetMarketType()`
- 代码转换：所有 `Convert*` 函数
- 编码处理：GBK/UTF-8 转换（或引用 ui 模块）
- 工具函数：`PadHKStockCode()`, `Min()`

**search.go (~300行)**
- `SearchStockBySymbol()` - 符号搜索聚合
- `SearchChineseStock()` - 中文名称搜索
- `TryAdvancedSearch()` - 高级搜索策略
- `GenerateSearchKeywords()` - 关键词变形

**china/tencent.go (~200行)**
- `FetchTencentAPI(symbol)` - 腾讯 API 实现
- `SearchStockByTencentAPI(keyword)` - 腾讯搜索
- `ConvertStockSymbolForTencent()` - 代码转换
- 腾讯搜索结果解析函数

**china/sina.go (~150行)**
- `FetchSinaAPI(symbol)` - 新浪 API 实现
- `SearchStockBySinaAPI(keyword)` - 新浪搜索
- 新浪结果解析函数

**china/eastmoney.go (~100行)**
- `TryEastMoneyHKTurnover(symbol)` - 港股换手率补充

**us/twelvedata.go (~200行)**
- `TryTwelveDataAPI(symbol)` - TwelveData 报价
- `SearchStockByTwelveDataAPI(keyword)` - TwelveData 搜索

**us/fmp.go (~100行)**
- `TryFMPFreeAPI(symbol)` - FMP 免费API

**us/yahoo.go (~150行)**
- `TryYahooFinanceAPI(symbol)` - Yahoo Finance API

### 关键挑战
1. **循环依赖**: API 模块需要调用 getStockPrice，而 getStockPrice 又需要调用各个 API
   - 解决方案：使用依赖注入或接口抽象

2. **日志函数依赖**: 所有 API 函数需要 logDebug/logInfo/logError
   - 解决方案：传递 logger 接口或使用全局 logger

3. **GBK 编码转换**: 需要 gbkToUtf8 函数
   - 解决方案：移动到 internal/encoding 或在 api 包内实现

---

## 模块 2: Intraday 模块 (`intraday.go` 1,535行 + `intraday_chart.go` 1,260行)

### 当前结构
- `intraday.go`: 分钟级数据采集、管理、存储
- `intraday_chart.go`: 图表渲染逻辑

### 目标结构

```
internal/intraday/
├── manager.go          # IntradayManager 类型和主要管理方法
├── collector.go        # 数据采集逻辑
├── worker.go           # Worker Pool 并发控制
├── storage.go          # 数据存储和加载（文件锁）
├── chart.go            # 图表渲染（intraday_chart.go 移动）
├── helpers.go          # 辅助函数
└── types.go            # IntradayData 等类型定义

```

### 函数职责分配

**manager.go (~200行)**
- `IntradayManager` 类型定义
- `NewIntradayManager()` - 构造函数
- `StartIntradayCollection()` - 启动采集
- `StopIntradayCollection()` - 停止采集
- `IsCollecting()` - 状态查询

**collector.go (~300行)**
- `collectIntradayData()` - 主采集逻辑
- 市场交易时间判断
- 数据验证和质量检查

**worker.go (~200行)**
- Worker 相关函数
- Semaphore 并发控制（max 10）
- Goroutine 生命周期管理

**storage.go (~300行)**
- `saveIntradayData()` - 保存数据
- `loadIntradayData()` - 加载数据
- 文件锁管理 (`intradayFileLocks`)
- 原子写入（temp file + rename）

**chart.go (1,260行 - 直接移动)**
- `renderIntradayChart()` - 主渲染函数
- 所有图表相关辅助函数
- Braille 字符绘图逻辑

**types.go (~100行)**
- `IntradayData` 结构定义
- `IntradayDataPoint` 结构定义
- 元数据结构

### 关键挑战
1. **API 依赖**: 需要调用 getStockPrice
   - 解决方案：通过接口注入或回调函数

2. **配置依赖**: 需要访问 Model.config
   - 解决方案：传递配置对象

3. **缓存更新**: 需要更新 stockPriceCache
   - 解决方案：通过消息传递而非直接修改

---

## 模块 3: Alert 模块 (`alert.go` 2,572行)

### 当前结构
- 包含告警检查、通知、UI 渲染、批量操作等所有告警相关功能
- 与 Model 深度耦合

### 目标结构

```
internal/alert/
├── checker.go          # 告警条件检查逻辑
├── notification.go     # 通知发送（跨平台）
├── render.go           # UI 渲染函数
├── batch.go            # 批量操作
├── helpers.go          # 辅助函数（findAlertByID 等）
└── types.go            # Alert 相关类型（可能已在 internal/types）

```

### 函数职责分配

**checker.go (~300行)**
- `checkAlerts()` - 主检查循环
- `checkAlertCondition()` - 单个告警条件判断
- 告警触发逻辑

**notification.go (~200行)**
- `sendNotification()` - 通知发送
- macOS notific notification-send
- Windows 通知
- 跨平台兼容性处理

**render.go (~1,200行)**
- `renderAlertManage()` - 告警管理主界面
- `renderAlertAdd()` - 添加告警界面
- `renderAlertEdit()` - 编辑告警界面
- 各种告警相关 UI 渲染

**batch.go (~500行)**
- 批量告警相关渲染
- 批量操作逻辑
- 按标签/市场批量添加

**helpers.go (~200行)**
- `findAlertByID()` - ID 查找
- 各种告警辅助函数

### 关键挑战
1. **与 Model 的深度耦合**: 大量函数是 Model 的方法
   - 解决方案：提取为接受 Model 参数的独立函数，或传递必要字段

2. **UI 状态依赖**: 需要访问多个 Model 字段
   - 解决方案：定义 AlertState 结构体包装相关字段

---

## 模块 4: 状态处理器 (`main.go` 中的 29个 handler 函数)

### 当前结构
- main.go 包含所有 29 个状态处理函数
- 每个 handle*() 函数处理一个状态
- Update() 和 View() 方法路由到各个处理器

### 目标结构

```
internal/states/
├── menu/
│   ├── main_menu.go        # handleMainMenu()
│   └── language.go         # handleLanguageSelection()
├── portfolio/
│   ├── monitoring.go       # handleMonitoring()
│   ├── adding.go           # handleAddingStock()
│   ├── editing.go          # handleEditingStock()
│   └── sorting.go          # handlePortfolioSorting()
├── watchlist/
│   ├── viewing.go          # handleWatchlistViewing()
│   ├── tagging.go          # handleWatchlistTagging()
│   ├── tag_select.go       # handleWatchlistTagSelect()
│   ├── tag_manage.go       # handleWatchlistTagManage()
│   ├── tag_edit.go         # handleWatchlistTagEdit()
│   ├── group_select.go     # handleWatchlistGroupSelect()
│   └── sorting.go          # handleWatchlistSorting()
├── search/
│   ├── searching.go        # handleSearchingStock()
│   ├── result.go           # handleSearchResult()
│   └── actions.go          # handleSearchResultWithActions()
├── alert/
│   ├── manage.go           # handleAlertManage()
│   ├── add.go              # handleAlertAdd()
│   ├── edit.go             # handleAlertEdit()
│   ├── batch.go            # handleAlertBatch*()
│   ├── frequency.go        # handleAlertFrequencySelect()
│   └── stock_alert.go      # handleStockAlertManage()
└── chart/
    └── intraday.go         # handleIntradayChartViewing()

```

### 函数签名统一为
```go
func Handle[StateName](msg tea.Msg, m *Model) (tea.Model, tea.Cmd)
```

### 关键挑战
1. **Model 访问**: 所有处理器需要访问和修改 Model
   - 解决方案：传递 Model 指针，处理器函数修改并返回

2. **代码重复**: 许多处理器有相似的键盘处理逻辑
   - 解决方案：提取共用的输入处理辅助函数

---

## 模块 5: App 核心 (`main.go` 3,259行 → ~100行)

### 当前结构
- main.go 包含 Model、Init、Update、View、所有状态处理器、所有渲染函数

### 目标结构

```
internal/app/
├── model.go            # Model 结构定义
├── init.go             # Init() 方法
├── update.go           # Update() 方法和路由
└── view.go             # View() 方法和路由

main.go (精简到 ~100行)
├── main() 函数
├── 配置加载
├── 应用初始化
└── TUI 启动

```

### main.go 精简版本

```go
package main

import (
    "log"
    tea "github.com/charmbracelet/bubbletea"
    "stock-monitor/internal/app"
    "stock-monitor/internal/data"
    "stock-monitor/internal/log"
)

func main() {
    // 初始化日志
    log.InitLogger()
    defer log.CloseLogger()

    // 加载配置和数据
    config := data.LoadConfig()
    portfolio := data.LoadPortfolio()
    watchlist := data.LoadWatchlist()
    alerts := data.LoadAlertData()

    // 创建应用
    model := app.NewModel(config, portfolio, watchlist, alerts)

    // 启动 TUI
    p := tea.NewProgram(model, tea.WithAltScreen())
    if err := p.Start(); err != nil {
        log.Fatal("Application error: %v", err)
    }
}
```

---

## 重构优先级和时间表

### P0 - 核心优化（可立即进行）
1. ✅ 类型系统模块化（已完成）
2. ✅ UI 工具模块化（已完成）
3. ✅ 数据持久化模块化（已完成）
4. 📋 添加代码组织注释标记
5. 📋 创建模块边界文档（本文档）

### P1 - 独立模块提取（低风险）
1. 📋 Intraday Chart 模块（独立性强）
2. 📋 Alert Notification 模块（独立性强）
3. 📋 API Helpers 模块（工具函数）

### P2 - 业务逻辑重构（中等风险）
1. 📋 API 模块拆分（需要解决循环依赖）
2. 📋 Alert 业务逻辑提取
3. 📋 Intraday 管理逻辑提取

### P3 - 架构重构（高风险，需要大量测试）
1. 📋 状态处理器提取
2. 📋 main.go 精简
3. 📋 完全的 Model 解耦

---

## 当前代码统计

```
文件                   行数    目标拆分后最大文件
====================================================
main.go              3,259    ~300行 (app/update.go)
alert.go             2,572    ~300行 (分5-6个文件)
intraday.go          1,535    ~300行 (分5个文件)
api.go               1,355    ~300行 (分8-10个文件)
intraday_chart.go    1,260    ~300行 (可能保持或拆分2个)
watchlist.go           883    ~300行 (分2-3个文件)
====================================================
Total: ~11,000行待重构代码
```

---

## 技术债务和注意事项

### 已知问题
1. **Model 结构巨大**: 230+ 字段，需要逻辑分组
2. **全局状态**: 许多函数依赖全局 Model
3. **测试覆盖不足**: 核心业务逻辑缺少单元测试
4. **循环依赖风险**: API/Intraday/Alert 相互依赖

### 重构原则
1. **保持向后兼容**: 数据格式不变
2. **渐进式迁移**: 一次一个模块
3. **持续测试**: 每次迁移后完整测试
4. **文档先行**: 先定义接口再实现
5. **无破坏性更改**: 旧代码逐步废弃而非立即删除

### 长期目标
1. **清晰的模块边界**: 每个模块职责单一
2. **可测试性**: 所有业务逻辑可单独测试
3. **可扩展性**: 易于添加新功能
4. **代码可读性**: 单个文件 < 300行

---

## 结论

由于 stock-monitor 项目规模较大（~15,000行代码），完全的模块化重构是一个多阶段工程。

**Round 1-2（已完成）**: 基础模块提取，项目结构清晰化
**Round 3-5（本文档）**: 定义重构蓝图和模块边界

下一步建议：
1. 采用本文档作为重构指南
2. 优先完成 P1 级别的独立模块提取
3. 逐步进行 P2、P3 级别的重构
4. 每个模块提取后立即编写单元测试
5. 保持项目始终可编译和可运行

---

**文档版本**: 1.0
**创建日期**: 2026-01-19
**最后更新**: 2026-01-19
