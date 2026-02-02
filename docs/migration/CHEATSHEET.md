# 阿里云迁移速查表 (Cheatsheet)

> 🔖 **快速参考** - 常用命令和配置一览

---

## 🚀 初始化

```bash
# 一键初始化所有 worktree
cd /Users/micago/Desktop/index/video-auto-dubbing
./scripts/setup_migration_worktrees.sh
```

---

## 📂 快速切换

```bash
cd ~/worktrees/video-dubbing

# 使用快捷脚本
source goto.sh main    # 主迁移分支
source goto.sh asr     # ASR 任务
source goto.sh oss     # OSS 任务
source goto.sh llm     # LLM 任务
source goto.sh tts     # TTS 任务
source goto.sh cleanup # 清理任务

# 或直接 cd
cd aliyun-migration
cd asr-default
cd oss-default
```

---

## 🔄 常用 Git 命令

### 提交更改

```bash
# 在当前 worktree 提交
git add .
git commit -m "feat(asr): implement feature X"
git push origin $(git branch --show-current)

# 批量提交所有 worktree
cd ~/worktrees/video-dubbing
./commit_all.sh "feat: unified aliyun platform"
```

### 合并分支

```bash
# 在主迁移分支合并子任务
cd ~/worktrees/video-dubbing/aliyun-migration
git merge feature/aliyun-asr-default
git push origin feature/aliyun-unified-migration
```

### 查看状态

```bash
# 查看所有 worktree 状态
cd ~/worktrees/video-dubbing
./check_status.sh

# 查看当前 worktree 状态
git status
git log --oneline -5
```

---

## 🧪 测试命令

### 单元测试

```bash
# 测试所有包
go test ./... -v

# 测试特定包
go test ./worker/internal/asr/... -v
go test ./shared/storage/... -v
go test ./worker/internal/translate/... -v
go test ./worker/internal/tts/... -v

# 带覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 集成测试

```bash
# 启动测试环境
docker-compose -f docker-compose.test.yml up -d

# 运行集成测试
go test ./tests/integration/... -v

# 清理
docker-compose -f docker-compose.test.yml down
```

### 性能测试

```bash
# 运行性能对比测试
./scripts/performance_comparison.sh

# 基准测试
go test -bench=. ./worker/internal/asr/...
```

---

## 🔧 配置文件位置

| 配置项 | 文件路径 |
|-------|---------|
| **共享配置** | `shared/config/config.go` |
| **Worker配置** | `worker/internal/config/config.go` |
| **API配置** | `api/internal/config/config.go` |
| **环境变量** | `.env` (根目录) |
| **Docker** | `docker-compose.yml` |

---

## 📝 配置示例

### 阿里云 ASR

```bash
# .env
ASR_BACKEND=aliyun
ALIYUN_ASR_API_KEY=sk-xxxxxxxxxxxxx
ALIYUN_ASR_MODEL=qwen3-asr-flash
ALIYUN_ASR_ENABLE_ITN=true
```

### 阿里云 OSS

```bash
# .env
STORAGE_BACKEND=oss
OSS_ENDPOINT=oss-cn-beijing.aliyuncs.com
OSS_BUCKET=video-dubbing
OSS_ACCESS_KEY_ID=LTAI5...
OSS_ACCESS_KEY_SECRET=xxx
OSS_USE_SSL=true
```

### 阿里云 LLM (DashScope)

```bash
# .env
DASHSCOPE_LLM_API_KEY=sk-xxxxxxxxxxxxx
DASHSCOPE_LLM_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
DASHSCOPE_LLM_MODEL=qwen-turbo
DASHSCOPE_LLM_RPS=5.0
```

### 阿里云 TTS

```bash
# .env
TTS_BACKEND=aliyun
DASHSCOPE_API_KEY=sk-xxxxxxxxxxxxx
DASHSCOPE_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
DASHSCOPE_MODEL=qwen-tts-flash
```

---

## 🏗️ 项目结构

```
video-auto-dubbing/
├── api/                  # API 服务
│   └── internal/
│       ├── config/      # API 配置
│       ├── handlers/    # 请求处理
│       └── service/     # 业务逻辑
├── worker/              # Worker 服务
│   └── internal/
│       ├── asr/         # ASR 客户端
│       ├── tts/         # TTS 客户端
│       ├── translate/   # 翻译客户端
│       └── storage/     # 存储客户端
├── shared/              # 共享代码
│   ├── config/          # 共享配置
│   ├── storage/         # 存储抽象层
│   └── queue/           # 消息队列
├── docs/                # 文档
│   ├── migration/       # ⭐ 迁移文档
│   ├── guides/          # 操作指南
│   └── troubleshooting/ # 故障排除
└── scripts/             # 脚本工具
```

---

## 🔍 文件路径速查

### ASR 相关

```
worker/internal/asr/aliyun_client.go       # 阿里云 ASR 客户端
worker/internal/asr/volcengine_client.go   # 火山引擎 ASR 客户端
worker/internal/asr/client.go              # ASR 客户端工厂
```

### OSS 相关

```
shared/storage/oss_storage.go              # 阿里云 OSS 客户端
shared/storage/storage.go                  # MinIO 客户端
shared/storage/factory.go                  # 存储工厂
```

### LLM 相关

```
worker/internal/translate/dashscope_client.go  # DashScope 翻译客户端
worker/internal/translate/factory.go           # 翻译客户端工厂
```

### TTS 相关

```
worker/internal/tts/aliyun_client.go       # 阿里云 TTS 客户端
worker/internal/tts/vllm_client.go         # IndexTTS vLLM 客户端
worker/internal/tts/client.go              # TTS 客户端工厂
```

---

## 📊 进度追踪

### 查看总体进度

```bash
cat docs/migration/MIGRATION_CHECKLIST.md | grep -E "^\[[ x]\]" | wc -l
```

### 更新任务状态

在 `docs/migration/MIGRATION_CHECKLIST.md` 中：

```markdown
- [ ] 未完成任务
- [x] 已完成任务
```

---

## 🐛 故障排除速查

### Worktree 问题

```bash
# 问题: worktree 已存在
git worktree remove ~/worktrees/video-dubbing/asr-default --force
git worktree prune

# 问题: 分支已存在
git branch -D feature/aliyun-asr-default
```

### Go 模块问题

```bash
# 清理并重新下载
go clean -modcache
go mod download
go mod tidy
```

### Docker 问题

```bash
# 清理缓存
docker system prune -a

# 重新构建
docker-compose build --no-cache
docker-compose up -d
```

### 合并冲突

```bash
# 查看冲突文件
git status

# 解决后
git add .
git commit -m "merge: resolve conflicts"
```

---

## 📚 文档链接

| 文档 | 路径 |
|-----|------|
| **迁移总导航** | [docs/migration/README.md](README.md) |
| **快速启动** | [docs/migration/QUICK_START_GUIDE.md](QUICK_START_GUIDE.md) |
| **完整计划** | [docs/migration/ALIYUN_MIGRATION_PLAN.md](ALIYUN_MIGRATION_PLAN.md) |
| **检查清单** | [docs/migration/MIGRATION_CHECKLIST.md](MIGRATION_CHECKLIST.md) |
| **速查表** | [docs/migration/CHEATSHEET.md](CHEATSHEET.md) ⭐ 当前 |

---

## 🔗 外部资源

| 资源 | 链接 |
|-----|------|
| **阿里云百炼控制台** | https://bailian.console.aliyun.com/ |
| **DashScope API 文档** | https://help.aliyun.com/zh/model-studio/ |
| **阿里云 OSS 控制台** | https://oss.console.aliyun.com/ |
| **Qwen ASR API** | https://help.aliyun.com/zh/model-studio/qwen-asr-api-reference |
| **Qwen-TTS API** | https://help.aliyun.com/zh/model-studio/developer-reference/text-to-speech-api-details |

---

## ⌨️ IDE 配置

### VS Code 推荐扩展

```json
{
  "recommendations": [
    "golang.go",
    "eamodio.gitlens",
    "ms-azuretools.vscode-docker",
    "streetsidesoftware.code-spell-checker"
  ]
}
```

### 调试配置

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Worker",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/worker/main.go",
      "envFile": "${workspaceFolder}/.env"
    }
  ]
}
```

---

## 📋 提交规范

```bash
# 功能开发
git commit -m "feat(asr): add aliyun client"
git commit -m "feat(oss): implement migration tool"

# Bug 修复
git commit -m "fix(tts): resolve audio quality issue"

# 文档
git commit -m "docs(migration): update checklist"

# 测试
git commit -m "test(translate): add dashscope tests"

# 重构
git commit -m "refactor(storage): optimize factory pattern"

# 性能优化
git commit -m "perf(asr): improve polling efficiency"

# 样式修改
git commit -m "style: format code with gofmt"

# 构建/CI
git commit -m "build: update docker compose"
```

---

## 🎯 关键指标

| 指标 | 目标 |
|-----|------|
| **单元测试覆盖率** | > 80% |
| **集成测试通过率** | 100% |
| **性能提升** | > 20% |
| **成本降低** | > 30% |
| **迁移完成时间** | 14天 |

---

## 🆘 紧急联系

| 问题类型 | 联系方式 |
|---------|---------|
| **技术问题** | 团队群聊 |
| **进度阻塞** | 项目负责人 |
| **环境问题** | 运维团队 |
| **API 配额** | 阿里云技术支持 |

---

**最后更新**: 2026-02-02
**版本**: v1.0

**💡 提示**: 将此文档加入书签，随时查阅！
