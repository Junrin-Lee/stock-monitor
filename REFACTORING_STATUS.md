# Refactoring Status - Stock Monitor

## Current State (2026-01-20 - Updated)

### Completed Refactoring (Rounds 1-4.4)

The following modules have been successfully extracted to `internal/` packages:

| Module | Location | Lines | Status |
|--------|----------|-------|--------|
| **Types** | internal/types | ~400 | ✅ Complete |
| **Constants** | internal/consts | ~80 | ✅ Complete |
| **API Layer** | internal/api | ~1,355 | ✅ Complete |
| **UI Utilities** | internal/ui | ~700 | ✅ Complete |
| **Watchlist UI** | internal/ui/watchlist | ~635 | ✅ Complete |
| **Alert UI** | internal/ui/alert | ~1,125 | ✅ Complete |
| **Intraday Core** | internal/intraday | ~1,701 | ✅ Complete |
| **Sorting** | internal/sort | ~140 | ✅ Complete |
| **Alert Frequency** | internal/alert | ~160 | ✅ Complete |
| **i18n** | internal/i18n | ~57 | ✅ Complete |
| **Logging** | internal/log | ~190 | ✅ Complete |
| **Data/Cache** | internal/data | ~270 | ✅ Complete |
| **Market/Timezone** | internal/market | ~450 | ✅ Complete |

### Recently Completed (Rounds 4.1-4.4)

**Round 3.2: Extract Columns Module**
```
Commit: 7ff18cc
- Moved columns.go → internal/ui/columns.go (492 lines)
- Updated main.go with adapter methods (204 lines added)
- All tests passing (26/26)
```

**Round 4.1: Extract Persistence Layer**
```
Commit: 60ba6d4
- Refactored persistence.go to use internal/data package
- Reduced from 415 to 205 lines (50% reduction)
- Created adapter methods for Portfolio/Watchlist/Config/Alert
- Updated alert_frequency_test.go
- All tests passing (31/31)
```

**Round 4.2: Extract Watchlist UI Handlers**
```
Commit: 202a7ec
- Extracted watchlist business logic to internal/ui/watchlist (635 lines total)
  * filter.go (172 lines) - filtering, caching, cursor management
  * tags.go (274 lines) - tag management, grouping, position finding
  * view.go (189 lines) - view rendering functions
- Refactored watchlist.go: 884 → 481 lines (-45.5% reduction)
- Converted WatchlistStock and TagGroup to type aliases
- All method calls replaced with function calls
- All tests passing (31/31)
```

**Round 4.3: Extract Alert UI Logic**
```
Commit: b3bf7e4
- Extracted alert UI logic to internal/ui/alert package (1,125 lines total)
  * checker.go (62 lines) - alert condition checking logic
  * helpers.go (132 lines) - helper functions (tag/market filtering, parsing)
  * notification.go (99 lines) - notification sending (macOS/Linux/Windows)
  * view.go (832 lines) - UI rendering functions (manage, add, edit, batch views)
- Refactored alert.go: 2,511 → 1,581 lines (-37.0% reduction, 930 lines)
- Created internal/ui/types.go - shared UI type definitions
- Added type conversion adapters for Alert, StockData, Cache, Portfolio, Watchlist
- Fixed type compatibility between main and internal packages
- All tests passing (31/31)
```

**Round 4.4: Extract Intraday Module**
```
Date: 2026-01-20
- Extracted intraday.go (1,536 lines) to internal/intraday package (1,701 lines total)
  * types.go (35 lines) - IntradayDataPoint, IntradayData, SaveDecision, CollectionMode
  * manager.go (168 lines) - IntradayManager core logic, worker pool management
  * worker.go (152 lines) - Smart worker, data fetching logic
  * trading.go (294 lines) - Trading hours, market state, data completeness checks
  * storage.go (169 lines) - File I/O, thread-safe locks, backward compatibility
  * helpers.go (239 lines) - Data merging, comparison, save decision logic
  * api.go (644 lines) - Multi-API integration (Tencent, Sina, EastMoney, Yahoo)
- Updated intraday_chart.go to use new internal/intraday package
- Created ModelInterface for loose coupling between main and intraday modules
- Exported necessary functions for external use and testing
- Root intraday.go reduced to 145 lines (adapter only)
- Fixed all test references to use internal/intraday package
- All intraday tests passing (6/6 test suites)
```

**Build Status**: ✅ Passing
**Test Status**: ✅ All intraday tests passing (TestConvert*, TestCompare*, TestShouldSave*)

### Remaining Work (Round 4 continuation)

Large files still in root directory that need refactoring:

| File | Lines | Target Location | Priority | Status |
|------|-------|----------------|----------|--------|
| **main.go** | 3,469 | Keep as entry point | N/A | ✅ Keep |
| **alert.go** | 1,581 | internal/ui/alert | High | ✅ Complete |
| **intraday.go** | 145 | Adapter only | Done | ✅ Complete |
| **intraday_chart.go** | 1,292 | Keep (chart rendering) | Medium | ✅ Integrated |
| **watchlist.go** | 481 | Adapter only | Done | ✅ Complete |
| **types.go** | 353 | Adapter only | Low | ⏳ Pending |
| **holiday_worker.go** | 278 | Partially migrated | Low | ⏳ Pending |
| **persistence.go** | 205 | Adapter only | Done | ✅ Complete |
| **ui_utils.go** | 194 | internal/ui (fully) | Low | ⏳ Pending |
| **logger.go** | 189 | internal/log (fully) | Low | ⏳ Pending |
| **timezone.go** | 172 | internal/market (fully) | Low | ⏳ Pending |
| **format.go** | 157 | internal/ui (fully) | Low | ⏳ Pending |
| **cache.go** | 149 | internal/data (fully) | Low | ⏳ Pending |
| **sort.go** | 140 | internal/sort (fully) | Low | ⏳ Pending |
| **alert_frequency.go** | 129 | internal/alert (fully) | Low | ⏳ Pending |

### Architecture Goals

1. **Main Package**: Should only contain:
   - Application entry point (main.go)
   - Bubble Tea Model and state machine orchestration
   - Adapter/bridge code between internal packages and Bubble Tea framework

2. **Internal Packages**: Should contain:
   - Business logic
   - Data structures
   - API integrations
   - UI rendering utilities
   - State-independent functionality

3. **Principles**:
   - Clear separation of concerns
   - Minimize circular dependencies
   - Keep Bubble Tea framework dependencies in main package
   - Internal packages should be framework-agnostic where possible

## Next Steps (Recommended Order)

### Round 4.1: Extract Persistence Layer
- Move persistence.go → internal/data/persistence.go
- Create clean interfaces for save/load operations
- Update main.go to use the new package

### Round 4.2: Extract Intraday Module
- Move intraday.go → internal/intraday/collector.go
- Move intraday_chart.go → internal/ui/chart/intraday.go
- Separate data collection from UI rendering
- Update main.go handlers

### Round 4.3: Extract UI Handlers
- Move alert.go → internal/ui/alert/handlers.go
- Move watchlist.go → internal/ui/watchlist/handlers.go
- Keep only Bubble Tea state machine in main.go

### Round 4.4: Complete UI Package
- Move remaining ui_utils.go to internal/ui/
- Move remaining format.go to internal/ui/
- Consolidate all UI rendering in internal/ui

### Round 4.5: Final Cleanup
- Remove duplicate type aliases from root
- Ensure all internal packages are properly exported
- Update documentation
- Final test pass

## Testing Strategy

For each refactoring round:
1. Run `go build -o cmd/stock-monitor` to verify compilation
2. Run `go test -v ./...` to verify all tests pass
3. Manual smoke test of key features
4. Commit changes with descriptive message

## Notes

- The project follows Go best practices with internal packages
- All refactorings maintain backward compatibility
- Tests provide safety net for refactoring
- Build and test success indicates safe refactoring
