package main

import (
	"fmt"
	"stock-monitor/internal/consts"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"stock-monitor/internal/types"
	"stock-monitor/internal/ui"
	alertui "stock-monitor/internal/ui/alert"
)

// ============================================================================
// Type Conversion Helpers
// ============================================================================

// Note: convertAlertsToTypes and convertAlertsFromTypes are defined in persistence.go

// convertCacheToTypes converts main cache map to types cache map
func convertCacheToTypes(cache map[string]*StockPriceCacheEntry) map[string]*types.StockPriceCacheEntry {
	result := make(map[string]*types.StockPriceCacheEntry)
	for k, v := range cache {
		var data *types.StockData
		if v.Data != nil {
			data = &types.StockData{
				Symbol:        v.Data.Symbol,
				Name:          v.Data.Name,
				Price:         v.Data.Price,
				Change:        v.Data.Change,
				ChangePercent: v.Data.ChangePercent,
				StartPrice:    v.Data.StartPrice,
				MaxPrice:      v.Data.MaxPrice,
				MinPrice:      v.Data.MinPrice,
				PrevClose:     v.Data.PrevClose,
				Volume:        v.Data.Volume,
				TurnoverRate:  v.Data.TurnoverRate,
			}
		}
		result[k] = &types.StockPriceCacheEntry{
			Data:       data,
			UpdateTime: v.UpdateTime,
		}
	}
	return result
}

// convertPortfolioToTypes converts main.Portfolio to types.Portfolio
func convertPortfolioToTypes(p Portfolio) types.Portfolio {
	stocks := make([]types.Stock, len(p.Stocks))
	for i, s := range p.Stocks {
		stocks[i] = types.Stock(s)
	}
	return types.Portfolio{Stocks: stocks}
}

// convertWatchlistToTypes converts main.Watchlist to types.Watchlist
func convertWatchlistToTypes(w Watchlist) types.Watchlist {
	stocks := make([]types.WatchlistStock, len(w.Stocks))
	for i, s := range w.Stocks {
		stocks[i] = types.WatchlistStock(s)
	}
	return types.Watchlist{Stocks: stocks}
}

// ============================================================================
// UUID Generation
// ============================================================================

// generateAlertID generates UUID v4 for alerts
func generateAlertID() string {
	return uuid.New().String()
}

// ============================================================================
// Alert Checking Logic
// ============================================================================

// checkAlertsMsg message for checking alerts
type checkAlertsMsg struct{}

// handleCheckAlerts checks all alert conditions
func (m *Model) handleCheckAlerts(msg checkAlertsMsg) (tea.Model, tea.Cmd) {
	// Build cache map for alert checking
	m.stockPriceMutex.RLock()
	cacheMap := convertCacheToTypes(m.stockPriceCache)
	m.stockPriceMutex.RUnlock()

	// Convert alerts to types.Alert for checking
	typesAlerts := convertAlertsToTypes(m.alertData.Alerts)

	// Check alerts using the alert package
	triggeredTypesAlerts := alertui.CheckAlerts(typesAlerts, cacheMap, func(a types.Alert) bool {
		return canTriggerInCurrentPeriod(Alert(a))
	})

	// Convert back to main.Alert
	triggeredAlerts := convertAlertsFromTypes(triggeredTypesAlerts)

	// Update triggered alerts
	if len(triggeredAlerts) > 0 {
		for i := range m.alertData.Alerts {
			for _, triggered := range triggeredAlerts {
				if m.alertData.Alerts[i].ID == triggered.ID {
					// Update trigger time
					m.alertData.Alerts[i].TriggeredAt = time.Now()

					// Disable one-time alerts
					if m.alertData.Alerts[i].Frequency == TriggerOnce || m.alertData.Alerts[i].Frequency == "" {
						m.alertData.Alerts[i].IsActive = false
					}

					logInfo("log.alert.triggered", m.alertData.Alerts[i].StockName, m.alertData.Alerts[i].StockCode)
					break
				}
			}
			// Update last checked time
			m.alertData.Alerts[i].LastChecked = time.Now()
		}

		// Save alert data
		m.alertData.LastCheck = time.Now().Format("2006-01-02T15:04:05Z07:00")
		m.saveAlertData()

		// Send notifications
		for _, alert := range triggeredAlerts {
			m.sendAlertNotification(alert)
		}
	}

	return m, nil
}

// sendAlertNotification sends alert notification
func (m *Model) sendAlertNotification(a Alert) {
	alertui.SendNotification(alertui.NotificationParams{
		Alert:   types.Alert(a),
		GetText: m.getText,
	})
}

// ============================================================================
// Helper Functions - Adapter Methods
// ============================================================================

// getStockAlerts gets all alerts for a specific stock
func (m *Model) getStockAlerts(stockCode string) []Alert {
	typesAlerts := make([]types.Alert, len(m.alertData.Alerts))
	for i, a := range m.alertData.Alerts {
		typesAlerts[i] = types.Alert(a)
	}
	result := alertui.GetStockAlerts(typesAlerts, stockCode)
	alerts := make([]Alert, len(result))
	for i, a := range result {
		alerts[i] = Alert(a)
	}
	return alerts
}

// getStocksByTag gets all stock codes with the specified tag
func (m *Model) getStocksByTag(tag string) []string {
	return alertui.GetStocksByTag(convertWatchlistToTypes(m.watchlist), tag)
}

// getStocksByMarket gets all stock codes in the specified market
func (m *Model) getStocksByMarket(marketType MarketType) []string {
	return alertui.GetStocksByMarket(
		convertWatchlistToTypes(m.watchlist),
		convertPortfolioToTypes(m.portfolio),
		types.MarketType(marketType),
	)
}

// parseStockCodes parses stock codes from input
func parseStockCodes(input string) []string {
	return alertui.ParseStockCodes(input)
}

// Note: GetAlertTypeFromCursor, GetAlertConditionFromCursor are defined in types.go
// Note: getFrequencyOptions is defined in alert_frequency.go

// ============================================================================
// consts.AlertManage State Handler
// ============================================================================

// handleAlertManage handles alert management state
func (m *Model) handleAlertManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // Return to main menu
		m.state = consts.MainMenu
		m.message = ""
		return m, nil

	case "a": // Add alert
		m.state = consts.SelectBatchStocks
		m.batchSelectStep = 0
		m.batchSelectedStocks = nil
		m.stockSelectionMap = make(map[string]bool)
		return m, nil

	case "g": // Batch add
		m.state = consts.AlertBatchMethodSelect
		m.batchSelectStep = 0
		return m, nil

	case "e": // Edit alert
		if len(m.alertData.Alerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.alertCursor >= len(m.alertData.Alerts) {
			return m, nil
		}

		// Save current selected alert
		m.currentAlert = m.alertData.Alerts[m.alertCursor]
		m.selectedAlertType = m.currentAlert.Type
		m.selectedAlertCondition = m.currentAlert.Condition
		m.alertThreshold = m.currentAlert.Threshold
		// Initialize frequency fields for editing
		m.selectedAlertFrequency = m.currentAlert.Frequency
		m.alertFrequencyDays = m.currentAlert.FrequencyDays
		m.alertFrequencyCursor = getFrequencyCursorFromValue(m.currentAlert.Frequency)

		m.state = consts.AlertEdit
		m.alertManageStep = 0
		return m, nil

	case "d": // Delete alert
		if len(m.alertData.Alerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.alertCursor >= len(m.alertData.Alerts) {
			return m, nil
		}

		deletedAlert := m.alertData.Alerts[m.alertCursor]
		m.alertData.Alerts = append(
			m.alertData.Alerts[:m.alertCursor],
			m.alertData.Alerts[m.alertCursor+1:]...,
		)
		m.saveAlertData()

		if m.alertCursor >= len(m.alertData.Alerts) && m.alertCursor > 0 {
			m.alertCursor--
		}

		m.message = fmt.Sprintf(m.getText("alert.deleteSuccess"), deletedAlert.StockName)
		return m, nil

	case " ": // Toggle alert enable/disable
		if len(m.alertData.Alerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.alertCursor >= len(m.alertData.Alerts) {
			return m, nil
		}

		m.alertData.Alerts[m.alertCursor].IsActive = !m.alertData.Alerts[m.alertCursor].IsActive
		m.alertData.Alerts[m.alertCursor].UpdatedAt = time.Now()
		m.saveAlertData()

		alert := m.alertData.Alerts[m.alertCursor]
		if alert.IsActive {
			m.message = fmt.Sprintf(m.getText("alert.toggle.enabled"), alert.StockName)
		} else {
			m.message = fmt.Sprintf(m.getText("alert.toggle.disabled"), alert.StockName)
		}
		return m, nil

	case "enter": // View alert details
		if len(m.alertData.Alerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.alertCursor >= len(m.alertData.Alerts) {
			return m, nil
		}

		alert := m.alertData.Alerts[m.alertCursor]
		typeText := alertui.GetAlertTypeText(alert.Type, m.getText)

		details := fmt.Sprintf("%s: %s %s %s %.2f | %s: %s",
			m.getText("alertDetailStock"), alert.StockName, typeText, alert.Condition, alert.Threshold,
			m.getText("alertDetailCreated"), alert.CreatedAt.Format("2006-01-02 15:04"))

		m.message = details
		return m, nil

	case "up", "k", "w": // Scroll up
		if m.alertCursor > 0 {
			m.alertCursor--
		}
		return m, nil

	case "down", "j", "s": // Scroll down
		if m.alertCursor < len(m.alertData.Alerts)-1 {
			m.alertCursor++
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewAlertManage renders alert management interface
func (m *Model) viewAlertManage() string {
	m.stockPriceMutex.RLock()
	cacheMap := convertCacheToTypes(m.stockPriceCache)
	m.stockPriceMutex.RUnlock()

	typesAlerts := convertAlertsToTypes(m.alertData.Alerts)

	return alertui.RenderAlertManage(alertui.AlertManageViewParams{
		ViewParams: alertui.ViewParams{
			GetText:         m.getText,
			Config:          types.Config(m.config),
			StockPriceCache: cacheMap,
		},
		Alerts:      typesAlerts,
		AlertCursor: m.alertCursor,
		Message:     m.message,
	})
}

// ============================================================================
// consts.StockAlertManage State Handler
// ============================================================================

// handleStockAlertManage handles stock alert details state
func (m *Model) handleStockAlertManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // Return to source list
		m.state = m.previousState
		m.message = ""
		m.stockAlertAlerts = nil
		return m, nil

	case "a": // Add alert
		m.state = consts.AlertAdd
		m.alertManageStep = 0
		m.alertInput = ""
		m.alertInputCursor = 0
		m.currentAlert = Alert{
			StockCode: m.stockAlertCode,
			StockName: m.stockAlertName,
			Type:      AlertTypePrice,
			Condition: ">",
			Threshold: 0,
			IsActive:  true,
			CreatedAt: time.Now(),
		}
		return m, nil

	case "e": // Edit alert
		if len(m.stockAlertAlerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.stockAlertCursor >= len(m.stockAlertAlerts) {
			return m, nil
		}

		m.currentAlert = m.stockAlertAlerts[m.stockAlertCursor]
		m.selectedAlertType = m.currentAlert.Type
		m.selectedAlertCondition = m.currentAlert.Condition
		m.alertThreshold = m.currentAlert.Threshold
		// Initialize frequency fields for editing
		m.selectedAlertFrequency = m.currentAlert.Frequency
		m.alertFrequencyDays = m.currentAlert.FrequencyDays
		m.alertFrequencyCursor = getFrequencyCursorFromValue(m.currentAlert.Frequency)

		m.state = consts.AlertEdit
		m.alertManageStep = 0
		return m, nil

	case "d": // Delete alert
		if len(m.stockAlertAlerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.stockAlertCursor >= len(m.stockAlertAlerts) {
			return m, nil
		}

		deletedAlert := m.stockAlertAlerts[m.stockAlertCursor]

		// Delete from global alert list
		for i, alert := range m.alertData.Alerts {
			if alert.ID == deletedAlert.ID {
				m.alertData.Alerts = append(
					m.alertData.Alerts[:i],
					m.alertData.Alerts[i+1:]...,
				)
				break
			}
		}

		// Also delete from stock alert list
		m.stockAlertAlerts = append(
			m.stockAlertAlerts[:m.stockAlertCursor],
			m.stockAlertAlerts[m.stockAlertCursor+1:]...,
		)

		if m.stockAlertCursor >= len(m.stockAlertAlerts) && m.stockAlertCursor > 0 {
			m.stockAlertCursor--
		}

		m.saveAlertData()
		m.message = fmt.Sprintf(m.getText("alert.deleteSuccess"), deletedAlert.StockName)
		return m, nil

	case " ": // Toggle alert enable/disable
		if len(m.stockAlertAlerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.stockAlertCursor >= len(m.stockAlertAlerts) {
			return m, nil
		}

		alertID := m.stockAlertAlerts[m.stockAlertCursor].ID

		for i := range m.alertData.Alerts {
			if m.alertData.Alerts[i].ID == alertID {
				m.alertData.Alerts[i].IsActive = !m.alertData.Alerts[i].IsActive
				m.alertData.Alerts[i].UpdatedAt = time.Now()
				m.stockAlertAlerts[m.stockAlertCursor].IsActive = m.alertData.Alerts[i].IsActive
				break
			}
		}
		m.saveAlertData()

		alert := m.stockAlertAlerts[m.stockAlertCursor]
		if alert.IsActive {
			m.message = fmt.Sprintf(m.getText("alert.toggle.enabled"), alert.StockName)
		} else {
			m.message = fmt.Sprintf(m.getText("alert.toggle.disabled"), alert.StockName)
		}
		return m, nil

	case "enter": // View alert details
		if len(m.stockAlertAlerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.stockAlertCursor >= len(m.stockAlertAlerts) {
			return m, nil
		}

		alert := m.stockAlertAlerts[m.stockAlertCursor]
		typeText := alertui.GetAlertTypeText(alert.Type, m.getText)

		details := fmt.Sprintf("%s %s %.2f | %s: %s",
			typeText, alert.Condition, alert.Threshold,
			m.getText("alertDetailCreated"), alert.CreatedAt.Format("2006-01-02 15:04"))

		m.message = details
		return m, nil

	case "up", "k", "w": // Scroll up
		if m.stockAlertCursor > 0 {
			m.stockAlertCursor--
		}
		return m, nil

	case "down", "j", "s": // Scroll down
		if m.stockAlertCursor < len(m.stockAlertAlerts)-1 {
			m.stockAlertCursor++
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewStockAlertManage renders stock alert details interface
func (m *Model) viewStockAlertManage() string {
	m.stockPriceMutex.RLock()
	cacheMap := convertCacheToTypes(m.stockPriceCache)
	m.stockPriceMutex.RUnlock()

	typesAlerts := convertAlertsToTypes(m.stockAlertAlerts)

	return alertui.RenderStockAlertManage(alertui.StockAlertViewParams{
		ViewParams: alertui.ViewParams{
			GetText:         m.getText,
			Config:          types.Config(m.config),
			StockPriceCache: cacheMap,
		},
		StockCode:   m.stockAlertCode,
		StockName:   m.stockAlertName,
		Alerts:      typesAlerts,
		AlertCursor: m.stockAlertCursor,
		Message:     m.message,
	})
}

// ============================================================================
// consts.AlertAdd State Handler (Type/Condition/Threshold)
// ============================================================================

// handleAlertAdd handles add alert state
func (m *Model) handleAlertAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.alertManageStep {
	case 0: // Select alert type
		return m.handleAlertTypeSelect(msg)
	case 1: // Select condition
		return m.handleAlertConditionSelect(msg)
	case 2: // Input threshold
		return m.handleAlertThresholdInput(msg)
	case 3: // Select trigger frequency
		return m.handleAlertFrequencySelectStep(msg)
	case 4: // Input custom days
		return m.handleAlertFrequencyDaysInput(msg)
	default:
		return m, nil
	}
}

// handleAlertTypeSelect handles alert type selection
func (m *Model) handleAlertTypeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return
		if m.previousState == consts.StockAlertManage {
			m.state = consts.StockAlertManage
		} else {
			m.state = consts.AlertManage
		}
		m.message = ""
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 2 { // 3 types
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		m.selectedAlertType = GetAlertTypeFromCursor(m.tagSelectCursor)
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertConditionSelect handles condition selection
func (m *Model) handleAlertConditionSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 3 { // 4 conditions
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		m.selectedAlertCondition = GetAlertConditionFromCursor(m.tagSelectCursor)
		m.alertManageStep = 2
		m.alertInput = ""
		m.alertInputCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertThresholdInput handles threshold input
func (m *Model) handleAlertThresholdInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	case "enter":
		var threshold float64
		_, err := fmt.Sscanf(m.alertInput, "%f", &threshold)
		if err != nil || m.alertInput == "" {
			m.message = m.getText("alert.invalidThreshold")
			return m, nil
		}

		m.alertThreshold = threshold
		m.alertManageStep = 3
		m.alertFrequencyCursor = 0
		m.selectedAlertFrequency = ""
		return m, nil

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			char := msg.String()
			if (char >= "0" && char <= "9") || char == "." || char == "-" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// handleAlertFrequencySelectStep handles trigger frequency selection
func (m *Model) handleAlertFrequencySelectStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	frequencyOptions := getFrequencyOptions()

	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 2
		m.alertInput = fmt.Sprintf("%.2f", m.alertThreshold)
		return m, nil

	case "up", "k", "w":
		if m.alertFrequencyCursor > 0 {
			m.alertFrequencyCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.alertFrequencyCursor < len(frequencyOptions)-1 {
			m.alertFrequencyCursor++
		}
		return m, nil

	case "enter", " ":
		m.selectedAlertFrequency = frequencyOptions[m.alertFrequencyCursor]

		if m.selectedAlertFrequency == TriggerEveryNDays {
			m.alertManageStep = 4
			m.alertInput = ""
			m.alertInputCursor = 0
		} else {
			return m.createAlertWithFrequency()
		}
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertFrequencyDaysInput handles custom days input
func (m *Model) handleAlertFrequencyDaysInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 3
		m.alertFrequencyCursor = 4
		return m, nil

	case "enter":
		var days int
		_, err := fmt.Sscanf(m.alertInput, "%d", &days)
		if err != nil || days <= 0 {
			m.message = m.getText("alert.frequency.invalidDays")
			return m, nil
		}
		m.alertFrequencyDays = days
		return m.createAlertWithFrequency()

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			char := msg.String()
			if char >= "0" && char <= "9" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// createAlertWithFrequency creates alert with selected frequency
func (m *Model) createAlertWithFrequency() (tea.Model, tea.Cmd) {
	alert := Alert{
		ID:            generateAlertID(),
		StockCode:     m.currentAlert.StockCode,
		StockName:     m.currentAlert.StockName,
		Type:          m.selectedAlertType,
		Condition:     m.selectedAlertCondition,
		Threshold:     m.alertThreshold,
		IsActive:      true,
		Frequency:     m.selectedAlertFrequency,
		FrequencyDays: m.alertFrequencyDays,
		CreatedAt:     time.Now(),
		TriggeredAt:   time.Time{},
		LastChecked:   time.Time{},
		BatchTag:      m.batchAlertTag,
	}

	m.alertData.Alerts = append(m.alertData.Alerts, alert)
	m.saveAlertData()

	m.message = fmt.Sprintf(m.getText("alert.createSuccess"), alert.StockName)

	// Return to source state
	if m.previousState == consts.StockAlertManage {
		m.stockAlertAlerts = m.getStockAlerts(m.stockAlertCode)
		m.state = consts.StockAlertManage
	} else {
		m.state = consts.AlertManage
	}

	// Reset state
	m.alertManageStep = 0
	m.tagSelectCursor = 0
	m.alertInput = ""
	m.batchAlertTag = ""
	m.selectedAlertFrequency = ""
	m.alertFrequencyDays = 0
	m.alertFrequencyCursor = 0

	return m, nil
}

// viewAlertAdd renders add alert interface
func (m *Model) viewAlertAdd() string {
	return alertui.RenderAlertAdd(alertui.AlertAddViewParams{
		GetText:              m.getText,
		StockCode:            m.currentAlert.StockCode,
		StockName:            m.currentAlert.StockName,
		Step:                 m.alertManageStep,
		TagSelectCursor:      m.tagSelectCursor,
		SelectedAlertType:    types.AlertType(m.selectedAlertType),
		SelectedCondition:    m.selectedAlertCondition,
		AlertThreshold:       m.alertThreshold,
		AlertInput:           m.alertInput,
		AlertFrequencyCursor: m.alertFrequencyCursor,
		Message:              m.message,
	})
}

// ============================================================================
// consts.AlertEdit State Handler
// ============================================================================

// handleAlertEdit handles edit alert state
func (m *Model) handleAlertEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.alertManageStep {
	case 0: // Select alert type
		return m.handleAlertEditTypeSelect(msg)
	case 1: // Select condition
		return m.handleAlertEditConditionSelect(msg)
	case 2: // Input threshold
		return m.handleAlertEditThresholdInput(msg)
	case 3: // Select trigger frequency
		return m.handleAlertEditFrequencySelect(msg)
	case 4: // Input custom days (for EveryNDays)
		return m.handleAlertEditFrequencyDaysInput(msg)
	default:
		return m, nil
	}
}

// handleAlertEditTypeSelect handles edit alert type selection
func (m *Model) handleAlertEditTypeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return
		if m.previousState == consts.StockAlertManage {
			m.state = consts.StockAlertManage
		} else {
			m.state = consts.AlertManage
		}
		m.message = ""
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 2 { // 3 types
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		m.selectedAlertType = GetAlertTypeFromCursor(m.tagSelectCursor)
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertEditConditionSelect handles edit condition selection
func (m *Model) handleAlertEditConditionSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 3 { // 4 conditions
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		conditions := []string{">", "<", ">=", "<="}
		m.selectedAlertCondition = conditions[m.tagSelectCursor]

		m.alertManageStep = 2
		m.alertInput = fmt.Sprintf("%.2f", m.currentAlert.Threshold)
		m.alertInputCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertEditThresholdInput handles edit threshold input
func (m *Model) handleAlertEditThresholdInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	case "enter":
		var threshold float64
		_, err := fmt.Sscanf(m.alertInput, "%f", &threshold)
		if err != nil || m.alertInput == "" {
			m.message = m.getText("alert.invalidThreshold")
			return m, nil
		}

		m.alertThreshold = threshold
		// Move to frequency selection instead of saving immediately
		m.alertManageStep = 3
		// m.alertFrequencyCursor is already initialized at entry
		return m, nil

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			char := msg.String()
			if (char >= "0" && char <= "9") || char == "." || char == "-" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// handleAlertEditFrequencySelect handles edit alert frequency selection
func (m *Model) handleAlertEditFrequencySelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	frequencyOptions := getFrequencyOptions()

	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 2
		m.alertInput = fmt.Sprintf("%.2f", m.alertThreshold)
		return m, nil

	case "up", "k", "w":
		if m.alertFrequencyCursor > 0 {
			m.alertFrequencyCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.alertFrequencyCursor < len(frequencyOptions)-1 {
			m.alertFrequencyCursor++
		}
		return m, nil

	case "enter", " ":
		m.selectedAlertFrequency = frequencyOptions[m.alertFrequencyCursor]

		if m.selectedAlertFrequency == TriggerEveryNDays {
			m.alertManageStep = 4
			// Pre-fill with existing custom days if available
			if m.alertFrequencyDays > 0 {
				m.alertInput = fmt.Sprintf("%d", m.alertFrequencyDays)
			} else {
				m.alertInput = ""
			}
			m.alertInputCursor = 0
		} else {
			return m.saveEditedAlert()
		}
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertEditFrequencyDaysInput handles edit alert custom days input
func (m *Model) handleAlertEditFrequencyDaysInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 3
		m.alertFrequencyCursor = 4
		return m, nil

	case "enter":
		var days int
		_, err := fmt.Sscanf(m.alertInput, "%d", &days)
		if err != nil || days <= 0 {
			m.message = m.getText("alert.frequency.invalidDays")
			return m, nil
		}
		m.alertFrequencyDays = days
		return m.saveEditedAlert()

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			char := msg.String()
			if char >= "0" && char <= "9" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// saveEditedAlert saves the edited alert with all updated fields
func (m *Model) saveEditedAlert() (tea.Model, tea.Cmd) {
	// Determine if frequency changed from once to periodic
	wasOnceOrEmpty := m.currentAlert.Frequency == TriggerOnce || m.currentAlert.Frequency == ""
	nowPeriodic := m.selectedAlertFrequency != TriggerOnce && m.selectedAlertFrequency != ""

	// Update alert
	for i, alert := range m.alertData.Alerts {
		if alert.ID == m.currentAlert.ID {
			m.alertData.Alerts[i].Type = m.selectedAlertType
			m.alertData.Alerts[i].Condition = m.selectedAlertCondition
			m.alertData.Alerts[i].Threshold = m.alertThreshold
			// Update frequency fields
			m.alertData.Alerts[i].Frequency = m.selectedAlertFrequency
			m.alertData.Alerts[i].FrequencyDays = m.alertFrequencyDays
			// Update timestamp
			m.alertData.Alerts[i].UpdatedAt = time.Now()

			// Handle triggered once-alert converted to periodic
			if wasOnceOrEmpty && nowPeriodic && !m.alertData.Alerts[i].TriggeredAt.IsZero() {
				// Reset TriggeredAt to allow immediate re-trigger
				m.alertData.Alerts[i].TriggeredAt = time.Time{}
				// Also re-enable the alert if it was disabled
				m.alertData.Alerts[i].IsActive = true
			}
			break
		}
	}

	m.saveAlertData()
	m.message = m.getText("alert.editSuccess")

	// Return to source state
	if m.previousState == consts.StockAlertManage {
		m.stockAlertAlerts = m.getStockAlerts(m.stockAlertCode)
		m.state = consts.StockAlertManage
	} else {
		m.state = consts.AlertManage
	}

	// Reset state
	m.alertManageStep = 0
	m.tagSelectCursor = 0
	m.alertInput = ""
	m.selectedAlertFrequency = ""
	m.alertFrequencyDays = 0
	m.alertFrequencyCursor = 0

	return m, nil
}

// viewAlertEdit renders edit alert interface
func (m *Model) viewAlertEdit() string {
	return alertui.RenderAlertAdd(alertui.AlertAddViewParams{
		GetText:              m.getText,
		StockCode:            m.currentAlert.StockCode,
		StockName:            m.currentAlert.StockName,
		Step:                 m.alertManageStep,
		TagSelectCursor:      m.tagSelectCursor,
		SelectedAlertType:    types.AlertType(m.selectedAlertType),
		SelectedCondition:    m.selectedAlertCondition,
		AlertThreshold:       m.alertThreshold,
		AlertInput:           m.alertInput,
		AlertFrequencyCursor: m.alertFrequencyCursor,
		Message:              m.message,
	})
}

// ============================================================================
// Batch Alert State Handlers
// ============================================================================

// handleAlertBatchMethodSelect handles batch add method selection
func (m *Model) handleAlertBatchMethodSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // Return to alert management
		m.state = consts.AlertManage
		m.message = ""
		return m, nil

	case "up", "k", "w": // Scroll up
		if m.batchSelectStep > 0 {
			m.batchSelectStep--
		}
		return m, nil

	case "down", "j", "s": // Scroll down
		if m.batchSelectStep < 2 {
			m.batchSelectStep++
		}
		return m, nil

	case "enter", " ":
		switch m.batchSelectStep {
		case 0: // Batch by tag
			m.state = consts.AlertBatchByTag
			m.batchAlertTag = ""
			m.tagManageCursor = 0

		case 1: // Batch by market
			m.state = consts.AlertBatchByMarket
			m.batchSelectedMarket = ""
			m.marketCursor = 0

		case 2: // Batch by stock list
			m.state = consts.SelectBatchStocks
			m.batchSelectStep = 0
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewAlertBatchMethodSelect renders batch method selection interface
func (m *Model) viewAlertBatchMethodSelect() string {
	return alertui.RenderBatchMethodSelect(alertui.BatchMethodSelectViewParams{
		GetText: m.getText,
		Cursor:  m.batchSelectStep,
		Message: m.message,
	})
}

// handleAlertBatchByTag handles batch add by tag
func (m *Model) handleAlertBatchByTag(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // Return to method selection
		m.state = consts.AlertBatchMethodSelect
		m.batchAlertTag = ""
		return m, nil

	case "up", "k", "w": // Scroll up
		if m.tagManageCursor > 0 {
			m.tagManageCursor--
		}
		return m, nil

	case "down", "j", "s": // Scroll down
		if m.tagManageCursor < len(m.availableTags)-1 {
			m.tagManageCursor++
		}
		return m, nil

	case "enter", " ": // Confirm selection
		if m.tagManageCursor < len(m.availableTags) {
			m.batchAlertTag = m.availableTags[m.tagManageCursor]

			m.batchSelectedStocks = m.getStocksByTag(m.batchAlertTag)

			if len(m.batchSelectedStocks) == 0 {
				m.message = m.getText("alertBatchEmptyTag")
				return m, nil
			}

			m.state = consts.AlertBatchAdd
			m.alertManageStep = 0
			m.tagSelectCursor = 0
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewAlertBatchByTag renders batch add by tag interface
func (m *Model) viewAlertBatchByTag() string {
	return alertui.RenderBatchByTag(alertui.BatchByTagViewParams{
		GetText:       m.getText,
		AvailableTags: m.availableTags,
		Cursor:        m.tagManageCursor,
		Message:       m.message,
		GetStockCount: func(tag string) int {
			return len(m.getStocksByTag(tag))
		},
	})
}

// handleAlertBatchByMarket handles batch add by market
func (m *Model) handleAlertBatchByMarket(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // Return to method selection
		m.state = consts.AlertBatchMethodSelect
		m.batchSelectedMarket = ""
		return m, nil

	case "up", "k", "w": // Scroll up
		if m.marketCursor > 0 {
			m.marketCursor--
		}
		return m, nil

	case "down", "j", "s": // Scroll down
		if m.marketCursor < 2 {
			m.marketCursor++
		}
		return m, nil

	case "enter", " ": // Confirm selection
		var marketType MarketType
		switch m.marketCursor {
		case 0:
			marketType = MarketChina
		case 1:
			marketType = MarketUS
		case 2:
			marketType = MarketHongKong
		}

		m.batchSelectedStocks = m.getStocksByMarket(marketType)

		if len(m.batchSelectedStocks) == 0 {
			m.message = m.getText("alertBatchEmptyMarket")
			return m, nil
		}

		m.state = consts.AlertBatchAdd
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// viewAlertBatchByMarket renders batch add by market interface
func (m *Model) viewAlertBatchByMarket() string {
	return alertui.RenderBatchByMarket(alertui.BatchByMarketViewParams{
		GetText: m.getText,
		Cursor:  m.marketCursor,
		Message: m.message,
		GetStockCount: func(marketType MarketType) int {
			return len(m.getStocksByMarket(marketType))
		},
	})
}

// handleSelectBatchStocks handles batch stock source selection
func (m *Model) handleSelectBatchStocks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // Return
		if m.state == consts.SelectBatchStocks && m.previousState != consts.AlertManage {
			m.state = consts.AlertBatchMethodSelect
		} else {
			m.state = consts.AlertManage
		}
		m.message = ""
		m.batchSelectedStocks = nil
		m.batchCodeInput = ""
		return m, nil

	case "up", "k", "w": // Scroll up
		if m.batchSelectStep > 0 {
			m.batchSelectStep--
		}
		return m, nil

	case "down", "j", "s": // Scroll down
		if m.batchSelectStep < 2 {
			m.batchSelectStep++
		}
		return m, nil

	case "enter", " ":
		switch m.batchSelectStep {
		case 0: // From watchlist
			m.state = consts.SelectBatchFromWatchlist
			m.batchStockSource = BatchSourceWatchlist
			m.stockSelectionMap = make(map[string]bool)
			m.watchlistCursor = 0

		case 1: // From portfolio
			m.state = consts.SelectBatchFromPortfolio
			m.batchStockSource = BatchSourcePortfolio
			m.stockSelectionMap = make(map[string]bool)
			m.portfolioCursor = 0

		case 2: // Manual input
			m.state = consts.InputBatchCodes
			m.batchStockSource = BatchSourceManual
			m.batchCodeInput = ""
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewSelectBatchStocks renders batch stock source selection interface
func (m *Model) viewSelectBatchStocks() string {
	return alertui.RenderSelectBatchStocks(alertui.SelectBatchStocksViewParams{
		GetText: m.getText,
		Cursor:  m.batchSelectStep,
		Message: m.message,
	})
}

// handleSelectBatchFromWatchlist handles batch selection from watchlist
func (m *Model) handleSelectBatchFromWatchlist(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleBatchStockSelection(msg, true)
}

// handleSelectBatchFromPortfolio handles batch selection from portfolio
func (m *Model) handleSelectBatchFromPortfolio(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleBatchStockSelection(msg, false)
}

// handleBatchStockSelection handles batch stock selection common logic
func (m *Model) handleBatchStockSelection(msg tea.KeyMsg, isWatchlist bool) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // Return to source selection
		m.state = consts.SelectBatchStocks
		m.stockSelectionMap = make(map[string]bool)
		return m, nil

	case "enter": // Confirm selection
		m.batchSelectedStocks = []string{}
		for code, selected := range m.stockSelectionMap {
			if selected {
				m.batchSelectedStocks = append(m.batchSelectedStocks, code)
			}
		}

		if len(m.batchSelectedStocks) == 0 {
			m.message = m.getText("alertBatchNoSelection")
			return m, nil
		}

		m.state = consts.AlertBatchAdd
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case " ": // Toggle selection
		if isWatchlist {
			filteredStocks := m.getFilteredWatchlist()
			if m.watchlistCursor < len(filteredStocks) {
				stock := filteredStocks[m.watchlistCursor]
				code := stock.Code
				m.stockSelectionMap[code] = !m.stockSelectionMap[code]
			}
		} else {
			if m.portfolioCursor < len(m.portfolio.Stocks) {
				stock := m.portfolio.Stocks[m.portfolioCursor]
				code := stock.Code
				m.stockSelectionMap[code] = !m.stockSelectionMap[code]
			}
		}
		return m, nil

	case "up", "k", "w": // Navigate up
		if isWatchlist {
			if m.watchlistCursor > 0 {
				m.watchlistCursor--
			}
		} else {
			if m.portfolioCursor > 0 {
				m.portfolioCursor--
			}
		}
		return m, nil

	case "down", "j", "s": // Navigate down
		if isWatchlist {
			filteredStocks := m.getFilteredWatchlist()
			if m.watchlistCursor < len(filteredStocks)-1 {
				m.watchlistCursor++
			}
		} else {
			if m.portfolioCursor < len(m.portfolio.Stocks)-1 {
				m.portfolioCursor++
			}
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewSelectBatchFromWatchlist renders watchlist batch selection interface
func (m *Model) viewSelectBatchFromWatchlist() string {
	filteredStocks := m.getFilteredWatchlist()
	stocks := make([]ui.SelectableStock, len(filteredStocks))
	for i, s := range filteredStocks {
		stocks[i] = ui.SelectableStock{
			Code: s.Code,
			Name: s.Name,
		}
	}

	return alertui.RenderBatchStockList(alertui.BatchStockListViewParams{
		GetText:         m.getText,
		Config:          m.config,
		Stocks:          stocks,
		Cursor:          m.watchlistCursor,
		SelectionMap:    m.stockSelectionMap,
		Message:         m.message,
		IsFromWatchlist: true,
	})
}

// viewSelectBatchFromPortfolio renders portfolio batch selection interface
func (m *Model) viewSelectBatchFromPortfolio() string {
	stocks := make([]ui.SelectableStock, len(m.portfolio.Stocks))
	for i, s := range m.portfolio.Stocks {
		stocks[i] = ui.SelectableStock{
			Code:      s.Code,
			Name:      s.Name,
			Quantity:  s.Quantity,
			CostPrice: s.CostPrice,
		}
	}

	return alertui.RenderBatchStockList(alertui.BatchStockListViewParams{
		GetText:         m.getText,
		Config:          m.config,
		Stocks:          stocks,
		Cursor:          m.portfolioCursor,
		SelectionMap:    m.stockSelectionMap,
		Message:         m.message,
		IsFromWatchlist: false,
	})
}

// handleInputBatchCodes handles manual stock code input
func (m *Model) handleInputBatchCodes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // Return to source selection
		m.state = consts.SelectBatchStocks
		m.batchCodeInput = ""
		return m, nil

	case "enter": // Confirm input
		codes := parseStockCodes(m.batchCodeInput)
		if len(codes) == 0 {
			m.message = m.getText("alertBatchInvalidCodes")
			return m, nil
		}

		m.batchSelectedStocks = codes
		m.state = consts.AlertBatchAdd
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case "backspace":
		if len(m.batchCodeInput) > 0 {
			m.batchCodeInput = m.batchCodeInput[:len(m.batchCodeInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.batchCodeInput += msg.String()
		}
		return m, nil
	}
}

// viewInputBatchCodes renders manual code input interface
func (m *Model) viewInputBatchCodes() string {
	return alertui.RenderInputBatchCodes(alertui.InputBatchCodesViewParams{
		GetText: m.getText,
		Input:   m.batchCodeInput,
		Message: m.message,
	})
}

// handleAlertBatchAdd handles batch add alert (set rules)
func (m *Model) handleAlertBatchAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.alertManageStep {
	case 0: // Select alert type
		return m.handleBatchAlertTypeSelect(msg)
	case 1: // Select condition
		return m.handleBatchAlertConditionSelect(msg)
	case 2: // Input threshold
		return m.handleBatchAlertThresholdInput(msg)
	case 3: // Select trigger frequency
		return m.handleBatchAlertFrequencySelect(msg)
	case 4: // Input custom days
		return m.handleBatchAlertFrequencyDaysInput(msg)
	default:
		return m, nil
	}
}

// handleBatchAlertTypeSelect handles batch alert type selection
func (m *Model) handleBatchAlertTypeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return
		m.state = consts.AlertBatchMethodSelect
		m.message = ""
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 2 {
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		m.selectedAlertType = GetAlertTypeFromCursor(m.tagSelectCursor)
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleBatchAlertConditionSelect handles batch alert condition selection
func (m *Model) handleBatchAlertConditionSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 3 {
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		m.selectedAlertCondition = GetAlertConditionFromCursor(m.tagSelectCursor)
		m.alertManageStep = 2
		m.alertInput = ""
		m.alertInputCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleBatchAlertThresholdInput handles batch alert threshold input
func (m *Model) handleBatchAlertThresholdInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // Return to previous step
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	case "enter":
		var threshold float64
		_, err := fmt.Sscanf(m.alertInput, "%f", &threshold)
		if err != nil || m.alertInput == "" {
			m.message = m.getText("alert.invalidThreshold")
			return m, nil
		}

		m.alertThreshold = threshold
		m.alertManageStep = 3
		m.alertFrequencyCursor = 0
		m.selectedAlertFrequency = ""
		return m, nil

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			char := msg.String()
			if (char >= "0" && char <= "9") || char == "." || char == "-" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// viewAlertBatchAdd renders batch add alert interface
func (m *Model) viewAlertBatchAdd() string {
	return alertui.RenderBatchAdd(alertui.BatchAddViewParams{
		AlertAddViewParams: alertui.AlertAddViewParams{
			GetText:              m.getText,
			Step:                 m.alertManageStep,
			TagSelectCursor:      m.tagSelectCursor,
			SelectedAlertType:    types.AlertType(m.selectedAlertType),
			SelectedCondition:    m.selectedAlertCondition,
			AlertThreshold:       m.alertThreshold,
			AlertInput:           m.alertInput,
			AlertFrequencyCursor: m.alertFrequencyCursor,
			Message:              m.message,
		},
		SelectedStocks: m.batchSelectedStocks,
	})
}

// handleBatchAlertFrequencySelect handles batch alert frequency selection
func (m *Model) handleBatchAlertFrequencySelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	frequencyOptions := getFrequencyOptions()

	switch msg.String() {
	case "esc", "q":
		m.alertManageStep = 2
		m.alertInput = fmt.Sprintf("%.2f", m.alertThreshold)
		return m, nil

	case "up", "k", "w":
		if m.alertFrequencyCursor > 0 {
			m.alertFrequencyCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.alertFrequencyCursor < len(frequencyOptions)-1 {
			m.alertFrequencyCursor++
		}
		return m, nil

	case "enter", " ":
		m.selectedAlertFrequency = frequencyOptions[m.alertFrequencyCursor]

		if m.selectedAlertFrequency == TriggerEveryNDays {
			m.alertManageStep = 4
			m.alertInput = ""
			m.alertInputCursor = 0
		} else {
			return m.createBatchAlertsWithFrequency()
		}
		return m, nil

	default:
		return m, nil
	}
}

// handleBatchAlertFrequencyDaysInput handles batch alert custom days input
func (m *Model) handleBatchAlertFrequencyDaysInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.alertManageStep = 3
		m.alertFrequencyCursor = 4
		return m, nil

	case "enter":
		var days int
		_, err := fmt.Sscanf(m.alertInput, "%d", &days)
		if err != nil || days <= 0 {
			m.message = m.getText("alert.frequency.invalidDays")
			return m, nil
		}
		m.alertFrequencyDays = days
		return m.createBatchAlertsWithFrequency()

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			char := msg.String()
			if char >= "0" && char <= "9" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// createBatchAlertsWithFrequency creates batch alerts with selected frequency
func (m *Model) createBatchAlertsWithFrequency() (tea.Model, tea.Cmd) {
	addedCount := 0
	for _, code := range m.batchSelectedStocks {
		stockName := code

		m.stockPriceMutex.RLock()
		if cacheEntry, exists := m.stockPriceCache[code]; exists && cacheEntry.Data != nil {
			stockName = cacheEntry.Data.Name
		}
		m.stockPriceMutex.RUnlock()

		alert := Alert{
			ID:            generateAlertID(),
			StockCode:     code,
			StockName:     stockName,
			Type:          m.selectedAlertType,
			Condition:     m.selectedAlertCondition,
			Threshold:     m.alertThreshold,
			IsActive:      true,
			Frequency:     m.selectedAlertFrequency,
			FrequencyDays: m.alertFrequencyDays,
			CreatedAt:     time.Now(),
			TriggeredAt:   time.Time{},
			LastChecked:   time.Time{},
			BatchTag:      m.batchAlertTag,
		}

		m.alertData.Alerts = append(m.alertData.Alerts, alert)
		addedCount++
	}

	m.saveAlertData()
	m.message = fmt.Sprintf(m.getText("alert.batch.success"), addedCount)

	m.state = consts.AlertManage
	m.alertManageStep = 0
	m.tagSelectCursor = 0
	m.alertInput = ""
	m.batchAlertTag = ""
	m.batchSelectedStocks = nil
	m.stockSelectionMap = make(map[string]bool)
	m.selectedAlertFrequency = ""
	m.alertFrequencyDays = 0
	m.alertFrequencyCursor = 0

	return m, nil
}
