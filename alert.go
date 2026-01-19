package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ============================================================================
// UUID Generation
// ============================================================================

// generateAlertID 生成 UUID v4 告警 ID
func generateAlertID() string {
	return uuid.New().String()
}

// ============================================================================
// Alert Checking Logic
// ============================================================================

// checkAlertsMsg 检查告警消息
type checkAlertsMsg struct{}

// handleCheckAlerts 检查所有告警条件
func (m *Model) handleCheckAlerts(msg checkAlertsMsg) (tea.Model, tea.Cmd) {
	triggeredAlerts := []Alert{}

	for i, alert := range m.alertData.Alerts {
		if !alert.IsActive {
			continue
		}

		// 检查是否到达触发周期
		if !canTriggerInCurrentPeriod(alert) {
			continue
		}

		// 获取当前股票价格
		m.stockPriceMutex.RLock()
		cacheEntry, exists := m.stockPriceCache[alert.StockCode]
		m.stockPriceMutex.RUnlock()

		if !exists || cacheEntry.Data == nil {
			continue
		}

		stockData := cacheEntry.Data

		// 检查告警条件
		isTriggered := false
		switch alert.Type {
		case AlertTypePrice:
			isTriggered = checkPriceAlert(stockData, alert)
		case AlertTypeRate:
			isTriggered = checkRateAlert(stockData, alert)
		case AlertTypeVolume:
			isTriggered = checkVolumeAlert(stockData, alert)
		}

		if isTriggered {
			// 更新触发时间
			alert.TriggeredAt = time.Now()

			// 仅一次性告警禁用，周期性告警保持激活
			if alert.Frequency == TriggerOnce || alert.Frequency == "" {
				alert.IsActive = false
			}
			// 周期性告警保持 IsActive = true

			triggeredAlerts = append(triggeredAlerts, alert)
			m.alertData.Alerts[i] = alert

			logInfo("log.alert.triggered", alert.StockName, alert.StockCode)
		}

		// 更新最后检查时间
		alert.LastChecked = time.Now()
		m.alertData.Alerts[i] = alert
	}

	// 保存告警配置
	if len(triggeredAlerts) > 0 {
		m.alertData.LastCheck = time.Now().Format("2006-01-02T15:04:05Z07:00")
		m.saveAlertData()
	}

	// 发送通知
	for _, alert := range triggeredAlerts {
		m.sendAlertNotification(alert)
	}

	return m, nil
}

// checkPriceAlert 检查价格告警
func checkPriceAlert(stockData *StockData, alert Alert) bool {
	return CheckNumericCondition(stockData.Price, alert.Threshold, alert.Condition)
}

// checkRateAlert 检查涨跌幅告警
func checkRateAlert(stockData *StockData, alert Alert) bool {
	return CheckNumericCondition(stockData.ChangePercent, alert.Threshold, alert.Condition)
}

// checkVolumeAlert 检查成交量告警
func checkVolumeAlert(stockData *StockData, alert Alert) bool {
	volumeFloat := float64(stockData.Volume)
	return CheckNumericCondition(volumeFloat, alert.Threshold, alert.Condition)
}

// ============================================================================
// Notification System
// ============================================================================

// sendAlertNotification 发送告警通知
func (m *Model) sendAlertNotification(alert Alert) {
	var title, message string

	switch alert.Type {
	case AlertTypePrice:
		title = m.getText("alertNotificationPriceTitle")
		message = fmt.Sprintf("%s (%s) %s %s %.2f",
			alert.StockName, alert.StockCode,
			m.getText("alertNotificationPrice"),
			alert.Condition, alert.Threshold)
	case AlertTypeRate:
		title = m.getText("alertNotificationRateTitle")
		message = fmt.Sprintf("%s (%s) %s %s %.2f%%",
			alert.StockName, alert.StockCode,
			m.getText("alertNotificationRate"),
			alert.Condition, alert.Threshold)
	case AlertTypeVolume:
		title = m.getText("alertNotificationVolumeTitle")
		message = fmt.Sprintf("%s (%s) %s %s %.0f",
			alert.StockName, alert.StockCode,
			m.getText("alertNotificationVolume"),
			alert.Condition, alert.Threshold)
	}

	// 根据平台选择通知方式
	switch runtime.GOOS {
	case "darwin": // macOS
		m.sendMacOSNotification(title, message)
	case "linux": // Linux
		m.sendLinuxNotification(title, message)
	case "windows": // Windows
		m.sendWindowsNotification(title, message)
	default:
		logWarn("log.alert.unsupportedPlatform", runtime.GOOS)
	}
}

// sendMacOSNotification 发送 macOS 通知 (terminal-notifier)
func (m *Model) sendMacOSNotification(title, message string) {
	// 从系统 PATH 查找 terminal-notifier (支持 Homebrew 等多种安装方式)
	binaryPath, err := exec.LookPath("terminal-notifier")
	if err != nil {
		logWarn("log.alert.terminalNotifierNotFound", "")
		return
	}

	cmd := exec.Command(
		binaryPath,
		"-title", title,
		"-message", message,
		"-sound", "default",
	)

	if err := cmd.Run(); err != nil {
		logError("log.alert.notificationFailed", err)
	} else {
		logInfo("log.alert.notificationSent", title)
	}
}

// sendLinuxNotification 发送 Linux 通知 (notify-send)
func (m *Model) sendLinuxNotification(title, message string) {
	cmd := exec.Command("notify-send", title, message)
	if err := cmd.Run(); err != nil {
		logWarn("log.alert.notifySendNotFound", "")
	} else {
		logInfo("log.alert.notificationSent", title)
	}
}

// sendWindowsNotification 发送 Windows 通知 (PowerShell)
func (m *Model) sendWindowsNotification(title, message string) {
	psScript := fmt.Sprintf(
		`[System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms"); [System.Windows.Forms.MessageBox]::Show("%s", "%s")`,
		message, title,
	)
	cmd := exec.Command("powershell", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		logWarn("log.alert.powershellNotFound", "")
	} else {
		logInfo("log.alert.notificationSent", title)
	}
}

// ============================================================================
// AlertManage State Handler
// ============================================================================

// handleAlertManage 处理告警管理状态
func (m *Model) handleAlertManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // 返回主菜单
		m.state = MainMenu
		m.message = ""
		return m, nil

	case "a": // 添加告警
		m.state = SelectBatchStocks
		m.batchSelectStep = 0
		m.batchSelectedStocks = nil
		m.stockSelectionMap = make(map[string]bool)
		return m, nil

	case "g": // 批量添加
		m.state = AlertBatchMethodSelect
		m.batchSelectStep = 0
		return m, nil

	case "e": // 编辑告警
		if len(m.alertData.Alerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.alertCursor >= len(m.alertData.Alerts) {
			return m, nil
		}

		// 保存当前选中的告警
		m.currentAlert = m.alertData.Alerts[m.alertCursor]
		m.selectedAlertType = m.currentAlert.Type
		m.selectedAlertCondition = m.currentAlert.Condition
		m.alertThreshold = m.currentAlert.Threshold

		m.state = AlertEdit
		m.alertManageStep = 0 // 从类型选择开始
		return m, nil

	case "d": // 删除告警
		if len(m.alertData.Alerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.alertCursor >= len(m.alertData.Alerts) {
			return m, nil
		}

		// 立即删除
		deletedAlert := m.alertData.Alerts[m.alertCursor]
		m.alertData.Alerts = append(
			m.alertData.Alerts[:m.alertCursor],
			m.alertData.Alerts[m.alertCursor+1:]...,
		)
		m.saveAlertData()

		// 调整光标位置
		if m.alertCursor >= len(m.alertData.Alerts) && m.alertCursor > 0 {
			m.alertCursor--
		}

		m.message = fmt.Sprintf(m.getText("alert.deleteSuccess"), deletedAlert.StockName)
		return m, nil

	case " ": // 切换告警启用/禁用状态
		if len(m.alertData.Alerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.alertCursor >= len(m.alertData.Alerts) {
			return m, nil
		}

		// 切换状态
		m.alertData.Alerts[m.alertCursor].IsActive = !m.alertData.Alerts[m.alertCursor].IsActive
		m.saveAlertData()

		// 显示切换结果
		alert := m.alertData.Alerts[m.alertCursor]
		if alert.IsActive {
			m.message = fmt.Sprintf(m.getText("alert.toggle.enabled"), alert.StockName)
		} else {
			m.message = fmt.Sprintf(m.getText("alert.toggle.disabled"), alert.StockName)
		}
		return m, nil

	case "enter": // 查看告警详情
		if len(m.alertData.Alerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.alertCursor >= len(m.alertData.Alerts) {
			return m, nil
		}

		alert := m.alertData.Alerts[m.alertCursor]
		typeText := ""
		switch alert.Type {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		details := fmt.Sprintf("%s: %s %s %s %.2f | %s: %s",
			m.getText("alertDetailStock"), alert.StockName, typeText, alert.Condition, alert.Threshold,
			m.getText("alertDetailCreated"), alert.CreatedAt.Format("2006-01-02 15:04"))

		m.message = details
		return m, nil

	case "up", "k", "w": // 向上滚动
		if m.alertCursor > 0 {
			m.alertCursor--
		}
		return m, nil

	case "down", "j", "s": // 向下滚动
		if m.alertCursor < len(m.alertData.Alerts)-1 {
			m.alertCursor++
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewAlertManage 渲染告警管理界面
func (m *Model) viewAlertManage() string {
	s := m.getText("alertTitle") + "\n\n"

	// 空列表提示
	if len(m.alertData.Alerts) == 0 {
		s += m.getText("alert.empty") + "\n\n"
		s += m.getText("alert.addFirst") + "\n\n"
		s += m.getText("alertHelp") + "\n"
		return s
	}

	// 创建表格
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// 表头
	t.AppendHeader(table.Row{
		m.getText("alertHeaderCode"),
		m.getText("alertHeaderName"),
		m.getText("alertHeaderType"),
		m.getText("alertHeaderCondition"),
		m.getText("alertHeaderThreshold"),
		m.getText("alertHeaderSwitch"),
		m.getText("alertHeaderFrequency"),
		m.getText("alertHeaderTriggeredAt"),
		m.getText("alertHeaderCreated"),
	})

	// 计算显示范围(分页)
	maxLines := m.config.Display.MaxLines
	startIndex := m.alertCursor
	if startIndex > len(m.alertData.Alerts)-maxLines {
		startIndex = len(m.alertData.Alerts) - maxLines
		if startIndex < 0 {
			startIndex = 0
		}
	}
	endIndex := startIndex + maxLines
	if endIndex > len(m.alertData.Alerts) {
		endIndex = len(m.alertData.Alerts)
	}

	// 数据行
	for i := startIndex; i < endIndex; i++ {
		alert := m.alertData.Alerts[i]

		// 开关状态（仅图标）
		var switchText string
		if alert.IsActive {
			switchText = "✓"
		} else {
			switchText = "✗"
		}

		// 触发时间（具体时间或 -）
		var triggeredText string
		if alert.TriggeredAt.IsZero() {
			triggeredText = "-"
		} else {
			triggeredText = alert.TriggeredAt.Format("2006-01-02 15:04:05")
		}

		// 触发频率
		frequencyText := m.getFrequencyDisplayText(alert.Frequency, alert.FrequencyDays)

		// 类型文本
		typeText := ""
		switch alert.Type {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		// 游标标记
		cursor := "  "
		if i == m.alertCursor {
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

	// 分页信息
	total := len(m.alertData.Alerts)
	currentPos := m.alertCursor + 1
	s += fmt.Sprintf("📊 %s (%d/%d)\n\n", m.getText("alertListTitle"), currentPos, total)

	// 帮助信息
	s += m.getText("alertHelp") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ============================================================================
// Helper Functions
// ============================================================================

// getStockAlerts 获取指定股票的所有告警
func (m *Model) getStockAlerts(stockCode string) []Alert {
	var alerts []Alert
	for _, alert := range m.alertData.Alerts {
		if alert.StockCode == stockCode {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// getStocksByTag 获取指定标签下的所有股票代码
func (m *Model) getStocksByTag(tag string) []string {
	codes := []string{}
	seen := make(map[string]bool)

	// 从自选列表中获取
	for _, stock := range m.watchlist.Stocks {
		for _, stockTag := range stock.Tags {
			if stockTag == tag {
				if !seen[stock.Code] {
					codes = append(codes, stock.Code)
					seen[stock.Code] = true
				}
				break
			}
		}
	}

	return codes
}

// getStocksByMarket 获取指定市场下的所有股票代码
func (m *Model) getStocksByMarket(marketType MarketType) []string {
	codes := []string{}
	seen := make(map[string]bool)

	// 从自选列表中获取
	for _, stock := range m.watchlist.Stocks {
		if stock.Market == marketType {
			if !seen[stock.Code] {
				codes = append(codes, stock.Code)
				seen[stock.Code] = true
			}
		}
	}

	// 从持股列表中获取
	for _, stock := range m.portfolio.Stocks {
		// 根据股票代码判断市场
		if isMarketType(stock.Code, marketType) {
			if !seen[stock.Code] {
				codes = append(codes, stock.Code)
				seen[stock.Code] = true
			}
		}
	}

	return codes
}

// isMarketType 判断股票代码是否属于指定市场
func isMarketType(stockCode string, marketType MarketType) bool {
	switch marketType {
	case MarketChina:
		return strings.HasPrefix(stockCode, "SH") || strings.HasPrefix(stockCode, "SZ")
	case MarketUS:
		return !strings.HasPrefix(stockCode, "SH") && !strings.HasPrefix(stockCode, "SZ") && !strings.HasPrefix(stockCode, "HK")
	case MarketHongKong:
		return strings.HasPrefix(stockCode, "HK")
	default:
		return false
	}
}

// parseStockCodes 解析股票代码（支持逗号和换行分隔）
func parseStockCodes(input string) []string {
	codes := []string{}

	// 先按换行分割,再按逗号分割
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ",")
		for _, part := range parts {
			code := strings.TrimSpace(part)
			if code != "" {
				codes = append(codes, code)
			}
		}
	}

	return codes
}

// ============================================================================
// StockAlertManage State Handler
// ============================================================================

// handleStockAlertManage 处理股票告警详情状态
func (m *Model) handleStockAlertManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // 返回来源列表
		m.state = m.previousState
		m.message = ""
		m.stockAlertAlerts = nil // 清空缓存
		return m, nil

	case "a": // 添加告警
		m.state = AlertAdd
		m.alertManageStep = 0
		m.alertInput = ""
		m.alertInputCursor = 0
		// 预先填入当前股票信息
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

	case "e": // 编辑告警
		if len(m.stockAlertAlerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.stockAlertCursor >= len(m.stockAlertAlerts) {
			return m, nil
		}

		// 保存当前选中的告警
		m.currentAlert = m.stockAlertAlerts[m.stockAlertCursor]
		m.selectedAlertType = m.currentAlert.Type
		m.selectedAlertCondition = m.currentAlert.Condition
		m.alertThreshold = m.currentAlert.Threshold

		m.state = AlertEdit
		m.alertManageStep = 0
		return m, nil

	case "d": // 删除告警
		if len(m.stockAlertAlerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.stockAlertCursor >= len(m.stockAlertAlerts) {
			return m, nil
		}

		// 立即删除
		deletedAlert := m.stockAlertAlerts[m.stockAlertCursor]

		// 从全局告警列表中删除
		for i, alert := range m.alertData.Alerts {
			if alert.ID == deletedAlert.ID {
				m.alertData.Alerts = append(
					m.alertData.Alerts[:i],
					m.alertData.Alerts[i+1:]...,
				)
				break
			}
		}

		// 同时从股票告警列表中删除
		m.stockAlertAlerts = append(
			m.stockAlertAlerts[:m.stockAlertCursor],
			m.stockAlertAlerts[m.stockAlertCursor+1:]...,
		)

		// 调整光标位置
		if m.stockAlertCursor >= len(m.stockAlertAlerts) && m.stockAlertCursor > 0 {
			m.stockAlertCursor--
		}

		m.saveAlertData()
		m.message = fmt.Sprintf(m.getText("alert.deleteSuccess"), deletedAlert.StockName)
		return m, nil

	case " ": // 切换告警启用/禁用状态
		if len(m.stockAlertAlerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.stockAlertCursor >= len(m.stockAlertAlerts) {
			return m, nil
		}

		// 获取当前选中的告警 ID
		alertID := m.stockAlertAlerts[m.stockAlertCursor].ID

		// 在全局列表中找到并切换状态
		for i := range m.alertData.Alerts {
			if m.alertData.Alerts[i].ID == alertID {
				m.alertData.Alerts[i].IsActive = !m.alertData.Alerts[i].IsActive
				// 同步更新本地缓存
				m.stockAlertAlerts[m.stockAlertCursor].IsActive = m.alertData.Alerts[i].IsActive
				break
			}
		}
		m.saveAlertData()

		// 显示切换结果
		alert := m.stockAlertAlerts[m.stockAlertCursor]
		if alert.IsActive {
			m.message = fmt.Sprintf(m.getText("alert.toggle.enabled"), alert.StockName)
		} else {
			m.message = fmt.Sprintf(m.getText("alert.toggle.disabled"), alert.StockName)
		}
		return m, nil

	case "enter": // 查看告警详情
		if len(m.stockAlertAlerts) == 0 {
			m.message = m.getText("alert.empty")
			return m, nil
		}
		if m.stockAlertCursor >= len(m.stockAlertAlerts) {
			return m, nil
		}

		alert := m.stockAlertAlerts[m.stockAlertCursor]
		typeText := ""
		switch alert.Type {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		details := fmt.Sprintf("%s %s %.2f | %s: %s",
			typeText, alert.Condition, alert.Threshold,
			m.getText("alertDetailCreated"), alert.CreatedAt.Format("2006-01-02 15:04"))

		m.message = details
		return m, nil

	case "up", "k", "w": // 向上滚动
		if m.stockAlertCursor > 0 {
			m.stockAlertCursor--
		}
		return m, nil

	case "down", "j", "s": // 向下滚动
		if m.stockAlertCursor < len(m.stockAlertAlerts)-1 {
			m.stockAlertCursor++
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewStockAlertManage 渲染股票告警详情界面
func (m *Model) viewStockAlertManage() string {
	// 获取当前股票价格
	m.stockPriceMutex.RLock()
	cacheEntry, exists := m.stockPriceCache[m.stockAlertCode]
	m.stockPriceMutex.RUnlock()

	s := fmt.Sprintf("=== %s (%s) - %s ===\n\n",
		m.stockAlertName, m.stockAlertCode, m.getText("alertManagement"))

	// 显示当前价格信息
	if exists && cacheEntry.Data != nil {
		stockData := cacheEntry.Data
		changeStr := fmt.Sprintf("%.2f%%", stockData.ChangePercent)
		if stockData.ChangePercent >= 0 {
			changeStr = "+" + changeStr
		}
		s += fmt.Sprintf("%s: %.2f (%s) | %s: %.2f\n\n",
			m.getText("alertCurrentPrice"), stockData.Price, changeStr,
			m.getText("alertPrevClose"), stockData.PrevClose)
	} else {
		s += fmt.Sprintf("%s: -\n\n", m.getText("alertCurrentPrice"))
	}

	// 空列表提示
	if len(m.stockAlertAlerts) == 0 {
		s += m.getText("alert.stock.empty") + "\n\n"
		s += m.getText("alertStockEmptyHint1") + "\n"
		s += m.getText("alertStockEmptyHint2") + "\n\n"
		s += m.getText("alert.stockHelp") + "\n"
		return s
	}

	// 创建表格
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// 表头
	t.AppendHeader(table.Row{
		m.getText("alertHeaderCode"),
		m.getText("alertHeaderName"),
		m.getText("alertHeaderType"),
		m.getText("alertHeaderCondition"),
		m.getText("alertHeaderThreshold"),
		m.getText("alertHeaderSwitch"),
		m.getText("alertHeaderFrequency"),
		m.getText("alertHeaderTriggeredAt"),
		m.getText("alertHeaderCreated"),
	})

	// 计算显示范围
	maxLines := m.config.Display.MaxLines
	startIndex := m.stockAlertCursor
	if startIndex > len(m.stockAlertAlerts)-maxLines {
		startIndex = len(m.stockAlertAlerts) - maxLines
		if startIndex < 0 {
			startIndex = 0
		}
	}
	endIndex := startIndex + maxLines
	if endIndex > len(m.stockAlertAlerts) {
		endIndex = len(m.stockAlertAlerts)
	}

	// 数据行
	for i := startIndex; i < endIndex; i++ {
		alert := m.stockAlertAlerts[i]

		// 开关状态（仅图标）
		var switchText string
		if alert.IsActive {
			switchText = "✓"
		} else {
			switchText = "✗"
		}

		// 类型文本
		typeText := ""
		switch alert.Type {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		// 触发频率
		frequencyText := m.getFrequencyDisplayText(alert.Frequency, alert.FrequencyDays)

		// 触发时间
		triggeredTime := "-"
		if !alert.TriggeredAt.IsZero() {
			triggeredTime = alert.TriggeredAt.Format("2006-01-02 15:04:05")
		}

		// 游标标记
		cursor := "  "
		if i == m.stockAlertCursor {
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

	s += m.getText("alertStockListTitle") + "\n\n"
	s += t.Render() + "\n\n"

	// 分页信息
	total := len(m.stockAlertAlerts)
	currentPos := m.stockAlertCursor + 1
	s += fmt.Sprintf("📊 %s %s (%d/%d)\n\n",
		m.stockAlertName, m.getText("alertListTitle"), currentPos, total)

	// 帮助信息
	s += m.getText("alert.stockHelp") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ============================================================================
// AlertAdd State Handler (Type/Condition/Threshold)
// ============================================================================

// handleAlertAdd 处理添加告警状态
func (m *Model) handleAlertAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.alertManageStep {
	case 0: // 选择告警类型
		return m.handleAlertTypeSelect(msg)
	case 1: // 选择条件
		return m.handleAlertConditionSelect(msg)
	case 2: // 输入阈值
		return m.handleAlertThresholdInput(msg)
	case 3: // 选择触发频率
		return m.handleAlertFrequencySelectStep(msg)
	case 4: // 输入自定义天数
		return m.handleAlertFrequencyDaysInput(msg)
	default:
		return m, nil
	}
}

// handleAlertTypeSelect 处理告警类型选择
func (m *Model) handleAlertTypeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回
		if m.previousState == StockAlertManage {
			m.state = StockAlertManage
		} else {
			m.state = AlertManage
		}
		m.message = ""
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 2 { // 3种类型
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		// 设置告警类型
		m.selectedAlertType = GetAlertTypeFromCursor(m.tagSelectCursor)

		// 进入条件选择
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertConditionSelect 处理条件选择
func (m *Model) handleAlertConditionSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回上一步
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 3 { // 4种条件
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		// 设置条件
		m.selectedAlertCondition = GetAlertConditionFromCursor(m.tagSelectCursor)

		// 进入阈值输入
		m.alertManageStep = 2
		m.alertInput = ""
		m.alertInputCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertThresholdInput 处理阈值输入
func (m *Model) handleAlertThresholdInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回上一步
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	case "enter":
		// 验证输入
		var threshold float64
		_, err := fmt.Sscanf(m.alertInput, "%f", &threshold)
		if err != nil || m.alertInput == "" {
			m.message = m.getText("alert.invalidThreshold")
			return m, nil
		}

		// 保存阈值，进入频率选择步骤
		m.alertThreshold = threshold
		m.alertManageStep = 3 // 进入频率选择
		m.alertFrequencyCursor = 0
		m.selectedAlertFrequency = ""
		return m, nil

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		// 只接受数字、小数点和负号
		if len(msg.String()) == 1 {
			char := msg.String()
			if (char >= "0" && char <= "9") || char == "." || char == "-" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// handleAlertFrequencySelectStep 处理触发频率选择
func (m *Model) handleAlertFrequencySelectStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	frequencyOptions := getFrequencyOptions()

	switch msg.String() {
	case "esc", "q": // 返回上一步
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
			// 需要输入自定义天数
			m.alertManageStep = 4
			m.alertInput = ""
			m.alertInputCursor = 0
		} else {
			// 直接创建告警
			return m.createAlertWithFrequency()
		}
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertFrequencyDaysInput 处理自定义天数输入
func (m *Model) handleAlertFrequencyDaysInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回上一步
		m.alertManageStep = 3
		m.alertFrequencyCursor = 4 // 定位到"自定义天数"选项
		return m, nil

	case "enter":
		// 验证输入
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
		// 只接受数字
		if len(msg.String()) == 1 {
			char := msg.String()
			if char >= "0" && char <= "9" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// createAlertWithFrequency 使用选定的频率创建告警
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

	// 返回来源状态
	if m.previousState == StockAlertManage {
		m.stockAlertAlerts = m.getStockAlerts(m.stockAlertCode)
		m.state = StockAlertManage
	} else {
		m.state = AlertManage
	}

	// 重置状态
	m.alertManageStep = 0
	m.tagSelectCursor = 0
	m.alertInput = ""
	m.batchAlertTag = ""
	m.selectedAlertFrequency = ""
	m.alertFrequencyDays = 0
	m.alertFrequencyCursor = 0

	return m, nil
}

// viewAlertAdd 渲染添加告警界面
func (m *Model) viewAlertAdd() string {
	s := "=== " + m.getText("alertAddTitle") + " ===\n\n"

	s += fmt.Sprintf("%s: %s (%s)\n\n",
		m.getText("alertStock"), m.currentAlert.StockName, m.currentAlert.StockCode)

	switch m.alertManageStep {
	case 0: // 选择类型
		s += m.getText("alertSelectType") + "\n\n"

		types := []string{
			m.getText("alertTypePrice"),
			m.getText("alertTypeRate"),
			m.getText("alertTypeVolume"),
		}

		for i, typeText := range types {
			prefix := "  "
			if i == m.tagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, typeText)
		}

		s += "\n" + m.getText("alertHelp.select") + "\n"

	case 1: // 选择条件
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n\n", m.getText("alertType"), typeText)
		s += m.getText("alertSelectCondition") + "\n\n"

		conditions := []string{
			m.getText("alertConditionAbove"),
			m.getText("alertConditionBelow"),
			m.getText("alertConditionAboveEq"),
			m.getText("alertConditionBelowEq"),
		}

		for i, condText := range conditions {
			prefix := "  "
			if i == m.tagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, condText)
		}

		s += "\n" + m.getText("alertHelp.select") + "\n"

	case 2: // 输入阈值
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n", m.getText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n\n", m.getText("alertCondition"), m.selectedAlertCondition)

		s += m.getText("alertInputThreshold") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", m.alertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += m.getText("alertHelp.input") + "\n"

	case 3: // 选择触发频率
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n", m.getText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n", m.getText("alertCondition"), m.selectedAlertCondition)
		s += fmt.Sprintf("%s: %.2f\n\n", m.getText("alert.threshold"), m.alertThreshold)

		s += m.getText("alert.selectFrequency") + "\n\n"

		frequencyTexts := []string{
			m.getText("alert.frequency.once"),
			m.getText("alert.frequency.daily"),
			m.getText("alert.frequency.weekly"),
			m.getText("alert.frequency.monthly"),
			m.getText("alert.frequency.everyNDays.option"),
		}

		for i, freqText := range frequencyTexts {
			prefix := "  "
			if i == m.alertFrequencyCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, freqText)
		}

		s += "\n" + m.getText("alertHelp.select") + "\n"

	case 4: // 输入自定义天数
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n", m.getText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n", m.getText("alertCondition"), m.selectedAlertCondition)
		s += fmt.Sprintf("%s: %.2f\n", m.getText("alert.threshold"), m.alertThreshold)
		s += fmt.Sprintf("%s: %s\n\n", m.getText("alert.frequency"), m.getText("alert.frequency.everyNDays.option"))

		s += m.getText("alert.frequency.enterDays") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", m.alertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += m.getText("alertHelp.input") + "\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ============================================================================
// AlertEdit State Handler
// ============================================================================

// handleAlertEdit 处理编辑告警状态
func (m *Model) handleAlertEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 编辑流程与添加相同,复用相同的步骤
	switch m.alertManageStep {
	case 0: // 选择告警类型
		return m.handleAlertEditTypeSelect(msg)
	case 1: // 选择条件
		return m.handleAlertEditConditionSelect(msg)
	case 2: // 输入阈值
		return m.handleAlertEditThresholdInput(msg)
	default:
		return m, nil
	}
}

// handleAlertEditTypeSelect 处理编辑告警类型选择
func (m *Model) handleAlertEditTypeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回
		if m.previousState == StockAlertManage {
			m.state = StockAlertManage
		} else {
			m.state = AlertManage
		}
		m.message = ""
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 2 { // 3种类型
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		// 设置告警类型
		m.selectedAlertType = GetAlertTypeFromCursor(m.tagSelectCursor)

		// 进入条件选择
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertEditConditionSelect 处理编辑条件选择
func (m *Model) handleAlertEditConditionSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回上一步
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case "up", "k", "w":
		if m.tagSelectCursor > 0 {
			m.tagSelectCursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.tagSelectCursor < 3 { // 4种条件
			m.tagSelectCursor++
		}
		return m, nil

	case "enter", " ":
		// 设置条件
		conditions := []string{">", "<", ">=", "<="}
		m.selectedAlertCondition = conditions[m.tagSelectCursor]

		// 进入阈值输入
		m.alertManageStep = 2
		m.alertInput = fmt.Sprintf("%.2f", m.currentAlert.Threshold)
		m.alertInputCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleAlertEditThresholdInput 处理编辑阈值输入
func (m *Model) handleAlertEditThresholdInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回上一步
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	case "enter":
		// 验证输入
		var threshold float64
		_, err := fmt.Sscanf(m.alertInput, "%f", &threshold)
		if err != nil || m.alertInput == "" {
			m.message = m.getText("alert.invalidThreshold")
			return m, nil
		}

		// 更新告警
		for i, alert := range m.alertData.Alerts {
			if alert.ID == m.currentAlert.ID {
				m.alertData.Alerts[i].Type = m.selectedAlertType
				m.alertData.Alerts[i].Condition = m.selectedAlertCondition
				m.alertData.Alerts[i].Threshold = threshold
				break
			}
		}

		m.saveAlertData()
		m.message = m.getText("alert.editSuccess")

		// 返回来源状态
		if m.previousState == StockAlertManage {
			// 刷新股票告警列表
			m.stockAlertAlerts = m.getStockAlerts(m.stockAlertCode)
			m.state = StockAlertManage
		} else {
			m.state = AlertManage
		}

		// 重置状态
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		m.alertInput = ""

		return m, nil

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		// 只接受数字、小数点和负号
		if len(msg.String()) == 1 {
			char := msg.String()
			if (char >= "0" && char <= "9") || char == "." || char == "-" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// viewAlertEdit 渲染编辑告警界面
func (m *Model) viewAlertEdit() string {
	s := "=== " + m.getText("alertEditTitle") + " ===\n\n"

	s += fmt.Sprintf("%s: %s (%s)\n\n",
		m.getText("alertStock"), m.currentAlert.StockName, m.currentAlert.StockCode)

	// 复用 viewAlertAdd 的逻辑
	switch m.alertManageStep {
	case 0: // 选择类型
		s += m.getText("alertSelectType") + "\n\n"

		types := []string{
			m.getText("alertTypePrice"),
			m.getText("alertTypeRate"),
			m.getText("alertTypeVolume"),
		}

		for i, typeText := range types {
			prefix := "  "
			if i == m.tagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, typeText)
		}

		s += "\n" + m.getText("alertHelp.select") + "\n"

	case 1: // 选择条件
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n\n", m.getText("alertType"), typeText)
		s += m.getText("alertSelectCondition") + "\n\n"

		conditions := []string{
			m.getText("alertConditionAbove"),
			m.getText("alertConditionBelow"),
			m.getText("alertConditionAboveEq"),
			m.getText("alertConditionBelowEq"),
		}

		for i, condText := range conditions {
			prefix := "  "
			if i == m.tagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, condText)
		}

		s += "\n" + m.getText("alertHelp.select") + "\n"

	case 2: // 输入阈值
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n", m.getText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n\n", m.getText("alertCondition"), m.selectedAlertCondition)

		s += m.getText("alertInputThreshold") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", m.alertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += m.getText("alertHelp.input") + "\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ============================================================================
// Batch Alert State Handlers
// ============================================================================

// handleAlertBatchMethodSelect 处理批量添加方式选择
func (m *Model) handleAlertBatchMethodSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // 返回告警管理
		m.state = AlertManage
		m.message = ""
		return m, nil

	case "up", "k", "w": // 向上选择
		if m.batchSelectStep > 0 {
			m.batchSelectStep--
		}
		return m, nil

	case "down", "j", "s": // 向下选择
		if m.batchSelectStep < 2 {
			m.batchSelectStep++
		}
		return m, nil

	case "enter", " ":
		// 根据选择的批量方式进入相应状态
		switch m.batchSelectStep {
		case 0: // 按标签添加
			m.state = AlertBatchByTag
			m.batchAlertTag = ""
			m.tagManageCursor = 0

		case 1: // 按市场添加
			m.state = AlertBatchByMarket
			m.batchSelectedMarket = ""
			m.marketCursor = 0

		case 2: // 按股票列表添加
			m.state = SelectBatchStocks
			m.batchSelectStep = 0
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewAlertBatchMethodSelect 渲染批量添加方式选择界面
func (m *Model) viewAlertBatchMethodSelect() string {
	s := "=== " + m.getText("alertBatchMethodTitle") + " ===\n\n"

	options := []string{
		m.getText("alertBatchByTag"),
		m.getText("alertBatchByMarket"),
		m.getText("alertBatchByStocks"),
	}

	for i, option := range options {
		prefix := "  "
		if i == m.batchSelectStep {
			prefix = "► "
		}
		s += fmt.Sprintf("%s%d. %s\n", prefix, i+1, option)
	}

	s += "\n" + m.getText("alertHelp.select") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// handleAlertBatchByTag 处理按标签批量添加
func (m *Model) handleAlertBatchByTag(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // 返回方式选择
		m.state = AlertBatchMethodSelect
		m.batchAlertTag = ""
		return m, nil

	case "up", "k", "w": // 向上选择
		if m.tagManageCursor > 0 {
			m.tagManageCursor--
		}
		return m, nil

	case "down", "j", "s": // 向下选择
		if m.tagManageCursor < len(m.availableTags)-1 {
			m.tagManageCursor++
		}
		return m, nil

	case "enter", " ": // 确认选择
		if m.tagManageCursor < len(m.availableTags) {
			m.batchAlertTag = m.availableTags[m.tagManageCursor]

			// 获取该标签下的所有股票
			m.batchSelectedStocks = m.getStocksByTag(m.batchAlertTag)

			if len(m.batchSelectedStocks) == 0 {
				m.message = m.getText("alertBatchEmptyTag")
				return m, nil
			}

			m.state = AlertBatchAdd
			m.alertManageStep = 0
			m.tagSelectCursor = 0
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewAlertBatchByTag 渲染按标签批量添加界面
func (m *Model) viewAlertBatchByTag() string {
	s := "=== " + m.getText("alertBatchByTagTitle") + " ===\n\n"
	s += m.getText("alertSelectTagPrompt") + "\n\n"

	if len(m.availableTags) == 0 {
		s += m.getText("alertNoTagsAvailable") + "\n\n"
		s += m.getText("alertHelp.back") + "\n"
		return s
	}

	for i, tag := range m.availableTags {
		prefix := "  "
		if i == m.tagManageCursor {
			prefix = "► "
		}

		// 统计该标签下的股票数量
		stockCount := len(m.getStocksByTag(tag))
		s += fmt.Sprintf("%s%s (%d %s)\n", prefix, tag, stockCount, m.getText("alertStocksCount"))
	}

	s += "\n" + m.getText("alertHelp.select") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// handleAlertBatchByMarket 处理按市场批量添加
func (m *Model) handleAlertBatchByMarket(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // 返回方式选择
		m.state = AlertBatchMethodSelect
		m.batchSelectedMarket = ""
		return m, nil

	case "up", "k", "w": // 向上选择
		if m.marketCursor > 0 {
			m.marketCursor--
		}
		return m, nil

	case "down", "j", "s": // 向下选择
		if m.marketCursor < 2 {
			m.marketCursor++
		}
		return m, nil

	case "enter", " ": // 确认选择
		// 根据选择的市场获取股票
		var marketType MarketType
		switch m.marketCursor {
		case 0:
			marketType = MarketChina
		case 1:
			marketType = MarketUS
		case 2:
			marketType = MarketHongKong
		}

		// 获取该市场下的所有股票(从自选和持股中)
		m.batchSelectedStocks = m.getStocksByMarket(marketType)

		if len(m.batchSelectedStocks) == 0 {
			m.message = m.getText("alertBatchEmptyMarket")
			return m, nil
		}

		m.state = AlertBatchAdd
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// viewAlertBatchByMarket 渲染按市场批量添加界面
func (m *Model) viewAlertBatchByMarket() string {
	s := "=== " + m.getText("alertBatchByMarketTitle") + " ===\n\n"
	s += m.getText("alertSelectMarketPrompt") + "\n\n"

	markets := []struct {
		name       string
		marketType MarketType
	}{
		{m.getText("alertMarketChina"), MarketChina},
		{m.getText("alertMarketUS"), MarketUS},
		{m.getText("alertMarketHK"), MarketHongKong},
	}

	for i, market := range markets {
		prefix := "  "
		if i == m.marketCursor {
			prefix = "► "
		}

		// 统计该市场的股票数量
		stockCount := len(m.getStocksByMarket(market.marketType))
		s += fmt.Sprintf("%s%s (%d %s)\n", prefix, market.name, stockCount, m.getText("alertStocksCount"))
	}

	s += "\n" + m.getText("alertHelp.select") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// handleSelectBatchStocks 处理选择批量股票来源
func (m *Model) handleSelectBatchStocks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // 返回
		if m.state == SelectBatchStocks && m.previousState != AlertManage {
			// 如果从AlertManage进来的,返回AlertBatchMethodSelect
			m.state = AlertBatchMethodSelect
		} else {
			m.state = AlertManage
		}
		m.message = ""
		m.batchSelectedStocks = nil
		m.batchCodeInput = ""
		return m, nil

	case "up", "k", "w": // 向上选择
		if m.batchSelectStep > 0 {
			m.batchSelectStep--
		}
		return m, nil

	case "down", "j", "s": // 向下选择
		if m.batchSelectStep < 2 {
			m.batchSelectStep++
		}
		return m, nil

	case "enter", " ":
		// 根据选择的来源进入不同状态
		switch m.batchSelectStep {
		case 0: // 从自选列表选择
			m.state = SelectBatchFromWatchlist
			m.batchStockSource = BatchSourceWatchlist
			m.stockSelectionMap = make(map[string]bool)
			m.watchlistCursor = 0

		case 1: // 从持股列表选择
			m.state = SelectBatchFromPortfolio
			m.batchStockSource = BatchSourcePortfolio
			m.stockSelectionMap = make(map[string]bool)
			m.portfolioCursor = 0

		case 2: // 手动输入
			m.state = InputBatchCodes
			m.batchStockSource = BatchSourceManual
			m.batchCodeInput = ""
		}
		return m, nil

	default:
		return m, nil
	}
}

// viewSelectBatchStocks 渲染批量股票来源选择界面
func (m *Model) viewSelectBatchStocks() string {
	s := "=== " + m.getText("alert.batch.byStocksTitle") + " ===\n\n"

	options := []string{
		m.getText("alert.batch.fromWatchlist"),
		m.getText("alert.batch.fromPortfolio"),
		m.getText("alert.batch.manualInput"),
	}

	for i, option := range options {
		prefix := "  "
		if i == m.batchSelectStep {
			prefix = "► "
		}
		s += fmt.Sprintf("%s%d. %s\n", prefix, i+1, option)
	}

	s += "\n" + m.getText("alertHelp.select") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// handleSelectBatchFromWatchlist 处理从自选列表选择
func (m *Model) handleSelectBatchFromWatchlist(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleBatchStockSelection(msg, true)
}

// handleSelectBatchFromPortfolio 处理从持股列表选择
func (m *Model) handleSelectBatchFromPortfolio(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleBatchStockSelection(msg, false)
}

// handleBatchStockSelection 处理批量股票选择的通用逻辑
func (m *Model) handleBatchStockSelection(msg tea.KeyMsg, isWatchlist bool) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // 返回来源选择
		m.state = SelectBatchStocks
		m.stockSelectionMap = make(map[string]bool)
		return m, nil

	case "enter": // 确认选择
		// 收集选中的股票代码
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

		// 进入批量添加状态
		m.state = AlertBatchAdd
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case " ": // 切换选择状态
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

	case "up", "k", "w": // 向上导航
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

	case "down", "j", "s": // 向下导航
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

// viewSelectBatchFromWatchlist 渲染从自选列表选择界面
func (m *Model) viewSelectBatchFromWatchlist() string {
	s := "=== " + m.getText("alertSelectFromWatchlistTitle") + " ===\n\n"

	filteredStocks := m.getFilteredWatchlist()
	if len(filteredStocks) == 0 {
		s += m.getText("emptyWatchlist") + "\n\n"
		s += m.getText("alertHelp.back") + "\n"
		return s
	}

	s += m.getText("alertMultiSelectPrompt") + "\n\n"

	// 显示股票列表
	maxLines := m.config.Display.MaxLines
	startIndex := m.watchlistCursor
	if startIndex > len(filteredStocks)-maxLines {
		startIndex = len(filteredStocks) - maxLines
		if startIndex < 0 {
			startIndex = 0
		}
	}
	endIndex := startIndex + maxLines
	if endIndex > len(filteredStocks) {
		endIndex = len(filteredStocks)
	}

	for i := startIndex; i < endIndex; i++ {
		stock := filteredStocks[i]

		// 选择标记
		checkbox := "  "
		if m.stockSelectionMap[stock.Code] {
			checkbox = "✓ "
		}

		// 游标标记
		cursor := " "
		if i == m.watchlistCursor {
			cursor = "►"
		}

		s += fmt.Sprintf("%s%s[%d] %s - %s\n", cursor, checkbox, i+1, stock.Code, stock.Name)
	}

	// 统计选中数量
	selectedCount := 0
	for _, selected := range m.stockSelectionMap {
		if selected {
			selectedCount++
		}
	}

	s += fmt.Sprintf("\n%s: %d %s\n\n", m.getText("alertSelectedCount"), selectedCount, m.getText("alertStocksCount"))
	s += m.getText("alert.batch.multiSelectHelp") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// viewSelectBatchFromPortfolio 渲染从持股列表选择界面
func (m *Model) viewSelectBatchFromPortfolio() string {
	s := "=== " + m.getText("alertSelectFromPortfolioTitle") + " ===\n\n"

	if len(m.portfolio.Stocks) == 0 {
		s += m.getText("emptyPortfolio") + "\n\n"
		s += m.getText("alertHelp.back") + "\n"
		return s
	}

	s += m.getText("alertMultiSelectPrompt") + "\n\n"

	// 显示股票列表
	maxLines := m.config.Display.MaxLines
	startIndex := m.portfolioCursor
	if startIndex > len(m.portfolio.Stocks)-maxLines {
		startIndex = len(m.portfolio.Stocks) - maxLines
		if startIndex < 0 {
			startIndex = 0
		}
	}
	endIndex := startIndex + maxLines
	if endIndex > len(m.portfolio.Stocks) {
		endIndex = len(m.portfolio.Stocks)
	}

	for i := startIndex; i < endIndex; i++ {
		stock := m.portfolio.Stocks[i]

		// 选择标记
		checkbox := "  "
		if m.stockSelectionMap[stock.Code] {
			checkbox = "✓ "
		}

		// 游标标记
		cursor := " "
		if i == m.portfolioCursor {
			cursor = "►"
		}

		s += fmt.Sprintf("%s%s[%d] %s - %s (%d%s, %s%.2f)\n",
			cursor, checkbox, i+1, stock.Code, stock.Name,
			stock.Quantity, m.getText("alertSharesUnit"),
			m.getText("alertCostLabel"), stock.CostPrice)
	}

	// 统计选中数量
	selectedCount := 0
	for _, selected := range m.stockSelectionMap {
		if selected {
			selectedCount++
		}
	}

	s += fmt.Sprintf("\n%s: %d %s\n\n", m.getText("alertSelectedCount"), selectedCount, m.getText("alertStocksCount"))
	s += m.getText("alert.batch.multiSelectHelp") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// handleInputBatchCodes 处理手动输入股票代码
func (m *Model) handleInputBatchCodes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m": // 返回来源选择
		m.state = SelectBatchStocks
		m.batchCodeInput = ""
		return m, nil

	case "enter": // 确认输入
		// 解析股票代码
		codes := parseStockCodes(m.batchCodeInput)
		if len(codes) == 0 {
			m.message = m.getText("alertBatchInvalidCodes")
			return m, nil
		}

		m.batchSelectedStocks = codes
		m.state = AlertBatchAdd
		m.alertManageStep = 0
		m.tagSelectCursor = 0
		return m, nil

	case "backspace":
		if len(m.batchCodeInput) > 0 {
			m.batchCodeInput = m.batchCodeInput[:len(m.batchCodeInput)-1]
		}
		return m, nil

	default:
		// 接受所有字符(包括逗号和换行)
		if len(msg.String()) == 1 {
			m.batchCodeInput += msg.String()
		}
		return m, nil
	}
}

// viewInputBatchCodes 渲染手动输入股票代码界面
func (m *Model) viewInputBatchCodes() string {
	s := "=== " + m.getText("alertInputCodesTitle") + " ===\n\n"
	s += m.getText("alertInputCodesPrompt") + "\n\n"

	s += "┌────────────────────────────────────────────┐\n"
	// 显示输入内容(支持多行)
	lines := strings.Split(m.batchCodeInput, "\n")
	displayLines := 5
	for i := 0; i < displayLines; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		s += fmt.Sprintf("│ %-42s │\n", line)
	}
	s += "└────────────────────────────────────────────┘\n\n"

	s += m.getText("alertInputCodesExample") + "\n\n"
	s += m.getText("alertHelp.input") + "\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// handleAlertBatchAdd 处理批量添加告警(设置规则)
func (m *Model) handleAlertBatchAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 复用单股票添加的步骤逻辑
	switch m.alertManageStep {
	case 0: // 选择告警类型
		return m.handleBatchAlertTypeSelect(msg)
	case 1: // 选择条件
		return m.handleBatchAlertConditionSelect(msg)
	case 2: // 输入阈值
		return m.handleBatchAlertThresholdInput(msg)
	case 3: // 选择触发频率
		return m.handleBatchAlertFrequencySelect(msg)
	case 4: // 输入自定义天数
		return m.handleBatchAlertFrequencyDaysInput(msg)
	default:
		return m, nil
	}
}

// handleBatchAlertTypeSelect 处理批量告警类型选择
func (m *Model) handleBatchAlertTypeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回
		// 返回到上一个状态
		m.state = AlertBatchMethodSelect
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
		// 设置告警类型
		m.selectedAlertType = GetAlertTypeFromCursor(m.tagSelectCursor)

		// 进入条件选择
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleBatchAlertConditionSelect 处理批量告警条件选择
func (m *Model) handleBatchAlertConditionSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回上一步
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
		// 设置条件
		m.selectedAlertCondition = GetAlertConditionFromCursor(m.tagSelectCursor)

		// 进入阈值输入
		m.alertManageStep = 2
		m.alertInput = ""
		m.alertInputCursor = 0
		return m, nil

	default:
		return m, nil
	}
}

// handleBatchAlertThresholdInput 处理批量告警阈值输入
func (m *Model) handleBatchAlertThresholdInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q": // 返回上一步
		m.alertManageStep = 1
		m.tagSelectCursor = 0
		return m, nil

	case "enter":
		// 验证输入
		var threshold float64
		_, err := fmt.Sscanf(m.alertInput, "%f", &threshold)
		if err != nil || m.alertInput == "" {
			m.message = m.getText("alert.invalidThreshold")
			return m, nil
		}

		// 保存阈值，进入频率选择步骤
		m.alertThreshold = threshold
		m.alertManageStep = 3 // 进入频率选择
		m.alertFrequencyCursor = 0
		m.selectedAlertFrequency = ""
		return m, nil

	case "backspace":
		if len(m.alertInput) > 0 {
			m.alertInput = m.alertInput[:len(m.alertInput)-1]
		}
		return m, nil

	default:
		// 只接受数字、小数点和负号
		if len(msg.String()) == 1 {
			char := msg.String()
			if (char >= "0" && char <= "9") || char == "." || char == "-" {
				m.alertInput += char
			}
		}
		return m, nil
	}
}

// viewAlertBatchAdd 渲染批量添加告警界面
func (m *Model) viewAlertBatchAdd() string {
	s := "=== " + m.getText("alertBatchAddTitle") + " ===\n\n"

	s += fmt.Sprintf("%s: %d %s\n\n",
		m.getText("alertBatchStockCount"),
		len(m.batchSelectedStocks),
		m.getText("alertStocksCount"))

	// 显示前5只股票作为示例
	previewCount := 5
	if len(m.batchSelectedStocks) < previewCount {
		previewCount = len(m.batchSelectedStocks)
	}
	for i := 0; i < previewCount; i++ {
		s += fmt.Sprintf("  [%d] %s\n", i+1, m.batchSelectedStocks[i])
	}
	if len(m.batchSelectedStocks) > previewCount {
		s += fmt.Sprintf("  ... (%s %d %s)\n",
			m.getText("alertAndMore"),
			len(m.batchSelectedStocks)-previewCount,
			m.getText("alertStocksCount"))
	}
	s += "\n"

	// 复用单股票添加的UI
	switch m.alertManageStep {
	case 0: // 选择类型
		s += m.getText("alertSelectType") + "\n\n"

		types := []string{
			m.getText("alertTypePrice"),
			m.getText("alertTypeRate"),
			m.getText("alertTypeVolume"),
		}

		for i, typeText := range types {
			prefix := "  "
			if i == m.tagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, typeText)
		}

		s += "\n" + m.getText("alertHelp.select") + "\n"

	case 1: // 选择条件
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n\n", m.getText("alertType"), typeText)
		s += m.getText("alertSelectCondition") + "\n\n"

		conditions := []string{
			m.getText("alertConditionAbove"),
			m.getText("alertConditionBelow"),
			m.getText("alertConditionAboveEq"),
			m.getText("alertConditionBelowEq"),
		}

		for i, condText := range conditions {
			prefix := "  "
			if i == m.tagSelectCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, condText)
		}

		s += "\n" + m.getText("alertHelp.select") + "\n"

	case 2: // 输入阈值
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n", m.getText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n\n", m.getText("alertCondition"), m.selectedAlertCondition)

		s += m.getText("alertInputThreshold") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", m.alertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += m.getText("alertHelp.input") + "\n"

	case 3: // 选择触发频率 (批量添加)
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n", m.getText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n", m.getText("alertCondition"), m.selectedAlertCondition)
		s += fmt.Sprintf("%s: %.2f\n\n", m.getText("alert.threshold"), m.alertThreshold)

		s += m.getText("alert.selectFrequency") + "\n\n"

		frequencyTexts := []string{
			m.getText("alert.frequency.once"),
			m.getText("alert.frequency.daily"),
			m.getText("alert.frequency.weekly"),
			m.getText("alert.frequency.monthly"),
			m.getText("alert.frequency.everyNDays.option"),
		}

		for i, freqText := range frequencyTexts {
			prefix := "  "
			if i == m.alertFrequencyCursor {
				prefix = "► "
			}
			s += fmt.Sprintf("%s%s\n", prefix, freqText)
		}

		s += "\n" + m.getText("alertHelp.select") + "\n"

	case 4: // 输入自定义天数 (批量添加)
		typeText := ""
		switch m.selectedAlertType {
		case AlertTypePrice:
			typeText = m.getText("alertTypePrice")
		case AlertTypeRate:
			typeText = m.getText("alertTypeRate")
		case AlertTypeVolume:
			typeText = m.getText("alertTypeVolume")
		}

		s += fmt.Sprintf("%s: %s\n", m.getText("alertType"), typeText)
		s += fmt.Sprintf("%s: %s\n", m.getText("alertCondition"), m.selectedAlertCondition)
		s += fmt.Sprintf("%s: %.2f\n", m.getText("alert.threshold"), m.alertThreshold)
		s += fmt.Sprintf("%s: %s\n\n", m.getText("alert.frequency"), m.getText("alert.frequency.everyNDays.option"))

		s += m.getText("alert.frequency.enterDays") + "\n"
		s += "┌────────────────────────────────────────────┐\n"
		s += fmt.Sprintf("│ %-42s │\n", m.alertInput)
		s += "└────────────────────────────────────────────┘\n\n"

		s += m.getText("alertHelp.input") + "\n"
	}

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// handleBatchAlertFrequencySelect 处理批量告警频率选择
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

// handleBatchAlertFrequencyDaysInput 处理批量告警自定义天数输入
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

// createBatchAlertsWithFrequency 使用选定的频率批量创建告警
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

	m.state = AlertManage
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
