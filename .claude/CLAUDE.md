# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Stock Monitor** — Go 1.25 TUI app (Bubble Tea framework) for real-time stock tracking across A-shares, US, and HK markets. Bilingual (zh/en). Current version: v9.1.

## Essential Commands

A `Makefile` provides the canonical shortcuts — prefer it over raw `go` invocations:

```bash
make build              # Build to cmd/stock-monitor
make run                # go run main.go
make test               # All tests
make test-race          # Race detector (essential when touching concurrency)
make test-coverage      # HTML coverage report
make check              # fmt + vet + test
make release-snapshot   # GoReleaser snapshot (no publish)

# Run a single test
go test -v ./internal/alert/ -run TestFrequencyDaily
```

## Architecture

### Layered design (read top-down)

```
UI (Bubble Tea)        → main.go + handlers_*.go + internal/ui/
Business logic         → internal/alert/, internal/intraday/, internal/market/, internal/sort/
Data layer             → internal/data/ (persistence + 30s TTL cache)
External integration   → internal/api/ (china/, us/, hongkong/) with fallback chains
Cross-cutting          → internal/log/, internal/i18n/, internal/types/, internal/consts/
```

### Hybrid layout: root-level legacy + `internal/` (in-progress refactor)

Root `*.go` files (`handlers_*.go`, `intraday.go`, `alert_frequency.go`, `persistence.go`, `cache.go`, `types.go`, etc.) are **active legacy code** being gradually moved into `internal/`. When changing behavior, check **both** locations — root files often call into `internal/` packages but still contain logic. `main.go` is the state-machine entry point and stays there.

### API fallback chains (`internal/api/`)

- A-shares: Tencent → Sina → East Money (GBK encoding, requires `golang.org/x/text/encoding/simplifiedchinese`)
- US: Yahoo / FMP → TwelveData
- HK: Finnhub → East Money
- On full failure, UI shows `"-"` — never crash.

### Intraday collection (`internal/intraday/`)

Worker-pool pattern: `manager.go` holds a `chan struct{}` semaphore capped at **10 concurrent workers**. Auto-stops a stock's collection at ~90% completeness. Storage path: `data/intraday/{code}/{date}.json`.

### Alert system (`internal/alert/`)

Frequency modes: Once / Daily / Weekly / Monthly / Custom (N days). `frequency.go` does O(1) boundary checks — when adding a mode, add edge-case tests for month-end / week-wrap / DST.

## Non-obvious rules (must follow)

1. **Never mutate the Bubble Tea `Model` from a goroutine.** Send a `tea.Msg` via `tea.Cmd` instead. Direct mutation is the #1 race-condition source here. v9.1 introduced `atomic.Pointer` for `globalModel` access — preserve that pattern.

2. **All persistence writes are atomic** — write to `path + ".tmp"`, then `os.Rename`. Required by `internal/data/persistence.go` and `internal/intraday/storage.go`. Don't bypass.

3. **Per-stock in-flight guards** prevent concurrent fetch races (added in v9.1). When adding a new fetch path, register the in-flight marker before the API call and release in `defer`.

4. **Cache TTL is 30 seconds** (`internal/data/cache.go`). Don't shorten — APIs throttle. Reads use `RWMutex.RLock`.

5. **Chinese characters are 2 cells wide.** Use the `go-pretty` library for table alignment, not raw `len()`.

6. **Holiday/timezone awareness lives in `internal/market/`.** Use `IsMarketOpen` / `FindNextTradingDay` — they handle HKEX Closing Auction (16:00–16:10), A-share lunch break, and US half-days.

7. **Path traversal validation** is applied to all user-controlled paths (intraday storage, log rotation). When adding a new path-construction code path, mirror the validation in `internal/intraday/storage.go`.

## Configuration

- `cmd/conf/config.yml` — user config (auto-generated on first run from `config_demo.yaml`)
- `i18n/zh.json`, `i18n/en.json` — translations; add new keys to **both**, naming as `module.feature.item`
- `data/portfolio.json`, `data/watchlist.json`, `data/alert_data.json` — runtime state (JSON, atomic writes)

## Adding features (where to start)

| Task | Primary file |
|------|--------------|
| New API source | `internal/api/{market}/`, register in `internal/api/stock.go` |
| New alert type | Add to `internal/types/alert.go`; checker in `internal/ui/alert/checker.go`; frequency in `internal/alert/frequency.go`; i18n keys in both languages |
| New background worker | New `internal/{name}/` with manager (semaphore) + worker; wire into `internal/app/init.go` |
| Column / table change | `internal/ui/columns.go` and `internal/ui/table/builder.go` |
| New market | `internal/market/timezone.go` + new `internal/api/{market}/` |

## Testing notes

- Edge-case heavy areas: alert frequency (month/week boundaries), market hours (DST, holidays), API code conversion (`601138` ↔ `601138.SS`, `0700` ↔ `0700.HK`).
- `make test-race` is mandatory after changes to: `internal/data/`, `internal/intraday/`, anything touching `globalModel`.
- `internal/testutil/` provides fixtures/helpers — use them rather than hand-rolling test stocks.
