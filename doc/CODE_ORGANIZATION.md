# Code Organization Map

This document provides a quick reference for navigating the stock-monitor codebase.

## File Organization

### Root Directory (Main Business Logic)

#### Core Application (~3,259 lines)
- `main.go` - Main application entry, Model definition, 29 state handlers, Update/View methods

#### Business Modules
- `api.go` (~1,355 lines) - Stock data API integration (Tencent, Sina, TwelveData, Yahoo, East Money)
- `intraday.go` (~1,535 lines) - Minute-level data collection and management
- `intraday_chart.go` (~1,260 lines) - Intraday chart rendering with Braille characters
- `alert.go` (~2,572 lines) - Alert system (checking, notification, batch operations)
- `watchlist.go` (~883 lines) - Watchlist management and tag operations

#### Supporting Modules
- `persistence.go` (~414 lines) - Data persistence (JSON/YAML I/O)
- `cache.go` (~127 lines) - Stock price caching (30s TTL, RWMutex)
- `alert_frequency.go` (~161 lines) - Alert frequency control logic
- `holiday_worker.go` (~278 lines) - Trading day calculation
- `timezone.go` (~172 lines) - Market-specific timezone handling

#### Utilities
- `ui_utils.go` (~194 lines) - UI rendering utilities (table, Chinese width)
- `format.go` (~156 lines) - Number/price/percentage formatting
- `color.go` - Color utilities
- `scroll.go` - Pagination logic
- `i18n.go` - I18n text retrieval

#### Logging
- `logger.go` (~189 lines) - Structured logging with zap (v5.8)
- `log.go` (~129 lines) - Logging interface with i18n

#### Type Definitions
- `types.go` (~425 lines) - Core data structures (Stock, Alert, Config, etc.)
- `consts.go` - Constants and enums

### Internal Packages (Modularized Code)

#### internal/types/
- `stock.go` - Stock-related type definitions
- `alert.go` - Alert-related types
- `watchlist.go` - Watchlist types
- `portfolio.go` - Portfolio types
- `config.go` - Configuration types
- `cache.go` - Cache types
- `intraday.go` - Intraday data types
- `interfaces.go` - Interface definitions

#### internal/ui/
- `color.go` - Color scheme utilities
- `format.go` - Formatting functions
- `input.go` - Input handling
- `scroll.go` - Scroll/pagination

#### internal/data/
- `persistence.go` - Data I/O operations
- `migration.go` - Data migration logic

#### internal/market/
- `timezone.go` - Timezone handling
- `trading.go` - Trading hours/calendar

#### internal/sort/
- `sorter.go` - Sorting engine interface and implementation

#### internal/alert/
- `frequency.go` - Frequency control (canTrigger logic)

#### internal/log/
- `logger.go` - Logger implementation

#### internal/i18n/
- `i18n.go` - I18n text retrieval

#### internal/consts/
- `consts.go` - Application constants

### Configuration and Data

#### cmd/conf/
- `config.yml` - User configuration (auto-generated)
- `config_demo.yaml` - Configuration template

#### data/
- `portfolio.json` - Portfolio stocks
- `watchlist.json` - Watchlist stocks with tags
- `alert_data.json` - Alert configurations
- `intraday/` - Minute-level data (by code/date)

### Internationalization

#### i18n/
- `zh.json` - Chinese translations
- `en.json` - English translations

### Tests

- `alert_frequency_test.go` (~419 lines) - Alert frequency tests (12 cases)
- `intraday_test.go` (~328 lines) - Intraday collection tests (6 cases)
- `api_test.go` (~87 lines) - API integration tests (3 cases)
- `types_test.go` - Type tests (5 cases)

### Documentation

#### doc/
- `ARCHITECTURE_REFACTORING.md` - Refactoring roadmap and module boundaries
- `README.md` - Main documentation
- `README_EN.md` - English documentation
- `changelogs/` - Version history
- `issues/` - Feature documentation
- `plans/` - Implementation plans

## Quick Navigation Guide

### To find:

**State handling logic** → `main.go` (29 handler functions)
- `handleMainMenu()`, `handleMonitoring()`, `handleAlertManage()`, etc.

**API integration** → `api.go`
- Search functions, price fetching, multi-API fallback

**Data collection** → `intraday.go`
- Background worker pool, minute-level data collection

**Alert system** → `alert.go` + `alert_frequency.go`
- Alert checking, notification, frequency control

**UI rendering** → `main.go` View() + `alert.go` render functions
- Portfolio table, watchlist, alert management UI

**Data structures** → `types.go` + `internal/types/*.go`
- Stock, Alert, Portfolio, Watchlist, Config

**Configuration** → `persistence.go` + `internal/data/`
- Load/save operations, migrations

**Formatting** → `format.go` + `internal/ui/format.go`
- Price, percentage, number formatting

**Internationalization** → `i18n.go` + `internal/i18n/` + `i18n/*.json`
- Text retrieval, translations

**Testing** → `*_test.go` files
- Alert frequency, intraday, API tests

## Module Refactoring Status

- ✅ **Completed**: Types, UI utilities, Data persistence, Market logic, Sort
- 📋 **Planned**: API, Intraday, Alert, State handlers (see ARCHITECTURE_REFACTORING.md)
- 🔄 **In Progress**: Documentation and organization

## Code Statistics

```
Main Business Logic:
  main.go              3,259 lines  (State machine, UI, handlers)
  alert.go             2,572 lines  (Alert system)
  intraday.go          1,535 lines  (Data collection)
  api.go               1,355 lines  (API integration)
  intraday_chart.go    1,260 lines  (Chart rendering)
  watchlist.go           883 lines  (Watchlist management)

Supporting:
  persistence.go         414 lines
  types.go               425 lines
  holiday_worker.go      278 lines

Internal Packages:
  ~2,888 lines across 20 files

Tests:
  ~900 lines (26 test cases)

Total: ~15,000 lines of Go code
```

---

For detailed refactoring plans, see `doc/ARCHITECTURE_REFACTORING.md`.
