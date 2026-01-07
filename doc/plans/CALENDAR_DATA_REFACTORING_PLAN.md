# 日历数据结构重构计划

## 概述

重构 `./data/calendars/` 目录下的日历数据结构，解决当前存在的数据冗余问题，并建立规范化的多市场日历数据管理体系。

**当前问题**:
- `china_holidays_2025.json` 包含 2023、2024、2025、2026 年数据
- `china_holidays_2026.json` 包含相同的 2023、2024、2025、2026 年数据
- 两个文件内容完全相同，只是 `year` 字段和文件名不同
- 数据结构不利于未来扩展（US、HK 等市场）

**重构目标**:
- ✅ 消除数据冗余
- ✅ 建立清晰的目录结构：`data/calendars/{市场代码}/{年份}.json`
- ✅ 为未来扩展做准备（支持 US、HK 等多市场）
- ✅ 确保数据完整性和正确性

**预计实现时间**: 2-3 小时

---

## 当前状态分析

### 现有文件结构

```
data/calendars/
├── china_holidays_2025.json  (3,249 bytes)
└── china_holidays_2026.json  (3,249 bytes)
```

### 数据结构

```json
{
  "market": "china",
  "year": 2025,
  "holidays": {
    "20221231": true,
    "20230101": true,
    "20230102": true,
    "20230121": true,
    ...  // 包含多年度数据
  },
  "comp_days": {
    "20230128": true,
    "20230423": true,
    ...  // 包含多年度调休日
  },
  "updated_at": "2025-12-31T17:38:13+08:00"
}
```

### 数据冗余统计

| 年份 | 2025文件 | 2026文件 | 状态 |
|------|----------|----------|------|
| 2023 | ✅ 存在 | ✅ 存在 | ❌ 冗余 |
| 2024 | ✅ 存在 | ✅ 存在 | ❌ 冗余 |
| 2025 | ✅ 存在 | ✅ 存在 | ❌ 冗余 |
| 2026 | ✅ 存在 | ✅ 存在 | ❌ 冗余 |

**两个文件完全相同**，总计 6,498 字节，实际只需约 3,200 字节（50% 冗余）。

### 代码实现状态

| 组件 | 状态 | 说明 |
|------|------|------|
| 日历数据文件 | ✅ 已存在 | `data/calendars/china_holidays_*.json` |
| `TradingStateHoliday` | ✅ 已定义 | `types.go` 枚举值 |
| `getTradingState()` | ⚠️ 未实现 | 只有 TODO 注释 |
| `findPreviousTradingDay()` | ⚠️ 未实现 | 需要假日检测逻辑 |
| 日历文件加载 | ❌ 未实现 | 从未加载或使用 |

**关键发现**: 日历数据文件存在，但代码中从未加载或使用这些数据。

---

## 新目录结构设计

### 目标结构

```
data/calendars/
├── CN/
│   ├── 2023.json  # 2023年中国市场节假日和调休日
│   ├── 2024.json  # 2024年中国市场节假日和调休日
│   ├── 2025.json  # 2025年中国市场节假日和调休日
│   └── 2026.json  # 2026年中国市场节假日和调休日
├── US/            # 未来扩展：美国市场
│   └── 2025.json
└── HK/            # 未来扩展：香港市场
    └── 2025.json
```

### 文件命名规范

- **目录名**: 市场代码（CN, US, HK 等），大写
- **文件名**: `{年份}.json`，4 位数字年份
- **市场标识**: JSON 文件中的 `market` 字段使用大写市场代码

### 数据结构更新

```json
{
  "market": "CN",           // 从 "china" 改为 "CN"
  "year": 2025,             // 仅包含该年份的数据
  "holidays": {
    "20250101": true,       // 仅 2025 年的假日
    "20250128": true,
    ...
  },
  "comp_days": {
    "20250126": true,       // 仅 2025 年的调休日
    "20250208": true,
    ...
  },
  "updated_at": "2025-12-31T17:38:13+08:00"
}
```

---

## 数据迁移方案

### 方案概述

创建 Go 迁移工具 `tools/migrate_calendar_structure.go`，实现以下功能：

1. ✅ 读取现有日历文件
2. ✅ 按年份分割假日和调休日数据
3. ✅ 创建新的目录结构
4. ✅ 生成每年独立的 JSON 文件
5. ✅ 生成迁移报告
6. ✅ 验证数据完整性


---

## 实施步骤

### 步骤 1: 创建迁移工具

```bash
# 创建 tools 目录（如果不存在）
mkdir -p tools

# 创建迁移工具文件
cat > tools/migrate_calendar_structure.go << 'EOF'
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

// HolidayCalendar 日历数据结构
type HolidayCalendar struct {
    Market   string          `json:"market"`
    Year     int             `json:"year"`
    Holidays map[string]bool `json:"holidays"`
    CompDays map[string]bool `json:"comp_days"`
    UpdatedAt string         `json:"updated_at"`
}

func main() {
    fmt.Println("🚀 开始日历数据迁移...")
    
    // 1. 读取现有数据
    calendarData, err := loadCalendar("data/calendars/china_holidays_2025.json")
    if err != nil {
        fmt.Printf("❌ 读取日历文件失败: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("✅ 读取成功: 原始市场=%s, 年份=%d\n", calendarData.Market, calendarData.Year)
    
    // 2. 按年份分割数据
    calendarsByYear := splitCalendarByYear(calendarData)
    fmt.Printf("✅ 数据分割完成: 共 %d 年\n", len(calendarsByYear))
    
    // 3. 创建新目录
    cnDir := "data/calendars/CN"
    if err := os.MkdirAll(cnDir, 0755); err != nil {
        fmt.Printf("❌ 创建 CN 目录失败: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("✅ 创建目录: %s\n", cnDir)
    
    // 4. 保存每年的数据
    savedCount := 0
    for year, data := range calendarsByYear {
        filePath := filepath.Join(cnDir, fmt.Sprintf("%d.json", year))
        if err := saveCalendar(filePath, data); err != nil {
            fmt.Printf("❌ 保存 %d 年数据失败: %v\n", year, err)
            continue
        }
        savedCount++
        fmt.Printf("✅ 已保存: %s (假日:%d, 调休:%d)\n", 
            filePath, len(data.Holidays), len(data.CompDays))
    }
    
    if savedCount == 0 {
        fmt.Println("❌ 没有文件被保存")
        os.Exit(1)
    }
    
    // 5. 生成迁移报告
    printMigrationReport(calendarsByYear)
    
    fmt.Println("\n✅ 迁移完成！")
    fmt.Println("\n📝 后续步骤:")
    fmt.Println("1. 检查生成的文件: data/calendars/CN/")
    fmt.Println("2. 验证数据正确性")
    fmt.Println("3. 删除旧的冗余文件（确认无误后）")
    fmt.Println("4. 更新相关代码以支持新的目录结构")
}

func loadCalendar(filePath string) (*HolidayCalendar, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    
    var calendar HolidayCalendar
    if err := json.Unmarshal(data, &calendar); err != nil {
        return nil, err
    }
    
    return &calendar, nil
}

func splitCalendarByYear(calendar *HolidayCalendar) map[int]*HolidayCalendar {
    result := make(map[int]*HolidayCalendar)
    
    // 提取年份：20250101 -> 2025
    splitByYear := func(dateMap map[string]bool) map[int]map[string]bool {
        years := make(map[int]map[string]bool)
        for dateStr := range dateMap {
            if len(dateStr) >= 4 {
                year := 0
                fmt.Sscanf(dateStr, "%4d", &year)
                if year > 0 {
                    if years[year] == nil {
                        years[year] = make(map[string]bool)
                    }
                    years[year][dateStr] = true
                }
            }
        }
        return years
    }
    
    holidaysByYear := splitByYear(calendar.Holidays)
    compDaysByYear := splitByYear(calendar.CompDays)
    
    // 合并所有年份（holidays + comp_days）
    allYears := make([]int, 0, len(holidaysByYear)+len(compDaysByYear))
    yearSet := make(map[int]bool)
    for y := range holidaysByYear {
        if !yearSet[y] {
            yearSet[y] = true
            allYears = append(allYears, y)
        }
    }
    for y := range compDaysByYear {
        if !yearSet[y] {
            yearSet[y] = true
            allYears = append(allYears, y)
        }
    }
    sort.Ints(allYears)
    
    // 为每年创建独立结构
    for _, year := range allYears {
        result[year] = &HolidayCalendar{
            Market:   "CN",  // 统一使用大写市场代码
            Year:     year,
            Holidays: make(map[string]bool),
            CompDays: make(map[string]bool),
        }
        
        if holidays, ok := holidaysByYear[year]; ok {
            result[year].Holidays = holidays
        }
        if compDays, ok := compDaysByYear[year]; ok {
            result[year].CompDays = compDays
        }
    }
    
    return result
}

func saveCalendar(filePath string, calendar *HolidayCalendar) error {
    data, err := json.MarshalIndent(calendar, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(filePath, data, 0644)
}

func printMigrationReport(calendars map[int]*HolidayCalendar) {
    fmt.Println("\n" + strings.Repeat("=", 70))
    fmt.Println("📊 迁移报告")
    fmt.Println(strings.Repeat("=", 70))
    
    years := make([]int, 0, len(calendars))
    for year := range calendars {
        years = append(years, year)
    }
    sort.Ints(years)
    
    totalHolidays := 0
    totalCompDays := 0
    
    for _, year := range years {
        cal := calendars[year]
        holidaysCount := len(cal.Holidays)
        compDaysCount := len(cal.CompDays)
        
        fmt.Printf("\n📅 年份: %d\n", year)
        fmt.Printf("   假日数: %d\n", holidaysCount)
        fmt.Printf("   调休日数: %d\n", compDaysCount)
        fmt.Printf("   总天数: %d\n", holidaysCount+compDaysCount)
        
        // 显示前3个假日和调休日（示例）
        if holidaysCount > 0 {
            holidayDates := getSortedDates(cal.Holidays)
            fmt.Printf("   假日示例: %v ...\n", holidayDates[:min(3, len(holidayDates))])
        }
        if compDaysCount > 0 {
            compDates := getSortedDates(cal.CompDays)
            fmt.Printf("   调休示例: %v ...\n", compDates[:min(3, len(compDates))])
        }
        
        totalHolidays += holidaysCount
        totalCompDays += compDaysCount
    }
    
    fmt.Println("\n" + strings.Repeat("-", 70))
    fmt.Printf("总计: %d 年, %d 个假日, %d 个调休日\n", 
        len(calendars), totalHolidays, totalCompDays)
}

func getSortedDates(dateMap map[string]bool) []string {
    dates := make([]string, 0, len(dateMap))
    for date := range dateMap {
        dates = append(dates, date)
    }
    sort.Strings(dates)
    return dates
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
EOF
```


### 步骤 2: 运行迁移工具

```bash
# 执行迁移
go run tools/migrate_calendar_structure.go
```

**预期输出**:
```
🚀 开始日历数据迁移...
✅ 读取成功: 原始市场=china, 年份=2025
✅ 数据分割完成: 共 4 年
✅ 创建目录: data/calendars/CN
✅ 已保存: data/calendars/CN/2023.json (假日:21, 调休:7)
✅ 已保存: data/calendars/CN/2024.json (假日:21, 调休:7)
✅ 已保存: data/calendars/CN/2025.json (假日:26, 调休:5)
✅ 已保存: data/calendars/CN/2026.json (假日:21, 调休:5)

======================================================================
📊 迁移报告
======================================================================

📅 年份: 2023
   假日数: 21
   调休日数: 7
   总天数: 28
   假日示例: [20221231 20230101 20230102] ...
   调休示例: [20230128 20230129 20230423] ...

[...]

------------------------------------------------------------------
总计: 4 年, 89 个假日, 24 个调休日

✅ 迁移完成！

📝 后续步骤:
1. 检查生成的文件: data/calendars/CN/
2. 验证数据正确性
3. 删除旧的冗余文件（确认无误后）
4. 更新相关代码以支持新的目录结构
```

### 步骤 3: 验证数据

```bash
# 查看新生成的文件
ls -lh data/calendars/CN/

# 查看每个文件的内容
for file in data/calendars/CN/*.json; do
    echo "=== $(basename $file) ==="
    cat $file
    echo ""
done

# 验证 JSON 格式
for file in data/calendars/CN/*.json; do
    python3 -m json.tool "$file" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "✅ $(basename $file): JSON 格式正确"
    else
        echo "❌ $(basename $file): JSON 格式错误"
    fi
done

# 统计每年的数据量
echo -e "\n📊 数据统计:"
for file in data/calendars/CN/*.json; do
    year=$(basename "$file" .json)
    holidays=$(grep -o '"202[0-9]*01[0-9]":' "$file" | wc -l | tr -d ' ')
    comp_days=$(grep -o '"202[0-9]*0[1-9][0-9]":' "$file" | grep -v '01' | wc -l | tr -d ' ')
    size=$(wc -c < "$file" | tr -d ' ')
    echo "  $year: 假日=$holidays, 调休=$comp_days, 大小=${size}字节"
done
```

### 步骤 4: 备份并清理旧文件

```bash
# 创建备份目录
mkdir -p data/calendars/backup

# 备份旧文件
mv data/calendars/china_holidays_2025.json data/calendars/backup/
mv data/calendars/china_holidays_2026.json data/calendars/backup/

echo "✅ 已备份旧文件到 data/calendars/backup/"

# 确认迁移无误后，可以删除备份（可选）
# rm -rf data/calendars/backup/
```

---

## 预期结果

### 迁移后的目录结构

```
data/calendars/
├── CN/
│   ├── 2023.json  (~800 bytes)
│   ├── 2024.json  (~800 bytes)
│   ├── 2025.json  (~900 bytes)
│   └── 2026.json  (~800 bytes)
└── backup/        # 可选：备份旧文件
    ├── china_holidays_2025.json (3,249 bytes)
    └── china_holidays_2026.json (3,249 bytes)
```

### 文件示例（2025.json）

```json
{
  "market": "CN",
  "year": 2025,
  "holidays": {
    "20250101": true,
    "20250128": true,
    "20250129": true,
    "20250130": true,
    "20250131": true,
    "20250201": true,
    "20250202": true,
    "20250203": true,
    "20250204": true,
    "20250404": true,
    "20250405": true,
    "20250406": true,
    "20250501": true,
    "20250502": true,
    "20250503": true,
    "20250504": true,
    "20250505": true,
    "20250531": true,
    "20250601": true,
    "20250602": true,
    "20251001": true,
    "20251002": true,
    "20251003": true,
    "20251004": true,
    "20251005": true,
    "20251006": true,
    "20251007": true,
    "20251008": true
  },
  "comp_days": {
    "20250126": true,
    "20250208": true,
    "20250427": true,
    "20250928": true,
    "20251011": true
  },
  "updated_at": "2025-12-31T17:38:13+08:00"
}
```

### 数据统计

| 年份 | 假日数 | 调休日数 | 文件大小 | 节省空间 |
|------|--------|----------|----------|----------|
| 2023 | 21 | 7 | ~800 bytes | - |
| 2024 | 21 | 7 | ~800 bytes | - |
| 2025 | 26 | 5 | ~900 bytes | - |
| 2026 | 21 | 5 | ~800 bytes | - |
| **总计** | **89** | **24** | **~3,300 bytes** | **~3,200 bytes (49%)** |

---

## 后续代码更新建议

虽然当前代码未实现假日检测逻辑，但为了未来扩展，建议：

### 1. 添加日历数据结构

在 `types.go` 中添加:

```go
// HolidayCalendar 日历数据
type HolidayCalendar struct {
    Market   string          `json:"market"`
    Year     int             `json:"year"`
    Holidays map[string]bool `json:"holidays"`
    CompDays map[string]bool `json:"comp_days"`
    UpdatedAt string         `json:"updated_at"`
}

// LoadCalendar 加载指定市场和年份的日历
func LoadCalendar(market string, year int) (*HolidayCalendar, error) {
    filePath := fmt.Sprintf("data/calendars/%s/%d.json", market, year)
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read calendar file: %w", err)
    }
    
    var calendar HolidayCalendar
    if err := json.Unmarshal(data, &calendar); err != nil {
        return nil, fmt.Errorf("failed to parse calendar file: %w", err)
    }
    
    return &calendar, nil
}
```

### 2. 实现假日检测逻辑

在 `intraday.go` 中更新 `getTradingState()`:

```go
func getTradingState(now time.Time, marketType MarketType) TradingState {
    weekday := now.Weekday()
    
    // 检查周末
    if weekday == time.Saturday || weekday == time.Sunday {
        return TradingStateWeekend
    }
    
    // 检查假日
    if isHoliday(now, marketType) {
        return TradingStateHoliday
    }
    
    // 获取市场配置
    config := getMarketConfig(marketType)
    nowTime := now.Format("15:04")
    
    // 检查交易时段
    for _, session := range config.TradingSessions {
        if nowTime >= session.StartTime && nowTime <= session.EndTime {
            return TradingStateLive
        }
    }
    
    // 检查盘前
    firstSession := config.TradingSessions[0]
    if nowTime < firstSession.StartTime {
        return TradingStatePreMarket
    }
    
    // 盘后
    return TradingStatePostMarket
}

// isHoliday 检查指定日期是否为假日
func isHoliday(date time.Time, marketType MarketType) bool {
    marketCode := getMarketCode(marketType)
    calendar, err := LoadCalendar(marketCode, date.Year())
    if err != nil {
        // 如果加载失败，返回 false（默认不是假日）
        debugPrint("警告: 无法加载日历数据 - %v", err)
        return false
    }
    
    dateStr := date.Format("20060102")
    return calendar.Holidays[dateStr]
}

// getMarketCode 获取市场代码
func getMarketCode(marketType MarketType) string {
    switch marketType {
    case MarketChina:
        return "CN"
    case MarketUS:
        return "US"
    case MarketHK:
        return "HK"
    default:
        return "CN"
    }
}
```


### 3. 更新 `findPreviousTradingDay()`

```go
func findPreviousTradingDay(date time.Time, marketType MarketType) time.Time {
    prevDay := date.AddDate(0, 0, -1)
    maxAttempts := 10 // 防止无限循环
    attempts := 0
    
    for attempts < maxAttempts {
        weekday := prevDay.Weekday()
        
        // 跳过周末
        if weekday == time.Saturday || weekday == time.Sunday {
            prevDay = prevDay.AddDate(0, 0, -1)
            attempts++
            continue
        }
        
        // 跳过假日
        if isHoliday(prevDay, marketType) {
            prevDay = prevDay.AddDate(0, 0, -1)
            attempts++
            continue
        }
        
        return prevDay
    }
    
    // 如果找不到，返回原日期
    debugPrint("警告: 无法找到前一个交易日")
    return date
}
```

---

## 测试计划

### 1. 单元测试

创建 `tools/calendar_test.go`:

```go
package main

import (
    "testing"
    "time"
)

func TestSplitCalendarByYear(t *testing.T) {
    // 测试数据分割逻辑
    calendar := &HolidayCalendar{
        Market: "CN",
        Year:   2025,
        Holidays: map[string]bool{
            "20230101": true,
            "20240101": true,
            "20250101": true,
            "20260101": true,
        },
        CompDays: map[string]bool{
            "20230128": true,
            "20240128": true,
            "20250128": true,
        },
    }
    
    result := splitCalendarByYear(calendar)
    
    if len(result) != 4 {
        t.Errorf("期望 4 年，实际 %d 年", len(result))
    }
    
    // 验证 2025 年数据
    cal2025, ok := result[2025]
    if !ok {
        t.Error("缺少 2025 年数据")
    }
    
    if len(cal2025.Holidays) != 1 {
        t.Errorf("2025 年期望 1 个假日，实际 %d 个", len(cal2025.Holidays))
    }
    
    if len(cal2025.CompDays) != 1 {
        t.Errorf("2025 年期望 1 个调休日，实际 %d 个", len(cal2025.CompDays))
    }
}

func TestValidateDates(t *testing.T) {
    // 测试有效日期格式
    validDates := map[string]bool{
        "20230101": true,
        "20231231": true,
    }
    if !validateDates(validDates) {
        t.Error("有效日期验证失败")
    }
    
    // 测试无效日期格式
    invalidDates := map[string]bool{
        "2023-01-01": true,
        "2023010":    true,
        "abc":        true,
    }
    if validateDates(invalidDates) {
        t.Error("无效日期验证应该失败")
    }
}

func TestLoadCalendar(t *testing.T) {
    // 测试加载 2025 年日历
    calendar, err := LoadCalendar("CN", 2025)
    if err != nil {
        t.Fatalf("加载日历失败: %v", err)
    }
    
    if calendar.Market != "CN" {
        t.Errorf("期望市场代码 'CN'，实际 '%s'", calendar.Market)
    }
    
    if calendar.Year != 2025 {
        t.Errorf("期望年份 2025，实际 %d", calendar.Year)
    }
    
    if len(calendar.Holidays) == 0 {
        t.Error("假日数据不应为空")
    }
}
```

运行测试:

```bash
go test -v ./tools/
```

### 2. 集成测试

```bash
# 测试日历加载
go test -v ./tools/

# 测试假日检测
# 在 intraday.go 中添加测试用例
```

### 3. 手动验证

```bash
# 验证已知假日
# 2025年1月1日应该是假日
echo "测试: 2025-01-01"
python3 -c "
import json
with open('data/calendars/CN/2025.json') as f:
    data = json.load(f)
    print('假日:', data['holidays'].get('20250101', False))
"

# 验证已知调休日
# 2025年1月26日应该是调休日
echo "测试: 2025-01-26"
python3 -c "
import json
with open('data/calendars/CN/2025.json') as f:
    data = json.load(f)
    print('调休:', data['comp_days'].get('20250126', False))
"
```

---

## 回滚计划

如果迁移出现问题，可以快速回滚:

```bash
# 恢复旧文件
cp data/calendars/backup/china_holidays_*.json data/calendars/

# 删除新目录
rm -rf data/calendars/CN/

echo "✅ 已回滚到迁移前状态"
```

---

## 风险与注意事项

### 风险

1. **数据丢失**: 迁移过程中可能丢失数据
   - **缓解措施**: 先备份，验证无误后再删除
   
2. **格式错误**: JSON 文件可能格式错误
   - **缓解措施**: 使用验证工具检查

3. **代码兼容性**: 其他代码可能依赖旧文件路径
   - **缓解措施**: 全面搜索代码库中的文件引用

4. **数据不完整**: 迁移后数据可能不完整
   - **缓解措施**: 详细的验证和测试

### 注意事项

1. **备份优先**: 务必先备份旧文件
2. **逐步验证**: 每一步都验证结果
3. **代码更新**: 同步更新相关代码
4. **文档更新**: 更新相关文档
5. **团队沟通**: 通知团队成员变更

---

## 时间估算

| 任务 | 预计时间 |
|------|----------|
| 创建迁移工具 | 30 分钟 |
| 测试迁移工具 | 30 分钟 |
| 执行数据迁移 | 10 分钟 |
| 验证数据 | 20 分钟 |
| 更新代码（可选） | 1-2 小时 |
| 测试代码（可选） | 30 分钟 |
| 文档更新 | 20 分钟 |
| **总计** | **3-4 小时** |

---

## 成功标准

- ✅ 所有年份数据成功迁移到新目录
- ✅ 每个文件只包含对应年份的数据
- ✅ JSON 格式正确无误
- ✅ 数据完整性验证通过
- ✅ 文件大小合理（节省约 50% 空间）
- ✅ 备份文件已保存
- ✅ 相关代码已更新（如需要）

---

## 参考资料

- 现有日历文件: `data/calendars/china_holidays_*.json`
- 类型定义: `types.go` 中的 `TradingState` 和 `MarketConfig`
- 交易状态逻辑: `intraday.go` 中的 `getTradingState()`
- 其他计划文档: `doc/plans/` 目录下的其他计划

---

**文档版本**: v1.0  
**创建日期**: 2026-01-07  
**最后更新**: 2026-01-07  
**作者**: AI Assistant

