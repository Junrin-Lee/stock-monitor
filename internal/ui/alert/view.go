package alert

import (
	"fmt"
	"stock-monitor/internal/types"
	"stock-monitor/internal/ui"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

// ViewParams contains common parameters for alert view rendering
type ViewParams struct {
	GetText         func(string) string
	Config          types.Config
	StockPriceCache map[string]*types.StockPriceCacheEntry
}

// AlertManageViewParams contains parameters for alert management view
type AlertManageViewParams struct {
	ViewParams
	Alerts      []types.Alert
	AlertCursor int
	Message     string
}

// RenderAlertManage renders the alert management interface
func RenderAlertManage(params AlertManageViewParams) string {
	s := params.GetText("alertTitle") + "\n\n"

	// Empty list prompt
	if len(params.Alerts) == 0 {
		s += params.GetText("alert.empty") + "\n\n"
		s += params.GetText("alert.addFirst") + "\n\n"
		s += params.GetText("alertHelp") + "\n"
		return s
	}

	// Create table
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// Table headers
	t.AppendHeader(table.Row{
		params.GetText("alertHeaderCode"),
		params.GetText("alertHeaderName"),
		params.GetText("alertHeaderType"),
		params.GetText("alertHeaderCondition"),
		params.GetText("alertHeaderThreshold"),
		params.GetText("alertHeaderSwitch"),
		params.GetText("alertHeaderFrequency"),
		params.GetText("alertHeaderTriggeredAt"),
		params.GetText("alertHeaderCreated"),
	})

	// Calculate display range (pagination)
	maxLines := params.Config.Display.MaxLines
	startIndex := params.AlertCursor
	if startIndex > len(params.Alerts)-maxLines {
		startIndex = len(params.Alerts) - maxLines
		if startIndex < 0 {
			startIndex = 0
		}
	}
	endIndex := startIndex + maxLines
	if endIndex > len(params.Alerts) {
		endIndex = len(params.Alerts)
	}

	// Data rows
	for i := startIndex; i < endIndex; i++ {
		alert := params.Alerts[i]

		// Switch state (icon only)
		switchText := "✗"
		if alert.IsActive {
			switchText = "✓"
		}

		// Trigger time
		triggeredText := "-"
		if !alert.TriggeredAt.IsZero() {
			triggeredText = alert.TriggeredAt.Format("2006-01-02 15:04:05")
		}

		// Frequency text
		frequencyText := GetFrequencyDisplayText(alert.Frequency, alert.FrequencyDays, params.GetText)

		// Type text
		typeText := GetAlertTypeText(alert.Type, params.GetText)

		// Cursor marker
		cursor := "  "
		if i == params.AlertCursor {
			cursor = "► "
		}

		row := table.Row{
			cursor + alert.StockCode,
			alert.StockName,
			typeText,
			alert.Condition,
			fmt.Sprintf("%.2f", alert.Threshold),
			switchText,
			frequencyText,
			triggeredText,
			alert.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		t.AppendRow(row)
		t.AppendSeparator()
	}

	s += t.Render() + "\n\n"

	// Pagination info
	total := len(params.Alerts)
	currentPos := params.AlertCursor + 1
	s += fmt.Sprintf("📊 %s (%d/%d)\n\n", params.GetText("alertListTitle"), currentPos, total)

	// Help info
	s += params.GetText("alertHelp") + "\n"

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// StockAlertViewParams contains parameters for stock alert view
type StockAlertViewParams struct {
	ViewParams
	StockCode       string
	StockName       string
	Alerts          []types.Alert
	AlertCursor     int
	Message         string
}

// RenderStockAlertManage renders stock-specific alert interface
func RenderStockAlertManage(params StockAlertViewParams) string {
	s := fmt.Sprintf("=== %s (%s) - %s ===\n\n",
		params.StockName, params.StockCode, params.GetText("alertManagement"))

	// Display current price info
	cacheEntry, exists := params.StockPriceCache[params.StockCode]
	if exists && cacheEntry.Data != nil {
		stockData := cacheEntry.Data
		changeStr := fmt.Sprintf("%.2f%%", stockData.ChangePercent)
		if stockData.ChangePercent >= 0 {
			changeStr = "+" + changeStr
		}
		s += fmt.Sprintf("%s: %.2f (%s) | %s: %.2f\n\n",
			params.GetText("alertCurrentPrice"), stockData.Price, changeStr,
			params.GetText("alertPrevClose"), stockData.PrevClose)
	} else {
		s += fmt.Sprintf("%s: -\n\n", params.GetText("alertCurrentPrice"))
	}

	// Empty list prompt
	if len(params.Alerts) == 0 {
		s += params.GetText("alert.stock.empty") + "\n\n"
		s += params.GetText("alertStockEmptyHint1") + "\n"
		s += params.GetText("alertStockEmptyHint2") + "\n\n"
		s += params.GetText("alert.stockHelp") + "\n"
		return s
	}

	// Create table
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// Table headers
	t.AppendHeader(table.Row{
		params.GetText("alertHeaderCode"),
		params.GetText("alertHeaderName"),
		params.GetText("alertHeaderType"),
		params.GetText("alertHeaderCondition"),
		params.GetText("alertHeaderThreshold"),
		params.GetText("alertHeaderSwitch"),
		params.GetText("alertHeaderFrequency"),
		params.GetText("alertHeaderTriggeredAt"),
		params.GetText("alertHeaderCreated"),
	})

	// Calculate display range
	maxLines := params.Config.Display.MaxLines
	startIndex := params.AlertCursor
	if startIndex > len(params.Alerts)-maxLines {
		startIndex = len(params.Alerts) - maxLines
		if startIndex < 0 {
			startIndex = 0
		}
	}
	endIndex := startIndex + maxLines
	if endIndex > len(params.Alerts) {
		endIndex = len(params.Alerts)
	}

	// Data rows
	for i := startIndex; i < endIndex; i++ {
		alert := params.Alerts[i]

		// Switch state
		switchText := "✗"
		if alert.IsActive {
			switchText = "✓"
		}

		// Type text
		typeText := GetAlertTypeText(alert.Type, params.GetText)

		// Frequency text
		frequencyText := GetFrequencyDisplayText(alert.Frequency, alert.FrequencyDays, params.GetText)

		// Trigger time
		triggeredTime := "-"
		if !alert.TriggeredAt.IsZero() {
			triggeredTime = alert.TriggeredAt.Format("2006-01-02 15:04:05")
		}

		// Cursor marker
		cursor := "  "
		if i == params.AlertCursor {
			cursor = "► "
		}

		row := table.Row{
			cursor + alert.StockCode,
			alert.StockName,
			typeText,
			alert.Condition,
			fmt.Sprintf("%.2f", alert.Threshold),
			switchText,
			frequencyText,
			triggeredTime,
			alert.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		t.AppendRow(row)
		t.AppendSeparator()
	}

	s += params.GetText("alertStockListTitle") + "\n\n"
	s += t.Render() + "\n\n"

	// Pagination info
	total := len(params.Alerts)
	currentPos := params.AlertCursor + 1
	s += fmt.Sprintf("📊 %s %s (%d/%d)\n\n",
		params.StockName, params.GetText("alertListTitle"), currentPos, total)

	// Help info
	s += params.GetText("alert.stockHelp") + "\n"

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// AlertAddViewParams contains parameters for alert add view
type AlertAddViewParams struct {
	GetText              func(string) string
	StockCode            string
	StockName            string
	Step                 int
	TagSelectCursor      int
	SelectedAlertType    types.AlertType
	SelectedCondition    string
	AlertThreshold       float64
	AlertInput           string
	AlertFrequencyCursor int
	Message              string
}

// RenderAlertAdd renders the add alert interface
func RenderAlertAdd(params AlertAddViewParams) string {
	s := "=== " + params.GetText("alertAddTitle") + " ===\n\n"

	s += fmt.Sprintf("%s: %s (%s)\n\n",
		params.GetText("alertStock"), params.StockName, params.StockCode)

	switch params.Step {
	case 0: // Select type
		s += params.GetText("alertSelectType") + "\n\n"

		types := []string{
			params.GetText("alertTypePrice"),
			params.GetText("alertTypeRate"),
			params.GetText("alertTypeVolume"),
		}

		for i, typeText := range types {
			prefix := "  "
			if i == params.TagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, typeText)
		}

		s += "\n" + params.GetText("alertHelp.select") + "\n"

	case 1: // Select condition
		typeText := GetAlertTypeText(params.SelectedAlertType, params.GetText)

		s += fmt.Sprintf("%s: %s\n\n", params.GetText("alertType"), typeText)
		s += params.GetText("alertSelectCondition") + "\n\n"

		conditions := []string{
			params.GetText("alertConditionAbove"),
			params.GetText("alertConditionBelow"),
			params.GetText("alertConditionAboveEq"),
			params.GetText("alertConditionBelowEq"),
		}

		for i, condText := range conditions {
			prefix := "  "
			if i == params.TagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, condText)
		}

		s += "\n" + params.GetText("alertHelp.select") + "\n"

	case 2: // Input threshold
		typeText := GetAlertTypeText(params.SelectedAlertType, params.GetText)

		s += fmt.Sprintf("%s: %s\n", params.GetText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n\n", params.GetText("alertCondition"), params.SelectedCondition)

		s += params.GetText("alertInputThreshold") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", params.AlertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += params.GetText("alertHelp.input") + "\n"

	case 3: // Select frequency
		typeText := GetAlertTypeText(params.SelectedAlertType, params.GetText)

		s += fmt.Sprintf("%s: %s\n", params.GetText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n", params.GetText("alertCondition"), params.SelectedCondition)
		s += fmt.Sprintf("%s: %.2f\n\n", params.GetText("alert.threshold"), params.AlertThreshold)

		s += params.GetText("alert.selectFrequency") + "\n\n"

		frequencyTexts := []string{
			params.GetText("alert.frequency.once"),
			params.GetText("alert.frequency.daily"),
			params.GetText("alert.frequency.weekly"),
			params.GetText("alert.frequency.monthly"),
			params.GetText("alert.frequency.everyNDays.option"),
		}

		for i, freqText := range frequencyTexts {
			prefix := "  "
			if i == params.AlertFrequencyCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, freqText)
		}

		s += "\n" + params.GetText("alertHelp.select") + "\n"

	case 4: // Input custom days
		typeText := GetAlertTypeText(params.SelectedAlertType, params.GetText)

		s += fmt.Sprintf("%s: %s\n", params.GetText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n", params.GetText("alertCondition"), params.SelectedCondition)
		s += fmt.Sprintf("%s: %.2f\n", params.GetText("alert.threshold"), params.AlertThreshold)
		s += fmt.Sprintf("%s: %s\n\n", params.GetText("alert.frequency"), params.GetText("alert.frequency.everyNDays.option"))

		s += params.GetText("alert.frequency.enterDays") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", params.AlertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += params.GetText("alertHelp.input") + "\n"
	}

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// BatchMethodSelectViewParams contains parameters for batch method selection view
type BatchMethodSelectViewParams struct {
	GetText   func(string) string
	Cursor    int
	Message   string
}

// RenderBatchMethodSelect renders the batch method selection interface
func RenderBatchMethodSelect(params BatchMethodSelectViewParams) string {
	s := "=== " + params.GetText("alertBatchMethodTitle") + " ===\n\n"

	options := []string{
		params.GetText("alertBatchByTag"),
		params.GetText("alertBatchByMarket"),
		params.GetText("alertBatchByStocks"),
	}

	for i, option := range options {
		prefix := "  "
		if i == params.Cursor {
			prefix = "► "
		}
		s += fmt.Sprintf("%s%d. %s\n", prefix, i+1, option)
	}

	s += "\n" + params.GetText("alertHelp.select") + "\n"

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// BatchByTagViewParams contains parameters for batch by tag view
type BatchByTagViewParams struct {
	GetText       func(string) string
	AvailableTags []string
	Cursor        int
	Message       string
	GetStockCount func(string) int
}

// RenderBatchByTag renders batch add by tag interface
func RenderBatchByTag(params BatchByTagViewParams) string {
	s := "=== " + params.GetText("alertBatchByTagTitle") + " ===\n\n"
	s += params.GetText("alertSelectTagPrompt") + "\n\n"

	if len(params.AvailableTags) == 0 {
		s += params.GetText("alertNoTagsAvailable") + "\n\n"
		s += params.GetText("alertHelp.back") + "\n"
		return s
	}

	for i, tag := range params.AvailableTags {
		prefix := "  "
		if i == params.Cursor {
			prefix = "► "
		}

		stockCount := params.GetStockCount(tag)
		s += fmt.Sprintf("%s%s (%d %s)\n", prefix, tag, stockCount, params.GetText("alertStocksCount"))
	}

	s += "\n" + params.GetText("alertHelp.select") + "\n"

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// BatchByMarketViewParams contains parameters for batch by market view
type BatchByMarketViewParams struct {
	GetText       func(string) string
	Cursor        int
	Message       string
	GetStockCount func(types.MarketType) int
}

// RenderBatchByMarket renders batch add by market interface
func RenderBatchByMarket(params BatchByMarketViewParams) string {
	s := "=== " + params.GetText("alertBatchByMarketTitle") + " ===\n\n"
	s += params.GetText("alertSelectMarketPrompt") + "\n\n"

	markets := []struct {
		name       string
		marketType types.MarketType
	}{
		{params.GetText("alertMarketChina"), types.MarketChina},
		{params.GetText("alertMarketUS"), types.MarketUS},
		{params.GetText("alertMarketHK"), types.MarketHongKong},
	}

	for i, market := range markets {
		prefix := "  "
		if i == params.Cursor {
			prefix = "► "
		}

		stockCount := params.GetStockCount(market.marketType)
		s += fmt.Sprintf("%s%s (%d %s)\n", prefix, market.name, stockCount, params.GetText("alertStocksCount"))
	}

	s += "\n" + params.GetText("alertHelp.select") + "\n"

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// SelectBatchStocksViewParams contains parameters for stock selection view
type SelectBatchStocksViewParams struct {
	GetText func(string) string
	Cursor  int
	Message string
}

// RenderSelectBatchStocks renders batch stock source selection interface
func RenderSelectBatchStocks(params SelectBatchStocksViewParams) string {
	s := "=== " + params.GetText("alert.batch.byStocksTitle") + " ===\n\n"

	options := []string{
		params.GetText("alert.batch.fromWatchlist"),
		params.GetText("alert.batch.fromPortfolio"),
		params.GetText("alert.batch.manualInput"),
	}

	for i, option := range options {
		prefix := "  "
		if i == params.Cursor {
			prefix = "► "
		}
		s += fmt.Sprintf("%s%d. %s\n", prefix, i+1, option)
	}

	s += "\n" + params.GetText("alertHelp.select") + "\n"

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// BatchStockListViewParams contains parameters for batch stock list view
type BatchStockListViewParams struct {
	GetText        func(string) string
	Config         types.Config
	Stocks         []ui.SelectableStock
	Cursor         int
	SelectionMap   map[string]bool
	Message        string
	IsFromWatchlist bool
}

// RenderBatchStockList renders stock list for batch selection
func RenderBatchStockList(params BatchStockListViewParams) string {
	var s string
	if params.IsFromWatchlist {
		s = "=== " + params.GetText("alertSelectFromWatchlistTitle") + " ===\n\n"
	} else {
		s = "=== " + params.GetText("alertSelectFromPortfolioTitle") + " ===\n\n"
	}

	if len(params.Stocks) == 0 {
		if params.IsFromWatchlist {
			s += params.GetText("emptyWatchlist") + "\n\n"
		} else {
			s += params.GetText("emptyPortfolio") + "\n\n"
		}
		s += params.GetText("alertHelp.back") + "\n"
		return s
	}

	s += params.GetText("alertMultiSelectPrompt") + "\n\n"

	// Display stock list
	maxLines := params.Config.Display.MaxLines
	startIndex := params.Cursor
	if startIndex > len(params.Stocks)-maxLines {
		startIndex = len(params.Stocks) - maxLines
		if startIndex < 0 {
			startIndex = 0
		}
	}
	endIndex := startIndex + maxLines
	if endIndex > len(params.Stocks) {
		endIndex = len(params.Stocks)
	}

	for i := startIndex; i < endIndex; i++ {
		stock := params.Stocks[i]

		// Selection marker
		checkbox := "  "
		if params.SelectionMap[stock.Code] {
			checkbox = "✓ "
		}

		// Cursor marker
		cursor := " "
		if i == params.Cursor {
			cursor = "►"
		}

		if params.IsFromWatchlist {
			s += fmt.Sprintf("%s%s[%d] %s - %s\n", cursor, checkbox, i+1, stock.Code, stock.Name)
		} else {
			s += fmt.Sprintf("%s%s[%d] %s - %s (%d%s, %s%.2f)\n",
				cursor, checkbox, i+1, stock.Code, stock.Name,
				stock.Quantity, params.GetText("alertSharesUnit"),
				params.GetText("alertCostLabel"), stock.CostPrice)
		}
	}

	// Count selected
	selectedCount := 0
	for _, selected := range params.SelectionMap {
		if selected {
			selectedCount++
		}
	}

	s += fmt.Sprintf("\n%s: %d %s\n\n", params.GetText("alertSelectedCount"), selectedCount, params.GetText("alertStocksCount"))
	s += params.GetText("alert.batch.multiSelectHelp") + "\n"

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// InputBatchCodesViewParams contains parameters for manual code input view
type InputBatchCodesViewParams struct {
	GetText func(string) string
	Input   string
	Message string
}

// RenderInputBatchCodes renders manual stock code input interface
func RenderInputBatchCodes(params InputBatchCodesViewParams) string {
	s := "=== " + params.GetText("alertInputCodesTitle") + " ===\n\n"
	s += params.GetText("alertInputCodesPrompt") + "\n\n"

	s += "┌────────────────────────────────────────────┐\n"
	// Display input content (multi-line support)
	lines := strings.Split(params.Input, "\n")
	displayLines := 5
	for i := 0; i < displayLines; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		s += fmt.Sprintf("│ %-42s │\n", line)
	}
	s += "└────────────────────────────────────────────┘\n\n"

	s += params.GetText("alertInputCodesExample") + "\n\n"
	s += params.GetText("alertHelp.input") + "\n"

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// BatchAddViewParams contains parameters for batch add view
type BatchAddViewParams struct {
	AlertAddViewParams
	SelectedStocks []string
}

// RenderBatchAdd renders batch add alert interface
func RenderBatchAdd(params BatchAddViewParams) string {
	s := "=== " + params.GetText("alertBatchAddTitle") + " ===\n\n"

	s += fmt.Sprintf("%s: %d %s\n\n",
		params.GetText("alertBatchStockCount"),
		len(params.SelectedStocks),
		params.GetText("alertStocksCount"))

	// Display first 5 stocks as preview
	previewCount := 5
	if len(params.SelectedStocks) < previewCount {
		previewCount = len(params.SelectedStocks)
	}
	for i := 0; i < previewCount; i++ {
		s += fmt.Sprintf("  [%d] %s\n", i+1, params.SelectedStocks[i])
	}
	if len(params.SelectedStocks) > previewCount {
		s += fmt.Sprintf("  ... (%s %d %s)\n",
			params.GetText("alertAndMore"),
			len(params.SelectedStocks)-previewCount,
			params.GetText("alertStocksCount"))
	}
	s += "\n"

	// Reuse single stock add UI
	switch params.Step {
	case 0: // Select type
		s += params.GetText("alertSelectType") + "\n\n"

		types := []string{
			params.GetText("alertTypePrice"),
			params.GetText("alertTypeRate"),
			params.GetText("alertTypeVolume"),
		}

		for i, typeText := range types {
			prefix := "  "
			if i == params.TagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, typeText)
		}

		s += "\n" + params.GetText("alertHelp.select") + "\n"

	case 1: // Select condition
		typeText := GetAlertTypeText(params.SelectedAlertType, params.GetText)

		s += fmt.Sprintf("%s: %s\n\n", params.GetText("alertType"), typeText)
		s += params.GetText("alertSelectCondition") + "\n\n"

		conditions := []string{
			params.GetText("alertConditionAbove"),
			params.GetText("alertConditionBelow"),
			params.GetText("alertConditionAboveEq"),
			params.GetText("alertConditionBelowEq"),
		}

		for i, condText := range conditions {
			prefix := "  "
			if i == params.TagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, condText)
		}

		s += "\n" + params.GetText("alertHelp.select") + "\n"

	case 2: // Input threshold
		typeText := GetAlertTypeText(params.SelectedAlertType, params.GetText)

		s += fmt.Sprintf("%s: %s\n", params.GetText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n\n", params.GetText("alertCondition"), params.SelectedCondition)

		s += params.GetText("alertInputThreshold") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", params.AlertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += params.GetText("alertHelp.input") + "\n"

	case 3: // Select frequency
		typeText := GetAlertTypeText(params.SelectedAlertType, params.GetText)

		s += fmt.Sprintf("%s: %s\n", params.GetText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n", params.GetText("alertCondition"), params.SelectedCondition)
		s += fmt.Sprintf("%s: %.2f\n\n", params.GetText("alert.threshold"), params.AlertThreshold)

		s += params.GetText("alert.selectFrequency") + "\n\n"

		frequencyTexts := []string{
			params.GetText("alert.frequency.once"),
			params.GetText("alert.frequency.daily"),
			params.GetText("alert.frequency.weekly"),
			params.GetText("alert.frequency.monthly"),
			params.GetText("alert.frequency.everyNDays.option"),
		}

		for i, freqText := range frequencyTexts {
			prefix := "  "
			if i == params.AlertFrequencyCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, freqText)
		}

		s += "\n" + params.GetText("alertHelp.select") + "\n"

	case 4: // Input custom days
		typeText := GetAlertTypeText(params.SelectedAlertType, params.GetText)

		s += fmt.Sprintf("%s: %s\n", params.GetText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n", params.GetText("alertCondition"), params.SelectedCondition)
		s += fmt.Sprintf("%s: %.2f\n", params.GetText("alert.threshold"), params.AlertThreshold)
		s += fmt.Sprintf("%s: %s\n\n", params.GetText("alert.frequency"), params.GetText("alert.frequency.everyNDays.option"))

		s += params.GetText("alert.frequency.enterDays") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", params.AlertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += params.GetText("alertHelp.input") + "\n"
	}

	if params.Message != "" {
		s += "\n" + params.Message + "\n"
	}

	return s
}

// GetAlertTypeText returns localized alert type text
func GetAlertTypeText(alertType types.AlertType, getText func(string) string) string {
	switch alertType {
	case types.AlertTypePrice:
		return getText("alertTypePrice")
	case types.AlertTypeRate:
		return getText("alertTypeRate")
	case types.AlertTypeVolume:
		return getText("alertTypeVolume")
	default:
		return ""
	}
}

// GetFrequencyDisplayText returns localized frequency text
func GetFrequencyDisplayText(frequency types.TriggerFrequency, customDays int, getText func(string) string) string {
	switch frequency {
	case types.TriggerOnce:
		return getText("alert.frequency.once")
	case types.TriggerDaily:
		return getText("alert.frequency.daily")
	case types.TriggerWeekly:
		return getText("alert.frequency.weekly")
	case types.TriggerMonthly:
		return getText("alert.frequency.monthly")
	case types.TriggerEveryNDays:
		return fmt.Sprintf(getText("alert.frequency.everyNDays.display"), customDays)
	default:
		return getText("alert.frequency.once")
	}
}
