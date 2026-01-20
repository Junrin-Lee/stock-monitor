package app

import (
	"fmt"
	"os"
	"stock-monitor/internal/ui"
)

// InitializeApp 执行应用程序初始化
// 创建必要的目录并初始化 UI 组件
func InitializeApp() error {
	// 创建必要目录
	dirs := []string{"data", "cmd/conf", "i18n"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 初始化 UI 列注册表
	ui.InitColumnRegistry()

	return nil
}
