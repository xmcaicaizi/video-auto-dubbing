# 贡献指南

感谢您对本项目的关注！本文档将帮助您了解如何参与项目开发。

## 快速开始

1. **Fork 项目**并克隆到本地
2. **阅读规范文档**：
   - [代码规范](docs/coding-standards.md)
   - [代码审查规范](docs/code-review.md)
   - [提交信息规范](docs/commit-message.md)
3. **设置开发环境**（见下文）
4. **创建特性分支**并开始开发
5. **提交代码**前运行检查工具
6. **创建 Pull Request**

## 开发环境设置

### 前置要求

- **Go**: 1.21+
- **Python**: 3.11+
- **uv**: Python 包管理器（用于 TTS 服务）
- **Docker**: 20.10+（用于本地测试）
- **Make**: 用于运行统一命令（可选，也可直接运行脚本）

### 安装依赖

#### Go 服务

```bash
cd api  # 或 worker
go mod download
```

#### Python TTS 服务

```bash
cd tts_service
uv venv
source .venv/bin/activate  # Windows: .venv\Scripts\activate
uv sync
```

## 开发工作流

### 1. 创建分支

```bash
# 从 main 分支创建新分支
git checkout main
git pull origin main
git checkout -b feature/your-feature-name

# 或修复 bug
git checkout -b fix/bug-description
```

**分支命名规范：**
- `feature/`: 新功能
- `fix/`: Bug 修复
- `refactor/`: 重构
- `docs/`: 文档更新
- `test/`: 测试相关

### 2. 开发代码

- 遵循 [代码规范](docs/coding-standards.md)
- 编写清晰的注释和文档
- 为新功能编写测试

### 3. 运行检查

在提交代码前，**必须**运行以下检查：

```bash
# 使用 Makefile（推荐）
make format    # 格式化代码
make lint      # 静态检查
make test      # 运行测试

# 或一次性运行所有检查
make ci
```

**手动运行（如果 Makefile 不可用）：**

#### Go 服务

```bash
cd api  # 或 worker

# 格式化
gofmt -w .
goimports -w .

# 静态检查
golangci-lint run

# 测试
go test ./...
```

#### Python TTS 服务

```bash
cd tts_service

# 格式化
uv run black .
uv run ruff check . --fix

# 静态检查（严格模式）
uv run ruff check .
uv run mypy .  # 如果配置了 mypy

# 测试
uv run pytest
```

### 4. 提交代码

#### 提交信息规范

遵循 [提交信息规范](docs/commit-message.md)，使用 Google-style 格式：

```
简短描述变更内容（动词开头）

详细说明变更内容、动机和影响。

测试说明：如何验证这个变更。
```

**示例：**

```
添加任务状态查询 API

实现了 GET /api/v1/tasks/:id 接口，支持查询任务处理状态。
添加了 Redis 缓存以减少数据库查询压力。

测试说明：
- 单元测试覆盖所有状态码
- 集成测试验证缓存命中
- 手动测试使用 curl 验证 API 响应
```

#### 提交前检查清单

- [ ] 代码已格式化（`make format`）
- [ ] 静态检查通过（`make lint`）
- [ ] 所有测试通过（`make test`）
- [ ] 新增代码包含测试
- [ ] 提交信息符合规范
- [ ] 无硬编码的密钥或敏感信息

### 5. 创建 Pull Request

1. **推送分支到远程**

```bash
git push origin feature/your-feature-name
```

2. **在 GitHub 创建 PR**

- 填写清晰的 PR 标题和描述
- 使用 PR 模板（如已配置）
- 关联相关 Issue（如适用）

3. **等待代码审查**

- 至少需要 1 个审查者批准
- 所有 CI 检查必须通过
- 及时响应审查意见

## 代码检查工具

### 统一入口（推荐）

项目提供 `Makefile` 作为统一入口：

```bash
make help          # 查看所有可用命令
make format        # 格式化所有代码
make lint          # 运行所有静态检查
make test          # 运行所有测试
make ci            # 运行完整 CI 检查（format + lint + test）
```

### 各语言工具

#### Go

- **gofmt**: 代码格式化（Go 官方工具）
- **goimports**: 管理导入（自动添加/删除导入）
- **golangci-lint**: 静态检查（集成多个 linter）

**安装：**

```bash
# goimports
go install golang.org/x/tools/cmd/goimports@latest

# golangci-lint
# macOS
brew install golangci-lint
# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
# Windows
choco install golangci-lint
```

#### Python

- **black**: 代码格式化
- **ruff**: 静态检查和格式化（快速替代 flake8 + isort）
- **mypy**: 类型检查（可选，严格模式推荐）
- **pytest**: 测试框架

**配置：** 已在 `tts_service/pyproject.toml` 中配置

#### Shell

- **shfmt**: Shell 脚本格式化
- **shellcheck**: Shell 脚本静态检查

**安装：**

```bash
# shfmt
go install mvdan.cc/sh/v3/cmd/shfmt@latest

# shellcheck
# macOS
brew install shellcheck
# Linux
# 见 https://github.com/koalaman/shellcheck#installing
```

## 测试指南

### Go 测试

```bash
cd api  # 或 worker

# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/services

# 显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Python 测试

```bash
cd tts_service

# 运行所有测试
uv run pytest

# 运行特定测试文件
uv run pytest tests/test_tts_engine.py

# 显示覆盖率
uv run pytest --cov=app --cov-report=html

# 运行并显示详细输出
uv run pytest -v
```

## 文档贡献

### 更新文档

- 代码变更时，同步更新相关文档
- API 变更必须更新 `docs/api-contracts.md`
- 架构变更必须更新 `docs/architecture.md`

### 文档格式

- 使用 Markdown 格式
- 遵循现有文档风格
- 代码示例要完整可运行

## 问题报告

### Bug 报告

在 GitHub Issues 中报告 bug，包含：

1. **问题描述**：清晰描述问题
2. **复现步骤**：如何复现问题
3. **预期行为**：应该发生什么
4. **实际行为**：实际发生了什么
5. **环境信息**：OS、版本、配置等
6. **日志/错误信息**：相关日志或错误堆栈

### 功能请求

在 GitHub Issues 中提出功能请求，包含：

1. **功能描述**：想要什么功能
2. **使用场景**：为什么需要这个功能
3. **建议实现**：如何实现（如已有想法）

## 代码审查流程

详见 [代码审查规范](docs/code-review.md)。

**关键点：**

- 所有代码必须经过审查才能合并
- 至少需要 1 个审查者批准
- 保持礼貌和专业
- 及时响应审查意见

## 发布流程

（待项目成熟后补充）

## 获取帮助

- 查看 [文档索引](README.md#文档索引)
- 在 GitHub Issues 中提问
- 联系项目维护者

## 行为准则

- 保持尊重和礼貌
- 接受建设性批评
- 关注代码质量
- 帮助他人学习和成长

感谢您的贡献！

