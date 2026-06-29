package main

import (
	"fmt"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"stock-monitor/internal/consts"
)

// ========== Main Menu Handlers ==========

// handleMainMenu handles the main menu state
func (m *Model) handleMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "w":
		m.currentMenuItem = MoveCursorUp(m.currentMenuItem)
		m.message = "" // 清除消息
	case "down", "j", "s":
		m.currentMenuItem = MoveCursorDown(m.currentMenuItem, len(m.menuItems)-1)
		m.message = "" // 清除消息
	case "enter", " ":
		return m.executeMenuItem()
	case "q", "ctrl+c":
		m.stopIntradayDataCollection()
		m.stopSearchIntradayWorker()
		m.savePortfolio()
		m.saveWatchlist()
		m.saveAlertData()
		return m, tea.Quit
	}
	return m, nil
}

// executeMenuItem executes the selected menu item
func (m *Model) executeMenuItem() (tea.Model, tea.Cmd) {
	m.message = "" // 清除之前的消息

	if m.currentMenuItem >= len(m.menuItems) {
		return m, nil
	}

	item := m.menuItems[m.currentMenuItem]

	switch item.Key {
	case "stockList": // 股票列表
		logInfo("log.action.enterPortfolio")
		m.state = consts.Monitoring
		m.resetPortfolioCursor() // 重置游标到第一只股票
		m.lastUpdate = time.Now()

		// 启动分时数据采集
		m.startIntradayDataCollection()

		return m, m.tickCmd()

	case "watchlist": // 自选股票
		logInfo("log.action.enterWatchlist")
		m.state = consts.WatchlistViewing
		m.resetWatchlistCursor() // 重置游标到第一只股票
		m.cursor = 0
		m.message = ""
		m.lastUpdate = time.Now()

		// 启动分时数据采集
		m.startIntradayDataCollection()

		// 立即启动数据更新，而不等待定时器
		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())

		// 强制启动股价数据更新
		if stockPriceCmd := m.startStockPriceUpdates(); stockPriceCmd != nil {
			cmds = append(cmds, stockPriceCmd)
		}

		return m, tea.Batch(cmds...)

	case "stockSearch": // 股票搜索
		logInfo("log.action.enterSearch")
		m.state = consts.SearchingStock
		m.searchInput = ""
		m.searchResult = nil
		m.searchFromWatchlist = false
		m.message = ""
		return m, nil

	case "alertManagement": // 告警管理
		logInfo("log.action.enterAlertManagement")
		m.state = consts.AlertManage
		m.alertCursor = 0
		alertData, corrupted := loadAlertData()
		m.alertData = alertData
		if corrupted {
			m.alertDataCorrupted = true
		}
		return m, nil

	case "sector.title": // 板块行情
		logInfo("log.action.enterSector")
		m.state = consts.SectorViewing
		m.sectorCursor = 0
		m.sectorScrollPos = 0
		m.message = ""
		m.lastUpdate = time.Now()

		// 启动定时刷新并获取初始板块数据
		var cmds []tea.Cmd
		cmds = append(cmds, m.tickCmd())                        // 启动定时刷新循环
		cmds = append(cmds, m.fetchSectorListCmd(m.sectorType)) // 立即获取板块数据
		return m, tea.Batch(cmds...)

	case "language": // 语言选择
		logInfo("log.action.enterLanguage")
		m.state = consts.LanguageSelection
		m.languageCursor = 0
		if m.language == consts.English {
			m.languageCursor = 1
		}
		return m, nil

	case "exit": // 退出
		logInfo("log.action.exit")
		m.stopIntradayDataCollection()
		m.stopSearchIntradayWorker()
		m.savePortfolio()
		m.saveWatchlist()
		m.saveAlertData()
		return m, tea.Quit
	}

	return m, nil
}

// viewMainMenu renders the main menu view
func (m *Model) viewMainMenu() string {
	s := m.getText("title") + "\n\n"

	for i, item := range m.menuItems {
		prefix := "  "
		if i == m.currentMenuItem {
			prefix = "► "
		}

		// 基于 Key 识别语言选项，显示当前语言状态
		if item.Key == "language" {
			langStatus := m.getText("english")
			if m.language == consts.Chinese {
				langStatus = m.getText("chinese")
			}
			s += fmt.Sprintf("%s%s: %s\n", prefix, item.Label, langStatus)
		} else {
			s += fmt.Sprintf("%s%s\n", prefix, item.Label)
		}
	}

	s += "\n"
	if runtime.GOOS == "windows" {
		s += m.getText("keyHelpWin") + "\n"
	} else {
		s += m.getText("keyHelp") + "\n"
	}
	s += "==================================================\n"

	if m.message != "" {
		s += "\n" + m.message + "\n"
	}

	return s
}

// ========== Language Selection Handlers ==========

// handleLanguageSelection handles the language selection state
func (m *Model) handleLanguageSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = consts.MainMenu
		m.message = "" // 清除消息
		return m, nil
	case "up", "k", "w":
		m.languageCursor = MoveCursorUp(m.languageCursor)
	case "down", "j", "s":
		m.languageCursor = MoveCursorDown(m.languageCursor, 1) // 0: consts.Chinese, 1: consts.English
	case "enter", " ":
		// 选择语言
		if m.languageCursor == 0 {
			m.language = consts.Chinese
			m.config.System.Language = "zh"
		} else {
			m.language = consts.English
			m.config.System.Language = "en"
		}
		// 保存配置到文件
		if err := saveConfig(m.config); err != nil {
			m.message = fmt.Sprintf("Warning: Failed to save config: %v", err)
		}
		// 更新菜单项以反映新语言
		m.menuItems = getMenuItems(m.language)
		m.state = consts.MainMenu
		m.message = ""
		return m, nil
	}
	return m, nil
}

// viewLanguageSelection renders the language selection view
func (m *Model) viewLanguageSelection() string {
	s := m.getText("languageTitle") + "\n\n"
	s += m.getText("selectLanguage") + "\n\n"

	// 语言选项 - 使用 i18n 系统获取翻译文本
	languages := []string{
		m.getText("chinese"),
		m.getText("english"),
	}

	for i, lang := range languages {
		prefix := "  "
		if i == m.languageCursor {
			prefix = "► "
		}
		s += fmt.Sprintf("%s%s\n", prefix, lang)
	}

	s += "\n" + m.getText("languageHelp") + "\n"

	return s
}
