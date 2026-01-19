package main

import (
	"stock-monitor/internal/ui"

	"github.com/jedib0t/go-pretty/v6/text"
)

// ColorUtils 颜色工具类 - 重新导出自 internal/ui
type ColorUtils = ui.ColorUtils

// NewColorUtils 创建颜色工具实例
func NewColorUtils() *ColorUtils {
	return ui.NewColorUtils()
}

// GetSupportedColors 为了向后兼容，提供包级别的函数
func GetSupportedColors() map[string]text.Color {
	return ui.NewColorUtils().GetSupportedColors()
}
