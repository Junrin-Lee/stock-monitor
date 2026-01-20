# 重构工具集成指南

## 快速开始

本指南帮助您将新创建的重构工具集成到现有代码中。

## 文件清单

```
新创建的文件:
├── internal/app/router.go                           # 状态路由器
├── internal/ui/table/builder.go                     # 表格构建器
├── internal/errors/errors.go                        # 错误处理
├── internal/service/portfolio/service.go            # 投资组合服务
├── internal/data/portfolio_repository.go            # 数据仓储
└── doc/CODE_REFACTORING_REPORT.md                   # 完整报告
```

## 集成步骤

### 步骤 1: 集成状态路由器 (30 分钟)

#### 1.1 修改 main.go

**当前代码** (main.go:157-250):
```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.state {
    case MainMenu:
        newModel, cmd = m.handleMainMenu(msg)
    case AddingStock:
        newModel, cmd = m.handleAddingStock(msg)
    // ... 27 more cases
    }
}
```

**改为**:
```go
import "stock-monitor/internal/app"

type Model struct {
    // ... existing fields
    router *app.Router  // 添加这一行
}

func (m *Model) initRouter() {
    m.router = app.NewRouter()

    // 注册所有状态处理器
    m.router.Register(MainMenu, func(msg tea.Msg) (tea.Model, tea.Cmd) {
        return m.handleMainMenu(msg.(tea.KeyMsg))
    })
    m.router.Register(AddingStock, func(msg tea.Msg) (tea.Model, tea.Cmd) {
        return m.handleAddingStock(msg.(tea.KeyMsg))
    })
    // ... 注册其他状态
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var newModel tea.Model
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        // 尝试使用路由器
        if model, tcmd, ok := m.router.Route(m.state, msg); ok {
            newModel, cmd = model, tcmd
        } else {
            newModel, cmd = m, nil
        }
    // ... 其他消息类型保持不变
    }

    return newModel, cmd
}
```

#### 1.2 在 main() 函数中初始化

```go
func main() {
    // ... existing code

    m := Model{
        // ... existing fields
    }

    m.initRouter()  // 添加这一行

    p := tea.NewProgram(&m, tea.WithAltScreen())
    // ...
}
```

### 步骤 2: 使用表格构建器 (1 小时)

#### 2.1 修改 handlers_portfolio.go

**找到** `viewMonitoring()` 函数中的表格渲染代码:

**当前代码** (~100 行):
```go
func (m *Model) viewMonitoring() string {
    t := table.NewWriter()
    t.SetStyle(table.StyleLight)

    // 手动构建表头
    header := table.Row{}
    // ... 很多代码

    // 手动添加行
    for i, stock := range stocks {
        row := table.Row{}
        // ... 很多代码
    }

    return t.Render()
}
```

**改为** (~30 行):
```go
import tablebuilder "stock-monitor/internal/ui/table"

func (m *Model) viewMonitoring() string {
    columns := m.GetPortfolioColumns()
    builder := tablebuilder.NewBuilder()

    // 链式调用构建表格
    builder.
        WithColumns(columns).
        WithHeader(m.GeneratePortfolioHeader()).
        WithCursor(m.portfolioCursor).
        WithScroll(m.portfolioScrollPos, m.portfolioScrollPos+m.config.Display.MaxLines)

    // 添加数据行
    stocks := m.portfolio.Stocks
    startIndex, endIndex := 0, len(stocks)
    if len(stocks) > m.config.Display.MaxLines {
        startIndex, endIndex = builder.GetVisibleRange(m.config.Display.MaxLines)
    }

    for i := startIndex; i < endIndex; i++ {
        row := m.GeneratePortfolioRow(&stocks[i], i, startIndex, endIndex)
        builder.AddRow(row)
    }

    // 添加总计行
    totalProfit, totalProfitRate, totalMarketValue := m.calculateTotals()
    builder.WithFooter(m.GeneratePortfolioTotalRow(totalProfit, totalProfitRate, totalMarketValue))

    return builder.Build()
}
```

### 步骤 3: 统一错误处理 (30 分钟)

#### 3.1 修改 internal/api/stock.go

**当前代码**:
```go
func GetStockPrice(symbol string) *types.StockData {
    data := china.TryTencentAPI(symbol)
    if data.Price > 0 {
        return data
    }

    log.Error("log.api.tencentFail")
    return nil
}
```

**改为**:
```go
import apperrors "stock-monitor/internal/errors"

func GetStockPrice(symbol string) (*types.StockData, error) {
    data := china.TryTencentAPI(symbol)
    if data != nil && data.Price > 0 {
        return data, nil
    }

    // 记录错误并返回
    err := apperrors.NewAPIError("Tencent API failed", nil).
        WithContext("symbol", symbol)
    err.LogError()

    // 尝试备用 API
    data = china.TrySinaAPI(symbol)
    if data != nil && data.Price > 0 {
        return data, nil
    }

    return nil, apperrors.NewAPIError("all APIs failed", nil).
        WithContext("symbol", symbol)
}
```

### 步骤 4: 引入服务层 (2-3 小时)

#### 4.1 创建服务容器

```go
// internal/app/container.go
package app

import (
    "stock-monitor/internal/api"
    "stock-monitor/internal/data"
    "stock-monitor/internal/service/portfolio"
    "stock-monitor/internal/types"
)

type Container struct {
    Config              *types.Config
    PortfolioRepo       *data.PortfolioRepository
    PortfolioService    *portfolio.Service
    APIClient           *api.Client
    Cache               *data.CacheManager
}

func NewContainer(config *types.Config, dataDir string) *Container {
    // 创建仓储
    portfolioRepo := data.NewPortfolioRepository(dataDir)

    // 创建 API 客户端
    apiClient := api.NewClient(config)

    // 创建缓存
    cache := data.NewCacheManager(30 * time.Second)

    // 创建服务
    portfolioService := portfolio.NewService(portfolioRepo, apiClient, cache)

    return &Container{
        Config:           config,
        PortfolioRepo:    portfolioRepo,
        PortfolioService: portfolioService,
        APIClient:        apiClient,
        Cache:            cache,
    }
}
```

#### 4.2 修改 Model 结构

```go
type Model struct {
    state      AppState
    container  *app.Container  // 替代直接依赖

    // UI 状态
    cursor     int
    scrollPos  int
    input      string
    message    string

    // 其他必要字段 (减少到 ~50 个)
}
```

#### 4.3 使用服务层

**handlers_portfolio.go 中的 handleMonitoringActions**:

**当前代码**:
```go
case "d":
    // 直接操作 Model
    removedStock := m.portfolio.Stocks[m.portfolioCursor]
    m.portfolio.Stocks = append(m.portfolio.Stocks[:m.portfolioCursor],
                                 m.portfolio.Stocks[m.portfolioCursor+1:]...)
    m.savePortfolio()
```

**改为**:
```go
case "d":
    // 使用服务层
    stock := m.container.PortfolioService.GetStock(code)
    if err := m.container.PortfolioService.RemoveStock(stock.Code); err != nil {
        m.message = m.getText("removeFailed")
        if apperrors.IsType(err, apperrors.ErrTypeNotFound) {
            m.message = m.getText("stockNotFound")
        }
        return m, nil
    }

    m.message = fmt.Sprintf(m.getText("removeSuccess"), stock.Name, stock.Code)
```

## 测试验证

### 1. 编译测试

```bash
go build -o cmd/stock-monitor
```

如果遇到编译错误:
- 检查 import 路径
- 确保新文件在正确的目录
- 运行 `go mod tidy`

### 2. 功能测试

```bash
# 运行程序
./cmd/stock-monitor

# 测试以下功能:
# 1. 进入持股监控
# 2. 添加股票
# 3. 删除股票
# 4. 查看自选
```

### 3. 性能对比

**测试指标**:
- 启动时间
- 添加股票响应时间
- 刷新数据时间
- 内存使用

```bash
# 使用 pprof 分析
go build -o cmd/stock-monitor
time ./cmd/stock-monitor  # 测试启动时间
```

## 渐进式迁移策略

### 阶段 1: 验证 (本周)

- ✅ 集成路由器 (不改变现有逻辑)
- ✅ 在 1 个 handler 中试用表格构建器
- ✅ 保持原有代码作为备份

### 阶段 2: 推广 (下周)

- 在所有 handler 中使用表格构建器
- 统一错误处理到 API 层
- 创建 Watchlist 和 Alert 服务

### 阶段 3: 完善 (下月)

- 完整的服务层
- 测试覆盖
- 性能优化

## 回滚方案

如果遇到问题,可以快速回滚:

```bash
# 1. 备份当前代码
cp main.go main.go.backup
cp handlers_portfolio.go handlers_portfolio.go.backup

# 2. 如果需要回滚
mv main.go.backup main.go
mv handlers_portfolio.go.backup handlers_portfolio.go

# 3. 删除新文件 (可选)
rm internal/app/router.go
rm internal/ui/table/builder.go
rm internal/errors/errors.go
```

## 常见问题

### Q1: 编译错误 "undefined: types.XXX"

**解决**: 检查 import 语句,确保引入了正确的包:
```go
import "stock-monitor/internal/types"
```

### Q2: 类型不匹配

**解决**: 使用现有的转换函数,或直接使用 types.* 类型

### Q3: 性能下降

**解决**:
1. 检查是否引入了不必要的转换
2. 使用 pprof 定位瓶颈
3. 考虑添加缓存

### Q4: 测试失败

**解决**:
1. 更新测试代码以匹配新接口
2. 添加 mock 对象用于测试
3. 逐个修复失败的测试

## 性能优化检查清单

- [ ] 消除不必要的类型转换
- [ ] 缓存频繁访问的数据
- [ ] 使用对象池减少内存分配
- [ ] 批量处理 API 请求
- [ ] 使用 sync.Map 替代 map+mutex (热路径)

## 下一步

1. ✅ 按照步骤 1 集成状态路由器
2. ✅ 测试基本功能
3. ✅ 按照步骤 2 使用表格构建器
4. ✅ 逐步引入服务层

## 获取帮助

如果遇到问题:
1. 查看 `doc/CODE_REFACTORING_REPORT.md`
2. 检查现有代码注释
3. 参考创建的示例代码

---

**文档版本**: v1.0
**更新时间**: 2026-01-20
