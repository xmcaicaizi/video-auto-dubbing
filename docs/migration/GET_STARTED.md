# 🚀 立即开始阿里云迁移

> ⚡ **3步快速开始** - 从现在开始，5分钟内启动开发环境

---

## Step 1: 运行初始化脚本 (1分钟)

```bash
cd /Users/micago/Desktop/index/video-auto-dubbing
./scripts/setup_migration_worktrees.sh
```

**这个脚本会**:
- ✅ 创建 6 个独立的工作目录 (worktree)
- ✅ 创建 5 个开发分支
- ✅ 生成 3 个辅助脚本
- ✅ 初始化任务文档

**预期输出**:
```
[INFO] 开始初始化 Git Worktree 环境...
[INFO] 检查 Git 仓库状态...
[SUCCESS] Git 仓库检查通过
[INFO] 创建 worktree 根目录: /Users/micago/worktrees/video-dubbing
[SUCCESS] 目录创建成功
...
[SUCCESS] 初始化完成！

================================
Worktree 环境初始化完成！
================================
```

---

## Step 2: 阅读快速指南 (2分钟)

```bash
# 方式1: 在终端查看
cat docs/migration/QUICK_START_GUIDE.md

# 方式2: 用编辑器打开
code docs/migration/QUICK_START_GUIDE.md

# 方式3: 在浏览器查看
open docs/migration/QUICK_START_GUIDE.md
```

**重点关注**:
- 📂 工作目录结构
- 🔧 常用命令
- 🐛 故障排除

---

## Step 3: 选择任务开始开发 (2分钟)

```bash
cd ~/worktrees/video-dubbing

# 根据你的兴趣/专长选择一个任务
source goto.sh asr      # ASR 迁移
# 或
source goto.sh oss      # OSS 迁移
# 或
source goto.sh llm      # LLM 翻译迁移
# 或
source goto.sh tts      # TTS 迁移
# 或
source goto.sh cleanup  # 清理遗留服务
```

**开始编码**:
```bash
# 查看任务说明
cat docs/migration/TASK_*.md

# 打开编辑器
code .

# 运行测试
go test ./... -v

# 提交更改
git add .
git commit -m "feat: implement feature X"
git push
```

---

## 🎯 我应该选择哪个任务？

### 如果你熟悉...

- **语音识别/ASR**: 选择 `asr` 任务
  ```bash
  source goto.sh asr
  ```

- **对象存储/云存储**: 选择 `oss` 任务
  ```bash
  source goto.sh oss
  ```

- **大语言模型/翻译**: 选择 `llm` 任务
  ```bash
  source goto.sh llm
  ```

- **语音合成/音频处理**: 选择 `tts` 任务
  ```bash
  source goto.sh tts
  ```

- **系统清理/DevOps**: 选择 `cleanup` 任务
  ```bash
  source goto.sh cleanup
  ```

---

## 📚 需要帮助？

### 查看完整文档

```bash
# 迁移总导航
cat docs/migration/README.md

# 完整迁移计划
cat docs/migration/ALIYUN_MIGRATION_PLAN.md

# 任务检查清单
cat docs/migration/MIGRATION_CHECKLIST.md

# 速查表（常用命令）
cat docs/migration/CHEATSHEET.md
```

### 常见问题

**Q: Worktree 是什么？**
A: Git Worktree 允许你同时在多个分支上工作，每个分支有独立的工作目录。

**Q: 我可以同时开发多个任务吗？**
A: 可以！这就是使用 Worktree 的好处，你可以在不同目录并行开发。

**Q: 如果初始化脚本失败了怎么办？**
A: 查看 [故障排除文档](QUICK_START_GUIDE.md#故障排除)

**Q: 我需要配置阿里云 API Key 吗？**
A: 开发阶段不需要，测试时才需要。参考 [配置指南](../guides/DASHSCOPE_LLM_SETUP.md)

---

## ✅ 验证环境

### 检查 Worktree 是否创建成功

```bash
git worktree list
```

**预期输出**:
```
/Users/micago/Desktop/index/video-auto-dubbing           abc1234 [main]
/Users/micago/worktrees/video-dubbing/aliyun-migration  def5678 [feature/aliyun-unified-migration]
/Users/micago/worktrees/video-dubbing/asr-default       ghi9012 [feature/aliyun-asr-default]
/Users/micago/worktrees/video-dubbing/oss-default       jkl3456 [feature/aliyun-oss-default]
...
```

### 检查辅助脚本

```bash
cd ~/worktrees/video-dubbing
ls -l *.sh
```

**预期输出**:
```
-rwxr-xr-x  1 user  staff  goto.sh
-rwxr-xr-x  1 user  staff  check_status.sh
-rwxr-xr-x  1 user  staff  commit_all.sh
```

---

## 🎊 你已经准备好了！

现在你可以：

1. ✅ **并行开发**: 每个任务独立工作目录
2. ✅ **快速切换**: 使用 `goto.sh` 在任务间切换
3. ✅ **独立测试**: 每个任务可单独测试
4. ✅ **灵活合并**: 完成后合并到主分支

---

## 💡 小贴士

1. **收藏速查表**: `docs/migration/CHEATSHEET.md` 包含所有常用命令
2. **每日检查进度**: 使用 `./check_status.sh` 查看所有任务状态
3. **批量提交**: 使用 `./commit_all.sh "message"` 一键提交所有更改
4. **遵循规范**: 提交信息使用 `feat:`, `fix:`, `docs:` 等前缀

---

## 🚀 立即行动

```bash
# 复制粘贴这3行命令，立即开始！
cd /Users/micago/Desktop/index/video-auto-dubbing
./scripts/setup_migration_worktrees.sh
cd ~/worktrees/video-dubbing && source goto.sh asr
```

**祝开发顺利！🎉**

---

**下一步**: 查看 [快速启动指南](QUICK_START_GUIDE.md) 了解详细开发流程
