package watchlist

import (
	"sort"
	"stock-monitor/internal/types"
)

// ============================================================================
// 标签管理函数 - 框架无关的核心逻辑
// ============================================================================

// IsMarketTag 判断标签是否为市场标签（用于迁移清理）
func IsMarketTag(tag string) bool {
	marketTags := []string{"A股", "A-Share", "美股", "US Stock", "港股", "HK Stock"}
	for _, mt := range marketTags {
		if tag == mt {
			return true
		}
	}
	return false
}

// RenameTagForAllStocks 批量更新所有使用指定标签的股票，将旧标签替换为新标签
func RenameTagForAllStocks(stocks []types.WatchlistStock, oldTag, newTag string) ([]types.WatchlistStock, int) {
	updatedCount := 0

	for i := range stocks {
		stock := &stocks[i]
		hasOldTag := false

		// 检查股票是否有旧标签
		for j, tag := range stock.Tags {
			if tag == oldTag {
				// 替换为新标签
				stock.Tags[j] = newTag
				hasOldTag = true
			}
		}

		if hasOldTag {
			updatedCount++
		}
	}

	return stocks, updatedCount
}

// GetAvailableTags 获取所有可用的标签（包括市场标签）
func GetAvailableTags(stocks []types.WatchlistStock, getMarketTagName func(types.MarketType) string) []string {
	tagMap := make(map[string]bool)

	// 添加所有市场标签
	for _, stock := range stocks {
		if stock.Market != "" {
			marketTag := getMarketTagName(stock.Market)
			if marketTag != "-" {
				tagMap[marketTag] = true
			}
		}
	}

	// 添加用户自定义标签
	for _, stock := range stocks {
		for _, tag := range stock.Tags {
			if tag != "" && tag != "-" {
				tagMap[tag] = true
			}
		}
	}

	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}

	return tags
}

// GetMarketTags 获取市场标签（固定顺序：A股 -> 美股 -> 港股）
// 只返回自选列表中实际存在的市场标签
func GetMarketTags(stocks []types.WatchlistStock, getMarketTagName func(types.MarketType) string) []string {
	// 检查自选列表中存在哪些市场
	hasMarket := make(map[types.MarketType]bool)
	for _, stock := range stocks {
		if stock.Market != "" {
			hasMarket[stock.Market] = true
		}
	}

	// 按固定顺序返回：中国 -> 美国 -> 香港
	tags := make([]string, 0, 3)

	if hasMarket[types.MarketChina] {
		tags = append(tags, getMarketTagName(types.MarketChina))
	}
	if hasMarket[types.MarketUS] {
		tags = append(tags, getMarketTagName(types.MarketUS))
	}
	if hasMarket[types.MarketHongKong] {
		tags = append(tags, getMarketTagName(types.MarketHongKong))
	}

	return tags
}

// GetUserTags 获取用户自定义标签（字母排序）
func GetUserTags(stocks []types.WatchlistStock) []string {
	// 收集所有唯一的用户标签
	tagSet := make(map[string]bool)
	for _, stock := range stocks {
		for _, tag := range stock.Tags {
			if tag != "" && tag != "-" {
				tagSet[tag] = true
			}
		}
	}

	// 转换为切片
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	// 字母排序
	sort.Strings(tags)

	return tags
}

// GetExistingMarkets 获取自选列表中存在的市场类型（用于组合过滤）
func GetExistingMarkets(stocks []types.WatchlistStock) map[types.MarketType]bool {
	hasMarket := make(map[types.MarketType]bool)
	for _, stock := range stocks {
		if stock.Market != "" {
			hasMarket[stock.Market] = true
		}
	}
	return hasMarket
}

// GetMarketOptions 获取市场选项列表（包含"全部市场"选项）
func GetMarketOptions(stocks []types.WatchlistStock) []types.MarketType {
	options := []types.MarketType{""} // "" 表示"全部市场"

	hasMarket := GetExistingMarkets(stocks)
	if hasMarket[types.MarketChina] {
		options = append(options, types.MarketChina)
	}
	if hasMarket[types.MarketUS] {
		options = append(options, types.MarketUS)
	}
	if hasMarket[types.MarketHongKong] {
		options = append(options, types.MarketHongKong)
	}

	return options
}

// GetUserTagOptions 获取用户标签选项列表（包含"全部标签"选项）
func GetUserTagOptions(stocks []types.WatchlistStock, allTagsText string) []string {
	options := []string{allTagsText} // "全部标签"
	options = append(options, GetUserTags(stocks)...)
	return options
}

// GetTagGroups 获取分类的标签分组（用于分组选择视图）
func GetTagGroups(stocks []types.WatchlistStock, getMarketTagName func(types.MarketType) string, marketTagsText, userTagsText string) []types.TagGroup {
	groups := make([]types.TagGroup, 0, 2)

	// 添加市场标签分组
	marketTags := GetMarketTags(stocks, getMarketTagName)
	if len(marketTags) > 0 {
		groups = append(groups, types.TagGroup{
			Name: marketTagsText,
			Tags: marketTags,
		})
	}

	// 添加用户标签分组
	userTags := GetUserTags(stocks)
	if len(userTags) > 0 {
		groups = append(groups, types.TagGroup{
			Name: userTagsText,
			Tags: userTags,
		})
	}

	return groups
}

// GetTagAtCursor 获取当前光标位置的标签名称
// 如果光标越界则返回空字符串
func GetTagAtCursor(groups []types.TagGroup, cursor int) string {
	currentPos := 0

	for _, group := range groups {
		// 检查光标是否在当前分组的范围内
		groupEndPos := currentPos + len(group.Tags)
		if cursor >= currentPos && cursor < groupEndPos {
			localIndex := cursor - currentPos
			return group.Tags[localIndex]
		}
		currentPos = groupEndPos
	}

	return ""
}

// GetTotalTagCount 获取所有分组的标签总数
func GetTotalTagCount(groups []types.TagGroup) int {
	count := 0
	for _, group := range groups {
		count += len(group.Tags)
	}
	return count
}

// FindTagPositionInGroups 查找标签在分组中的位置
// 返回标签的全局位置索引，如果未找到则返回 -1
func FindTagPositionInGroups(groups []types.TagGroup, tagName string) int {
	if tagName == "" {
		return -1
	}

	currentPos := 0
	for _, group := range groups {
		for _, tag := range group.Tags {
			if tag == tagName {
				return currentPos
			}
			currentPos++
		}
	}

	return -1 // 未找到
}

// FindMarketTagPosition 查找市场标签的位置索引(考虑全部市场选项)
// 返回: 0=全部市场, 1=第一个市场标签, 2=第二个市场标签...
// 如果未找到则返回 -1
func FindMarketTagPosition(stocks []types.WatchlistStock, marketFilter types.MarketType, getMarketTagName func(types.MarketType) string) int {
	if marketFilter == "" {
		return 0 // "全部市场"在索引0
	}

	marketTags := GetMarketTags(stocks, getMarketTagName)
	targetTag := getMarketTagName(marketFilter)

	for i, tag := range marketTags {
		if tag == targetTag {
			return i + 1 // +1 因为全部市场占了索引0
		}
	}

	return -1 // 未找到
}

// FindUserTagPosition 查找用户标签的位置索引(考虑全部标签选项)
// 返回: 0=全部标签, 1=第一个用户标签, 2=第二个用户标签...
// 如果未找到则返回 -1
func FindUserTagPosition(stocks []types.WatchlistStock, tagName string) int {
	if tagName == "" {
		return 0 // "全部标签"在索引0
	}

	userTags := GetUserTags(stocks)
	for i, tag := range userTags {
		if tag == tagName {
			return i + 1 // +1 因为全部标签占了索引0
		}
	}

	return -1 // 未找到
}
