# 让 Sparkline 走势列在所有表格中可见

## Context

v9.0 开发已完成 Sparkline 底层能力（`sparkline.go`、`loader.go`、`columns.go` 注册、`main.go` 行生成 case），但用户在持股列表、自选列表、板块成分股中均看不到走势列。原因有两个：

1. **用户配置 `cmd/conf/config.yml`** 的 `portfolio_columns` 和 `watchlist_columns` 中未包含 `trend`，而列系统是配置驱动的——不在配置中的列不会显示。
2. **板块成分股（Sector）使用独立的硬编码列系统**（`SectorStockColumn`），完全未集成 Sparkline。

---

## 修改清单

### 修改 1: 用户配置添加 trend 列

**文件**: `cmd/conf/config.yml`

- 第 12-26 行 `portfolio_columns` 列表末尾添加 `- trend`
- 第 27-39 行 `watchlist_columns` 列表末尾添加 `- trend`

效果：Portfolio 和 Watchlist 表格立即显示走势列。

### 修改 2: Sector 成分股列定义添加走势列

**文件**: `internal/ui/sector/columns.go`

- 第 44 行（换手率之后）添加新列：`{Header: "走势", Width: 14, SortField: -1}`

### 修改 3: Sector 渲染函数签名添加 sparkline 闭包参数

**文件**: `internal/ui/sector/view.go`

- `RenderSectorStockList()` (第 149-160 行) 函数签名新增参数：`sparklineFunc func(code string) string`
- 第 266 行（换手率 append）之后、第 268 行（`t.AppendRow(row)`）之前，追加 sparkline 列值：
  ```go
  // 走势
  if sparklineFunc != nil {
      row = append(row, sparklineFunc(stock.Code))
  } else {
      row = append(row, "─────────────")
  }
  ```

设计选择：传入闭包而非 map，保持 `RenderSectorStockList` 为纯函数（不依赖 Model），同时复用 `getSparklineForStock()` 内建的缓存机制。

### 修改 4: Sector handler 调用处传入闭包

**文件**: `handlers_sector.go`

- `viewSectorStockList()` (第 363-377 行) 调用 `sector.RenderSectorStockList()` 时新增最后一个参数：`m.getSparklineForStock`

### 关键复用

| 已有能力 | 文件 | 直接复用 |
|---------|------|---------|
| `getSparklineForStock()` | `format.go:103-130` | Sector 通过闭包传入 |
| `sparkline.Generate()` | `internal/ui/sparkline/sparkline.go` | 被 getSparklineForStock 内部调用 |
| `intraday.LoadLatestPrices()` | `internal/intraday/loader.go` | 被 getSparklineForStock 内部调用 |
| sparklineCache | `types.go:245-247` | 自动命中缓存 |

不需要创建任何新文件或新函数。

---

## 验证

1. `go build ./...` 编译通过
2. `go test -v ./...` 所有测试通过
3. 运行 `go run main.go`：
   - 进入持股列表（Portfolio）→ 最右侧出现"走势"列，有分时数据的股票显示 `▁▂▃▄▅▆▇` 趋势图，无数据的显示 `────────────`
   - 切换到自选列表（Watchlist）→ 同样出现"走势"列
   - 进入板块 → 选择板块 → 成分股列表最右侧出现"走势"列
4. A 股红涨绿跌、美股绿涨红跌颜色正确
