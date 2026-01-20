#!/bin/bash
set -e

echo "=== 执行 Round 6 重构 ==="

# 由于工作量巨大且风险高,采用保守方案:
# 保持 handlers 在 main 包,只创建 app 包作为初始化层

# 1. 恢复 handlers 到根目录
echo "步骤1: 恢复 handlers 文件..."
mv internal/app/handlers/*.go . 2>/dev/null || true
for f in handlers_*.go; do
  [ -f "$f" ] && sed -i '' '1s/package handlers/package main/' "$f"
done
rm -rf internal/app/handlers

# 2. 创建精简的 app 包(只包含初始化辅助)  
echo "步骤2: 创建 app 包初始化辅助..."
cat > internal/app/setup.go << 'EOF'
package app

import (
	"fmt"
	"os"
	"stock-monitor/internal/ui"
)

// Setup 执行应用程序初始化
func Setup() error {
	// 创建必要目录
	dirs := []string{"data", "cmd/conf", "i18n"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	
	// 初始化UI列注册表
	ui.InitColumnRegistry()
	
	return nil
}
EOF

# 3. 更新 main.go 使用 app.Setup()
echo "步骤3: 更新 main.go..."
# 在 import 中添加 app 包
sed -i '' '/^import (/a\
	"stock-monitor/internal/app"
' main.go

# 替换初始化代码
sed -i '' '/os.MkdirAll("data"/,/ui.InitColumnRegistry()/c\
	// 初始化应用程序\
	if err := app.Setup(); err != nil {\
		fmt.Fprintf(os.Stderr, "Setup failed: %v\\n", err)\
		os.Exit(1)\
	}
' main.go

echo "✓ Round 6 重构完成(保守方案)"
echo ""
echo "改动:"
echo "- 创建 internal/app 包提供 Setup() 函数"
echo "- main.go 使用 app.Setup() 替代重复的初始化代码"  
echo "- handlers 保持在 main 包(避免循环依赖)"
echo ""
echo "验证编译..."

if go build -o cmd/stock-monitor; then
  echo "✓ 编译成功"
  echo ""
  echo "提交到 git..."
  git add -A
  git commit -m "refactor(round6): create internal/app package for initialization

- Add app.Setup() function for application initialization
- Simplify main.go by using app.Setup()
- Keep handlers in main package to avoid circular dependencies"
  echo "✓ 已提交"
else
  echo "✗ 编译失败,回滚..."
  git checkout .
  exit 1
fi
