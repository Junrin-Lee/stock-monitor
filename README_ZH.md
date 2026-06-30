# Stock Monitor

[![Version](https://img.shields.io/badge/version-v9.2-blue.svg)]()
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)]()
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey.svg)]()

> **AI Generated Repository**: 本项目完全由AI生成，包括代码架构、功能实现和项目文档。

**多语言文档**: [中文](README_ZH.md) | [English](README.md)

---

## 📖 项目简介

专业的命令行股票监控工具，基于 Bubble Tea 框架构建现代化终端 UI，支持 **A 股、美股、港股** 实时行情追踪与投资组合管理。

---

## ⚡ 快速安装

选择你的操作系统，一分钟内即可开始使用！

### macOS

**方式 1: Homebrew（推荐 ⭐）**

```bash
# 添加 tap 并安装
brew tap Junrin-Lee/stock-monitor https://github.com/Junrin-Lee/stock-monitor.git
brew install stock-monitor

# 运行
stock-monitor

# 升级到最新版本
brew upgrade stock-monitor
```

**方式 2: DMG 安装包**

1. 从 [Releases](https://github.com/Junrin-Lee/stock-monitor/releases) 下载 `stock-monitor_vX.X.X_macOS_universal.dmg`
2. 双击打开 DMG 文件
3. 将 `stock-monitor` 拖拽到 `/Applications` 或其他目录
4. 首次运行：右键 → 打开（绕过 Gatekeeper 警告）

**方式 3: 手动安装**

```bash
# 下载最新版本（替换 vX.X.X 为实际版本号）
curl -LO https://github.com/Junrin-Lee/stock-monitor/releases/latest/download/stock-monitor_vX.X.X_macOS_universal.tar.gz
tar -xzf stock-monitor_vX.X.X_macOS_universal.tar.gz

# 移动到 PATH
sudo mv stock-monitor /usr/local/bin/

# 运行
stock-monitor
```

**macOS 额外配置 - 安装 terminal-notifier（用于告警通知）**

```bash
brew install terminal-notifier
```

> 💡 告警系统需要 `terminal-notifier` 来发送系统通知。如果未安装，告警仍会在应用内显示，但不会弹出系统通知。

---

### Linux

**Debian / Ubuntu（推荐 ⭐）**

```bash
# 下载 .deb 包（替换 vX.X.X 为实际版本号）
wget https://github.com/Junrin-Lee/stock-monitor/releases/latest/download/stock-monitor_vX.X.X_linux_amd64.deb

# 安装
sudo dpkg -i stock-monitor_vX.X.X_linux_amd64.deb

# 运行
stock-monitor
```

**RHEL / Fedora / CentOS**

```bash
# 下载 .rpm 包
wget https://github.com/Junrin-Lee/stock-monitor/releases/latest/download/stock-monitor_vX.X.X_linux_amd64.rpm

# 安装
sudo rpm -i stock-monitor_vX.X.X_linux_amd64.rpm

# 运行
stock-monitor
```

**其他发行版（tar.gz）**

```bash
# 下载并解压
wget https://github.com/Junrin-Lee/stock-monitor/releases/latest/download/stock-monitor_vX.X.X_Linux_x86_64.tar.gz
tar -xzf stock-monitor_vX.X.X_Linux_x86_64.tar.gz

# 移动到 PATH
sudo mv stock-monitor /usr/local/bin/

# 运行
stock-monitor
```

---

### Windows

1. 从 [Releases](https://github.com/Junrin-Lee/stock-monitor/releases) 下载 `stock-monitor_vX.X.X_Windows_x86_64.zip`
2. 解压到任意目录（如 `C:\Program Files\stock-monitor\`）
3. （可选）将目录添加到系统 PATH 环境变量
4. 打开 PowerShell 或 CMD，运行：

```powershell
# 导航到解压目录
cd "C:\Program Files\stock-monitor"

# 运行程序
.\stock-monitor.exe

# 查看版本
.\stock-monitor.exe --version
```

---

### 从源代码编译（开发者）

如果你想参与开发或使用最新代码：

```bash
# 需要 Go 1.25+
go version

# 克隆项目
git clone https://github.com/Junrin-Lee/stock-monitor.git
cd stock-monitor

# 下载依赖
go mod download

# 编译
go build -o stock-monitor

# 运行
./stock-monitor
```

首次运行会自动创建配置文件和数据目录。

### 添加第一只股票

进入程序后：
1. 选择"持股列表"
2. 按 `A` 键添加股票
3. 输入代码（如 `SH601138` 或 `AAPL`）
4. 输入成本价和持股数
5. 确认添加

支持的代码格式：
- A 股：`SH601138` / `SZ000001` （或无前缀）
- 美股：`AAPL` / `TSLA`
- 港股：`HK00700` / `0700.HK`

---

## ✨ 核心功能

| 功能 | 说明 |
|------|------|
| **实时监控** | 5 秒间隔自动刷新股价，掌握每个交易时刻 |
| **投资组合** | 完整的持仓管理,支持添加、修改、删除 |
| **自选列表** | 独立的股票关注列表，支持多标签分组 |
| **板块市场** 📊 | 浏览板块行情（地域、行业、概念）实时数据 |
| **走势迷你图** 📈 | 行内 Unicode 方块趋势图 (▁▂▃▄▅▆▇█)，直观展示价格走势 |
| **盘前盘后** 🌙 | 美股盘前/盘后实时价格和涨跌幅显示 |
| **分时图表** 📈 | 分时走势图带昨收零轴与涨跌双色；支持多周期 K 线 (T/D/W/M/Q/Y) |
| **告警系统** ⏰ | 价格/涨跌幅/成交量告警，5 种触发频率，跨平台通知 |
| **全球市场** | 支持 A 股、美股、港股等主流市场 |
| **多语言** | 完整的中英文双语界面支持 |
| **本地存储** | 原子写入、数据损坏保护，本地存储数据永不丢失 |

---

## 🎯 典型使用场景

### 场景 1：日内交易者
- 5 秒刷新，快速发现热点股
- 按涨幅排序，识别强势股
- 为关键价位设置一次性告警

→ [详细指南](doc/guides/use-cases.md#日内交易者)

### 场景 2：价值投资者
- 自选列表 + 标签分类，长期跟踪
- 为目标价设置每日告警
- 月度复盘，评估表现

→ [详细指南](doc/guides/use-cases.md#价值投资者)

### 场景 3：告警监控
- 价格告警：止损价、目标价
- 涨跌幅告警：强势机会
- 成交量告警：异常放量（A 股）

→ [完整告警指南](doc/guides/alert-system.md)

---

## 🎨 界面展示

### 持股列表
```
┌────────────────────────────────────────────────────────────┐
│  实时监控 - 持股列表              每 5 秒自动刷新         │
├──────────┬──────────┬────────┬─────────┬──────────┬────────┤
│   代码   │   名称   │  现价  │今日涨幅 │ 盈亏率   │ 市值   │
├──────────┼──────────┼────────┼─────────┼──────────┼────────┤
│ SH601138 │ 工业富联 │ 61.63  │ -2.91%  │ +18.6%   │ 30,815 │
│ SZ000001 │ 平安银行 │ 12.45  │ +1.22%  │ +11.2%   │ 12,450 │
│ AAPL     │ 苹果公司 │189.95  │ +0.85%  │  +8.5%   │ 18,995 │
└──────────┴──────────┴────────┴─────────┴──────────┴────────┘
```

### 基本快捷键
| 按键 | 功能 |
|------|------|
| `↑↓` `W` `S` | 上下导航 |
| `A` | 添加股票 |
| `E` | 编辑选中 |
| `D` | 删除选中 |
| `S` | 排序设置 |
| `V` | 查看分时图 |
| `ESC` / `Q` | 返回上级 |
| `Ctrl+C` | 退出程序 |

更多快捷键：[⌨️ 键盘快捷键](doc/guides/keyboard-shortcuts.md)

---

## 📚 完整文档导航

### 🚀 快速上手

- [📖 5 分钟快速开始](doc/guides/getting-started.md) - 环境配置、安装、首次使用
- [⌨️ 键盘快捷键速查](doc/guides/keyboard-shortcuts.md) - 所有快捷键和操作
- [🔔 告警系统完整指南](doc/guides/alert-system.md) - 告警类型、频率、最佳实践

### ⚙️ 配置与场景

- [🔧 配置说明](doc/guides/configuration.md) - 所有配置选项、表格列自定义
- [💡 使用场景与最佳实践](doc/guides/use-cases.md) - 不同投资风格的推荐配置

### 🔍 技术文档

- [🏗️ 系统架构](doc/technical/architecture.md) - 分层设计、并发控制、性能优化
- [📁 项目结构](doc/technical/project-structure.md) - 代码组织、模块职责
- [🌐 API 集成](doc/technical/api-integration.md) - 支持的市场、代码格式、数据源
- [🆘 故障排除](doc/technical/troubleshooting.md) - 常见问题解决方案

### 📋 版本历史

- [v9.2](doc/changelogs/v9.2.md) - 分时图表重构（昨收零轴、asciigraph 迁移、自适应 Y 轴）
- [v9.1](doc/changelogs/v9.1.md) - 数据安全加固 + 多市场时区
- [v9.0](doc/changelogs/v9.0.md) - Sparkline 走势图 + 盘前盘后行情
- [v8.1](doc/changelogs/v8.1.md) - 跨平台发布自动化
- [v8.0](doc/changelogs/v8.0.md) - 板块市场查看
- [v7.1](doc/changelogs/v7.1.md) - 告警频率编辑
- [v7.0](doc/changelogs/v7.0.md) - 架构重构、模块化
- [v6.0](doc/changelogs/v6.0.md) - 告警系统发布
- [完整版本历史](doc/changelogs/)

---

## 🎓 学习路径

**新手建议**：按以下顺序学习

```
第 1 步：快速开始 (5 分钟)
  └─ [快速开始指南](doc/guides/getting-started.md)

第 2 步：掌握操作 (10 分钟)
  └─ [键盘快捷键](doc/guides/keyboard-shortcuts.md)

第 3 步：深入告警 (20 分钟)
  └─ [告警系统指南](doc/guides/alert-system.md)

第 4 步：优化配置 (10 分钟)
  └─ [配置说明](doc/guides/configuration.md)

第 5 步：寻找场景 (按需)
  └─ [使用场景](doc/guides/use-cases.md)
```

---

## 🛠️ 常用命令

### 开发相关

```bash
# 编译
go build -o cmd/stock-monitor

# 编译 + 竞态检测（调试用）
go build -race -o cmd/stock-monitor

# 运行
./cmd/stock-monitor

# 测试
go test -v ./...

# 查看代码统计
wc -l *.go | tail -1
```

---

## 🌟 核心特性

✅ **专业级功能**
- 多维度数据：现价、成本、盈亏、涨跌幅等
- 实时计算：持仓盈亏、盈亏率、市值
- 智能排序：11 个持股字段 + 7 个自选字段

✅ **告警系统** (v6.0+)
- 3 种告警类型：价格、涨跌幅、成交量
- 5 种触发频率：一次、每日、每周、每月、自定义
- 跨平台通知：macOS / Linux / Windows

✅ **用户友好**
- Vim 风格导航：方向键、WASD、J/K
- 多标签管理：灵活分类、快速筛选
- 分时图表：昨收零轴 + 涨跌双色，一眼看清当日涨跌

✅ **开发者友好**
- 清晰的分层架构
- 模块化设计（15 个独立包）
- 40+ 单元测试
- 详细的技术文档

---

## 💬 获取帮助

- 📖 查看完整文档：[doc/](doc/)
- 🆘 故障排除：[故障排除指南](doc/technical/troubleshooting.md)
- 🐛 报告 Bug：[GitHub Issues](https://github.com/Junrin-Lee/stock-monitor/issues)

---

**温馨提示**：请在交易时间使用获得最准确数据。本工具仅供投资参考，投资有风险，入市需谨慎。

---

<div align="center">

[⬆ 回到顶部](#stock-monitor)

</div>
