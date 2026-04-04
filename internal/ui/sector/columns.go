package sector

import "stock-monitor/internal/consts"

// SectorColumn 板块列表列定义
type SectorColumn struct {
	Header    string           // 列标题
	Width     int              // 列宽度
	SortField consts.SortField // 对应的排序字段
}

// GetSectorColumns 获取板块列表列配置
func GetSectorColumns() []SectorColumn {
	return []SectorColumn{
		{Header: "板块名称", Width: 16, SortField: consts.SortBySectorName},
		{Header: "涨跌幅", Width: 10, SortField: consts.SortBySectorChangePercent},
		{Header: "涨跌额", Width: 10, SortField: consts.SortBySectorChange},
		{Header: "总成交额", Width: 12, SortField: consts.SortBySectorTurnover},
		{Header: "换手率", Width: 10, SortField: consts.SortByTurnoverRate},
		{Header: "上涨", Width: 8, SortField: consts.SortBySectorRiseCount},
		{Header: "下跌", Width: 8, SortField: -1}, // 不支持排序
		{Header: "领涨股", Width: 12, SortField: -1},
		{Header: "领涨幅", Width: 10, SortField: -1},
	}
}

// SectorStockColumn 成分股列表列定义
type SectorStockColumn struct {
	Header    string           // 列标题
	Width     int              // 列宽度
	SortField consts.SortField // 对应的排序字段
}

// GetSectorStockColumns 获取成分股列表列配置
func GetSectorStockColumns() []SectorStockColumn {
	return []SectorStockColumn{
		{Header: "代码", Width: 12, SortField: consts.SortByCode},
		{Header: "名称", Width: 14, SortField: consts.SortByName},
		{Header: "现价", Width: 10, SortField: consts.SortByPrice},
		{Header: "涨跌幅", Width: 10, SortField: consts.SortByChangePercent},
		{Header: "涨跌额", Width: 10, SortField: consts.SortByChange},
		{Header: "成交量", Width: 12, SortField: consts.SortByVolume},
		{Header: "成交额", Width: 12, SortField: consts.SortBySectorTurnover},
		{Header: "换手率", Width: 10, SortField: consts.SortByTurnoverRate},
		{Header: "走势", Width: 14, SortField: -1},
	}
}
