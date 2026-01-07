package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type HolidayAPIResponse struct {
	Name    string                    `json:"Name"`
	Version string                    `json:"Version"`
	Years   map[string][]HolidayEntry `json:"Years"`
}

type HolidayEntry struct {
	Name      string   `json:"Name"`
	StartDate string   `json:"StartDate"`
	EndDate   string   `json:"EndDate"`
	Duration  int      `json:"Duration"`
	CompDays  []string `json:"CompDays"`
}

type CalendarData struct {
	Market    string          `json:"market"`
	Year      int             `json:"year"`
	Holidays  map[string]bool `json:"holidays"`
	CompDays  map[string]bool `json:"comp_days"`
	UpdatedAt string          `json:"updated_at"`
}

func StartHolidayWorker() {
	logInfo("log.holiday.workerStart")

	go func() {
		startTime := time.Now()
		year := time.Now().Year()

		apiResp, err := fetchHolidayDataWithRetry(3)
		if err != nil {
			logError("log.holiday.allRetryFailed", 3, err.Error())
			return
		}

		yearStr := fmt.Sprintf("%d", year)
		_, exists := apiResp.Years[yearStr]
		if !exists {
			logWarn("log.holiday.noDataForYear", year)
			return
		}

		logInfo("log.holiday.fetchSuccess", year)

		calendarData, err := transformToCalendarData(apiResp, year)
		if err != nil {
			logErrorDirect("Failed to transform calendar data: %v", err)
			return
		}

		existingData, err := loadExistingCalendar(year)
		if err != nil {
			if os.IsNotExist(err) {
				filePath := filepath.Join(".", "data", "calendars", "CN", fmt.Sprintf("%d.json", year))
				if err := saveCalendarData(calendarData, year); err != nil {
					logErrorDirect("Failed to save calendar data: %v", err)
					return
				}
				logInfo("log.holiday.createFile", filePath)
				duration := time.Since(startTime)
				logInfo("log.holiday.complete", duration)
				return
			}
			logErrorDirect("Failed to load existing calendar: %v", err)
			return
		}

		changes := diffCalendarData(existingData, calendarData)
		if changes == 0 {
			calendarData.UpdatedAt = time.Now().Format(time.RFC3339)
			if err := saveCalendarData(calendarData, year); err != nil {
				logErrorDirect("Failed to update timestamp: %v", err)
				return
			}
			logInfo("log.holiday.noChange")
		} else {
			filePath := filepath.Join(".", "data", "calendars", "CN", fmt.Sprintf("%d.json", year))
			if err := saveCalendarData(calendarData, year); err != nil {
				logErrorDirect("Failed to update calendar file: %v", err)
				return
			}
			logInfo("log.holiday.updateFile", filePath, changes)
		}

		duration := time.Since(startTime)
		logInfo("log.holiday.complete", duration)
	}()
}

func fetchHolidayDataWithRetry(maxRetries int) (*HolidayAPIResponse, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			backoff := time.Duration(1<<uint(i-1)) * time.Second
			logWarnDirect("Holiday worker retry %d/%d after %v", i+1, maxRetries, backoff)
			time.Sleep(backoff)
		}

		resp, err := fetchHolidayData()
		if err == nil {
			return resp, nil
		}
		lastErr = err
		logWarn("log.holiday.fetchFailed", err.Error())
	}
	return nil, lastErr
}

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

func transformToCalendarData(apiResp *HolidayAPIResponse, year int) (*CalendarData, error) {
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
		dates, err := expandDateRange(holiday.StartDate, holiday.EndDate)
		if err != nil {
			logWarnDirect("Failed to expand date range %s-%s: %v", holiday.StartDate, holiday.EndDate, err)
			continue
		}

		for _, date := range dates {
			calendarData.Holidays[date] = true
		}

		for _, compDay := range holiday.CompDays {
			t, err := time.Parse("2006-01-02", compDay)
			if err != nil {
				logWarnDirect("Failed to parse comp day %s: %v", compDay, err)
				continue
			}
			dateStr := t.Format("20060102")
			calendarData.CompDays[dateStr] = true
		}
	}

	calendarData.UpdatedAt = time.Now().Format(time.RFC3339)

	return calendarData, nil
}

func expandDateRange(startDate, endDate string) ([]string, error) {
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

func loadExistingCalendar(year int) (*CalendarData, error) {
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

func diffCalendarData(old, new *CalendarData) int {
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

func saveCalendarData(data *CalendarData, year int) error {
	dir := filepath.Join(".", "data", "calendars", "CN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%d.json", year))
	return os.WriteFile(filePath, jsonData, 0644)
}
