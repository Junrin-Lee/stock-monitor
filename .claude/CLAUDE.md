# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **Note**: This is an AI-generated repository. The entire project was created by AI, including code architecture, implementation, and documentation.

## Project Overview

**Stock Monitor** is a professional command-line TUI (Terminal User Interface) application for real-time stock price tracking, portfolio management, and watchlist analysis. Built with Go 1.25+ using the Bubble Tea framework, it supports A-shares (Shanghai/Shenzhen), US stocks, and Hong Kong stocks with bilingual support (Chinese/English).

**Current Version**: v9.1

**Recent Architecture**: v9.1 is a stability hardening release with 26 commits focused on data safety, race condition elimination, and multi-market timezone correctness. Key changes: `atomic.Pointer` for lock-free globalModel access, per-stock in-flight guards preventing concurrent fetch races, corruption tracking with automatic backup, atomic file writes across all persistence paths, holiday calendar integration with memory cache, HKEX Closing Auction Session (16:00-16:10) support, cache eviction for expired entries, and path traversal security validation. Built on v9.0's Sparkline trends and pre/post-market features with full backward compatibility.

## Essential Commands

### Build & Run
```bash
# Build the executable
go build -o cmd/stock-monitor

# Run directly without building
go run main.go

# Run the compiled binary
./cmd/stock-monitor

# Build with race detector (for concurrency debugging)
go build -race -o cmd/stock-monitor
```

### Testing
```bash
# Run all tests
go test -v ./...

# Run tests in specific package
go test -v ./internal/alert/...

# Run specific test
go test -v ./internal/intraday/ -run TestIntradayCollection

# Run tests with coverage report
go test -v ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Run alert frequency tests (12 comprehensive test cases)
go test -v -run TestFrequency ./internal/alert/

# Run intraday tests (market detection, data validation)
go test -v -run TestIntraday ./internal/intraday/
```

### Dependencies
```bash
# Download all dependencies
go mod download

# Verify and tidy dependencies
go mod verify && go mod tidy

# View dependency tree
go mod graph | head -20
```

## Codebase Architecture

### Overview: Modular Package Structure

The project has been refactored into a clean modular architecture:

```
stock-monitor/
├── main.go, *.go              # Root-level files (legacy, being phased out)
│
├── internal/                   # Core internal packages (NEW - main codebase)
│   ├── api/                   # Multi-source stock data fetching
│   │   ├── china/             # Tencent, Sina APIs for A-shares
│   │   ├── us/                # FMP, Yahoo, TwelveData for US stocks
│   │   ├── hongkong/          # East Money for HK stocks
│   │   ├── stock.go           # Unified API interface
│   │   ├── search.go          # Stock search functionality
│   │   ├── helpers.go         # Common utilities
│   │   └── common/            # Code converters, helpers
│   │
│   ├── app/                   # Application initialization
│   │   ├── init.go            # App setup, configuration loading
│   │   └── router.go          # State routing and handlers
│   │
│   ├── data/                  # Data layer (persistence, cache)
│   │   ├── persistence.go     # JSON/YAML storage operations
│   │   ├── cache.go           # 30-second TTL price cache
│   │   ├── portfolio_repository.go  # Portfolio data access
│   │   └── *_test.go          # Data layer tests
│   │
│   ├── alert/                 # Alert system (v6.0)
│   │   ├── frequency.go       # Trigger frequency logic (O(1) checking)
│   │   └── frequency_test.go  # 12 comprehensive test cases
│   │
│   ├── intraday/              # Background minute-by-minute data collection
│   │   ├── manager.go         # Worker pool orchestration (max 10 concurrent)
│   │   ├── worker.go          # Individual data collector goroutines
│   │   ├── storage.go         # Intraday data persistence
│   │   ├── trading.go         # Trading hours, market detection
│   │   ├── api.go             # Intraday API calls
│   │   ├── helpers.go         # Utility functions
│   │   ├── types.go           # Intraday data structures
│   │   └── *_test.go          # Intraday tests
│   │
│   ├── market/                # Market-specific logic
│   │   ├── timezone.go        # Multi-market timezone handling
│   │   ├── holiday.go         # Trading day/holiday detection
│   │   └── *_test.go          # Market tests
│   │
│   ├── service/               # Business logic services
│   │   └── portfolio/         # Portfolio service layer
│   │
│   ├── ui/                    # UI rendering and interaction
│   │   ├── alert/             # Alert UI views and handlers
│   │   ├── watchlist/         # Watchlist filtering and tagging UI
│   │   ├── sector/            # Sector market view and columns
│   │   ├── sparkline/         # Sparkline trend mini-charts (v9.0)
│   │   │   └── sparkline.go   # Unicode block rendering (▁▂▃▄▅▆▇█)
│   │   ├── table/             # Table rendering
│   │   ├── color.go           # Color/theme utilities
│   │   ├── columns.go         # Column metadata and configuration
│   │   ├── format.go          # Number/price/percentage formatting
│   │   ├── input.go           # Input field handling
│   │   ├── scroll.go          # Pagination and scrolling
│   │   ├── types.go           # UI state and message types
│   │   └── view.go            # View rendering (aggregated)
│   │
│   ├── log/                   # Logging system
│   │   └── logger.go          # zap-based logger with rotation
│   │
│   ├── i18n/                  # Internationalization
│   │   └── i18n.go            # Translation management
│   │
│   ├── consts/                # Application constants
│   │   └── consts.go          # App states, limits, defaults
│   │
│   ├── types/                 # Core data types
│   │   ├── common.go          # Common types (MarketType, Sort)
│   │   ├── stock.go           # Stock, StockData types
│   │   ├── alert.go           # Alert types (v6.0)
│   │   ├── config.go          # Configuration structures
│   │   ├── model.go           # Main application Model
│   │   ├── watchlist.go       # Watchlist types
│   │   ├── intraday.go        # Intraday-specific types
│   │   ├── interfaces.go      # Core interfaces
│   │   └── *_test.go          # Type tests
│   │
│   ├── sort/                  # Sorting engine
│   │   └── sorter.go          # Field sorting for portfolio/watchlist
│   │
│   ├── errors/                # Error definitions
│   │   └── errors.go          # Custom error types
│   │
│   └── testutil/              # Testing utilities
│       ├── fixtures.go        # Test data fixtures
│       └── helpers.go         # Test helper functions
│
├── cmd/
│   ├── stock-monitor         # Compiled executable
│   └── conf/
│       ├── config.yml        # User configuration (auto-generated)
│       └── config_demo.yaml  # Configuration template
│
├── data/
│   ├── portfolio.json        # Portfolio holdings
│   ├── watchlist.json        # Watched stocks with tags
│   ├── alert_data.json       # Alert configurations (v6.0)
│   └── intraday/             # Minute-by-minute data by code/date
│
├── i18n/
│   ├── zh.json              # Chinese translations (~250 keys)
│   └── en.json              # English translations (~250 keys)
│
└── doc/
    ├── changelogs/          # Version history
    └── issues/              # Feature documentation
```

### Code Statistics

| Metric | Value | Notes |
|--------|-------|-------|
| **Total Go Code** | ~22,500+ lines | Root (9,900+) + internal (12,600+) |
| **Main Application** | main.go | State machine entry point |
| **Core Packages** | 17 internal packages | API, data, UI, sparkline, intraday, market, etc. |
| **Test Coverage** | 40+ tests | Alert, intraday, market, data, sparkline |
| **Languages** | 2 (Chinese/English) | Full bilingual support |
| **Markets Supported** | 3 (A-share, US, HK) | 30+ data sources via fallback chains |

### Critical Architecture Pattern: Layered Design

```
┌──────────────────────────────────────────────────────────┐
│  UI Layer                                                │
│  - internal/ui/: Table, Color, Format, Input, Scroll    │
│  - Alert UI, Watchlist UI                               │
│  - main.go: State machine and event loop                │
├──────────────────────────────────────────────────────────┤
│  Business Logic Layer                                    │
│  - internal/alert/: Frequency control                   │
│  - internal/intraday/: Data collection manager          │
│  - internal/market/: Timezone, trading hours            │
│  - internal/sort/: Sorting engine                       │
├──────────────────────────────────────────────────────────┤
│  Data Layer                                              │
│  - internal/data/: Persistence (JSON/YAML)              │
│  - internal/data/: Cache (30s TTL, RWMutex)             │
│  - internal/types/: Data structures                     │
├──────────────────────────────────────────────────────────┤
│  External Integration Layer                             │
│  - internal/api/: Multi-API stock data fetching         │
│  - internal/intraday/: Background collection workers   │
│  - Fallback chains: Tencent → Sina → East Money        │
├──────────────────────────────────────────────────────────┤
│  Cross-Cutting Concerns                                 │
│  - internal/log/: zap-based logging                    │
│  - internal/i18n/: Translation management              │
│  - internal/consts/, errors/: Constants and errors     │
└──────────────────────────────────────────────────────────┘
```

## Module Reference

### High-Level Module Functions

#### API Layer (`internal/api/`)
- **Purpose**: Unified stock data fetching with multi-source fallback
- **Key Files**:
  - `stock.go` - Unified interface for all markets
  - `china/tencent.go`, `china/sina.go` - A-share data (primary + fallback)
  - `us/yahoo.go`, `us/fmp.go`, `us/twelvedata.go` - US stock data
  - `hongkong/eastmoney.go` - HK stock data
  - `search.go` - Stock search across markets
- **Fallback Strategy**:
  - A-shares: Tencent → Sina → East Money
  - US: Yahoo/FMP → TwelveData
  - HK: Finnhub → East Money
- **Design Pattern**: Primary → Secondary → Tertiary, display "-" on full failure
- **Character Encoding**: GBK ↔ UTF-8 conversion for Chinese APIs

#### Data Layer (`internal/data/`)
- **Purpose**: Local data persistence and caching
- **Modules**:
  - `persistence.go` - JSON/YAML storage (portfolio, watchlist, alerts, config)
  - `cache.go` - 30-second TTL cache with RWMutex (concurrent reads)
  - `portfolio_repository.go` - Repository pattern for portfolio access
- **Key Design**: Atomic writes (temp → rename), migration support for version upgrades

#### Intraday Collection (`internal/intraday/`)
- **Purpose**: Background collection of minute-by-minute stock data
- **Architecture**: Worker pool (max 10 concurrent) with intelligent management
- **Key Files**:
  - `manager.go` - Semaphore pattern for bounded concurrency
  - `worker.go` - Individual data collector goroutines
  - `storage.go` - Per-stock/per-date file organization
  - `trading.go` - Market hours detection, trading state
- **Behavior**: Smart auto-stop when ~90% data completeness reached
- **Storage**: `data/intraday/{code}/{date}.json`

#### Alert System (`internal/alert/` + `internal/ui/alert/`)
- **Purpose**: Price/rate/volume monitoring with intelligent frequency control
- **Frequency Modes**: Once, Daily, Weekly, Monthly, Custom (every N days)
- **Trigger Logic** (`frequency.go`): O(1) boundary checking (same day/week/month)
- **UI Views** (`internal/ui/alert/view.go`): Alert management, batch operations, notification preview
- **Notification System**: macOS (NSUserNotification), Linux (D-Bus), Windows (Toast)
- **Test Coverage**: 12 comprehensive tests covering all frequency modes and edge cases

#### Market Handling (`internal/market/`)
- **Timezone Support**: Asia/Shanghai (A-share), US/Eastern (US), Asia/Hong_Kong (HK)
- **Trading Hours Detection**: Identifies if market is in trading session
- **Holiday Handling**: Weekend detection, market-specific holidays
- **Use Case**: Intraday collection auto-stop, alert boundary checking

#### UI Layer (`internal/ui/`)
- **Table Rendering** (`table/builder.go`): Column-aware formatting with Chinese width handling
- **Watchlist UI** (`watchlist/`): Tag filtering, grouping, market classification
- **Alert UI** (`alert/`): Alert list, batch operation views, notification alerts
- **Color System** (`color.go`): Red/green/white for price changes, theme support
- **Formatting** (`format.go`): Price decimals, percentage display, number grouping

### Root-Level Files (Legacy, Being Phased Out)

These files will eventually be replaced by `internal/` package implementations:

| File | Status | Replacement Path |
|------|--------|------------------|
| `main.go` | Active | Stays as entry point, orchestrates `internal/app/` and `internal/ui/` |
| `handlers_*.go` | Active | Handlers moved to `internal/ui/` and `internal/app/` |
| `intraday.go`, `alert.go` | Legacy | Replace with `internal/intraday/`, `internal/alert/` |
| `api.go` | Legacy | Use `internal/api/` package |
| `types.go`, `persistence.go`, `cache.go` | Legacy | Use `internal/types/`, `internal/data/` |

## Adding New Features

### 1. Add Stock Data Source

**Task**: Add support for a new stock API

**Steps**:
1. Create new provider in `internal/api/{market}/` (e.g., `internal/api/china/eastmoney.go`)
2. Implement `FetchStockPrice(symbol string) *StockData` function
3. Add to fallback chain in `internal/api/stock.go`
4. Handle GBK encoding if Chinese API
5. Add tests in `internal/api/common/converters_test.go`

**Example**:
```go
// internal/api/china/eastmoney.go
package china

func FetchEastMoneyPrice(symbol string) *StockData {
    // Call API
    // Parse response
    // Convert GBK to UTF-8 if needed
    return &StockData{...}
}
```

### 2. Modify UI Display

**Task**: Change table columns, colors, or formatting

**Changes**:
1. **Column Configuration**: Edit `internal/ui/columns.go`
2. **Color Scheme**: Modify `internal/ui/color.go`
3. **Number Formatting**: Update `internal/ui/format.go`
4. **Table Layout**: Adjust `internal/ui/table/builder.go`
5. **Chinese Width**: Already handled via `go-pretty` library

### 3. Add New Alert Type

**Task**: Implement a new alert trigger condition (e.g., range breakout)

**Implementation**:
1. Add type to `internal/types/alert.go`: `AlertTypeRangeBreakout`
2. Implement checker in `internal/ui/alert/checker.go`
3. Add frequency boundary logic in `internal/alert/frequency.go`
4. Create UI handler in `internal/ui/alert/view.go`
5. Add tests in `internal/alert/frequency_test.go`
6. Update i18n: `i18n/zh.json`, `i18n/en.json`

### 4. Add New Background Task

**Task**: Implement a new worker (e.g., news fetcher)

**Pattern**:
1. Create package: `internal/news/`
2. Implement manager (semaphore pattern): `internal/news/manager.go`
3. Implement worker: `internal/news/worker.go`
4. Add to main app initialization in `internal/app/init.go`
5. Handle goroutine → message passing (never mutate Model from goroutine)
6. Add tests for worker logic

## Testing Strategy

### Test Organization

```
internal/
├── alert/
│   └── frequency_test.go      # 12 tests: all frequency modes + edge cases
├── intraday/
│   ├── *_test.go              # Manager, worker, storage tests
├── market/
│   └── timezone_test.go       # Timezone conversion tests
├── data/
│   └── *_test.go              # Cache TTL, persistence tests
└── api/
    ├── common/converters_test.go  # Code conversion tests
    └── helpers_test.go            # API helper tests
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Test specific concern
go test -v ./internal/alert/         # Alert frequency
go test -v ./internal/intraday/      # Data collection
go test -v ./internal/market/        # Timezone/holidays
go test -v ./internal/api/           # API integration

# Test with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Continuous testing with race detector
go test -race -v ./...
```

### Key Test Scenarios

1. **Alert Frequency** (`internal/alert/frequency_test.go`):
   - Daily boundary: last trigger on Dec 31, check on Jan 1
   - Weekly boundary: week-end wrap-around
   - Month boundary: Feb 28/29 handling
   - Custom intervals: every 3/7 days

2. **Intraday Collection** (`internal/intraday/*_test.go`):
   - Market detection (A-share, US, HK)
   - Trading hours accuracy
   - Worker pool semaphore behavior
   - Data completeness calculation

3. **API Fallback** (`internal/api/*_test.go`):
   - Code conversion (601138 → 601138.SS)
   - HK code handling (0700 → 0700.HK)
   - Fallback chain execution

4. **Market Operations** (`internal/market/*_test.go`):
   - Timezone conversions across markets
   - Holiday/trading day detection
   - Trading session identification

## Concurrency & Thread Safety Patterns

### 1. Cache Access (RWMutex)

```go
// internal/data/cache.go
type Cache struct {
    data  map[string]*CacheEntry
    mu    sync.RWMutex              // RWMutex for high read concurrency
}

// Non-blocking reads
func (c *Cache) Get(key string) *CacheEntry {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.data[key]
}

// Exclusive writes
func (c *Cache) Set(key string, entry *CacheEntry) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = entry
}
```

### 2. Worker Pool (Semaphore Pattern)

```go
// internal/intraday/manager.go
type Manager struct {
    workerPool chan struct{}     // Buffered channel: max 10 concurrent
    activeStocks sync.Map
    mu       sync.RWMutex
}

// StartWorker acquires and releases semaphore
func (m *Manager) StartWorker(code string) {
    m.workerPool <- struct{}{}   // Acquire (blocks if full)
    defer func() { <-m.workerPool }()  // Release
    // ... worker logic ...
}
```

### 3. File Locks (sync.Map Pattern)

```go
// internal/intraday/storage.go
var fileLocks sync.Map  // map[string]*sync.Mutex

func saveData(key string, data []byte) error {
    lock, _ := fileLocks.LoadOrStore(key, &sync.Mutex{})
    lock.(*sync.Mutex).Lock()
    defer lock.(*sync.Mutex).Unlock()

    // Atomic write: temp → rename
    tmpPath := path + ".tmp"
    os.WriteFile(tmpPath, data, 0644)
    os.Rename(tmpPath, path)
    return nil
}
```

### 4. Message Passing (Never Mutate Model from Goroutine)

```go
// Rule: Goroutines MUST NOT directly modify Model state
// Instead: Send messages via tea.Cmd

// ✅ Correct: Send message
go func() {
    data := fetchPriceFromAPI(symbol)
    cmdChan <- tea.Cmd(func() tea.Msg {
        return StockPriceUpdateMsg{Symbol: symbol, Data: data}
    })
}()

// ❌ Wrong: Direct mutation (race condition!)
go func() {
    model.stockPriceCache[symbol] = fetchPrice(symbol)  // WRONG!
}()
```

## Configuration System

### Config File: `cmd/conf/config.yml`

```yaml
system:
  language: zh                  # Language: zh, en
  auto_start: true             # Skip menu if data exists
  startup_module: portfolio     # Initial module: portfolio, watchlist
  debug_mode: false            # Enable debug logging

display:
  color_scheme: professional   # Theme: professional, simple
  decimal_places: 3            # Price precision
  table_style: light           # go-pretty style: light, bold
  max_lines: 10                # Rows per page
  portfolio_highlight: yellow  # Highlight color for holdings

update:
  refresh_interval: 5          # Price update frequency (seconds)
  auto_update: true            # Enable auto-refresh

intraday_collection:
  enable_auto_stop: true       # Smart stop at 90% completeness
  completeness_threshold: 90.0 # Data completeness %
  max_consecutive_errors: 5    # Max consecutive API failures

markets:
  china:
    timezone: "Asia/Shanghai"
    trading_sessions:
      - start_time: "09:30"
        end_time: "11:30"
      - start_time: "13:00"
        end_time: "15:00"
```

### Internationalization (i18n)

**Location**: `i18n/zh.json` (Chinese), `i18n/en.json` (English)

**Usage in code**:
```go
import "stock-monitor/internal/i18n"

// Get translated text
text := i18n.Get("alert.frequency.daily")

// With placeholders
text := i18n.Getf("log.cache.update", count)  // "Starting async update for %d stocks"
```

**Adding new translations**:
1. Add key to both `i18n/zh.json` and `i18n/en.json`
2. Use consistent key naming: `module.feature.item`
3. Example: `"alert.frequency.daily": "每天一次"` (zh), `"Daily"` (en)

## Common Debugging Workflows

### Stock Data Shows "-"

**Diagnosis**:
1. Check network connectivity
2. Verify stock code format in `internal/types/stock.go`
3. Check API fallback chain in `internal/api/stock.go`
4. Enable debug logging in config: `debug_mode: true`

**Debug Steps**:
```bash
# Test API directly
go run main.go  # Enter debug mode via 'D' key

# Check specific API provider
go test -v -run TestTencentAPI ./internal/api/china/

# Trace API calls with logging
grep -r "fetchTencentAPI" internal/api/
```

### Intraday Data Not Collecting

**Check**:
1. Market trading hours in `internal/market/trading.go`
2. Worker pool status in `internal/intraday/manager.go`
3. Storage permissions for `data/intraday/` directory
4. API responses in `internal/intraday/worker.go`

**Verify**:
```bash
# Check if workers are running
ps aux | grep stock-monitor

# Inspect intraday data directory
ls -la data/intraday/

# Test market detection
go test -v -run TestMarketTradingHours ./internal/market/
```

### Alert Not Triggering

**Check**:
1. Alert enabled status (✓ checkbox in UI)
2. Frequency logic in `internal/alert/frequency.go`
3. Holiday/trading day in `internal/market/holiday.go`
4. Price meets condition in `internal/ui/alert/checker.go`

**Debug**:
```bash
# Test frequency logic for all modes
go test -v ./internal/alert/

# Run specific frequency test
go test -v -run TestFrequencyDaily ./internal/alert/frequency_test.go

# Check alert last trigger time
cat data/alert_data.json | jq '.[] | select(.code=="AAPL") | .last_trigger_time'
```

### Concurrency Issues

**Use race detector**:
```bash
# Build with race detection
go build -race -o cmd/stock-monitor

# Run tests with race detector
go test -race -v ./...

# Run specific package with race detection
go test -race -v ./internal/data/
```

**Common issues**:
- Cache not protected by RWMutex (check `internal/data/cache.go`)
- Goroutine mutating Model directly (must use message passing)
- Worker pool semaphore misuse (check `internal/intraday/manager.go`)

## Important Patterns & Warnings

⚠️ **Never mutate Model from goroutines** - Always send messages via `tea.Cmd()`

⚠️ **API failures gracefully** - Display "-" for missing data, never crash on API errors

⚠️ **Chinese character width is 2 cells** - Use `go-pretty` library for alignment

⚠️ **Re-sort after price updates** - Sorting is not cached, recalculated with fresh data

⚠️ **GBK encoding for Chinese APIs** - Convert with `golang.org/x/text/encoding/simplifiedchinese`

⚠️ **30-second cache TTL** - Don't bypass or APIs get overwhelmed

⚠️ **Worker pool limit is 10** - Semaphore protects system resources, don't increase arbitrarily

⚠️ **Alert frequency relies on time boundaries** - Test edge cases (month-end, week-start, day transitions)

⚠️ **Market hours vary by region** - Use `internal/market/trading.go` for accurate times

⚠️ **File atomic writes** - Always write to temp file first, then rename to ensure atomicity

## Performance Considerations

| Component | Strategy | Limit |
|-----------|----------|-------|
| **Cache** | 30-second TTL | Prevents API overload |
| **API Calls** | Batched every 5 seconds | Not per-stock |
| **Worker Pool** | Semaphore (max 10) | Protects system resources |
| **Sorting** | On-demand, not cached | ~1000 stocks comfortable |
| **Intraday Data** | Auto-stop at 90% completeness | Saves bandwidth |
| **UI Rendering** | Pagination via max_lines | Typically 10-20 rows |

## Backward Compatibility

- **Data migrations**: `internal/data/persistence.go` handles format upgrades
- **Config migrations**: Automatic version detection and upgrade
- **File formats**: JSON stays compatible across versions
- **Alert system**: Gracefully handles missing alert_data.json on first run

## File Organization Quick Reference

| Task | Primary Location | Secondary |
|------|------------------|-----------|
| Add API source | `internal/api/{market}/` | Update `internal/api/stock.go` |
| Fix stock price display | `internal/ui/format.go` | Check `internal/api/` fallback |
| Modify alert UI | `internal/ui/alert/` | Update i18n keys |
| Change table layout | `internal/ui/table/builder.go` | Adjust column config |
| Add configuration option | `cmd/conf/config_demo.yaml` | Handle in `internal/data/persistence.go` |
| Add new market | `internal/market/timezone.go` | Add API provider in `internal/api/` |
| Fix concurrency issue | Use race detector: `go build -race` | Check RWMutex usage |
| Optimize performance | `internal/data/cache.go` | Check worker pool limits |

## Architecture Evolution Timeline

| Version | Key Changes |
|---------|-------------|
| **v9.1** | 🛡️ Data safety hardening (atomic writes, corruption tracking, backup); Race condition fixes (`atomic.Pointer` globalModel, per-stock in-flight guard, search worker race); Security (path traversal validation); Resource leaks (FD leak on log rotation, HTTP body leak); Multi-market timezone (holiday-aware `IsMarketOpen`/`FindNextTradingDay`, HKEX CAS 16:00-16:10); Holiday calendar with memory cache; Cache eviction; Strict weak ordering fix |
| **v9.0** | 📈 Sparkline trend mini-charts (Unicode ▁▂▃▄▅▆▇█); US stock pre/post-market prices; A-share auction & lunch break support; Multi-period charts (5D/1M/3M/1Y); Intraday Volume field; Sector sparkline integration; New packages: `sparkline/`, `intraday/loader.go` |
| **v8.1** | 🚀 Cross-platform release automation - GoReleaser multi-platform builds (Windows/macOS/Linux); GitHub Actions CI/CD pipeline; Homebrew formula support; Enhanced Makefile with release commands; Version info in binary |
| **v8.0** | 📊 Sector market view feature - Browse regional/industry/concept sectors with real-time data; Sort state optimization for CRUD operations; Alert UpdatedAt tracking |
| **v7.1** | 🔔 Alert frequency editing - Support modifying trigger frequency in edit flow; Smart state reset for Once→Periodic transitions |
| **v7.0** | 🏗️ Complete architecture refactoring to modular `internal/` packages; 15 independent modules; 40+ unit tests; clear layered design (UI → Business Logic → Data → External Integration) |
| **v6.0** | 🔔 Alert system with frequency control, batch operations, cross-platform notifications; 11 new UI states; 12 comprehensive test cases |
| v5.8 | Structured logging with zap |
| v5.7 | Tag grouping system |
| v5.6 | Search integration |
| v5.5 | Multi-market support with auto-detection |
| v5.0 | Complete modularization from monolithic structure |

## Future Development

1. **Phase out root-level files** - Move remaining logic to `internal/` packages
2. **Package boundaries** - Each `internal/` package should be independently testable
3. **API improvements** - Add caching layer for rarely-changing data (company info)
4. **Database migration** - For >1000 stocks, consider SQLite instead of JSON files
5. **Performance profiling** - Identify bottlenecks with pprof
6. **Additional markets** - Add support for crypto, futures, bonds
7. **Trading API integration** - Research in `doc/plans/TRADING_API_RESEARCH.md`

---

**Version**: v9.1 (Based on codebase state of April 8, 2026)
**Maintained for**: Claude Code v1.3.6+, Go 1.25+
**Status**: ✅ Production Ready - Data safety hardened, multi-market timezone, holiday calendar, race-free concurrency