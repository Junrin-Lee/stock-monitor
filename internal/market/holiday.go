package market

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// calendarCache caches loaded calendar data in memory to avoid repeated file reads.
var (
	calendarCacheMu   sync.RWMutex
	calendarCacheData = make(map[int]*CalendarData) // year → calendar
)

// Logger 日志记录器接口
type Logger interface {
	Info(key string, args ...interface{})
	Warn(key string, args ...interface{})
	Error(key string, args ...interface{})
	Debug(format string, args ...interface{})
	WarnDirect(format string, args ...interface{})
	ErrorDirect(format string, args ...interface{})
	DebugDirect(format string, args ...interface{})
}

// DefaultLogger 默认日志记录器（空实现）
type DefaultLogger struct{}

func (l *DefaultLogger) Info(key string, args ...interface{})          {}
func (l *DefaultLogger) Warn(key string, args ...interface{})          {}
func (l *DefaultLogger) Error(key string, args ...interface{})         {}
func (l *DefaultLogger) Debug(format string, args ...interface{})      {}
func (l *DefaultLogger) WarnDirect(format string, args ...interface{}) {}
func (l *DefaultLogger) ErrorDirect(format string, args ...interface{}) {}
func (l *DefaultLogger) DebugDirect(format string, args ...interface{}) {}

// HolidayAPIResponse 节假日 API 响应结构
type HolidayAPIResponse struct {
	Name    string                    `json:"Name"`
	Version string                    `json:"Version"`
	Years   map[string][]HolidayEntry `json:"Years"`
}

// HolidayEntry 节假日条目
type HolidayEntry struct {
	Name      string   `json:"Name"`
	StartDate string   `json:"StartDate"`
	EndDate   string   `json:"EndDate"`
	Duration  int      `json:"Duration"`
	CompDays  []string `json:"CompDays"`
}

// CalendarData 日历数据
type CalendarData struct {
	Market    string          `json:"market"`
	Year      int             `json:"year"`
	Holidays  map[string]bool `json:"holidays"`
	CompDays  map[string]bool `json:"comp_days"`
	UpdatedAt string          `json:"updated_at"`
}

// HolidayWorker 节假日数据同步工作器
type HolidayWorker struct {
	logger Logger
}

// NewHolidayWorker 创建节假日工作器
func NewHolidayWorker(logger Logger) *HolidayWorker {
	if logger == nil {
		logger = &DefaultLogger{}
	}
	return &HolidayWorker{logger: logger}
}

// Start 启动节假日数据同步
func (w *HolidayWorker) Start() {
	w.logger.Info("log.holiday.workerStart")

	go func() {
		startTime := time.Now()
		year := time.Now().Year()

		apiResp, err := w.fetchHolidayDataWithRetry(3)
		if err != nil {
			w.logger.Error("log.holiday.allRetryFailed", 3, err.Error())
			return
		}

		yearStr := fmt.Sprintf("%d", year)
		_, exists := apiResp.Years[yearStr]
		if !exists {
			w.logger.Warn("log.holiday.noDataForYear", year)
			return
		}

		w.logger.Info("log.holiday.fetchSuccess", year)

		calendarData, err := w.transformToCalendarData(apiResp, year)
		if err != nil {
			w.logger.ErrorDirect("Failed to transform calendar data: %v", err)
			return
		}

		existingData, err := LoadExistingCalendar(year)
		if err != nil {
			if os.IsNotExist(err) {
				filePath := filepath.Join(".", "data", "calendars", "CN", fmt.Sprintf("%d.json", year))
				if err := SaveCalendarData(calendarData, year); err != nil {
					w.logger.ErrorDirect("Failed to save calendar data: %v", err)
					return
				}
				w.logger.Info("log.holiday.createFile", filePath)
				duration := time.Since(startTime)
				w.logger.Info("log.holiday.complete", duration)
				return
			}
			w.logger.ErrorDirect("Failed to load existing calendar: %v", err)
			return
		}

		changes := DiffCalendarData(existingData, calendarData)
		if changes == 0 {
			calendarData.UpdatedAt = time.Now().Format(time.RFC3339)
			if err := SaveCalendarData(calendarData, year); err != nil {
				w.logger.ErrorDirect("Failed to update timestamp: %v", err)
				return
			}
			w.logger.Info("log.holiday.noChange")
		} else {
			filePath := filepath.Join(".", "data", "calendars", "CN", fmt.Sprintf("%d.json", year))
			if err := SaveCalendarData(calendarData, year); err != nil {
				w.logger.ErrorDirect("Failed to update calendar file: %v", err)
				return
			}
			w.logger.Info("log.holiday.updateFile", filePath, changes)
		}

		duration := time.Since(startTime)
		w.logger.Info("log.holiday.complete", duration)
	}()
}

// fetchHolidayDataWithRetry 带重试的获取节假日数据
func (w *HolidayWorker) fetchHolidayDataWithRetry(maxRetries int) (*HolidayAPIResponse, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			backoff := time.Duration(1<<uint(i-1)) * time.Second
			w.logger.WarnDirect("Holiday worker retry %d/%d after %v", i+1, maxRetries, backoff)
			time.Sleep(backoff)
		}

		resp, err := w.fetchHolidayData()
		if err == nil {
			return resp, nil
		}
		lastErr = err
		w.logger.Warn("log.holiday.fetchFailed", err.Error())
	}
	return nil, lastErr
}

// fetchHolidayData 获取节假日数据
func (w *HolidayWorker) fetchHolidayData() (*HolidayAPIResponse, error) {
	urls := []string{
		"https://www.shuyz.com/githubfiles/china-holiday-calender/master/holidayAPI.json",
		"https://raw.githubusercontent.com/lanceliao/china-holiday-calender/master/holidayAPI.json",
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, url := range urls {
		w.logger.DebugDirect("Holiday worker fetching: %s", url)
		resp, err := client.Get(url)
		if err != nil {
			w.logger.WarnDirect("Holiday worker HTTP error from %s: %v", url, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			w.logger.WarnDirect("Holiday worker HTTP %d from %s", resp.StatusCode, url)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			w.logger.WarnDirect("Holiday worker read error from %s: %v", url, err)
			continue
		}

		var apiResp HolidayAPIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			w.logger.WarnDirect("Holiday worker JSON parse error: %v", err)
			continue
		}

		return &apiResp, nil
	}

	return nil, fmt.Errorf("all API URLs failed")
}

// transformToCalendarData 转换为日历数据格式
func (w *HolidayWorker) transformToCalendarData(apiResp *HolidayAPIResponse, year int) (*CalendarData, error) {
	calendarData := &CalendarData{
		Market:   "CN",
		Year:     year,
		Holidays: make(map[string]bool),
		CompDays: make(map[string]bool),
	}

	yearStr := fmt.Sprintf("%d", year)
	holidays, exists := apiResp.Years[yearStr]
	if !exists {
		return nil, fmt.Errorf("no data for year %d", year)
	}

	for _, holiday := range holidays {
		dates, err := ExpandDateRange(holiday.StartDate, holiday.EndDate)
		if err != nil {
			w.logger.WarnDirect("Failed to expand date range %s-%s: %v", holiday.StartDate, holiday.EndDate, err)
			continue
		}

		for _, date := range dates {
			calendarData.Holidays[date] = true
		}

		for _, compDay := range holiday.CompDays {
			t, err := time.Parse("2006-01-02", compDay)
			if err != nil {
				w.logger.WarnDirect("Failed to parse comp day %s: %v", compDay, err)
				continue
			}
			dateStr := t.Format("20060102")
			calendarData.CompDays[dateStr] = true
		}
	}

	calendarData.UpdatedAt = time.Now().Format(time.RFC3339)

	return calendarData, nil
}

// ExpandDateRange 展开日期范围
func ExpandDateRange(startDate, endDate string) ([]string, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %v", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %v", err)
	}

	var dates []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("20060102"))
	}

	return dates, nil
}

// LoadExistingCalendar 加载现有日历数据
func LoadExistingCalendar(year int) (*CalendarData, error) {
	filePath := filepath.Join(".", "data", "calendars", "CN", fmt.Sprintf("%d.json", year))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var calendar CalendarData
	if err := json.Unmarshal(data, &calendar); err != nil {
		return nil, err
	}
	return &calendar, nil
}

// DiffCalendarData 比较日历数据差异
func DiffCalendarData(old, new *CalendarData) int {
	changes := 0

	for date := range new.Holidays {
		if !old.Holidays[date] {
			changes++
		}
	}

	for date := range old.Holidays {
		if !new.Holidays[date] {
			changes++
		}
	}

	for date := range new.CompDays {
		if !old.CompDays[date] {
			changes++
		}
	}

	for date := range old.CompDays {
		if !new.CompDays[date] {
			changes++
		}
	}

	return changes
}

// SaveCalendarData 保存日历数据（原子写：temp+rename 防止截断）
func SaveCalendarData(data *CalendarData, year int) error {
	invalidateCalendarCache(year)
	dir := filepath.Join(".", "data", "calendars", "CN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%d.json", year))

	tmp, err := os.CreateTemp(dir, fmt.Sprintf("%d.json.tmp.*", year))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(jsonData); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp to target: %w", err)
	}
	return nil
}

// getCachedCalendar returns calendar data from memory cache, loading from disk on first access.
func getCachedCalendar(year int) (*CalendarData, error) {
	calendarCacheMu.RLock()
	if cal, ok := calendarCacheData[year]; ok {
		calendarCacheMu.RUnlock()
		return cal, nil
	}
	calendarCacheMu.RUnlock()

	cal, err := LoadExistingCalendar(year)
	if err != nil {
		return nil, err
	}

	calendarCacheMu.Lock()
	calendarCacheData[year] = cal
	calendarCacheMu.Unlock()
	return cal, nil
}

// invalidateCalendarCache removes cached data for a year so next read reloads from disk.
func invalidateCalendarCache(year int) {
	calendarCacheMu.Lock()
	delete(calendarCacheData, year)
	calendarCacheMu.Unlock()
}

// IsHoliday 检查指定日期是否为节假日
func IsHoliday(date string, year int) bool {
	calendar, err := getCachedCalendar(year)
	if err != nil {
		return false
	}
	return calendar.Holidays[date]
}

// IsCompDay 检查指定日期是否为补班日
func IsCompDay(date string, year int) bool {
	calendar, err := getCachedCalendar(year)
	if err != nil {
		return false
	}
	return calendar.CompDays[date]
}

// IsTradingDay 检查指定日期是否为交易日
// 交易日 = 工作日（周一到周五）+ 补班日 - 节假日
func IsTradingDay(t time.Time) bool {
	date := t.Format("20060102")
	year := t.Year()

	// 先检查是否为节假日
	if IsHoliday(date, year) {
		return false
	}

	// 检查是否为补班日（补班日即使是周末也要交易）
	if IsCompDay(date, year) {
		return true
	}

	// 检查是否为周末
	weekday := t.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}
