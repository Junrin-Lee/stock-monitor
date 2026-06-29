package watchlist

import (
	"fmt"
	"stock-monitor/internal/types"
	"strings"
)

// ============================================================================
// 视图渲染 - 框架无关的显示逻辑
// ============================================================================

// GetTagsDisplay 获取股票标签的显示字符串（展示层动态组合市场标签和用户标签）
func GetTagsDisplay(stock *types.WatchlistStock, marketTag string) string {
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

// TaggingViewParams 打标签视图参数
type TaggingViewParams struct {
	Title          string
	StockName      string
	StockCode      string
	MarketLabel    string
	MarketTag      string
	CurrentTags    string
	InputPrompt    string
	TagInput       string
	TagInputCursor int
	HelpText       string
}

// RenderTaggingView 渲染打标签视图
func RenderTaggingView(params TaggingViewParams, formatTextWithCursor func(string, int) string) string {
	var s string

	s += params.Title + "\n\n"
	s += fmt.Sprintf("%s: %s (%s)\n", "股票", params.StockName, params.StockCode)
	s += fmt.Sprintf("%s: %s\n", params.MarketLabel, params.MarketTag)
	s += fmt.Sprintf("%s: %s\n\n", "当前标签", params.CurrentTags)
	s += params.InputPrompt + formatTextWithCursor(params.TagInput, params.TagInputCursor) + "\n\n"
	s += params.HelpText

	return s
}

// GroupSelectViewParams 分组选择视图参数
type GroupSelectViewParams struct {
	Title            string
	MarketTagsTitle  string
	UserTagsTitle    string
	AllMarketsText   string
	AllTagsText      string
	NoTagsText       string
	HelpText         string
	MarketTags       []string
	UserTags         []string
	Cursor           int
	FilterStep       int // 0=选择市场, 1=选择标签
	SelectedMarket   types.MarketType
	GetMarketTagName func(types.MarketType) string
}

// RenderGroupSelectView 渲染分组选择视图（两阶段选择，单页显示，支持全部选项）
func RenderGroupSelectView(params GroupSelectViewParams) string {
	var s string

	// 标题
	s += params.Title + "\n\n"

	// 检查是否有可用标签
	if len(params.MarketTags) == 0 && len(params.UserTags) == 0 {
		s += params.NoTagsText + "\n"
		s += "\n" + params.HelpText + "\n"
		return s
	}

	// 渲染市场标签分组
	s += fmt.Sprintf("--- %s ---\n", params.MarketTagsTitle)

	// 第一项：全部市场
	cursor := " "
	checkMark := " "
	if params.FilterStep == 0 && params.Cursor == 0 {
		cursor = ">" // 第一阶段第一项
	}
	if params.FilterStep == 1 && params.SelectedMarket == "" {
		checkMark = "✓" // 第二阶段显示勾选(全部市场)
	}
	s += fmt.Sprintf("%s %s %s\n", cursor, checkMark, params.AllMarketsText)

	// 其余市场标签
	for i, tag := range params.MarketTags {
		cursor = " "
		checkMark = " "

		// 第一阶段：显示光标(索引+1因为全部市场占了0)
		if params.FilterStep == 0 && i+1 == params.Cursor {
			cursor = ">"
		}

		// 第二阶段：显示已选市场的勾选标记
		if params.FilterStep == 1 && params.SelectedMarket != "" {
			selectedMarketTag := params.GetMarketTagName(params.SelectedMarket)
			if tag == selectedMarketTag {
				checkMark = "✓"
			}
		}

		s += fmt.Sprintf("%s %s %s\n", cursor, checkMark, tag)
	}
	s += "\n"

	// 渲染用户标签分组
	if len(params.UserTags) > 0 {
		s += fmt.Sprintf("--- %s ---\n", params.UserTagsTitle)

		// 第一项：全部标签
		cursor = " "
		if params.FilterStep == 1 && params.Cursor == 0 {
			cursor = ">" // 第二阶段第一项
		}
		s += fmt.Sprintf("%s %s\n", cursor, params.AllTagsText)

		// 其余用户标签
		for i, tag := range params.UserTags {
			cursor = " "

			// 仅在第二阶段显示光标(索引+1因为全部标签占了0)
			if params.FilterStep == 1 && i+1 == params.Cursor {
				cursor = ">"
			}

			s += fmt.Sprintf("%s %s\n", cursor, tag)
		}
		s += "\n"
	}

	// 帮助文本
	s += params.HelpText + "\n"

	return s
}

// RenderCurrentFilterStatus 渲染当前组合过滤状态（用于两步选择视图）
func RenderCurrentFilterStatus(marketFilter types.MarketType, userTagFilter string, getMarketTagName func(types.MarketType) string, filterLabel string) string {
	if marketFilter == "" && userTagFilter == "" {
		return ""
	}

	var parts []string
	if marketFilter != "" {
		parts = append(parts, getMarketTagName(marketFilter))
	}
	if userTagFilter != "" {
		parts = append(parts, userTagFilter)
	}

	return fmt.Sprintf("%s: %s\n\n", filterLabel, strings.Join(parts, " + "))
}
