# 系统架构

详细的架构设计和技术细节。

## 分层架构

```
┌─────────────────────────────────────────────────────────┐
│  UI Layer (main.go, ui_utils.go, color.go)             │
│  Bubble Tea Framework + Go-Pretty Tables               │
│  ← 用户交互，状态机，事件循环                           │
├─────────────────────────────────────────────────────────┤
│  Business Logic Layer (alert.go, intraday.go, sort.go) │
│  State Management + Feature Modules                     │
│  ← 业务规则，告警检查，数据排序                         │
├─────────────────────────────────────────────────────────┤
│  Data Layer (cache.go, persistence.go, types.go)       │
│  Local Storage + In-Memory Cache                        │
│  ← 数据访问，缓存管理，类型定义                         │
├─────────────────────────────────────────────────────────┤
│  External Integration (api.go, intraday.go)            │
│  Multi-API Fallback + Background Workers               │
│  ← API 调用，数据采集，错误处理                         │
├─────────────────────────────────────────────────────────┤
│  Cross-Cutting (i18n.go, log.go, consts.go, errors.go) │
│  Internationalization, Logging, Constants              │
│  ← 全局功能，日志记录，常量定义                         │
└─────────────────────────────────────────────────────────┘
```

## 核心模块

| 模块 | 代码行 | 职责 |
|------|--------|------|
| `main.go` | 3,259 | 状态机与事件处理（29个状态） |
| `alert.go` | 2,572 | 告警系统核心（3种类型、批量操作） |
| `intraday.go` | 1,535 | 分时采集（工作池、智能管理） |
| `api.go` | 1,355 | API集成（多源数据、自动容错） |
| `intraday_chart.go` | 1,260 | 分时图表（Braille字符渲染） |

## 技术栈

| 组件 | 版本 | 用途 |
|------|------|------|
| Bubble Tea | v1.3.6 | 终端 UI 框架 |
| Go-Pretty | v6.6.8 | 表格布局和格式化 |
| ntcharts | v0.3.1 | Braille 字符图表 |
| golang.org/x/text | v0.28.0 | GBK/UTF-8 编码转换 |
| YAML v3 | v3.0.1 | 配置文件处理 |

## 并发与线程安全

### 缓存同步（RWMutex）

```go
type Cache struct {
    data  map[string]*CacheEntry
    mu    sync.RWMutex  // 多读单写
}

// 高并发读
func (c *Cache) Get(key string) *CacheEntry {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.data[key]
}

// 独占写
func (c *Cache) Set(key string, entry *CacheEntry) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = entry
}
```

### 工作池（信号量模式）

```go
type Manager struct {
    workerPool chan struct{}  // 最多 10 并发
}

// 带限制的并发控制
func (m *Manager) StartWorker() {
    m.workerPool <- struct{}{}     // 获取
    defer func() { <-m.workerPool }()  // 释放
    // 执行工作
}
```

### 消息传递（无直接状态修改）

```go
// ✅ 正确：通过消息告知主线程
go func() {
    data := fetchData()
    cmdChan <- tea.Cmd(func() tea.Msg {
        return DataUpdateMsg{Data: data}
    })
}()

// ❌ 错误：直接修改（竞态条件）
go func() {
    model.data = fetchData()  // 危险！
}()
```

## 性能考虑

| 指标 | 值 | 优化方案 |
|------|-----|--------|
| 缓存 | 30秒 TTL | 避免 API 过载 |
| 刷新间隔 | 5秒 | 平衡实时性和负载 |
| 工作池限制 | 10 | 保护系统资源 |
| 告警检查 | O(1) | 使用时间边界 |

---

更多详细信息见项目的 CLAUDE.md 文档。
