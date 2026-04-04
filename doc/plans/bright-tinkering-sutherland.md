# 修复分时数据午休期间停止采集的问题

## Context

分时数据采集在 A 股午休时段 (11:30-13:00) 停止后不恢复。日志显示用户离开持股页面（按 ESC/Q/M）时所有 worker 被全部杀掉，重新进入时由于午休被误判为"盘后"，worker 以错误模式启动后很快自动停止。根本原因是 `GetTradingState` 没有定义"午休"状态，导致 11:30-13:00 被归类为 `PostMarket`。

## 根因分析

| # | 严重度 | 位置 | 问题 |
|---|--------|------|------|
| 1 | CRITICAL | `trading.go:62-70` | 午休时段 (11:30-13:00) 被 `else` 兜底返回 `PostMarket` |
| 2 | CRITICAL | `worker.go:129-145` | Live worker 检查到 `PostMarket` + 数据完整 → 永久停止 |
| 3 | HIGH | `trading.go:120-166` | `GetTradingDayForCollection` 午休时返回 `Complete` 或 `Historical` 模式 |
| 4 | HIGH | `worker.go:68-145` | 数据获取(goroutine) 与自动停止检查存在竞态：检查读取的是上一轮的旧数据 |
| 5 | MODERATE | `trading.go:65-66` | `now.After(13:00)` 在恰好 13:00:00 时为 false，漏掉边界秒 |

## 实施方案

### Step 1: 新增 `TradingStateLunchBreak` 状态

**文件**: `internal/types/intraday.go`

在 `TradingState` iota 末尾追加 `TradingStateLunchBreak`（不改变现有常量值）:

```go
const (
    TradingStatePreMarket  TradingState = iota // 0
    TradingStateLive                           // 1
    TradingStatePostMarket                     // 2
    TradingStateWeekend                        // 3
    TradingStateHoliday                        // 4
    TradingStateAuction                        // 5
    TradingStateLunchBreak                     // 6 - NEW
)
```

更新 `String()` 方法添加 `"LunchBreak"` case。

### Step 2: 修复 `GetTradingState` 检测午休

**文件**: `internal/intraday/trading.go:62-70`

替换当前逻辑为:

```go
if now.Before(morningStart) {
    return types.TradingStatePreMarket
} else if (!now.Before(morningStart) && now.Before(morningEnd)) ||
    (!now.Before(afternoonStart) && now.Before(afternoonEnd)) {
    return types.TradingStateLive
} else if !now.Before(morningEnd) && now.Before(afternoonStart) {
    return types.TradingStateLunchBreak  // 午休
} else if !now.Before(afternoonEnd) {
    return types.TradingStatePostMarket
} else {
    return types.TradingStatePostMarket  // 安全兜底
}
```

关键变化:
- `now.After(X)` → `!now.Before(X)` 解决边界秒问题（含等于）
- 显式检测 `morningEnd <= now < afternoonStart` 为午休
- 美股 `morningEnd == afternoonEnd`，午休条件数学上不可能满足，不影响

### Step 3: 修复 `GetTradingDayForCollection` 处理午休

**文件**: `internal/intraday/trading.go:137-166`

在 switch 中新增 case:

```go
case types.TradingStateLunchBreak:
    // 午休期间按 Live 模式处理当日数据
    return now.Format("20060102"), types.CollectionModeLive, nil
```

### Step 4: 修复 `isMarketOpenForConfig` 边界条件

**文件**: `internal/intraday/trading.go:264`

```go
// 旧: if now.After(startTime) && now.Before(endTime)
// 新: if !now.Before(startTime) && now.Before(endTime)
```

### Step 5: 修复 worker 午休逻辑

**文件**: `internal/intraday/worker.go`

**5a**: 在 Live 模式 market-closed continue 处（line 62-66），检测到午休时重置 `ConsecutiveSkips`:

```go
if mode == types.CollectionModeLive {
    if !IsMarketOpen(stockCode, im.model) {
        // 午休期间重置连续跳过计数，防止下午开盘后误触自动停止
        marketType := string(api.GetMarketType(stockCode))
        location, _ := GetMarketLocation(marketType)
        if location != nil {
            now := time.Now().In(location)
            if GetTradingState(now, marketType) == types.TradingStateLunchBreak {
                im.metadataMutex.Lock()
                if meta, exists := im.workerMetadata[stockCode]; exists {
                    meta.ConsecutiveSkips = 0
                }
                im.metadataMutex.Unlock()
            }
        }
        continue
    }
}
```

**5b**: 修复竞态 — 将自动停止检查（line 94-145）移到 fetch 之前执行，这样读取的是上一轮已完成的元数据:

```
原顺序: fetch goroutine → auto-stop check（读旧数据 ❌）
新顺序: auto-stop check（读上一轮数据 ✅）→ fetch goroutine
```

### Step 6: 补充 sector 视图的 stopIntradayDataCollection

**文件**: `handlers_sector.go`

在 sector stock list 和 sector viewing 的 ESC/Q 退出处理中补充 `m.stopIntradayDataCollection()` 调用。

### Step 7: 添加测试

**文件**: 新建 `internal/intraday/trading_test.go`

覆盖场景:
- A 股午休检测: 11:31, 12:00, 12:59 → `LunchBreak`
- 港股午休检测: 12:01, 12:30, 12:59 → `LunchBreak`
- 美股无午休: 任何时间不返回 `LunchBreak`
- 边界时间: 09:30=Live, 11:30=LunchBreak, 13:00=Live, 15:00=PostMarket
- `GetTradingDayForCollection` 午休时返回 `(today, Live, nil)`

## 实施顺序

1 → 2 → 3 → 4 → 5 → 6 → 7

## 验证方法

```bash
# 运行新增的交易状态测试
go test -v ./internal/intraday/ -run TestGetTradingState

# 运行所有 intraday 测试
go test -v ./internal/intraday/...

# 带竞态检测器编译运行
go build -race -o cmd/stock-monitor && ./cmd/stock-monitor

# 实际验证：在午休时段 (11:30-13:00) 进出持股页面，观察日志中 worker 是否保持运行
```

## 关键文件

- `internal/types/intraday.go` — 新增 `TradingStateLunchBreak` 常量
- `internal/intraday/trading.go` — 修复 `GetTradingState`、`GetTradingDayForCollection`、`isMarketOpenForConfig`
- `internal/intraday/worker.go` — 修复 worker 自动停止逻辑、竞态、午休跳过重置
- `handlers_sector.go` — 补充缺失的 `stopIntradayDataCollection()`
- `internal/intraday/trading_test.go` — 新增测试（新建文件）
