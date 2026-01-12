package main

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================================
// 市场标签函数
// ============================================================================

// getMarketTagName 根据市场类型和语言获取标签名称（展示层使用）
func (m *Model) getMarketTagName(market MarketType) string {
	switch market {
	case MarketChina:
		return m.getText("marketTag.china")
	case MarketUS:
		return m.getText("marketTag.us")
	case MarketHongKong:
		return m.getText("marketTag.hongkong")
	}
	return "-"
}

// isMarketTag 判断标签是否为市场标签（用于迁移清理）
func isMarketTag(tag string) bool {
	marketTags := []string{"A股", "A-Share", "美股", "US Stock", "港股", "HK Stock"}
	for _, mt := range marketTags {
		if tag == mt {
			return true
		}
	}
	return false
}

// ============================================================================
// 标签管理函数
// ============================================================================

// renameTagForAllStocks 批量更新所有使用指定标签的股票，将旧标签替换为新标签
func (m *Model) renameTagForAllStocks(oldTag, newTag string) int {
	updatedCount := 0

	for i := range m.watchlist.Stocks {
		stock := &m.watchlist.Stocks[i]
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

	return updatedCount
}

// getAvailableTags 获取所有可用的标签（包括市场标签）
func (m *Model) getAvailableTags() []string {
	tagMap := make(map[string]bool)

	// 添加所有市场标签
	for _, stock := range m.watchlist.Stocks {
		if stock.Market != "" {
			marketTag := m.getMarketTagName(stock.Market)
			if marketTag != "-" {
				tagMap[marketTag] = true
			}
		}
	}

	// 添加用户自定义标签
	for _, stock := range m.watchlist.Stocks {
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

// getMarketTags 获取市场标签（固定顺序：A股 -> 美股 -> 港股）- v5.6
// 只返回自选列表中实际存在的市场标签
func (m *Model) getMarketTags() []string {
	// 检查自选列表中存在哪些市场
	hasMarket := make(map[MarketType]bool)
	for _, stock := range m.watchlist.Stocks {
		if stock.Market != "" {
			hasMarket[stock.Market] = true
		}
	}

	// 按固定顺序返回：中国 -> 美国 -> 香港
	tags := make([]string, 0, 3)

	if hasMarket[MarketChina] {
		tags = append(tags, m.getMarketTagName(MarketChina))
	}
	if hasMarket[MarketUS] {
		tags = append(tags, m.getMarketTagName(MarketUS))
	}
	if hasMarket[MarketHongKong] {
		tags = append(tags, m.getMarketTagName(MarketHongKong))
	}

	return tags
}

// getUserTags 获取用户自定义标签（字母排序）- v5.6
func (m *Model) getUserTags() []string {
	// 收集所有唯一的用户标签
	tagSet := make(map[string]bool)
	for _, stock := range m.watchlist.Stocks {
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

// getExistingMarkets 获取自选列表中存在的市场类型（用于组合过滤）
func (m *Model) getExistingMarkets() map[MarketType]bool {
	hasMarket := make(map[MarketType]bool)
	for _, stock := range m.watchlist.Stocks {
		if stock.Market != "" {
			hasMarket[stock.Market] = true
		}
	}
	return hasMarket
}

// getMarketOptions 获取市场选项列表（包含"全部市场"选项）
func (m *Model) getMarketOptions() []MarketType {
	options := []MarketType{""} // "" 表示"全部市场"

	hasMarket := m.getExistingMarkets()
	if hasMarket[MarketChina] {
		options = append(options, MarketChina)
	}
	if hasMarket[MarketUS] {
		options = append(options, MarketUS)
	}
	if hasMarket[MarketHongKong] {
		options = append(options, MarketHongKong)
	}

	return options
}

// getUserTagOptions 获取用户标签选项列表（包含"全部标签"选项）
func (m *Model) getUserTagOptions() []string {
	options := []string{m.getText("filter.allTags")} // "全部标签"
	options = append(options, m.getUserTags()...)
	return options
}

// getTagGroups 获取分类的标签分组（用于分组选择视图）- v5.6
func (m *Model) getTagGroups() []TagGroup {
	groups := make([]TagGroup, 0, 2)

	// 添加市场标签分组
	marketTags := m.getMarketTags()
	if len(marketTags) > 0 {
		groups = append(groups, TagGroup{
			Name: m.getText("group.marketTags"),
			Tags: marketTags,
		})
	}

	// 添加用户标签分组
	userTags := m.getUserTags()
	if len(userTags) > 0 {
		groups = append(groups, TagGroup{
			Name: m.getText("group.userTags"),
			Tags: userTags,
		})
	}

	return groups
}

// getTagAtCursor 获取当前光标位置的标签名称 - v5.6
// 如果光标越界则返回空字符串
func (m *Model) getTagAtCursor() string {
	currentPos := 0

	for _, group := range m.tagGroups {
		// 检查光标是否在当前分组的范围内
		groupEndPos := currentPos + len(group.Tags)
		if m.cursor >= currentPos && m.cursor < groupEndPos {
			localIndex := m.cursor - currentPos
			return group.Tags[localIndex]
		}
		currentPos = groupEndPos
	}

	return ""
}

// getTotalTagCount 获取所有分组的标签总数 - v5.6
func (m *Model) getTotalTagCount() int {
	count := 0
	for _, group := range m.tagGroups {
		count += len(group.Tags)
	}
	return count
}

// findTagPositionInGroups 查找标签在分组中的位置 - v5.6
// 返回标签的全局位置索引，如果未找到则返回 -1
func (m *Model) findTagPositionInGroups(tagName string) int {
	if tagName == "" {
		return -1
	}

	currentPos := 0
	for _, group := range m.tagGroups {
		for _, tag := range group.Tags {
			if tag == tagName {
				return currentPos
			}
			currentPos++
		}
	}

	return -1 // 未找到
}

// ============================================================================
// WatchlistStock 标签方法
// ============================================================================

// hasTag 检查股票是否包含指定标签
func (stock *WatchlistStock) hasTag(tag string) bool {
	for _, t := range stock.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// addTag 添加标签到股票（避免重复）
func (stock *WatchlistStock) addTag(tag string) {
	if tag == "" || tag == "-" {
		return
	}
	if !stock.hasTag(tag) {
		stock.Tags = append(stock.Tags, tag)
	}
}

// removeTag 移除股票的标签
func (stock *WatchlistStock) removeTag(tag string) {
	for i, t := range stock.Tags {
		if t == tag {
			stock.Tags = append(stock.Tags[:i], stock.Tags[i+1:]...)
			break
		}
	}
}

// getTagsDisplay 获取股票标签的显示字符串（展示层动态组合市场标签和用户标签）
func (stock *WatchlistStock) getTagsDisplay(m *Model) string {
	// 从 market 字段生成市场标签
	marketTag := m.getMarketTagName(stock.Market)

	// 过滤用户自定义标签
	var validTags []string
	for _, tag := range stock.Tags {
		if tag != "" && tag != "-" {
			validTags = append(validTags, tag)
		}
	}

	// 组合市场标签 + 用户标签（市场标签优先）
	allTags := []string{marketTag}
	allTags = append(allTags, validTags...)

	// 如果只有市场标签且为 "-"，返回 "-"
	if len(allTags) == 1 && allTags[0] == "-" {
		return "-"
	}

	// 如果只有一个标签（市场标签）
	if len(allTags) == 1 {
		return allTags[0]
	}

	// 多个标签时，用逗号分隔，但如果太长则显示数量
	display := strings.Join(allTags, ",")
	totalLen := len(display)

	if totalLen > 15 {
		return fmt.Sprintf("%s+%d", allTags[0], len(allTags)-1)
	}

	return display
}

// ============================================================================
// 自选列表过滤和缓存
// ============================================================================

// getFilteredWatchlist 根据市场和用户标签过滤自选股票（支持组合过滤，使用 AND 逻辑）
func (m *Model) getFilteredWatchlist() []WatchlistStock {
	// 如果没有任何过滤条件，直接返回完整列表
	if m.selectedMarketFilter == "" && m.selectedUserTagFilter == "" {
		return m.watchlist.Stocks
	}

	// 检查缓存是否有效
	if m.isFilteredWatchlistValid &&
		m.cachedFilterMarket == m.selectedMarketFilter &&
		m.cachedFilterUserTag == m.selectedUserTagFilter {
		return m.cachedFilteredWatchlist
	}

	// 重新计算过滤结果并缓存（应用 AND 逻辑）
	var filtered []WatchlistStock
	for _, stock := range m.watchlist.Stocks {
		// 检查市场过滤条件
		if m.selectedMarketFilter != "" && stock.Market != m.selectedMarketFilter {
			continue // 市场不匹配，跳过
		}

		// 检查用户标签过滤条件
		if m.selectedUserTagFilter != "" && !stock.hasTag(m.selectedUserTagFilter) {
			continue // 用户标签不匹配，跳过
		}

		// 两个条件都满足（或未设置），加入过滤结果
		filtered = append(filtered, stock)
	}

	// 更新缓存
	m.cachedFilteredWatchlist = filtered
	m.cachedFilterMarket = m.selectedMarketFilter
	m.cachedFilterUserTag = m.selectedUserTagFilter
	m.isFilteredWatchlistValid = true

	return filtered
}

// invalidateWatchlistCache 使缓存失效的辅助函数
func (m *Model) invalidateWatchlistCache() {
	m.isFilteredWatchlistValid = false
	m.cachedFilteredWatchlist = nil
	m.cachedFilterTag = ""     // 保留用于向后兼容
	m.cachedFilterMarket = ""  // 清除市场过滤缓存
	m.cachedFilterUserTag = "" // 清除用户标签过滤缓存
}

// ============================================================================
// 自选列表光标和滚动管理
// ============================================================================

// resetWatchlistCursor 重置自选列表游标到第一只股票（基于过滤后的列表）
func (m *Model) resetWatchlistCursor() {
	filteredStocks := m.getFilteredWatchlist()
	if len(filteredStocks) > 0 {
		m.watchlistCursor = 0
		maxWatchlistLines := m.config.Display.MaxLines
		if len(filteredStocks) > maxWatchlistLines {
			// 显示前N条：滚动位置设置为显示从索引0开始的N条
			m.watchlistScrollPos = len(filteredStocks) - maxWatchlistLines
		} else {
			// 股票数量不超过显示行数，显示全部
			m.watchlistScrollPos = 0
		}
	} else {
		// 没有股票时重置
		m.watchlistCursor = 0
		m.watchlistScrollPos = 0
	}
}

// adjustWatchlistScroll 调整自选列表滚动位置（基于过滤后的列表）
func (m *Model) adjustWatchlistScroll(filteredStocks []WatchlistStock) {
	maxWatchlistLines := m.config.Display.MaxLines
	totalStocks := len(filteredStocks)

	if totalStocks <= maxWatchlistLines {
		m.watchlistScrollPos = 0
		return
	}

	// 确保光标在可见范围内
	endIndex := totalStocks - m.watchlistScrollPos
	startIndex := endIndex - maxWatchlistLines
	if startIndex < 0 {
		startIndex = 0
	}

	// 如果光标超出可见范围的上边界，调整滚动位置
	if m.watchlistCursor < startIndex {
		m.watchlistScrollPos = totalStocks - m.watchlistCursor - maxWatchlistLines
		if m.watchlistScrollPos < 0 {
			m.watchlistScrollPos = 0
		}
	}

	// 如果光标超出可见范围的下边界，调整滚动位置
	if m.watchlistCursor >= endIndex {
		m.watchlistScrollPos = totalStocks - m.watchlistCursor - 1
		if m.watchlistScrollPos < 0 {
			m.watchlistScrollPos = 0
		}
	}
}

// ============================================================================
// 自选列表股票管理
// ============================================================================

// isStockInWatchlist 检查股票是否已在自选列表中
func (m *Model) isStockInWatchlist(code string) bool {
	for _, stock := range m.watchlist.Stocks {
		if stock.Code == code {
			return true
		}
	}
	return false
}

// addToWatchlist 添加股票到自选列表
func (m *Model) addToWatchlist(code, name string) bool {
	if m.isStockInWatchlist(code) {
		return false // 已在列表中
	}

	// 识别市场类型
	market := getMarketType(code)

	watchStock := WatchlistStock{
		Code:   code,
		Name:   name,
		Market: market,     // 保存市场类型
		Tags:   []string{}, // 初始为空，不包含市场标签
	}
	// 将新股票插入到列表首位，而不是末尾
	m.watchlist.Stocks = append([]WatchlistStock{watchStock}, m.watchlist.Stocks...)
	m.invalidateWatchlistCache() // 使缓存失效
	m.watchlistIsSorted = false  // 添加自选股票后重置自选列表排序状态
	m.saveWatchlist()
	return true
}

// removeFromWatchlist 从自选列表删除股票
func (m *Model) removeFromWatchlist(index int) {
	if index >= 0 && index < len(m.watchlist.Stocks) {
		m.watchlist.Stocks = append(m.watchlist.Stocks[:index], m.watchlist.Stocks[index+1:]...)
		m.invalidateWatchlistCache() // 使缓存失效
		m.saveWatchlist()
		m.watchlistIsSorted = false // 删除自选股票后重置自选列表排序状态
	}
}

// ============================================================================
// 标签相关状态处理器
// ============================================================================

// handleWatchlistTagging 处理自选股票打标签
func (m *Model) handleWatchlistTagging(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.tagInput == "" {
			// 回到标签管理界面
			m.availableTags = m.getAvailableTags()
			m.state = WatchlistTagManage
			m.tagManageCursor = 0
			return m, nil
		}

		// 更新当前选中股票的标签（基于过滤后的列表）
		filteredStocks := m.getFilteredWatchlist()
		if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
			stockToTag := filteredStocks[m.watchlistCursor]

			// 在原始列表中找到该股票并添加标签
			for i, stock := range m.watchlist.Stocks {
				if stock.Code == stockToTag.Code {
					// 处理多个标签（逗号分隔）
					newTags := strings.Split(m.tagInput, ",")
					for _, tag := range newTags {
						tag = strings.TrimSpace(tag)
						if tag != "" && tag != "-" {
							m.watchlist.Stocks[i].addTag(tag)
						}
					}
					// 如果没有有效标签，确保至少有默认标签
					if len(m.watchlist.Stocks[i].Tags) == 0 {
						m.watchlist.Stocks[i].Tags = []string{"-"}
					}

					// 更新当前股票标签列表
					m.currentStockTags = make([]string, 0)
					for _, tag := range m.watchlist.Stocks[i].Tags {
						if tag != "" && tag != "-" {
							m.currentStockTags = append(m.currentStockTags, tag)
						}
					}
					break
				}
			}

			m.invalidateWatchlistCache() // 使缓存失效
			m.saveWatchlist()

			if m.language == Chinese {
				m.message = fmt.Sprintf("已为 %s 添加标签: %s",
					stockToTag.Name, m.tagInput)
			} else {
				m.message = fmt.Sprintf("Added tags to %s: %s",
					stockToTag.Name, m.tagInput)
			}
		}

		// 回到标签管理界面，更新可用标签列表
		m.availableTags = m.getAvailableTags()
		m.state = WatchlistTagManage
		m.tagManageCursor = 0
		m.tagInput = ""
		m.tagInputCursor = 0
		return m, nil
	case "esc", "q":
		// 回到标签管理界面
		m.availableTags = m.getAvailableTags()
		m.state = WatchlistTagManage
		m.tagManageCursor = 0
		m.tagInput = ""
		m.tagInputCursor = 0
		m.message = ""
		return m, nil
	case "left", "ctrl+b":
		// 光标左移
		if m.tagInputCursor > 0 {
			m.tagInputCursor--
		}
		return m, nil
	case "right", "ctrl+f":
		// 光标右移
		runes := []rune(m.tagInput)
		if m.tagInputCursor < len(runes) {
			m.tagInputCursor++
		}
		return m, nil
	case "home", "ctrl+a":
		// 光标移到开头
		m.tagInputCursor = 0
		return m, nil
	case "end", "ctrl+e":
		// 光标移到末尾
		m.tagInputCursor = len([]rune(m.tagInput))
		return m, nil
	case "backspace":
		// 删除光标前的字符
		m.tagInput, m.tagInputCursor = deleteRuneBeforeCursor(m.tagInput, m.tagInputCursor)
		return m, nil
	case "delete", "ctrl+d":
		// 删除光标处的字符
		m.tagInput, m.tagInputCursor = deleteRuneAtCursor(m.tagInput, m.tagInputCursor)
		return m, nil
	default:
		// 处理文本输入
		str := msg.String()
		if len(str) > 0 && str != "\n" && str != "\r" && !isControlKey(str) {
			m.tagInput, m.tagInputCursor = insertStringAtCursor(m.tagInput, m.tagInputCursor, str)
		}
		return m, nil
	}
}

// handleWatchlistGroupSelect 处理自选股票分组选择（两步选择：市场 → 用户标签）
func (m *Model) handleWatchlistGroupSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.filterSelectionStep {
	case 0:
		return m.handleMarketSelection(msg)
	case 1:
		return m.handleUserTagSelection(msg)
	}
	return m, nil
}

// handleMarketSelection 处理市场选择（第一步）
func (m *Model) handleMarketSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	markets := m.getMarketOptions()

	switch msg.String() {
	case "enter":
		// 选择市场并进入第二步
		if m.cursor >= 0 && m.cursor < len(markets) {
			m.selectedMarketFilter = markets[m.cursor]
		}
		m.filterSelectionStep = 1
		m.cursor = 0 // 重置光标用于第二步
		return m, nil

	case "esc", "q":
		// 取消并返回自选列表
		m.filterSelectionStep = 0
		m.state = WatchlistViewing
		m.message = ""
		return m, m.tickCmd()

	case "c":
		// 清除所有过滤
		m.selectedMarketFilter = ""
		m.selectedUserTagFilter = ""
		m.invalidateWatchlistCache()
		m.filterSelectionStep = 0
		m.state = WatchlistViewing
		m.resetWatchlistCursor()
		m.message = ""
		return m, m.tickCmd()

	case "up", "k", "w":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.cursor < len(markets)-1 {
			m.cursor++
		}
		return m, nil
	}

	return m, nil
}

// handleUserTagSelection 处理用户标签选择（第二步）
func (m *Model) handleUserTagSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tags := m.getUserTagOptions()

	switch msg.String() {
	case "enter":
		// 选择标签并应用组合过滤
		if m.cursor == 0 {
			m.selectedUserTagFilter = "" // "全部标签" 选项
		} else if m.cursor > 0 && m.cursor < len(tags) {
			// 跳过第一个"全部标签"选项，获取实际标签
			userTags := m.getUserTags()
			if m.cursor-1 < len(userTags) {
				m.selectedUserTagFilter = userTags[m.cursor-1]
			}
		}

		// 保存选择用于位置记忆
		m.lastSelectedGroupTag = m.selectedUserTagFilter

		m.invalidateWatchlistCache()
		m.filterSelectionStep = 0
		m.state = WatchlistViewing
		m.resetWatchlistCursor()
		m.message = ""
		return m, m.tickCmd()

	case "b", "backspace":
		// 返回第一步（市场选择）
		m.filterSelectionStep = 0
		m.cursor = 0
		return m, nil

	case "esc", "q":
		// 取消并返回自选列表（保持现有过滤）
		m.filterSelectionStep = 0
		m.state = WatchlistViewing
		m.message = ""
		return m, m.tickCmd()

	case "c":
		// 清除所有过滤
		m.selectedMarketFilter = ""
		m.selectedUserTagFilter = ""
		m.invalidateWatchlistCache()
		m.filterSelectionStep = 0
		m.state = WatchlistViewing
		m.resetWatchlistCursor()
		m.message = ""
		return m, m.tickCmd()

	case "up", "k", "w":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j", "s":
		if m.cursor < len(tags)-1 {
			m.cursor++
		}
		return m, nil
	}

	return m, nil
}

// ============================================================================
// 标签相关视图函数
// ============================================================================

// viewWatchlistTagging 打标签视图
func (m *Model) viewWatchlistTagging() string {
	var s string

	if m.language == Chinese {
		s += "=== 设置标签 ===\n\n"
	} else {
		s += "=== Set Tag ===\n\n"
	}

	filteredStocks := m.getFilteredWatchlist()
	if m.watchlistCursor >= 0 && m.watchlistCursor < len(filteredStocks) {
		stock := filteredStocks[m.watchlistCursor]
		marketTag := m.getMarketTagName(stock.Market)

		if m.language == Chinese {
			s += fmt.Sprintf("股票: %s (%s)\n", stock.Name, stock.Code)
			s += fmt.Sprintf("%s: %s\n", m.getText("marketInfo"), marketTag)
			s += fmt.Sprintf("当前标签: %s\n\n", stock.getTagsDisplay(m))
			s += "请输入新标签(多个标签用逗号分隔): " + formatTextWithCursor(m.tagInput, m.tagInputCursor) + "\n\n"
			s += "操作: ←/→移动光标, Enter确认, ESC/Q取消, Home/End跳转首尾"
		} else {
			s += fmt.Sprintf("Stock: %s (%s)\n", stock.Name, stock.Code)
			s += fmt.Sprintf("%s: %s\n", m.getText("marketInfo"), marketTag)
			s += fmt.Sprintf("Current tags: %s\n\n", stock.getTagsDisplay(m))
			s += "Enter new tags (comma separated): " + formatTextWithCursor(m.tagInput, m.tagInputCursor) + "\n\n"
			s += "Actions: ←/→ move cursor, Enter confirm, ESC/Q cancel, Home/End jump"
		}
	}

	return s
}

// renderCurrentFilterStatus 渲染当前组合过滤状态（用于两步选择视图）
func (m *Model) renderCurrentFilterStatus() string {
	if m.selectedMarketFilter == "" && m.selectedUserTagFilter == "" {
		return ""
	}

	var parts []string
	if m.selectedMarketFilter != "" {
		parts = append(parts, m.getMarketTagName(m.selectedMarketFilter))
	}
	if m.selectedUserTagFilter != "" {
		parts = append(parts, m.selectedUserTagFilter)
	}

	return fmt.Sprintf("%s: %s\n\n", m.getText("watchlist.currentFilter"), strings.Join(parts, " + "))
}

// viewWatchlistGroupSelectStep1 市场选择视图（第一步）
func (m *Model) viewWatchlistGroupSelectStep1() string {
	var s string
	s += m.getText("watchlist.selectGroup") + "\n\n"

	// 显示当前组合过滤状态
	s += m.renderCurrentFilterStatus()

	// 步骤指示
	s += fmt.Sprintf("--- %s ---\n", m.getText("filter.step1.selectMarket"))

	// 市场选项：全部 + 存在的市场
	markets := []struct {
		value MarketType
		label string
	}{
		{"", m.getText("filter.allMarkets")},
	}

	// 添加实际存在的市场
	hasMarket := m.getExistingMarkets()
	if hasMarket[MarketChina] {
		markets = append(markets, struct {
			value MarketType
			label string
		}{MarketChina, m.getText("marketTag.china")})
	}
	if hasMarket[MarketUS] {
		markets = append(markets, struct {
			value MarketType
			label string
		}{MarketUS, m.getText("marketTag.us")})
	}
	if hasMarket[MarketHongKong] {
		markets = append(markets, struct {
			value MarketType
			label string
		}{MarketHongKong, m.getText("marketTag.hongkong")})
	}

	// 渲染市场选项
	for i, market := range markets {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, market.label)
	}

	s += "\n" + m.getText("filter.step1.helpText") + "\n"
	return s
}

// viewWatchlistGroupSelectStep2 用户标签选择视图（第二步）
func (m *Model) viewWatchlistGroupSelectStep2() string {
	var s string
	s += m.getText("watchlist.selectGroup") + "\n\n"

	// 显示当前组合过滤状态
	s += m.renderCurrentFilterStatus()

	// 显示已选市场
	marketLabel := m.getText("filter.allMarkets")
	if m.selectedMarketFilter != "" {
		marketLabel = m.getMarketTagName(m.selectedMarketFilter)
	}
	s += fmt.Sprintf("%s: %s\n\n", m.getText("filter.selectedMarket"), marketLabel)

	// 步骤指示
	s += fmt.Sprintf("--- %s ---\n", m.getText("filter.step2.selectTag"))

	// 用户标签选项：全部 + 用户标签
	tags := m.getUserTagOptions()

	// 渲染标签选项
	for i, tag := range tags {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, tag)
	}

	s += "\n" + m.getText("filter.step2.helpText") + "\n"
	return s
}

// viewWatchlistGroupSelect 分组选择视图 (v5.6: 区分市场分组和用户标签)
func (m *Model) viewWatchlistGroupSelect() string {
	var s string

	// 标题
	s += m.getText("watchlist.selectGroup") + "\n\n"

	// 显示当前过滤状态
	if m.selectedTag != "" {
		s += fmt.Sprintf("%s: %s\n\n", m.getText("watchlist.currentFilter"), m.selectedTag)
	}

	// 检查是否有可用标签
	if len(m.tagGroups) == 0 {
		s += m.getText("watchlist.noTags") + "\n"
		s += "\n" + m.getText("group.helpText") + "\n"
		return s
	}

	// 渲染每个分组及其标签
	currentPos := 0
	for groupIdx, group := range m.tagGroups {
		// 分组之间添加空行
		if groupIdx > 0 {
			s += "\n"
		}

		// 分组标题（带分隔符）
		s += fmt.Sprintf("--- %s ---\n", group.Name)

		// 渲染该分组的标签
		for _, tag := range group.Tags {
			cursor := " "
			if currentPos == m.cursor {
				cursor = ">"
			}
			s += fmt.Sprintf("%s %s\n", cursor, tag)
			currentPos++
		}
	}

	// 帮助文本
	s += "\n" + m.getText("group.helpText") + "\n"

	return s
}
