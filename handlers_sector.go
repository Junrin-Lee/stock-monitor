package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"stock-monitor/internal/api"
	"stock-monitor/internal/api/china"
	"stock-monitor/internal/consts"
	"stock-monitor/internal/types"
	"stock-monitor/internal/ui/sector"
)

// ============================================================================
// Sector Viewing Handlers - 板块列表查看
// ============================================================================

func (m *Model) handleSectorViewing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "m":
		// 返回主菜单
		m.stopIntradayDataCollection()
		m.state = consts.MainMenu
		m.message = ""
		return m, nil

	case "up", "k":
		// 光标上移
		if m.sectorCursor > 0 {
			m.sectorCursor--
			// 调整滚动位置
			if m.sectorCursor < m.sectorScrollPos {
				m.sectorScrollPos = m.sectorCursor
			}
		}
		return m, nil

	case "down", "j":
		// 光标下移
		if m.sectorCursor < len(m.sectorList)-1 {
			m.sectorCursor++
			// 调整滚动位置
			if m.sectorCursor >= m.sectorScrollPos+m.maxLines {
				m.sectorScrollPos = m.sectorCursor - m.maxLines + 1
			}
		}
		return m, nil

	case "enter":
		// 进入成分股列表
		if len(m.sectorList) == 0 {
			m.message = "暂无板块数据"
			return m, nil
		}

		if m.sectorCursor >= 0 && m.sectorCursor < len(m.sectorList) {
			selectedSector := m.sectorList[m.sectorCursor]
			m.currentSectorCode = selectedSector.Code
			m.currentSectorName = selectedSector.Name
			m.currentSectorInfo = &selectedSector

			// 重置成分股状态
			m.sectorStocks = nil
			m.sectorStockCursor = 0
			m.sectorStockScrollPos = 0

			// 切换到成分股列表状态
			m.state = consts.SectorStockList

			// 异步加载成分股数据
			return m, m.fetchSectorStocksCmd(selectedSector.Code)
		}
		return m, nil

	case "t":
		// 切换板块类型
		if m.sectorType == types.SectorTypeIndustry {
			m.sectorType = types.SectorTypeConcept
		} else {
			m.sectorType = types.SectorTypeIndustry
		}

		// 重置状态
		m.sectorList = nil
		m.sectorCursor = 0
		m.sectorScrollPos = 0
		m.sectorIsSorted = false

		// 重新加载数据
		return m, m.fetchSectorListCmd(m.sectorType)

	case "s":
		// 打开排序菜单
		m.state = consts.SectorSorting
		// Smart position cursor to current sort field (板块列表)
		m.sortMenuCursor = m.findSectorSortFieldIndex(m.sectorSortField, true)
		return m, nil

	case "c":
		// 清除排序,恢复默认顺序(按名称升序)
		m.sectorSortField = consts.SortBySectorName
		m.sectorSortDirection = consts.SortAsc
		m.sectorIsSorted = false
		sortSectors(m.sectorList, m.sectorSortField, m.sectorSortDirection)
		m.message = "已恢复默认排序(按名称升序)"
		return m, nil
	}

	return m, nil
}

// ============================================================================
// Sector Stock List Handlers - 成分股列表
// ============================================================================

func (m *Model) handleSectorStockList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// 返回板块列表
		m.stopIntradayDataCollection()
		m.state = consts.SectorViewing
		m.message = ""
		return m, nil

	case "up", "k":
		// 光标上移
		if m.sectorStockCursor > 0 {
			m.sectorStockCursor--
			if m.sectorStockCursor < m.sectorStockScrollPos {
				m.sectorStockScrollPos = m.sectorStockCursor
			}
		}
		return m, nil

	case "down", "j":
		// 光标下移
		if m.sectorStockCursor < len(m.sectorStocks)-1 {
			m.sectorStockCursor++
			if m.sectorStockCursor >= m.sectorStockScrollPos+m.maxLines {
				m.sectorStockScrollPos = m.sectorStockCursor - m.maxLines + 1
			}
		}
		return m, nil

	case "a":
		// 添加到自选列表
		if len(m.sectorStocks) == 0 {
			m.message = "暂无成分股数据"
			return m, nil
		}

		if m.sectorStockCursor >= 0 && m.sectorStockCursor < len(m.sectorStocks) {
			selectedStock := m.sectorStocks[m.sectorStockCursor]

			// 检查是否已在自选列表
			for _, ws := range m.watchlist.Stocks {
				if ws.Code == selectedStock.Code {
					m.message = fmt.Sprintf("%s 已在自选列表中", selectedStock.Name)
					return m, nil
				}
			}

			// 添加到自选列表（包含市场类型）
			newWatchStock := WatchlistStock{
				Code:   selectedStock.Code,
				Name:   selectedStock.Name,
				Market: api.GetMarketType(selectedStock.Code),
			}
			m.watchlist.Stocks = append(m.watchlist.Stocks, newWatchStock)

			// 保存自选列表
			m.saveWatchlist()
			m.message = fmt.Sprintf("%s 已添加到自选列表", selectedStock.Name)
		}
		return m, nil

	case "v":
		// 查看分时图
		if len(m.sectorStocks) == 0 {
			m.message = "暂无成分股数据"
			return m, nil
		}

		if m.sectorStockCursor >= 0 && m.sectorStockCursor < len(m.sectorStocks) {
			selectedStock := m.sectorStocks[m.sectorStockCursor]
			m.chartViewStock = selectedStock.Code
			m.chartViewStockName = selectedStock.Name

			// 选择展示日期（与 worker 采集逻辑解耦，处理静默期等边缘情况）
			actualDate := getChartDisplayDate(selectedStock.Code, m)
			m.chartViewDate = actualDate
			m.previousState = consts.SectorStockList

			// 加载分时数据
			data, loadErr := m.loadIntradayDataForDate(
				selectedStock.Code,
				selectedStock.Name,
				actualDate,
			)

			if loadErr != nil {
				// 无数据 - 触发收集
				m.chartData = nil
				m.chartLoadError = nil
				m.state = consts.IntradayChartViewing
				return m, m.triggerIntradayDataCollection(
					selectedStock.Code,
					selectedStock.Name,
					actualDate,
				)
			}

			// 数据存在 - 显示图表
			m.chartData = data
			m.chartLoadError = nil
			m.chartIsCollecting = false
			m.state = consts.IntradayChartViewing
		}
		return m, nil

	case "s":
		// 打开排序菜单
		m.state = consts.SectorSorting
		// Smart position cursor to current sort field (成分股列表)
		m.sortMenuCursor = m.findSectorSortFieldIndex(m.sectorSortField, false)
		return m, nil

	case "c":
		// 清除排序
		m.sectorSortField = consts.SortByName
		m.sectorSortDirection = consts.SortAsc
		m.sectorIsSorted = false
		sortSectorStocks(m.sectorStocks, m.sectorSortField, m.sectorSortDirection)
		m.message = "已恢复默认排序(按名称升序)"
		return m, nil
	}

	return m, nil
}

// ============================================================================
// Sector Sorting Handlers - 板块排序
// ============================================================================

func (m *Model) handleSectorSorting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var sortFields []SortField

	// 根据当前状态选择排序选项
	if m.state == consts.SectorSorting {
		// 判断是板块列表还是成分股列表
		if m.currentSectorCode == "" {
			// 板块列表排序选项
			sortFields = []SortField{
				consts.SortBySectorName,
				consts.SortBySectorChangePercent,
				consts.SortBySectorChange,
				consts.SortBySectorTurnover,
				consts.SortByTurnoverRate,
				consts.SortBySectorRiseCount,
			}
		} else {
			// 成分股列表排序选项
			sortFields = []SortField{
				consts.SortByCode,
				consts.SortByName,
				consts.SortByPrice,
				consts.SortByChangePercent,
				consts.SortByChange,
				consts.SortByVolume,
				consts.SortBySectorTurnover,
				consts.SortByTurnoverRate,
			}
		}
	}

	switch msg.String() {
	case "esc", "q":
		// 返回上一状态
		if m.currentSectorCode == "" {
			m.state = consts.SectorViewing
		} else {
			m.state = consts.SectorStockList
		}
		return m, nil

	case "up", "k":
		if m.sortMenuCursor > 0 {
			m.sortMenuCursor--
		}
		return m, nil

	case "down", "j":
		if m.sortMenuCursor < len(sortFields)-1 {
			m.sortMenuCursor++
		}
		return m, nil

	case "enter", " ":
		// 选择排序字段
		if m.sortMenuCursor >= 0 && m.sortMenuCursor < len(sortFields) {
			selectedField := sortFields[m.sortMenuCursor]

			// 如果选择的字段与当前字段相同,切换排序方向
			if m.sectorSortField == selectedField {
				if m.sectorSortDirection == consts.SortAsc {
					m.sectorSortDirection = consts.SortDesc
				} else {
					m.sectorSortDirection = consts.SortAsc
				}
			} else {
				// 新字段,默认降序(除了名称/代码默认升序)
				m.sectorSortField = selectedField
				if selectedField == consts.SortBySectorName || selectedField == consts.SortByName || selectedField == consts.SortByCode {
					m.sectorSortDirection = consts.SortAsc
				} else {
					m.sectorSortDirection = consts.SortDesc
				}
			}

			m.sectorIsSorted = true

			// 执行排序
			if m.currentSectorCode == "" {
				// 板块列表排序
				m.sortSectorList()
				m.state = consts.SectorViewing
			} else {
				// 成分股排序
				m.sortSectorStocks()
				m.state = consts.SectorStockList
			}

			// 重置光标和滚动位置
			m.sectorCursor = 0
			m.sectorScrollPos = 0
			m.sectorStockCursor = 0
			m.sectorStockScrollPos = 0
		}
		return m, nil
	}

	return m, nil
}

// ============================================================================
// Sector View Functions - 视图渲染
// ============================================================================

func (m *Model) viewSectorViewing() string {
	updateTime := time.Now().Format("2006-01-02 15:04:05")
	return sector.RenderSectorList(
		m.sectorList,
		m.sectorType,
		m.sectorCursor,
		m.sectorScrollPos,
		m.maxLines,
		m.sectorSortField,
		m.sectorSortDirection,
		m.language,
		updateTime,
	)
}

func (m *Model) viewSectorStockList() string {
	updateTime := time.Now().Format("2006-01-02 15:04:05")
	return sector.RenderSectorStockList(
		m.currentSectorName,
		m.currentSectorInfo,
		m.sectorStocks,
		m.sectorStockCursor,
		m.sectorStockScrollPos,
		m.maxLines,
		m.sectorSortField,
		m.sectorSortDirection,
		m.language,
		updateTime,
		m.getSparklineForStock,
	)
}

func (m *Model) viewSectorSorting() string {
	s := "=== 排序选择 ===\n\n"
	s += "请选择排序字段:\n\n"

	var sortFields []SortField
	var fieldNames []string

	// 根据当前上下文选择排序选项
	if m.currentSectorCode == "" {
		// 板块列表排序
		sortFields = []SortField{
			consts.SortBySectorName,
			consts.SortBySectorChangePercent,
			consts.SortBySectorChange,
			consts.SortBySectorTurnover,
			consts.SortByTurnoverRate,
			consts.SortBySectorRiseCount,
		}
		fieldNames = []string{
			"板块名称",
			"涨跌幅",
			"涨跌额",
			"总成交额",
			"换手率",
			"上涨家数",
		}
	} else {
		// 成分股排序
		sortFields = []SortField{
			consts.SortByCode,
			consts.SortByName,
			consts.SortByPrice,
			consts.SortByChangePercent,
			consts.SortByChange,
			consts.SortByVolume,
			consts.SortBySectorTurnover,
			consts.SortByTurnoverRate,
		}
		fieldNames = []string{
			"股票代码",
			"股票名称",
			"现价",
			"涨跌幅",
			"涨跌额",
			"成交量",
			"成交额",
			"换手率",
		}
	}

	for i, field := range sortFields {
		prefix := "  "
		if i == m.sortMenuCursor {
			prefix = "► "
		}

		fieldName := fieldNames[i]
		if m.sectorIsSorted && m.sectorSortField == field {
			// 显示当前排序状态
			directionName := "升序"
			if m.sectorSortDirection == consts.SortDesc {
				directionName = "降序"
			}
			s += fmt.Sprintf("%s%s (%s)\n", prefix, fieldName, directionName)
		} else {
			s += fmt.Sprintf("%s%s\n", prefix, fieldName)
		}
	}

	s += "\n按回车选择 | ESC/Q 返回\n"
	return s
}

// ============================================================================
// Sector Data Fetching Commands - 数据获取命令
// ============================================================================

// fetchSectorListCmd 异步获取板块列表
func (m *Model) fetchSectorListCmd(sectorType types.SectorType) tea.Cmd {
	return func() tea.Msg {
		sectors, err := china.FetchSectorList(sectorType)
		if err != nil {
			logError("获取板块列表失败: %v", err)
			return SectorListMsg{Sectors: nil, Err: err}
		}
		return SectorListMsg{Sectors: sectors, Err: nil}
	}
}

// fetchSectorStocksCmd 异步获取成分股列表
func (m *Model) fetchSectorStocksCmd(sectorCode string) tea.Cmd {
	return func() tea.Msg {
		stocks, err := china.FetchSectorStocks(sectorCode)
		if err != nil {
			logError("获取成分股列表失败: %v", err)
			return SectorStocksMsg{Stocks: nil, Err: err}
		}
		return SectorStocksMsg{Stocks: stocks, Err: nil}
	}
}

// SectorListMsg 板块列表消息
type SectorListMsg struct {
	Sectors []types.Sector
	Err     error
}

// SectorStocksMsg 成分股列表消息
type SectorStocksMsg struct {
	Stocks []types.SectorStock
	Err    error
}

// ============================================================================
// Sector Sorting Functions - 排序函数
// ============================================================================

func (m *Model) sortSectorList() {
	if len(m.sectorList) == 0 {
		return
	}

	// 使用 sort package
	sortSectors(m.sectorList, m.sectorSortField, m.sectorSortDirection)
}

func (m *Model) sortSectorStocks() {
	if len(m.sectorStocks) == 0 {
		return
	}

	// 使用 sort package
	sortSectorStocks(m.sectorStocks, m.sectorSortField, m.sectorSortDirection)
}

// ============================================================================
// Sector Helper Functions - 辅助函数
// ============================================================================

// findSectorSortFieldIndex 查找板块排序字段在字段列表中的索引,未找到返回 0
// isSectorList: true=板块列表, false=成分股列表
func (m *Model) findSectorSortFieldIndex(field SortField, isSectorList bool) int {
	var sortFields []SortField

	if isSectorList {
		// 板块列表排序选项
		sortFields = []SortField{
			consts.SortBySectorName,
			consts.SortBySectorChangePercent,
			consts.SortBySectorChange,
			consts.SortBySectorTurnover,
			consts.SortByTurnoverRate,
			consts.SortBySectorRiseCount,
		}
	} else {
		// 成分股列表排序选项
		sortFields = []SortField{
			consts.SortByCode,
			consts.SortByName,
			consts.SortByPrice,
			consts.SortByChangePercent,
			consts.SortByChange,
			consts.SortByVolume,
			consts.SortBySectorTurnover,
			consts.SortByTurnoverRate,
		}
	}

	for i, f := range sortFields {
		if f == field {
			return i
		}
	}

	// 如果当前排序字段未找到,返回 0 (第一个字段)
	return 0
}
