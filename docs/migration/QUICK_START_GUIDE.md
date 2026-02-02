# 阿里云统一平台迁移 - 快速启动指南

> 🚀 **5分钟快速开始阿里云统一平台迁移开发**

---

## 📋 前置要求

### 必需工具
- [x] Git 2.30+
- [x] Go 1.21+
- [x] Docker & Docker Compose
- [x] 代码编辑器（VS Code 推荐）

### 必需权限
- [x] Git 仓库写权限
- [x] 阿里云账号和 API Key（用于测试）

---

## 🏃 30秒快速开始

```bash
# 1. 进入项目目录
cd /Users/micago/Desktop/index/video-auto-dubbing

# 2. 运行初始化脚本
chmod +x scripts/setup_migration_worktrees.sh
./scripts/setup_migration_worktrees.sh

# 3. 切换到你的工作目录
cd ~/worktrees/video-dubbing

# 4. 选择一个任务开始开发
source goto.sh asr    # ASR 迁移
# 或
source goto.sh oss    # OSS 迁移
# 或
source goto.sh llm    # LLM 迁移
# 或
source goto.sh tts    # TTS 迁移
```

✅ **就是这么简单！现在你可以开始开发了！**

---

## 📂 工作目录结构

初始化后，你会得到以下目录结构：

```
~/worktrees/video-dubbing/
├── aliyun-migration/          # 主迁移分支
│   └── docs/migration/        # 迁移文档
├── asr-default/               # ASR 迁移任务
├── oss-default/               # OSS 迁移任务
├── llm-default/               # LLM 翻译迁移任务
├── tts-default/               # TTS 迁移任务
├── cleanup-legacy/            # 清理遗留服务任务
├── goto.sh                    # 快速切换脚本
├── check_status.sh            # 状态检查脚本
└── commit_all.sh              # 批量提交脚本
```

---

## 🎯 选择你的任务

### 方案A: 我想负责 ASR 迁移

```bash
cd ~/worktrees/video-dubbing
source goto.sh asr

# 查看任务说明
cat docs/migration/TASK_ASR_DEFAULT.md

# 开始开发
# 1. 修改配置文件
code shared/config/config.go

# 2. 运行测试
go test ./worker/internal/asr/... -v

# 3. 提交更改
git add .
git commit -m "feat(asr): set aliyun as default backend"
git push origin feature/aliyun-asr-default
```

### 方案B: 我想负责 OSS 迁移

```bash
cd ~/worktrees/video-dubbing
source goto.sh oss

# 开始开发数据迁移工具
mkdir -p scripts
code scripts/migrate_minio_to_oss.go

# 测试 OSS 客户端
go test ./shared/storage/... -v
```

### 方案C: 我想负责 LLM 翻译迁移

```bash
cd ~/worktrees/video-dubbing
source goto.sh llm

# 优化 DashScope 客户端
code worker/internal/translate/dashscope_client.go

# 运行翻译测试
go test ./worker/internal/translate/... -v
```

### 方案D: 我想负责 TTS 迁移

```bash
cd ~/worktrees/video-dubbing
source goto.sh tts

# 优化阿里云 TTS 客户端
code worker/internal/tts/aliyun_client.go

# 创建音色映射配置
mkdir -p configs
code configs/voice_mapping.yaml
```

---

## 🔧 常用命令

### 检查所有任务状态

```bash
cd ~/worktrees/video-dubbing
./check_status.sh
```

输出示例：
```
========================================
Worktree: asr-default
========================================
Branch: feature/aliyun-asr-default
Status:
 M worker/internal/asr/aliyun_client.go
 M shared/config/config.go
```

### 批量提交所有更改

```bash
cd ~/worktrees/video-dubbing
./commit_all.sh "feat: implement aliyun unified platform"
```

这会：
1. 提交所有子任务的更改
2. 推送到远程仓库
3. 自动合并到主迁移分支

### 切换分支

```bash
cd ~/worktrees/video-dubbing

# 方式1: 使用快捷脚本
source goto.sh asr    # 切换到 ASR 任务
source goto.sh main   # 切换到主迁移分支

# 方式2: 直接 cd
cd asr-default
cd ../llm-default
```

### 运行测试

```bash
# 在当前 worktree 运行测试
go test ./... -v

# 运行特定包的测试
go test ./worker/internal/asr/... -v
go test ./shared/storage/... -v

# 运行集成测试
cd ~/worktrees/video-dubbing/aliyun-migration
./scripts/integration_test.sh
```

---

## 📝 开发工作流

### 每日工作流程

```bash
# 1. 早上拉取最新代码
cd ~/worktrees/video-dubbing/aliyun-migration
git pull origin feature/aliyun-unified-migration

# 2. 切换到你的任务分支
source ~/worktrees/video-dubbing/goto.sh asr

# 3. 开发、测试、提交
# ... 编写代码 ...
go test ./... -v
git add .
git commit -m "feat(asr): implement feature X"
git push origin feature/aliyun-asr-default

# 4. 晚上合并到主分支（可选）
cd ~/worktrees/video-dubbing/aliyun-migration
git merge feature/aliyun-asr-default
git push origin feature/aliyun-unified-migration
```

### 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```bash
# 功能开发
git commit -m "feat(asr): add aliyun client implementation"
git commit -m "feat(oss): implement minio to oss migration tool"

# Bug 修复
git commit -m "fix(tts): resolve voice mapping issue"

# 文档更新
git commit -m "docs(migration): update migration checklist"

# 测试
git commit -m "test(translate): add dashscope client unit tests"

# 重构
git commit -m "refactor(storage): optimize factory pattern"
```

---

## 🧪 测试策略

### 单元测试

```bash
# 测试单个文件
go test ./worker/internal/asr/aliyun_client_test.go -v

# 测试整个包
go test ./worker/internal/asr/... -v

# 带覆盖率
go test ./worker/internal/asr/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 集成测试

```bash
# 在主迁移分支运行
cd ~/worktrees/video-dubbing/aliyun-migration

# 启动测试环境
docker-compose -f docker-compose.test.yml up -d

# 运行集成测试
go test ./tests/integration/... -v

# 清理
docker-compose -f docker-compose.test.yml down
```

### 端到端测试

```bash
# 启动完整系统
cd ~/worktrees/video-dubbing/aliyun-migration
docker-compose up -d

# 运行端到端测试
go test ./tests/e2e/... -v

# 或使用脚本
./scripts/e2e_test.sh
```

---

## 🐛 故障排除

### 问题1: Worktree 已存在

```bash
# 错误: fatal: 'xxx' is already checked out at 'yyy'

# 解决方案: 清理旧的 worktree
git worktree remove ~/worktrees/video-dubbing/asr-default --force
git worktree prune

# 重新创建
cd ~/worktrees/video-dubbing/aliyun-migration
git worktree add ~/worktrees/video-dubbing/asr-default -b feature/aliyun-asr-default
```

### 问题2: 分支冲突

```bash
# 合并时遇到冲突
cd ~/worktrees/video-dubbing/aliyun-migration
git merge feature/aliyun-asr-default

# 解决冲突
git status  # 查看冲突文件
# 编辑冲突文件
git add .
git commit -m "merge: resolve conflicts from asr-default"
```

### 问题3: Go 模块问题

```bash
# 错误: module not found

# 解决方案: 清理并重新下载依赖
go clean -modcache
go mod download
go mod tidy
```

### 问题4: Docker 构建失败

```bash
# 清理 Docker 缓存
docker system prune -a

# 重新构建
docker-compose build --no-cache
docker-compose up -d
```

---

## 📚 相关文档

### 必读文档
1. [迁移计划](./ALIYUN_MIGRATION_PLAN.md) - 详细的迁移步骤和架构
2. [检查清单](./MIGRATION_CHECKLIST.md) - 任务进度追踪
3. [代码规范](../coding-standards.md) - 编码规范
4. [提交规范](../commit-message.md) - Git 提交规范

### API 文档
1. [阿里云 ASR](https://help.aliyun.com/zh/model-studio/qwen-asr-api-reference)
2. [阿里云 OSS](https://help.aliyun.com/zh/oss/)
3. [阿里云 TTS](https://help.aliyun.com/zh/model-studio/developer-reference/text-to-speech-api-details)
4. [DashScope LLM](https://help.aliyun.com/zh/model-studio/developer-reference/api-details)

---

## 💬 获取帮助

### 开发问题
- 查看项目 Wiki
- 搜索现有 Issues
- 在团队群里提问

### 紧急问题
- 联系项目负责人
- 发送邮件到团队邮箱

### 提交 Bug
```bash
# 创建 Bug 报告
git checkout -b fix/describe-the-bug
# 修复后提交 PR
```

---

## ✅ 下一步

选择一个任务开始开发：

```bash
# 🎯 ASR 迁移 (推荐新手)
cd ~/worktrees/video-dubbing/asr-default

# 🎯 OSS 迁移 (需要存储经验)
cd ~/worktrees/video-dubbing/oss-default

# 🎯 LLM 迁移 (需要 AI 经验)
cd ~/worktrees/video-dubbing/llm-default

# 🎯 TTS 迁移 (需要音频处理经验)
cd ~/worktrees/video-dubbing/tts-default
```

**祝开发顺利！🚀**

---

**最后更新**: 2026-02-02
**版本**: v1.0
**维护者**: 项目团队
