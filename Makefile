.PHONY: help format lint test ci clean install-tools

# 默认目标
help:
	@echo "可用命令："
	@echo "  make format      - 格式化所有代码（Go + Python + Shell）"
	@echo "  make lint        - 运行所有静态检查（Go + Python + Shell）"
	@echo "  make test        - 运行所有测试（Go + Python）"
	@echo "  make ci          - 运行完整 CI 检查（format + lint + test）"
	@echo "  make clean       - 清理临时文件"
	@echo "  make install-tools - 安装必要的工具"
	@echo ""
	@echo "语言特定命令："
	@echo "  make format-go   - 格式化 Go 代码"
	@echo "  make lint-go     - 检查 Go 代码"
	@echo "  make test-go     - 运行 Go 测试"
	@echo "  make format-py   - 格式化 Python 代码"
	@echo "  make lint-py     - 检查 Python 代码"
	@echo "  make test-py     - 运行 Python 测试"
	@echo "  make format-sh   - 格式化 Shell 脚本"
	@echo "  make lint-sh     - 检查 Shell 脚本"

# 检查工具是否安装
GOLANGCILINT := $(shell command -v golangci-lint 2> /dev/null)
GOIMPORTS := $(shell command -v goimports 2> /dev/null)
SHFMT := $(shell command -v shfmt 2> /dev/null)
SHELLCHECK := $(shell command -v shellcheck 2> /dev/null)

# 格式化所有代码
format: format-go format-py format-sh

# 静态检查所有代码
lint: lint-go lint-py lint-sh

# 运行所有测试
test: test-go test-py

# 完整 CI 检查
ci: format lint test
	@echo "✅ 所有检查通过！"

# 清理临时文件
clean:
	find . -type f -name "*.out" -delete
	find . -type f -name "coverage.out" -delete
	find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name ".pytest_cache" -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name ".ruff_cache" -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name ".mypy_cache" -exec rm -rf {} + 2>/dev/null || true

# ==================== Go ====================

# 格式化 Go 代码
format-go:
	@echo "📝 格式化 Go 代码..."
	@if [ -d "api" ]; then \
		echo "  格式化 api/..."; \
		cd api && gofmt -w . && ([ -n "$(GOIMPORTS)" ] && goimports -w . || echo "  警告: goimports 未安装，跳过导入管理"); \
	fi
	@if [ -d "worker" ]; then \
		echo "  格式化 worker/..."; \
		cd worker && gofmt -w . && ([ -n "$(GOIMPORTS)" ] && goimports -w . || echo "  警告: goimports 未安装，跳过导入管理"); \
	fi
	@echo "✅ Go 代码格式化完成"

# 检查 Go 代码
lint-go:
	@echo "🔍 检查 Go 代码..."
	@if [ -z "$(GOLANGCILINT)" ]; then \
		echo "❌ 错误: golangci-lint 未安装"; \
		echo "  安装方法: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi
	@failed=0; \
	if [ -d "api" ]; then \
		echo "  检查 api/..."; \
		cd api && golangci-lint run || failed=1; \
	fi; \
	if [ -d "worker" ]; then \
		echo "  检查 worker/..."; \
		cd worker && golangci-lint run || failed=1; \
	fi; \
	if [ $$failed -eq 1 ]; then \
		echo "❌ Go 代码检查失败"; \
		exit 1; \
	fi
	@echo "✅ Go 代码检查通过"

# 运行 Go 测试
test-go:
	@echo "🧪 运行 Go 测试..."
	@failed=0; \
	if [ -d "api" ]; then \
		echo "  测试 api/..."; \
		cd api && go test ./... -v || failed=1; \
	fi; \
	if [ -d "worker" ]; then \
		echo "  测试 worker/..."; \
		cd worker && go test ./... -v || failed=1; \
	fi; \
	if [ $$failed -eq 1 ]; then \
		echo "❌ Go 测试失败"; \
		exit 1; \
	fi
	@echo "✅ Go 测试通过"

# ==================== Python ====================

# 格式化 Python 代码
format-py:
	@echo "📝 格式化 Python 代码..."
	@if [ -d "tts_service" ]; then \
		echo "  格式化 tts_service/..."; \
		cd tts_service && uv run black . && uv run ruff check . --fix || exit 1; \
	fi
	@echo "✅ Python 代码格式化完成"

# 检查 Python 代码
lint-py:
	@echo "🔍 检查 Python 代码..."
	@if [ -d "tts_service" ]; then \
		echo "  检查 tts_service/..."; \
		cd tts_service && uv run ruff check . || exit 1; \
	fi
	@echo "✅ Python 代码检查通过"

# 运行 Python 测试
test-py:
	@echo "🧪 运行 Python 测试..."
	@if [ -d "tts_service" ]; then \
		echo "  测试 tts_service/..."; \
		cd tts_service && uv run pytest -v || exit 1; \
	fi
	@echo "✅ Python 测试通过"

# ==================== Shell ====================

# 格式化 Shell 脚本
format-sh:
	@echo "📝 格式化 Shell 脚本..."
	@if [ -z "$(SHFMT)" ]; then \
		echo "⚠️  警告: shfmt 未安装，跳过 Shell 脚本格式化"; \
		echo "  安装方法: go install mvdan.cc/sh/v3/cmd/shfmt@latest"; \
	else \
		find . -name "*.sh" -type f -not -path "*/\.*" -exec shfmt -w {} \; && echo "✅ Shell 脚本格式化完成"; \
	fi

# 检查 Shell 脚本
lint-sh:
	@echo "🔍 检查 Shell 脚本..."
	@if [ -z "$(SHELLCHECK)" ]; then \
		echo "⚠️  警告: shellcheck 未安装，跳过 Shell 脚本检查"; \
		echo "  安装方法: https://github.com/koalaman/shellcheck#installing"; \
	else \
		failed=0; \
		find . -name "*.sh" -type f -not -path "*/\.*" | while read -r file; do \
			echo "  检查 $$file..."; \
			shellcheck "$$file" || failed=1; \
		done; \
		if [ $$failed -eq 1 ]; then \
			echo "❌ Shell 脚本检查失败"; \
			exit 1; \
		fi; \
		echo "✅ Shell 脚本检查通过"; \
	fi

# ==================== 工具安装 ====================

# 安装必要的工具
install-tools:
	@echo "🔧 安装开发工具..."
	@echo "安装 goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest || echo "⚠️  goimports 安装失败（可能需要先安装 Go）"
	@echo "安装 shfmt..."
	@go install mvdan.cc/sh/v3/cmd/shfmt@latest || echo "⚠️  shfmt 安装失败（可能需要先安装 Go）"
	@echo ""
	@echo "⚠️  以下工具需要手动安装："
	@echo "  - golangci-lint: https://golangci-lint.run/usage/install/"
	@echo "  - shellcheck: https://github.com/koalaman/shellcheck#installing"
	@echo ""
	@echo "✅ 工具安装完成（部分工具需要手动安装）"

