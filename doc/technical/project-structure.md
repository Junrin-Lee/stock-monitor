# 项目结构

完整的代码组织和目录说明。

## 目录树

```
stock-monitor/
├── main.go                          # 核心应用入口
├── alert.go                         # 告警系统
├── intraday.go                      # 分时采集
├── api.go                           # API 集成
├── intraday_chart.go                # 分时图表
├── watchlist.go                     # 自选管理
├── types.go                         # 数据结构
├── persistence.go                   # 数据持久化
├── cache.go                         # 价格缓存
├── sort.go                          # 排序引擎
├── format.go                        # 格式化
├── logger.go                        # 日志系统
├── ...其他源文件...
│
├── cmd/
│   ├── stock-monitor               # 编译后的可执行文件
│   └── conf/
│       ├── config.yml              # 用户配置
│       └── config_demo.yaml        # 配置模板
│
├── data/
│   ├── portfolio.json              # 持仓数据
│   ├── watchlist.json              # 自选数据
│   ├── alert_data.json             # 告警配置
│   └── intraday/                   # 分时数据
│       ├── SH600000/
│       │   ├── 20260101.json
│       │   └── 20260102.json
│       └── AAPL/
│           └── 20260101.json
│
├── i18n/
│   ├── zh.json                     # 中文翻译
│   └── en.json                     # 英文翻译
│
├── doc/
│   ├── guides/                     # 使用指南
│   │   ├── getting-started.md
│   │   ├── alert-system.md
│   │   ├── keyboard-shortcuts.md
│   │   ├── configuration.md
│   │   └── use-cases.md
│   ├── technical/                  # 技术文档
│   │   ├── architecture.md
│   │   ├── project-structure.md
│   │   ├── api-integration.md
│   │   └── troubleshooting.md
│   ├── changelogs/                 # 版本历史
│   └── plans/                      # 实现计划
│
├── README.md                        # 主文档
├── README_EN.md                     # 英文文档
├── go.mod                           # Go 依赖
└── go.sum                           # 依赖锁定
```

## 核心模块职责

### UI 层

- `main.go` - 状态机、事件循环、用户交互处理
- `ui_utils.go` - 表格格式化、分页、中文宽度处理
- `format.go` - 数字格式化、价格显示、百分比计算
- `color.go` - 颜色主题、涨跌配色

### 业务逻辑层

- `alert.go` - 告警 CRUD、触发检查、批量操作
- `intraday.go` - 工作池管理、数据采集、市场检测
- `sort.go` - 多字段排序引擎
- `watchlist.go` - 自选标签、分组、筛选

### 数据层

- `types.go` - 数据结构定义
- `cache.go` - 30秒 TTL 缓存（RWMutex 并发保护）
- `persistence.go` - JSON/YAML 读写、数据迁移
- `alert_data.json` - 告警持久化存储

### 外部集成层

- `api.go` - 多数据源 API 调用、自动容错
- 支持：腾讯、新浪、东方财富、Finnhub、TwelveData

### 日志与国际化

- `logger.go` - zap 日志系统、日志轮转
- `i18n.go` - 多语言支持（中文/英文）
- `log.go` - 日志接口

### 测试文件

```
*_test.go 文件:
- alert_frequency_test.go      (12 个测试用例)
- intraday_test.go             (市场检测、采集模式测试)
- api_test.go                  (API 回退测试)
- ...
```

## 数据文件说明

### portfolio.json - 持仓数据

```json
[
  {
    "code": "SH601138",
    "name": "工业富联",
    "cost": 51.98,
    "quantity": 500
  }
]
```

### watchlist.json - 自选数据

```json
[
  {
    "code": "SH600000",
    "name": "浦发银行",
    "tags": ["银行", "持仓"]
  }
]
```

### alert_data.json - 告警配置

```json
[
  {
    "code": "SH601138",
    "type": "price",
    "condition": "higher",
    "value": 120.0,
    "frequency": "daily",
    "enabled": true
  }
]
```

### intraday/ - 分时数据

```
data/intraday/
├── SH600000/                    # 股票代码
│   ├── 20260101.json           # 日期
│   └── 20260102.json
└── AAPL/
    └── 20260101.json
```

**分时数据格式**：
```json
[
  {
    "timestamp": "09:30:00",
    "price": 61.50,
    "volume": 1000000
  }
]
```

## 配置文件说明

### config.yml

- 系统设置：语言、启动模块、调试模式
- 显示设置：颜色、小数位、表格样式
- 更新设置：刷新间隔、自动更新
- 列配置：持股/自选表格列
- 分时采集：智能停止、完整性阈值

详见：[配置说明](../guides/configuration.md)

## 国际化文件

### zh.json / en.json

每个文件包含约 250 条翻译条目：

```json
{
  "menu.portfolio": "持股列表 / Portfolio",
  "menu.watchlist": "自选列表 / Watchlist",
  "menu.alert": "告警管理 / Alerts",
  "action.add": "添加 / Add",
  "action.edit": "编辑 / Edit"
}
```

## 模块依赖关系

```
main.go (状态机)
  ├─ alert.go (告警核心)
  │  ├─ alert_frequency.go (频率控制)
  │  ├─ persistence.go (数据存储)
  │  └─ holiday_worker.go (交易日历)
  │
  ├─ intraday.go (分时采集)
  │  ├─ api.go (API 调用)
  │  ├─ market/timezone.go (时区处理)
  │  └─ persistence.go (数据存储)
  │
  ├─ watchlist.go (自选管理)
  │  ├─ sort.go (排序)
  │  └─ persistence.go (数据存储)
  │
  ├─ cache.go (价格缓存)
  ├─ api.go (多源数据获取)
  └─ ui_utils.go (UI 工具)
```

## 文件组织原则

- **单一职责**：每个文件聚焦一个功能模块
- **测试覆盖**：核心模块都有对应的 `*_test.go`
- **数据分离**：代码与数据分开存储
- **配置集中**：配置统一在 `config.yml`

## 代码统计

| 类别 | 行数 | 说明 |
|------|------|------|
| 源代码 | ~9,773 | 主程序逻辑 |
| 内部包 | ~12,090 | 模块化包结构 |
| 测试代码 | ~2,000+ | 40+ 个测试用例 |
| 总计 | ~21,863 | 完整项目 |

---

详见项目的 CLAUDE.md 文档获取更多信息。
