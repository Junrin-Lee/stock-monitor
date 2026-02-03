# Stock Monitor - Makefile
# 提供便捷的测试和构建命令

.PHONY: help test test-verbose test-coverage test-race test-bench clean build run

# 默认目标 - 显示帮助信息
help:
	@echo "Stock Monitor - 可用命令:"
	@echo ""
	@echo "  make test          - 运行所有单元测试"
	@echo "  make test-v        - 运行测试(详细输出)"
	@echo "  make test-coverage - 运行测试并生成覆盖率报告"
	@echo "  make test-race     - 运行测试(带并发竞争检测)"
	@echo "  make test-bench    - 运行性能基准测试"
	@echo "  make clean         - 清理测试缓存和临时文件"
	@echo "  make build         - 编译项目"
	@echo "  make run           - 运行程序"
	@echo ""

# 运行所有单元测试
test:
	@echo "🧪 运行单元测试..."
	@go test ./internal/api/... ./internal/data/... ./internal/market/... -count=1
	@echo "✅ 测试完成!"

# 运行测试(详细输出)
test-v:
	@echo "🧪 运行单元测试(详细模式)..."
	@go test -v ./internal/api/... ./internal/data/... ./internal/market/... -count=1

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "📊 生成测试覆盖率报告..."
	@go test -coverprofile=coverage.out ./internal/api/... ./internal/data/... ./internal/market/...
	@go tool cover -func=coverage.out | tail -1
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告已生成: coverage.html"

# 并发竞争检测
test-race:
	@echo "🔍 运行并发竞争检测..."
	@go test -race ./internal/api/... ./internal/data/... ./internal/market/... -count=1
	@echo "✅ 并发竞争检测完成!"

# 性能基准测试
test-bench:
	@echo "⚡ 运行性能基准测试..."
	@go test -bench=. -benchmem ./internal/api/... ./internal/data/... ./internal/market/...

# 测试特定包
test-api:
	@go test -v ./internal/api/... -count=1

test-data:
	@go test -v ./internal/data/... -count=1

test-market:
	@go test -v ./internal/market/... -count=1

# 清理测试缓存和临时文件
clean:
	@echo "🧹 清理测试缓存和临时文件..."
	@go clean -testcache
	@rm -f coverage.out coverage.html
	@rm -rf /tmp/stock-monitor-test-*
	@echo "✅ 清理完成!"

# 编译项目
build:
	@echo "🔨 编译项目..."
	@mkdir -p bin
	@go build -o bin/stock-monitor
	@echo "✅ 编译完成: bin/stock-monitor"

# 运行程序
run:
	@go run main.go

# 安装依赖
deps:
	@echo "📦 安装项目依赖..."
	@go mod download
	@go mod tidy
	@echo "✅ 依赖安装完成!"

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	@go fmt ./...
	@echo "✅ 格式化完成!"

# 代码静态检查
vet:
	@echo "🔍 运行静态检查..."
	@go vet ./...
	@echo "✅ 静态检查完成!"

# 完整检查(格式化+静态检查+测试)
check: fmt vet test
	@echo "✅ 所有检查通过!"

# 快速测试(不使用缓存)
test-quick:
	@go test ./internal/api/... ./internal/data/... ./internal/market/... -short -count=1

# 查看测试统计
test-stats:
	@echo "📈 测试统计信息:"
	@echo ""
	@echo "测试文件数:"
	@find internal -name "*_test.go" | wc -l
	@echo ""
	@echo "测试代码行数:"
	@find internal -name "*_test.go" -exec wc -l {} \; | awk '{sum+=$$1} END {print sum}'
	@echo ""
	@echo "测试函数数:"
	@grep -r "^func Test" internal --include="*_test.go" | wc -l
	@echo ""

# Release management
.PHONY: release-check release-snapshot release-dry-run

release-check:  ## Check GoReleaser configuration
	@echo "🔍 Checking GoReleaser configuration..."
	@goreleaser check
	@echo "✅ Configuration is valid!"

release-snapshot:  ## Build snapshot release (no publish)
	@echo "📦 Building snapshot release..."
	@goreleaser release --snapshot --clean
	@echo "✅ Snapshot built in dist/ directory"

release-dry-run:  ## Simulate release process (no publish)
	@echo "🧪 Simulating release process..."
	@goreleaser release --skip=publish --clean
	@echo "✅ Dry run completed!"
