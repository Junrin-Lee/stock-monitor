package table

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"stock-monitor/internal/ui"
	"stock-monitor/internal/consts"
)

// Builder 表格构建器
type Builder struct {
	columns      []*ui.ColumnMetadata
	rows         []table.Row
	headerRow    table.Row
	footerRow    table.Row
	cursor       int
	scrollStart  int
	scrollEnd    int
	isSorted     bool
	sortField    consts.SortField
	sortDir      consts.SortDirection
	style        table.Style
	hasStyle     bool // 标记是否设置了自定义样式
}

// NewBuilder 创建表格构建器
func NewBuilder() *Builder {
	return &Builder{
		columns: make([]*ui.ColumnMetadata, 0),
		rows:    make([]table.Row, 0),
	}
}

// WithColumns 设置列定义
func (b *Builder) WithColumns(columns []*ui.ColumnMetadata) *Builder {
	b.columns = columns
	return b
}

// WithHeader 设置表头
func (b *Builder) WithHeader(row table.Row) *Builder {
	b.headerRow = row
	return b
}

// AddRow 添加数据行
func (b *Builder) AddRow(row table.Row) *Builder {
	b.rows = append(b.rows, row)
	return b
}

// WithFooter 设置表尾
func (b *Builder) WithFooter(row table.Row) *Builder {
	b.footerRow = row
	return b
}

// WithCursor 设置游标位置
func (b *Builder) WithCursor(cursor int) *Builder {
	b.cursor = cursor
	return b
}

// WithScroll 设置滚动范围
func (b *Builder) WithScroll(start, end int) *Builder {
	b.scrollStart = start
	b.scrollEnd = end
	return b
}

// WithSorting 设置排序状态
func (b *Builder) WithSorting(sorted bool, field consts.SortField, dir consts.SortDirection) *Builder {
	b.isSorted = sorted
	b.sortField = field
	b.sortDir = dir
	return b
}

// WithStyle 设置表格样式
func (b *Builder) WithStyle(style table.Style) *Builder {
	b.style = style
	b.hasStyle = true
	return b
}

// Build 构建并渲染表格
func (b *Builder) Build() string {
	t := table.NewWriter()

	// 应用样式
	if b.hasStyle {
		t.SetStyle(b.style)
	} else {
		t.SetStyle(table.StyleLight)
	}

	// 设置表头
	if len(b.headerRow) > 0 {
		t.AppendHeader(b.headerRow)
	}

	// 添加数据行
	for _, row := range b.rows {
		t.AppendRow(row)
	}

	// 设置表尾
	if len(b.footerRow) > 0 {
		t.AppendFooter(b.footerRow)
	}

	return t.Render()
}

// GetVisibleRange 获取可见行范围
func (b *Builder) GetVisibleRange(maxLines int) (start, end int) {
	totalRows := len(b.rows)
	if totalRows <= maxLines {
		return 0, totalRows
	}

	start = b.cursor - maxLines/2
	if start < 0 {
		start = 0
	}

	end = start + maxLines
	if end > totalRows {
		end = totalRows
		start = end - maxLines
		if start < 0 {
			start = 0
		}
	}

	return start, end
}
