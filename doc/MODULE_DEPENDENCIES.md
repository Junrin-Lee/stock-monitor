# Module Dependency Graph

This document visualizes the dependencies between different modules in the stock-monitor project.

## Current Architecture (Simplified)

```
┌─────────────────────────────────────────────────────────────┐
│                         main.go                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  State Machine (29 states)                          │   │
│  │  - Init(), Update(), View()                         │   │
│  │  - State handlers (handle*)                         │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  Depends on ↓                                               │
└───────┬──────────────────────────┬───────────┬──────────────┘
        │                          │           │
        ↓                          ↓           ↓
┌──────────────┐          ┌──────────────┐  ┌──────────────┐
│   api.go     │          │  alert.go    │  │ intraday.go  │
│ (1,355 lines)│          │ (2,572 lines)│  │ (1,535 lines)│
│              │          │              │  │              │
│ - Search     │          │ - Checker    │  │ - Manager    │
│ - Price      │          │ - Notify     │  │ - Collector  │
│ - Multi-API  │          │ - Batch Ops  │  │ - Worker     │
└──────────────┘          └──────────────┘  └──────────────┘
        │                         │                  │
        └─────────────┬───────────┴──────────────────┘
                      ↓
           ┌─────────────────────┐
           │  Supporting Modules │
           └─────────────────────┘
                      │
        ┌─────────────┼────────────┬──────────────┐
        ↓             ↓            ↓              ↓
 ┌────────────┐ ┌──────────┐ ┌─────────┐  ┌────────────┐
 │persistence │ │  cache   │ │ types   │  │  ui_utils  │
 │   .go      │ │   .go    │ │  .go    │  │    .go     │
 └────────────┘ └──────────┘ └─────────┘  └────────────┘
        │                         │              │
        ↓                         ↓              ↓
 ┌────────────────────────────────────────────────────┐
 │              internal/ packages                    │
 │  ┌─────┐ ┌─────┐ ┌──────┐ ┌────┐ ┌────┐ ┌─────┐  │
 │  │types│ │data │ │market│ │sort│ │log │ │i18n │  │
 │  └─────┘ └─────┘ └──────┘ └────┘ └────┘ └─────┘  │
 └────────────────────────────────────────────────────┘
```

## Detailed Module Dependencies

### Layer 1: Core Application
- **main.go**: Depends on everything below
  - Direct dependencies: api, alert, intraday, watchlist, persistence, cache, types, ui_utils
  - Bubble Tea framework integration
  - 230+ Model fields

### Layer 2: Business Logic Modules

#### api.go
```
api.go
├── → types.go (StockData)
├── → ui_utils.go (gbkToUtf8)
├── → log.go (logging functions)
└── External APIs:
    ├── Tencent API
    ├── Sina API
    ├── TwelveData API
    ├── Yahoo Finance API
    └── East Money API
```

#### alert.go
```
alert.go
├── → types.go (Alert, TriggerFrequency)
├── → alert_frequency.go (canTriggerInCurrentPeriod)
├── → holiday_worker.go (trading day check)
├── → api.go (getStockPrice for checking)
├── → cache.go (stock price)
├── → log.go (logging)
└── → ui_utils.go (UI rendering)
```

#### intraday.go
```
intraday.go
├── → types.go (IntradayData)
├── → api.go (getStockPrice)
├── → cache.go (updateStockPriceCache)
├── → timezone.go (market timezone)
├── → holiday_worker.go (trading day)
├── → log.go (logging)
└── Worker Pool (max 10 concurrent)
```

### Layer 3: Supporting Modules

#### persistence.go
```
persistence.go
├── → types.go (Portfolio, Watchlist, Alert, Config)
├── → internal/data/ (data operations)
└── File I/O (JSON, YAML)
```

#### cache.go
```
cache.go
├── → types.go (StockData, StockPriceCacheEntry)
├── → sync.RWMutex (thread safety)
└── 30-second TTL
```

#### ui_utils.go
```
ui_utils.go
├── → types.go (Stock)
├── → format.go (formatting functions)
├── → go-pretty/table (rendering)
└── encoding/simplifiedchinese (GBK conversion)
```

### Layer 4: Internal Packages

```
internal/
├── types/       (no dependencies within project)
├── ui/          → types
├── data/        → types
├── market/      → types
├── sort/        → types
├── log/         → i18n
├── i18n/        (minimal dependencies)
└── consts/      (no dependencies)
```

## Circular Dependencies (Current Issues)

### Issue 1: api.go ↔ getStockPrice
- `api.go` contains `getStockPrice()`
- `getStockPrice()` calls individual API functions
- Individual API functions may need to call `getStockPrice()` again (recursive)
- **Impact**: Hard to modularize API providers

### Issue 2: alert.go → api.go → cache.go ← intraday.go
- `alert.go` calls `getStockPrice()` from `api.go`
- `intraday.go` calls `getStockPrice()` and updates cache
- All share same `cache.go` with RWMutex
- **Impact**: Tight coupling between modules

### Issue 3: Model is everywhere
- Almost all functions take `*Model` or access `Model` fields
- Model has 230+ fields
- No clear separation between data and UI state
- **Impact**: Hard to test, hard to refactor

## Proposed Architecture (Target State)

```
┌────────────────────────────────────────────────┐
│              main.go (~100 lines)              │
│  - Application bootstrap                      │
│  - Load config and data                       │
│  - Start TUI                                  │
└────────────────────────────────────────────────┘
                      │
                      ↓
┌────────────────────────────────────────────────┐
│           internal/app/ (new)                  │
│  ┌──────────────────────────────────────────┐ │
│  │  Model (data only, ~50 fields)           │ │
│  │  Init(), Update(), View()                │ │
│  └──────────────────────────────────────────┘ │
└────────────────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        ↓             ↓             ↓
┌─────────────┐ ┌──────────┐ ┌────────────┐
│internal/    │ │internal/ │ │internal/   │
│api/         │ │alert/    │ │intraday/   │
│             │ │          │ │            │
│ china/      │ │ checker  │ │ manager    │
│ us/         │ │ notifier │ │ collector  │
│ hongkong/   │ │ renderer │ │ storage    │
└─────────────┘ └──────────┘ └────────────┘
        │             │             │
        └─────────────┼─────────────┘
                      ↓
         ┌───────────────────────┐
         │  Shared Services      │
         │  - Cache              │
         │  - Logger             │
         │  - I18n               │
         └───────────────────────┘
                      ↓
         ┌───────────────────────┐
         │  Data Layer           │
         │  - Persistence        │
         │  - Types              │
         └───────────────────────┘
```

## Dependency Injection Strategy

To break circular dependencies, use dependency injection:

### Example: API Module
```go
// Current (tightly coupled)
func getStockPrice(symbol string) *StockData {
    // Directly calls tryTencentAPI, tryFinnhubAPI
}

// Target (dependency injection)
type StockPriceProvider interface {
    GetPrice(symbol string) (*StockData, error)
}

type APIClient struct {
    providers []StockPriceProvider
    cache     CacheService
    logger    LogService
}

func NewAPIClient(providers []StockPriceProvider, cache CacheService, logger LogService) *APIClient {
    // ...
}
```

### Example: Alert Module
```go
// Current (method on Model)
func (m *Model) checkAlerts() {
    // Access m.alerts, m.stockPriceCache, etc.
}

// Target (independent service)
type AlertService struct {
    priceProvider StockPriceProvider
    notifier      Notifier
    logger        LogService
}

func (s *AlertService) Check(alerts []Alert) []Alert {
    // Independent, testable logic
}
```

## Testing Strategy

### Current State
- 26 test cases, mostly focused on:
  - Alert frequency logic (12 tests)
  - Intraday collection (6 tests)
  - API integration (3 tests)
  - Types (5 tests)
- **Missing**: UI logic, state machine, business logic integration tests

### Target State
```
tests/
├── unit/                    # Pure unit tests
│   ├── api/                 # Each API provider
│   ├── alert/               # Alert logic
│   ├── intraday/            # Collection logic
│   └── cache/               # Cache TTL, concurrency
├── integration/             # Integration tests
│   ├── api_fallback_test.go
│   ├── alert_check_test.go
│   └── intraday_flow_test.go
└── e2e/                     # End-to-end (if needed)
    └── state_machine_test.go
```

## Module Size Targets

| Module | Current | Target Max | Files |
|--------|---------|------------|-------|
| main.go | 3,259 | 100 | 1 |
| api/* | 1,355 | 300/file | 8-10 |
| alert/* | 2,572 | 300/file | 5-6 |
| intraday/* | 2,795 | 300/file | 5 |
| states/* | 0 | 200/file | 20-25 |

## Refactoring Steps (Phased Approach)

### Phase 1: Foundation (Completed)
- ✅ Extract types to internal/types
- ✅ Extract UI utilities to internal/ui
- ✅ Extract data persistence to internal/data

### Phase 2: Independent Modules (Next)
- 📋 Extract intraday_chart.go (no dependencies)
- 📋 Extract alert notification (minimal dependencies)
- 📋 Extract API helpers (utility functions)

### Phase 3: Service Layer
- 📋 Create interfaces for major services
- 📋 Implement dependency injection
- 📋 Extract API client with providers

### Phase 4: State Machine
- 📋 Extract state handlers
- 📋 Create app/update.go, app/view.go
- 📋 Slim down Model

### Phase 5: Complete Migration
- 📋 Move all business logic to internal/
- 📋 main.go becomes pure bootstrap
- 📋 Full test coverage

## Conclusion

The current architecture has clear modules but tight coupling through:
1. **Model mega-structure** (230+ fields)
2. **Circular function calls** (api ↔ getStockPrice)
3. **Shared mutable state** (cache with RWMutex)

The target architecture uses:
1. **Dependency injection** to break cycles
2. **Service-oriented** design for business logic
3. **Clear layer separation** (UI → Services → Data)

This refactoring is a **multi-month effort** due to code size (~15,000 lines).
Proceed incrementally to maintain stability.

---

**See also**:
- `doc/ARCHITECTURE_REFACTORING.md` - Detailed refactoring plan
- `doc/CODE_ORGANIZATION.md` - File navigation guide
