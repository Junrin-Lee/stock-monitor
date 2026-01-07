# 中国节假日数据同步 Worker 实现计划

## 概述

构建一个启动时异步执行的 worker，从 lanceliao/china-holiday-calender API 获取当前年份的中国节假日数据，更新本地 `./data/calendars/CN/{year}.json` 文件。

## 数据源

- **API URL**: `https://www.shuyz.com/githubfiles/china-holiday-calender/master/holidayAPI.json`
- **备用 URL**: `https://raw.githubusercontent.com/lanceliao/china-holiday-calender/master/holidayAPI.json`
- **数据范围**: 2023-2026 年（API 提供）
- **数据来源**: 国务院官方公告

## API 响应格式

```json
{
  "Name": "ShuYZ中国节假日",
  "Version": "1.0",
  "Years": {
    "2026": [
      {
        "Name": "元旦",
        "StartDate": "2026-01-01",
        "EndDate": "2026-01-03",
        "Duration": 3,
        "CompDays": ["2026-01-04"],
        "URL": "...",
        "Memo": "..."
      }
    ]
  }
}
```

## 本地数据格式

```json
{
  "market": "CN",
  "year": 2026,
  "holidays": {
    "20260101": true,
    "20260102": true,
    "20260103": true
  },
  "comp_days": {
    "20260104": true
  },
  "updated_at": "2026-01-07T10:30:00+08:00"
}
```

## 实现步骤

### 步骤 1: 创建 worker 目录结构

```
worker/
  └── holiday_worker.go    # 节假日数据同步 worker
```

### 步骤 2: 实现 holiday_worker.go

#### 2.1 数据结构定义

```go
// HolidayAPIResponse API 响应结构
type HolidayAPIResponse struct {
    Name    string                       `json:"Name"`
    Version string                       `json:"Version"`
    Years   map[string][]HolidayEntry    `json:"Years"`
}

// HolidayEntry 单个节假日条目
type HolidayEntry struct {
    Name      string   `json:"Name"`
    StartDate string   `json:"StartDate"`  // YYYY-MM-DD
    EndDate   string   `json:"EndDate"`    // YYYY-MM-DD
    Duration  int      `json:"Duration"`
    CompDays  []string `json:"CompDays"`   // 补班日期
}

// CalendarData 本地日历数据格式
type CalendarData struct {
    Market    string          `json:"market"`
    Year      int             `json:"year"`
    Holidays  map[string]bool `json:"holidays"`   // YYYYMMDD -> true
    CompDays  map[string]bool `json:"comp_days"`  // YYYYMMDD -> true
    UpdatedAt string          `json:"updated_at"`
}
```

#### 2.2 核心函数

| 函数 | 职责 |
|------|------|
| `StartHolidayWorker()` | 启动协程，执行数据同步 |
| `fetchHolidayData()` | HTTP 请求获取 API 数据，支持备用 URL |
| `parseHolidayResponse()` | 解析 JSON 响应 |
| `transformToCalendarData()` | 转换 API 格式到本地格式 |
| `expandDateRange()` | 展开 StartDate~EndDate 为每日条目 |
| `loadExistingCalendar()` | 加载现有本地数据 |
| `diffCalendarData()` | 比较数据差异 |
| `saveCalendarData()` | 保存到 JSON 文件 |

#### 2.3 主要流程

```
StartHolidayWorker()
    └── go func() {
            1. 获取当前年份
            2. fetchWithRetry() - 指数退避重试 3 次
               ├── 首次重试: 1s
               ├── 二次重试: 2s
               └── 三次重试: 4s
            3. 解析并转换数据
            4. 加载本地现有数据
            5. 比较差异:
               ├── 不存在 → 创建新文件
               ├── 有差异 → 更新数据
               └── 无差异 → 仅更新 updated_at
            6. 保存文件
            7. 记录日志并退出
        }()
```

#### 2.4 指数退避重试实现

```go
func fetchWithRetry(maxRetries int) (*HolidayAPIResponse, error) {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        if i > 0 {
            backoff := time.Duration(1<<uint(i-1)) * time.Second // 1s, 2s, 4s
            time.Sleep(backoff)
            logWarnDirect("Holiday worker retry %d/%d after %v", i, maxRetries, backoff)
        }

        resp, err := fetchHolidayData()
        if err == nil {
            return resp, nil
        }
        lastErr = err
    }
    return nil, lastErr
}
```

#### 2.5 日期范围展开

```go
// 将 "2026-01-01" ~ "2026-01-03" 展开为 ["20260101", "20260102", "20260103"]
func expandDateRange(startDate, endDate string) ([]string, error) {
    start, _ := time.Parse("2006-01-02", startDate)
    end, _ := time.Parse("2006-01-02", endDate)

    var dates []string
    for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
        dates = append(dates, d.Format("20060102"))
    }
    return dates, nil
}
```

### 步骤 3: 在 main.go 中集成

在 `func main()` 中，日志初始化后启动 worker（约第 46 行，`defer globalLogger.Sync()` 之后）：

```go
// main.go 现有代码片段：
func main() {
    // 确保目录存在
    os.MkdirAll("data", 0755)
    os.MkdirAll("cmd/conf", 0755)
    os.MkdirAll("i18n", 0755)

    // 初始化日志系统（在所有初始化之前）
    logDir := filepath.Join(".", "data", "logs")
    logLevel := LogInfo
    if err := InitLogger(logDir, logLevel); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
        os.Exit(1)
    }
    defer globalLogger.Sync()

    // ===== 添加以下代码 =====
    // 启动节假日数据同步 worker（异步执行，不阻塞主流程）
    StartHolidayWorker()
    // ========================

    // 加载 i18n 文件
    loadI18nFiles()
    // ... 其余代码
}
```

**注意**: Worker 在后台异步执行，不会阻塞应用启动。即使网络请求失败，也不会影响主程序运行。

### 步骤 4: 添加 i18n 日志键

**i18n/zh.json:**
```json
{
  "log.holiday.workerStart": "节假日数据同步 worker 启动",
  "log.holiday.fetchSuccess": "成功获取 %d 年节假日数据",
  "log.holiday.noDataForYear": "API 中无 %d 年数据",
  "log.holiday.createFile": "创建新的日历文件: %s",
  "log.holiday.updateFile": "更新日历文件: %s (变更: %d 处)",
  "log.holiday.noChange": "日历数据无变化，仅更新时间戳",
  "log.holiday.complete": "节假日数据同步完成，耗时 %v",
  "log.holiday.fetchFailed": "获取节假日数据失败: %s",
  "log.holiday.allRetryFailed": "节假日数据同步失败（已重试 %d 次）: %s"
}
```

**i18n/en.json:**
```json
{
  "log.holiday.workerStart": "Holiday data sync worker started",
  "log.holiday.fetchSuccess": "Successfully fetched %d holiday data",
  "log.holiday.noDataForYear": "No data for year %d in API",
  "log.holiday.createFile": "Created new calendar file: %s",
  "log.holiday.updateFile": "Updated calendar file: %s (changes: %d)",
  "log.holiday.noChange": "Calendar data unchanged, only updated timestamp",
  "log.holiday.complete": "Holiday data sync completed in %v",
  "log.holiday.fetchFailed": "Failed to fetch holiday data: %s",
  "log.holiday.allRetryFailed": "Holiday data sync failed after %d retries: %s"
}
```

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `worker/holiday_worker.go` | 新建 | Worker 主体实现 |
| `main.go` | 修改 | 添加 StartHolidayWorker() 调用 |
| `i18n/zh.json` | 修改 | 添加中文日志键 |
| `i18n/en.json` | 修改 | 添加英文日志键 |

## HTTP 客户端模式（遵循 api.go 现有模式）

```go
func fetchHolidayData() (*HolidayAPIResponse, error) {
    urls := []string{
        "https://www.shuyz.com/githubfiles/china-holiday-calender/master/holidayAPI.json",
        "https://raw.githubusercontent.com/lanceliao/china-holiday-calender/master/holidayAPI.json",
    }

    client := &http.Client{Timeout: 10 * time.Second}

    for _, url := range urls {
        logDebugDirect("Holiday worker fetching: %s", url)
        resp, err := client.Get(url)
        if err != nil {
            logWarnDirect("Holiday worker HTTP error from %s: %v", url, err)
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            logWarnDirect("Holiday worker HTTP %d from %s", resp.StatusCode, url)
            continue
        }

        body, err := io.ReadAll(resp.Body)
        if err != nil {
            logWarnDirect("Holiday worker read error from %s: %v", url, err)
            continue
        }

        var apiResp HolidayAPIResponse
        if err := json.Unmarshal(body, &apiResp); err != nil {
            logWarnDirect("Holiday worker JSON parse error: %v", err)
            continue
        }

        return &apiResp, nil
    }

    return nil, fmt.Errorf("all API URLs failed")
}
```

## JSON 文件操作模式（遵循 persistence.go 现有模式）

```go
func saveCalendarData(data *CalendarData, year int) error {
    // 确保目录存在
    dir := filepath.Join(".", "data", "calendars", "CN")
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    // 使用 MarshalIndent 格式化输出
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }

    // 写入文件
    filePath := filepath.Join(dir, fmt.Sprintf("%d.json", year))
    return os.WriteFile(filePath, jsonData, 0644)
}

func loadExistingCalendar(year int) (*CalendarData, error) {
    filePath := filepath.Join(".", "data", "calendars", "CN", fmt.Sprintf("%d.json", year))
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err // 文件不存在或读取失败
    }

    var calendar CalendarData
    if err := json.Unmarshal(data, &calendar); err != nil {
        return nil, err
    }
    return &calendar, nil
}
```

## 关键设计决策

1. **协程自动释放**: Worker 完成后自动退出，无需手动管理
2. **资源最小化**: 仅在启动时执行一次，不占用持续资源
3. **数据保护**: 使用 diff 比较，避免无谓覆盖，保留 updated_at 时间戳
4. **容错设计**: 指数退避重试，双 URL 备用，完整错误日志
5. **遵循现有模式**: 使用 logInfo/logError 日志函数，复用 JSON 文件操作模式
6. **HTTP 超时**: 10 秒超时，与 api.go 中的腾讯 API 一致
7. **URL 备用**: 主 URL (shuyz.com) 失败时自动尝试 GitHub Raw URL

## 示例日志输出

**成功场景:**
```
[2026-01-07 10:30:00][INFO][log.holiday.workerStart][节假日数据同步 worker 启动]
[2026-01-07 10:30:01][INFO][log.holiday.fetchSuccess][成功获取 2026 年节假日数据]
[2026-01-07 10:30:01][INFO][log.holiday.noChange][日历数据无变化，仅更新时间戳]
[2026-01-07 10:30:01][INFO][log.holiday.complete][节假日数据同步完成，耗时 1.2s]
```

**失败场景:**
```
[2026-01-07 10:30:00][INFO][log.holiday.workerStart][节假日数据同步 worker 启动]
[2026-01-07 10:30:05][WARN][][Holiday worker retry 1/3 after 1s]
[2026-01-07 10:30:08][WARN][][Holiday worker retry 2/3 after 2s]
[2026-01-07 10:30:13][ERROR][log.holiday.allRetryFailed][节假日数据同步失败（已重试 3 次）: connection timeout]
```

## 测试计划

1. **单元测试**: `expandDateRange()`, `diffCalendarData()` 函数
2. **集成测试**: 模拟 API 响应，验证文件创建/更新逻辑
3. **手动测试**:
   - 删除本地文件，验证创建逻辑
   - 修改本地文件，验证 diff 更新逻辑
   - 断开网络，验证重试和错误日志