# ⚙️ 配置说明

详细的配置选项和自定义指南。

## 目录

- [配置文件位置](#配置文件位置)
- [基础配置](#基础配置)
- [显示配置](#显示配置)
- [表格列配置](#表格列配置)
- [分时数据采集](#分时数据采集)
- [配置示例](#配置示例)

---

## 配置文件位置

首次运行时自动创建：

**路径**：`cmd/conf/config.yml`

**相对于项目根目录**：
```
stock-monitor/
├── cmd/
│   ├── conf/
│   │   └── config.yml  ← 配置文件在这里
│   ├── stock-monitor   ← 可执行文件
│   └── ...
└── ...
```

---

## 基础配置

### system 配置段

```yaml
system:
  language: zh                    # 系统语言: zh (中文), en (英文)
  auto_start: true               # 有数据时自动进入监控: true/false
  startup_module: portfolio       # 启动模块: portfolio (持股), watchlist (自选)
  debug_mode: false              # 调试模式: true/false
```

| 配置项 | 可选值 | 说明 |
|--------|--------|------|
| `language` | `zh` / `en` | 中文或英文界面 |
| `auto_start` | `true` / `false` | true = 自动进入监控，false = 显示菜单 |
| `startup_module` | `portfolio` / `watchlist` | 优先启动哪个模块 |
| `debug_mode` | `true` / `false` | 开启调试日志 |

**示例**：
```yaml
system:
  language: zh              # 中文界面
  auto_start: false         # 首次启动显示菜单
  startup_module: watchlist # 优先显示自选列表
  debug_mode: true          # 开启调试
```

---

## 显示配置

### display 配置段

```yaml
display:
  color_scheme: professional     # 配色方案: professional (专业), simple (简洁)
  decimal_places: 3              # 价格小数位: 1-4
  table_style: light             # 表格样式: light, bold, simple
  max_lines: 10                  # 每页显示行数
  portfolio_highlight: yellow    # 持仓高亮颜色
```

| 配置项 | 可选值 | 默认值 | 说明 |
|--------|--------|--------|------|
| `color_scheme` | `professional` / `simple` | `professional` | 颜色主题 |
| `decimal_places` | 1-4 | 3 | 股价显示小数点位数 |
| `table_style` | `light` / `bold` / `simple` | `light` | 表格线条风格 |
| `max_lines` | 正整数 | 10 | 单页最多显示行数 |
| `portfolio_highlight` | 颜色名 | `yellow` | 持仓高亮色 |

**颜色选项**（用于 `portfolio_highlight`）：
```
black（黑）、red（红）、green（绿）、yellow（黄）
blue（蓝）、magenta（品红）、cyan（青）、white（白）
```

**示例**：
```yaml
display:
  color_scheme: professional    # 专业配色
  decimal_places: 2             # 显示 2 位小数
  table_style: bold             # 粗线表格
  max_lines: 20                 # 每页 20 行
  portfolio_highlight: green    # 持仓用绿色高亮
```

---

## 更新配置

### update 配置段

```yaml
update:
  refresh_interval: 5            # 刷新间隔(秒): 正整数
  auto_update: true              # 自动刷新: true/false
```

| 配置项 | 可选值 | 默认值 | 说明 |
|--------|--------|--------|------|
| `refresh_interval` | 1-60 | 5 | 多少秒刷新一次股价 |
| `auto_update` | `true` / `false` | `true` | 是否自动刷新 |

**说明**：
- `refresh_interval` 越小，更新越频繁，但网络压力更大
- 推荐值：3-10 秒
- 最小值：1 秒（风险：容易触发 API 限流）

**示例**：
```yaml
update:
  refresh_interval: 3            # 每 3 秒刷新一次
  auto_update: true              # 启用自动刷新
```

---

## 表格列配置

### 持股列表列配置

**所有可用列**（按默认顺序）：
```yaml
display:
  portfolio_columns:
    - cursor              # 必需：光标指示器
    - code                # 必需：股票代码
    - name                # 必需：股票名称
    - price               # 必需：现价
    - prev_close          # 可选：昨收价
    - open                # 可选：开盘价
    - high                # 可选：最高价
    - low                 # 可选：最低价
    - cost                # 可选：成本价
    - quantity            # 可选：持股数
    - today_change        # 可选：今日涨幅
    - position_profit     # 可选：持仓盈亏
    - profit_rate         # 可选：盈亏率
    - market_value        # 可选：市值
```

**配置示例**：

简洁模式（只显示核心信息）：
```yaml
display:
  portfolio_columns:
    - cursor
    - code
    - name
    - price
    - today_change
    - position_profit
    - profit_rate
```

详细模式（显示所有列）：
```yaml
display:
  portfolio_columns:
    - cursor
    - code
    - name
    - prev_close
    - open
    - high
    - low
    - price
    - cost
    - quantity
    - today_change
    - position_profit
    - profit_rate
    - market_value
```

### 自选列表列配置

**所有可用列**：
```yaml
display:
  watchlist_columns:
    - cursor              # 必需：光标指示器
    - tag                 # 必需：标签
    - code                # 必需：股票代码
    - name                # 必需：股票名称
    - price               # 必需：现价
    - prev_close          # 可选：昨收价
    - open                # 可选：开盘价
    - high                # 可选：最高价
    - low                 # 可选：最低价
    - today_change        # 可选：今日涨幅
    - turnover            # 可选：换手率
    - volume              # 可选：成交量
```

**配置示例**：

简洁模式：
```yaml
display:
  watchlist_columns:
    - cursor
    - tag
    - code
    - name
    - price
    - today_change
```

详细模式：
```yaml
display:
  watchlist_columns:
    - cursor
    - tag
    - code
    - name
    - price
    - prev_close
    - open
    - high
    - low
    - today_change
    - turnover
    - volume
```

**使用提示**：
- 列按配置中的顺序从左到右显示
- 注释掉某行即可隐藏该列（必需列除外）
- 必需列缺失时会自动补全
- 修改配置后重启应用生效

---

## 分时数据采集

### intraday_collection 配置段

```yaml
intraday_collection:
  enable_auto_stop: true            # 启用智能自动停止
  completeness_threshold: 90.0      # 完整性阈值(百分比)
  max_consecutive_errors: 5         # 最大连续错误次数
  min_datapoints: 20                # 最小数据点数量
```

| 配置项 | 默认值 | 范围 | 说明 |
|--------|--------|------|------|
| `enable_auto_stop` | true | true/false | 达到阈值时自动停止采集 |
| `completeness_threshold` | 90.0 | 50.0-100.0 | 认为数据完整的百分比 |
| `max_consecutive_errors` | 5 | 1-20 | 连续错误超过此数时停止 |
| `min_datapoints` | 20 | 10-100 | 最少需要的数据点数 |

**场景配置**：

低网络带宽：
```yaml
intraday_collection:
  enable_auto_stop: true
  completeness_threshold: 85.0      # 降低阈值，快速停止
  max_consecutive_errors: 3         # 较低容错
  min_datapoints: 15
```

数据质量优先：
```yaml
intraday_collection:
  enable_auto_stop: true
  completeness_threshold: 95.0      # 提高阈值，确保完整
  max_consecutive_errors: 10        # 较高容错
  min_datapoints: 50
```

---

## 完整配置示例

### 推荐配置（默认）

```yaml
system:
  language: zh
  auto_start: true
  startup_module: portfolio
  debug_mode: false

display:
  color_scheme: professional
  decimal_places: 3
  table_style: light
  max_lines: 10
  portfolio_highlight: yellow

update:
  refresh_interval: 5
  auto_update: true

display:
  portfolio_columns:
    - cursor
    - code
    - name
    - price
    - today_change
    - position_profit
    - profit_rate

  watchlist_columns:
    - cursor
    - tag
    - code
    - name
    - price
    - today_change

intraday_collection:
  enable_auto_stop: true
  completeness_threshold: 90.0
  max_consecutive_errors: 5
  min_datapoints: 20
```

### 快速响应配置（日内交易）

```yaml
system:
  language: zh
  auto_start: true
  startup_module: portfolio
  debug_mode: false

display:
  color_scheme: professional
  decimal_places: 2
  table_style: bold
  max_lines: 15
  portfolio_highlight: yellow

update:
  refresh_interval: 2               # 2 秒更新一次
  auto_update: true

display:
  portfolio_columns:                # 精简显示
    - cursor
    - code
    - name
    - price
    - today_change
    - position_profit

intraday_collection:
  enable_auto_stop: false           # 持续采集
  completeness_threshold: 100.0
  max_consecutive_errors: 10
  min_datapoints: 50
```

### 深度分析配置（价值投资）

```yaml
system:
  language: zh
  auto_start: true
  startup_module: portfolio
  debug_mode: true                  # 启用调试

display:
  color_scheme: professional
  decimal_places: 4                 # 更精细的价格
  table_style: light
  max_lines: 20
  portfolio_highlight: blue

update:
  refresh_interval: 10              # 10 秒更新一次
  auto_update: true

display:
  portfolio_columns:                # 显示所有信息
    - cursor
    - code
    - name
    - prev_close
    - open
    - high
    - low
    - price
    - cost
    - quantity
    - today_change
    - position_profit
    - profit_rate
    - market_value

intraday_collection:
  enable_auto_stop: true
  completeness_threshold: 95.0      # 优先质量
  max_consecutive_errors: 15
  min_datapoints: 100
```

---

## 配置修改后

**何时生效**：
- 修改配置后需要**重启应用**生效

**常见修改场景**：
```
场景 1: 显示行数太多/太少
  → 修改 display.max_lines

场景 2: 股价更新太快/太慢
  → 修改 update.refresh_interval

场景 3: 显示的列不够用/太多
  → 修改 display.portfolio_columns 或 watchlist_columns

场景 4: 需要更精细的数据
  → 修改 display.decimal_places

场景 5: 分时采集数据不完整
  → 调整 intraday_collection 参数
```

---

**相关文档**：
- 📖 [快速开始指南](getting-started.md)
- ⌨️ [键盘快捷键](keyboard-shortcuts.md)
- 🔔 [告警系统](alert-system.md)
