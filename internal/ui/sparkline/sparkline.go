package sparkline

import "github.com/charmbracelet/lipgloss"

// blocks 是 8 级高度的 Unicode 块字符，从低到高
var blocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Generate 使用 Unicode 块字符生成单行迷你趋势图
// prices: 价格序列（至少需要 2 个点）
// width: 目标字符宽度（推荐 12）
// prevClose: 昨收价，用于判断涨跌颜色（<=0 时回退到与序列首点比较）
// upColor: 上涨颜色（A股为红色 #ff4d4d，美股为绿色 #00cc66）
// downColor: 下跌颜色（A股为绿色 #00cc66，美股为红色 #ff4d4d）
// flatColor: 平盘颜色
func Generate(prices []float64, width int, prevClose float64, upColor, downColor, flatColor string) string {
	if len(prices) < 2 {
		return noDataPlaceholder(width)
	}

	// 降采样到目标宽度
	sampled := downsample(prices, width)

	// 找最大最小值用于归一化
	minVal, maxVal := sampled[0], sampled[0]
	for _, p := range sampled {
		if p < minVal {
			minVal = p
		}
		if p > maxVal {
			maxVal = p
		}
	}

	// 选取颜色：与昨收价对比判断涨跌方向（昨收价无效时回退到序列首点）
	ref := sampled[0]
	if prevClose > 0 {
		ref = prevClose
	}
	color := flatColor
	if sampled[len(sampled)-1] > ref {
		color = upColor
	} else if sampled[len(sampled)-1] < ref {
		color = downColor
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

	// 构建字符序列
	result := make([]rune, len(sampled))
	for i, p := range sampled {
		var idx int
		if maxVal == minVal {
			idx = len(blocks) - 1 // 价格完全不变，显示满格（平盘）
		} else {
			// 归一化到 0-7
			normalized := (p - minVal) / (maxVal - minVal)
			idx = int(normalized * 7)
			if idx > 7 {
				idx = 7
			}
		}
		result[i] = blocks[idx]
	}

	return style.Render(string(result))
}

// GenerateWithDefaults 使用语言/市场默认色彩规则生成 Sparkline
// prices: 价格序列
// prevClose: 昨收价，用于判断涨跌颜色（<=0 时回退到与序列首点比较）
// isAShare: 是否为 A 股（A股红涨绿跌；美股/港股绿涨红跌）
func GenerateWithDefaults(prices []float64, prevClose float64, isAShare bool) string {
	var upColor, downColor string
	if isAShare {
		upColor = "#ff4d4d"   // A股：涨 = 红
		downColor = "#00cc66" // A股：跌 = 绿
	} else {
		upColor = "#00cc66"   // 美股/港股：涨 = 绿
		downColor = "#ff4d4d" // 美股/港股：跌 = 红
	}
	return Generate(prices, 12, prevClose, upColor, downColor, "#888888")
}

// Downsample 将 prices 降采样到 targetWidth 个点（等距采样）
// 导出版本，供 loader.go 等外部使用
func Downsample(prices []float64, targetWidth int) []float64 {
	return downsample(prices, targetWidth)
}

// downsample 内部等距采样实现
func downsample(prices []float64, targetWidth int) []float64 {
	if targetWidth <= 1 || len(prices) <= targetWidth {
		return prices
	}
	result := make([]float64, targetWidth)
	step := float64(len(prices)-1) / float64(targetWidth-1)
	for i := 0; i < targetWidth; i++ {
		idx := int(float64(i) * step)
		if idx >= len(prices) {
			idx = len(prices) - 1
		}
		result[i] = prices[idx]
	}
	return result
}

// noDataPlaceholder 无数据时的占位符
func noDataPlaceholder(width int) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	placeholder := make([]rune, width)
	for i := range placeholder {
		placeholder[i] = '─'
	}
	return style.Render(string(placeholder))
}
