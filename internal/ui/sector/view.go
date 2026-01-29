package sector

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"

	"stock-monitor/internal/consts"
	"stock-monitor/internal/types"
	"stock-monitor/internal/ui"
)

// RenderSectorList 渲染板块列表视图
func RenderSectorList(
	sectors []types.Sector,
	sectorType types.SectorType,
	cursor int,
	scrollPos int,
	maxLines int,
	sortField consts.SortField,
	sortDir consts.SortDirection,
	language consts.Language,
	updateTime string,
) string {
	var sb strings.Builder

	// 标题
	sb.WriteString("=== 板块行情 ===\n")
	sb.WriteString(fmt.Sprintf("更新时间(5s): %s\n", updateTime))

	// 当前板块类型
	var typeText, switchHint string
	if sectorType == types.SectorTypeIndustry {
		typeText = "行业板块"
		switchHint = "按T切换至概念板块"
	} else {
		typeText = "概念板块"
		switchHint = "按T切换至行业板块"
	}
	sb.WriteString(fmt.Sprintf("当前: %s (%s)\n\n", typeText, switchHint))

	// 板块列表标题
	total := len(sectors)

	sb.WriteString(fmt.Sprintf(" 板块列表 (%d/%d)\n\n",
		cursor+1, total))

	// 如果没有数据
	if len(sectors) == 0 {
		sb.WriteString("暂无板块数据\n\n")
		sb.WriteString("↑/↓:移动光标 | Enter:查看成分股 | T:切换板块类型 | S:排序 | ESC/Q/M:返回主菜单\n")
		return sb.String()
	}

	// 创建表格
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// 表头
	columns := GetSectorColumns()
	headers := make(table.Row, 0, len(columns)+1)
	headers = append(headers, "") // 光标列

	for _, col := range columns {
		header := col.Header
		// 如果是排序字段,添加排序指示器
		if col.SortField == sortField {
			if sortDir == consts.SortAsc {
				header += "▲"
			} else {
				header += "▼"
			}
		}
		headers = append(headers, header)
	}
	t.AppendHeader(headers)

	// 数据行
	formatter := ui.NewFormatter(language)
	end := scrollPos + maxLines
	if end > len(sectors) {
		end = len(sectors)
	}

	for i := scrollPos; i < end; i++ {
		sector := sectors[i]
		row := make(table.Row, 0, len(columns)+1)

		// 光标
		if i == cursor {
			row = append(row, "►")
		} else {
			row = append(row, "")
		}

		// 板块名称
		row = append(row, sector.Name)

		// 涨跌幅
		row = append(row, formatter.FormatProfitRateWithColor(sector.ChangePercent))

		// 涨跌额
		row = append(row, formatter.FormatProfitWithColor(sector.Change))

		// 总成交额
		row = append(row, formatTurnover(sector.Turnover))

		// 换手率
		row = append(row, fmt.Sprintf("%.2f%%", sector.TurnoverRate))

		// 上涨
		row = append(row, fmt.Sprintf("%d", sector.RiseCount))

		// 下跌
		row = append(row, fmt.Sprintf("%d", sector.FallCount))

		// 领涨股
		if sector.LeaderName != "" {
			row = append(row, sector.LeaderName)
		} else {
			row = append(row, "-")
		}

		// 领涨幅
		if sector.LeaderCode != "" {
			row = append(row, formatter.FormatProfitRateWithColor(sector.LeaderChange))
		} else {
			row = append(row, "-")
		}

		t.AppendRow(row)
		// 添加行分隔符（最后一行除外）
		if i < end-1 {
			t.AppendSeparator()
		}
	}

	sb.WriteString(t.Render())
	sb.WriteString("\n\n")

	// 操作提示
	sb.WriteString("↑/↓:移动光标 | Enter:查看成分股 | T:切换板块类型 | S:排序 | ESC/Q/M:返回主菜单\n")

	return sb.String()
}

// RenderSectorStockList 渲染板块成分股列表视图
func RenderSectorStockList(
	sectorName string,
	sectorInfo *types.Sector,
	stocks []types.SectorStock,
	cursor int,
	scrollPos int,
	maxLines int,
	sortField consts.SortField,
	sortDir consts.SortDirection,
	language consts.Language,
	updateTime string,
) string {
	var sb strings.Builder

	// 标题
	sb.WriteString(fmt.Sprintf("=== %s - 成分股 ===\n", sectorName))
	sb.WriteString(fmt.Sprintf("更新时间(5s): %s\n\n", updateTime))

	// 板块概况
	if sectorInfo != nil {
		sb.WriteString(" 板块概况\n")

		overviewTable := table.NewWriter()
		overviewTable.SetStyle(table.StyleLight)

		// 表头
		overviewTable.AppendHeader(table.Row{"涨跌幅", "成交额", "上涨", "下跌", "换手率"})

		// 数据行
		overviewTable.AppendRow(table.Row{
			colorChangePercent(sectorInfo.ChangePercent, language),
			formatTurnover(sectorInfo.Turnover),
			fmt.Sprintf("%d家", sectorInfo.RiseCount),
			fmt.Sprintf("%d家", sectorInfo.FallCount),
			fmt.Sprintf("%.2f%%", sectorInfo.TurnoverRate),
		})

		sb.WriteString(overviewTable.Render())
		sb.WriteString("\n\n")
	}

	// 成分股列表标题
	total := len(stocks)

	sb.WriteString(fmt.Sprintf(" 成分股列表 (%d/%d)\n\n",
		cursor+1, total))

	// 如果没有数据
	if len(stocks) == 0 {
		sb.WriteString("暂无成分股数据\n\n")
		sb.WriteString("↑/↓:移动光标 | A:添加到自选 | V:查看分时图 | S:排序 | ESC/Q:返回板块列表\n")
		return sb.String()
	}

	// 创建表格
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// 表头
	columns := GetSectorStockColumns()
	headers := make(table.Row, 0, len(columns)+1)
	headers = append(headers, "") // 光标列

	for _, col := range columns {
		header := col.Header
		// 如果是排序字段,添加排序指示器
		if col.SortField == sortField {
			if sortDir == consts.SortAsc {
				header += "▲"
			} else {
				header += "▼"
			}
		}
		headers = append(headers, header)
	}
	t.AppendHeader(headers)

	// 数据行
	formatter := ui.NewFormatter(language)
	end := scrollPos + maxLines
	if end > len(stocks) {
		end = len(stocks)
	}

	for i := scrollPos; i < end; i++ {
		stock := stocks[i]
		row := make(table.Row, 0, len(columns)+1)

		// 光标
		if i == cursor {
			row = append(row, "►")
		} else {
			row = append(row, "")
		}

		// 代码
		row = append(row, stock.Code)

		// 名称
		row = append(row, stock.Name)

		// 现价
		row = append(row, fmt.Sprintf("%.3f", stock.Price))

		// 涨跌幅
		row = append(row, formatter.FormatProfitRateWithColor(stock.ChangePercent))

		// 涨跌额
		row = append(row, formatter.FormatProfitWithColor(stock.Change))

		// 成交量
		row = append(row, ui.FormatVolumeZh(stock.Volume))

		// 成交额
		row = append(row, formatTurnover(stock.Turnover))

		// 换手率
		row = append(row, fmt.Sprintf("%.2f%%", stock.TurnoverRate))

		t.AppendRow(row)
		// 添加行分隔符（最后一行除外）
		if i < end-1 {
			t.AppendSeparator()
		}
	}

	sb.WriteString(t.Render())
	sb.WriteString("\n\n")

	// 操作提示
	sb.WriteString("↑/↓:移动光标 | A:添加到自选 | V:查看分时图 | S:排序 | ESC/Q:返回板块列表\n")

	return sb.String()
}

// formatTurnover 格式化成交额（亿元）
func formatTurnover(turnover float64) string {
	if turnover >= 100000000 {
		return fmt.Sprintf("%.2f亿", turnover/100000000)
	} else if turnover >= 10000 {
		return fmt.Sprintf("%.2f万", turnover/10000)
	}
	return fmt.Sprintf("%.2f", turnover)
}

// colorChangePercent 根据涨跌幅上色
func colorChangePercent(changePercent float64, language consts.Language) string {
	formatter := ui.NewFormatter(language)
	return formatter.FormatProfitRateWithColor(changePercent)
}
