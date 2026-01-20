# Round 5 & 6 Implementation Guide

## Current Status
- **Line Count**: main.go (3,499), alert.go (1,581), watchlist.go (482), intraday_chart.go (1,292)
- **Total Handlers**: 49 handler functions across 4 files
- **Model Fields**: ~130 fields in the Model struct

## Problem Analysis

The original refactoring plan assumes:
1. State handlers can be easily extracted to `internal/states/` packages
2. Model can be moved to `internal/app` package
3. Clean separation is possible

**Reality Check:**
- Model has 130+ fields with complex interdependencies
- Handlers are tightly coupled to Model's private fields
- Many helper methods access multiple Model fields
- Bubble Tea message types are defined alongside Model
- Moving Model breaks all existing code (main.go: 3,499 lines)

## Recommended Pragmatic Approach

### Option A: Split Handlers in Root Package (Low Risk)
Keep Model where it is, split handlers into domain files:

```
main.go (reduced to ~300 lines)
  - main() function
  - Model struct
  - Init(), Update(), View() routers
  - Common helper functions

handlers_menu.go (~200 lines)
  - handleMainMenu()
  - handleLanguageSelection()
  - executeMenuItem()
  - Related view functions

handlers_portfolio.go (~800 lines)
  - handleMonitoring()
  - handleAddingStock()
  - handleEditingStock()
  - handlePortfolioSorting()
  - Related helpers and views

handlers_watchlist.go (~900 lines)
  - handleWatchlistViewing()
  - handleWatchlistTagging()
  - handleWatchlistTagSelect()
  - handleWatchlistTagManage()
  - handleWatchlistGroupSelect()
  - Related helpers and views

handlers_search.go (~400 lines)
  - handleSearchingStock()
  - handleSearchResult()
  - handleSearchResultWithActions()
  - Related views

handlers_alert.go (~1,600 lines from alert.go)
  - All handleAlert*() functions
  - Alert checking logic
  - Alert views

handlers_chart.go (~1,300 lines from intraday_chart.go)
  - handleIntradayChartViewing()
  - Chart rendering
  - Related helpers
```

**Benefits:**
- ✅ Reduces main.go from 3,499 → ~300 lines
- ✅ Organizes code by domain
- ✅ Zero risk - no package changes
- ✅ All handlers stay as Model methods
- ✅ No import path changes
-  ✅ Can be done incrementally

**Implementation Steps:**
1. Create `handlers_menu.go` - Move 2 menu handlers from main.go
2. Create `handlers_portfolio.go` - Move 4 portfolio handlers from main.go
3. Create `handlers_watchlist.go` - Move watchlist handlers from main.go + watchlist.go
4. Create `handlers_search.go` - Move 3 search handlers from main.go
5. Rename `alert.go` → `handlers_alert.go` (already organized)
6. Rename `intraday_chart.go` → `handlers_chart.go` (already organized)
7. Keep Model and types in main.go temporarily
8. Verify: `go build && go test ./...`

### Option B: Full Internal/App Migration (High Risk - NOT RECOMMENDED)

Create `internal/app` package and move everything:
- ❌ Requires updating 3,500+ lines of code
- ❌ Complex circular dependency issues
- ❌ Bubble Tea message types need migration
- ❌ All tests need import path updates
- ❌ High risk of breaking existing functionality
- ❌ Estimated time: 8-12 hours of careful work

## Recommended Implementation: Option A

### Step-by-Step Execution

#### Step 1: Extract Menu Handlers (handlers_menu.go)

```bash
# Create the new file
touch handlers_menu.go
```

Move these functions from main.go to handlers_menu.go:
- `handleMainMenu()`
- `executeMenuItem()`
- `handleLanguageSelection()`
- `viewMainMenu()`
- `viewLanguageSelection()`
- `getMenuItems()` (if it's a method)

#### Step 2: Extract Portfolio Handlers (handlers_portfolio.go)

Move from main.go:
- `handleMonitoring()`
- `handleMonitoringNavigation()`
- `handleMonitoringActions()`
- `handleMonitoringViews()`
- `handleAddingStock()`
- `handleEditingStock()`
- `handlePortfolioSorting()`
- `processAddingStep()`
- All related view methods
- Helper methods like `resetPortfolioCursor()`

#### Step 3: Extract Watchlist Handlers (handlers_watchlist.go)

Move from main.go:
- `handleWatchlistViewing()`
- `handleWatchlistTagging()`
- `handleWatchlistTagSelect()`
- `handleWatchlistTagManage()`
- `handleWatchlistTagRemoveSelect()`
- `handleWatchlistTagEdit()`
- `handleWatchlistGroupSelect()`
- `handleWatchlistSorting()`
- `handleWatchlistSearchConfirm()`

Move from watchlist.go:
- Any remaining handlers
- Related view methods

#### Step 4: Extract Search Handlers (handlers_search.go)

Move from main.go:
- `handleSearchingStock()`
- `handleSearchResult()`
- `handleSearchResultWithActions()`
- Related view methods

#### Step 5: Rename Alert File

```bash
mv alert.go handlers_alert.go
```

#### Step 6: Rename Chart File

```bash
mv intraday_chart.go handlers_chart.go
```

#### Step 7: Clean Up main.go

After extraction, main.go should contain:
- `package main` and imports
- `globalModel` variable
- Type definitions (Model, Stock, etc.)
- Message types (tickMsg, etc.)
- `main()` function
- `Init()` method
- `Update()` method (just routing)
- `View()` method (just routing)
- Common utility methods

#### Step 8: Verify

```bash
go build -o cmd/stock-monitor
go test ./...
./cmd/stock-monitor  # Manual testing
```

## File Size Targets After Refactoring

| File | Current | Target | Content |
|------|---------|--------|---------|
| main.go | 3,499 | ~400 | Entry, Model, routers, types |
| handlers_menu.go | - | ~250 | Menu + Language |
| handlers_portfolio.go | - | ~900 | Portfolio management |
| handlers_watchlist.go | ~482 | ~1,000 | Watchlist + tags |
| handlers_search.go | - | ~400 | Search functionality |
| handlers_alert.go | ~1,581 | ~1,581 | Alert management (rename) |
| handlers_chart.go | ~1,292 | ~1,292 | Chart viewing (rename) |
| **Total** | **6,854** | **5,823** | Better organized |

## Benefits of This Approach

1. **Minimal Risk**: No package structure changes
2. **Incremental**: Can be done file-by-file
3. **Testable**: Each step can be validated
4. **Maintainable**: Clear domain separation
5. **Rollback**: Easy to revert if issues arise
6. **Achieves Goal**: Main.go significantly reduced

## Why Not Full internal/app Migration?

The REFACTORING_STATUS.md document already identified:
> "Round 4.5 部分完成... 建议保持当前架构,已经足够优秀"

The current architecture is sound:
- `internal/*` packages contain pure business logic
- Root directory contains Bubble Tea TUI implementation
- This is the correct "onion architecture" pattern

Moving Model to `internal/app` would:
- Break the architectural pattern (TUI should be in main)
- Create circular dependencies
- Require massive refactoring (7,000+ lines)
- Provide minimal benefit

## Next Steps

1. **Execute Option A** - Split handlers in root package
2. **Update REFACTORING_STATUS.md** - Document completion
3. **Git Commit** - "refactor(round5): split state handlers by domain"
4. **Manual Testing** - Verify all 29 states work correctly

## Conclusion

Round 5 & 6 can be completed pragmatically by splitting handlers into domain files within the main package. This achieves the core goal (reduce main.go size, improve organization) without the risks of a full package migration.

The project is already well-architected. Further refactoring should focus on functionality, not structure.
