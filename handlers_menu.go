package main

import (
	"fmt"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		m.savePortfolio()
		return m, tea.Quit
	}
	return m, nil
}

// executeMenuItem executes the selected menu item
func (m *Model) executeMenuItem() (tea.Model, tea.Cmd) {
	m.message = "" // 清除之前的消息
	switch m.currentMenuItem {
	case 0: // 股票列表
		logInfo("log.action.enterPortfolio")
		m.state = Monitoring
		m.resetPortfolioCursor() // 重置游标到第一只股票
		m.lastUpdate = time.Now()

		// 启动分时数据采集
		m.startIntradayDataCollection()

		return m, m.tickCmd()
	case 1: // 自选股票
		logInfo("log.action.enterWatchlist")
		m.state = WatchlistViewing
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
	case 2: // 股票搜索
		logInfo("log.action.enterSearch")
		m.state = SearchingStock
		m.searchInput = ""
		m.searchResult = nil
		m.searchFromWatchlist = false
		m.message = ""
		return m, nil
	case 3: // 告警管理
		logInfo("log.action.enterAlertManagement")
		m.state = AlertManage
		m.alertCursor = 0
		m.alertData = loadAlertData()
		return m, nil
	case 4: // 语言选择页面
		logInfo("log.action.enterLanguage")
		m.state = LanguageSelection
		m.languageCursor = 0
		if m.language == English {
			m.languageCursor = 1
		}
		return m, nil
	case 5: // 退出
		logInfo("log.action.exit")
		m.savePortfolio()
		m.saveWatchlist()
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

		if i == 4 { // 语言选择
			langStatus := m.getText("english")
			if m.language == Chinese {
				langStatus = m.getText("chinese")
			}
			s += fmt.Sprintf("%s%s: %s\n", prefix, item, langStatus)
		} else {
			s += fmt.Sprintf("%s%s\n", prefix, item)
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
		m.state = MainMenu
		m.message = "" // 清除消息
		return m, nil
	case "up", "k", "w":
		m.languageCursor = MoveCursorUp(m.languageCursor)
	case "down", "j", "s":
		m.languageCursor = MoveCursorDown(m.languageCursor, 1) // 0: Chinese, 1: English
	case "enter", " ":
		// 选择语言
		if m.languageCursor == 0 {
			m.language = Chinese
			m.config.System.Language = "zh"
		} else {
			m.language = English
			m.config.System.Language = "en"
		}
		// 保存配置到文件
		if err := saveConfig(m.config); err != nil {
			m.message = fmt.Sprintf("Warning: Failed to save config: %v", err)
		}
		// 更新菜单项
		m.menuItems = m.getMenuItems()
		m.state = MainMenu
		m.message = ""
		return m, nil
	}
	return m, nil
}

// viewLanguageSelection renders the language selection view
func (m *Model) viewLanguageSelection() string {
	s := m.getText("languageTitle") + "\n\n"
	s += m.getText("selectLanguage") + "\n\n"

	// 语言选项
	languages := []string{
		"中文简体",
		"English",
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
