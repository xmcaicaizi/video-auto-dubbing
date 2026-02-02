# 🚀 统一阿里云百炼平台迁移

> **重大架构升级**: 将所有外部服务（ASR/OSS/LLM/TTS）统一迁移到阿里云百炼平台

---

## 📌 迁移状态

🔴 **状态**: 规划完成，待开始执行
📅 **开始日期**: 待定
⏱️ **预计工期**: 2-3周
👥 **参与人数**: 5-6人

---

## 🎯 为什么要迁移？

### 当前架构的痛点

- 🔥 **多平台管理复杂**: 火山引擎ASR + 智谱GLM + IndexTTS自建服务
- 💸 **运维成本高**: 需要维护GPU服务器运行IndexTTS
- 🐌 **性能瓶颈**: 本地TTS服务受限于单GPU性能
- 🔧 **配置分散**: 需要管理多个平台的API Key和配置

### 迁移后的收益

#### 💰 成本优化
- ✅ 移除GPU服务器维护成本（节省 $500+/月）
- ✅ MinIO自建存储迁移到阿里云OSS（按需付费）
- ✅ 统一计费平台，成本更透明

#### 🚀 性能提升
- ⚡ qwen-tts-flash 实时合成，低延迟
- ⚡ OSS CDN全球加速
- ⚡ 多区域高可用部署

#### 🛠️ 运维简化
- 📦 无需维护Docker容器和GPU环境
- 🔧 统一的阿里云API Key管理
- 📊 统一的监控和日志平台
- 🔄 更简单的扩容和降级

---

## 🗺️ 迁移范围

| 服务类型 | 当前方案 | 迁移目标 | 状态 |
|---------|---------|---------|------|
| **ASR** (语音识别) | 火山引擎 | 阿里云 Qwen ASR | ✅ 已支持，待设为默认 |
| **OSS** (对象存储) | MinIO (自建) | 阿里云 OSS | ✅ 已支持，待设为默认 |
| **LLM** (翻译) | 智谱GLM | 阿里云 DashScope | ✅ 已支持，待设为默认 |
| **TTS** (语音合成) | IndexTTS (自建) | 阿里云 Qwen-TTS | ✅ 已支持，待设为默认 |

**好消息**: 所有阿里云服务已经完成集成，迁移主要是**切换默认配置**和**优化现有实现**！

---

## 🏃 快速开始

### 第一步: 初始化开发环境

```bash
# 进入项目目录
cd /Users/micago/Desktop/index/video-auto-dubbing

# 运行初始化脚本（创建Git Worktree开发环境）
./scripts/setup_migration_worktrees.sh
```

这会创建以下工作目录：

```
~/worktrees/video-dubbing/
├── aliyun-migration/     # 主迁移分支 - 用于集成测试
├── asr-default/          # ASR 迁移任务
├── oss-default/          # OSS 迁移任务
├── llm-default/          # LLM 迁移任务
├── tts-default/          # TTS 迁移任务
└── cleanup-legacy/       # 清理遗留服务
```

### 第二步: 选择一个任务

```bash
cd ~/worktrees/video-dubbing

# 快速切换到任务目录
source goto.sh asr      # ASR 迁移
source goto.sh oss      # OSS 迁移
source goto.sh llm      # LLM 迁移
source goto.sh tts      # TTS 迁移
source goto.sh cleanup  # 清理遗留服务
```

### 第三步: 开始开发

```bash
# 查看任务说明
cat docs/migration/TASK_*.md

# 查看完整迁移计划
cat docs/migration/ALIYUN_MIGRATION_PLAN.md

# 开始开发
code .

# 运行测试
go test ./... -v

# 提交更改
git add .
git commit -m "feat: implement aliyun as default"
git push
```

---

## 📚 完整文档

所有迁移相关的文档都在 `docs/migration/` 目录下：

### 📖 主要文档

1. **[迁移总导航](docs/migration/README.md)**
   一站式导航，了解全局

2. **[快速启动指南](docs/migration/QUICK_START_GUIDE.md)** ⭐ 推荐先读
   5分钟快速开始，包含常用命令和故障排除

3. **[完整迁移计划](docs/migration/ALIYUN_MIGRATION_PLAN.md)**
   详细的5个阶段任务、Git Worktree策略、测试策略

4. **[迁移检查清单](docs/migration/MIGRATION_CHECKLIST.md)**
   40+任务的复选框、每日进度追踪

### 🔗 相关文档

- [阿里云 ASR 集成文档](docs/aliyun_asr_integration.md)
- [阿里云 TTS 集成文档](docs/aliyun_tts_integration.md)
- [DashScope LLM 配置指南](docs/guides/DASHSCOPE_LLM_SETUP.md)
- [架构概览](docs/ARCHITECTURE_OVERVIEW.md)

---

## ⏱️ 时间线

```
┌────────────────────────────────────────────────────────────┐
│                     迁移时间线 (14天)                       │
└────────────────────────────────────────────────────────────┘

Week 1 (Day 1-5):
  Day 1  │ ✅ Worktree环境初始化
  Day 2-3│ 🔄 ASR迁移 (设置阿里云为默认)
  Day 4-5│ 🔄 OSS迁移开始

Week 2 (Day 6-10):
  Day 6  │ ✅ OSS迁移完成
  Day 7-8│ 🔄 LLM翻译迁移
  Day 9-10│ 🔄 TTS迁移开始

Week 3 (Day 11-14):
  Day 11 │ ✅ TTS迁移完成
  Day 12-13│ 🔄 清理遗留服务
  Day 14 │ 🎉 最终验证和合并到main
```

---

## 🎯 任务分配

### Phase 1: ASR 迁移 (3天)
**工作目录**: `~/worktrees/video-dubbing/asr-default`

- [ ] 设置阿里云ASR为默认后端
- [ ] 优化异步轮询逻辑
- [ ] 实现优雅降级机制
- [ ] 更新文档

### Phase 2: OSS 迁移 (3天)
**工作目录**: `~/worktrees/video-dubbing/oss-default`

- [ ] 设置阿里云OSS为默认存储
- [ ] 开发MinIO→OSS数据迁移工具
- [ ] 实现CDN加速配置
- [ ] 数据完整性验证

### Phase 3: LLM 翻译迁移 (2天)
**工作目录**: `~/worktrees/video-dubbing/llm-default`

- [ ] 设置DashScope为默认翻译服务
- [ ] 优化批处理逻辑
- [ ] 实现翻译记忆库
- [ ] 质量对比测试

### Phase 4: TTS 迁移 (3天)
**工作目录**: `~/worktrees/video-dubbing/tts-default`

- [ ] 设置阿里云TTS为默认后端
- [ ] 实现流式合成支持
- [ ] 优化音色映射系统
- [ ] 移除IndexTTS服务依赖

### Phase 5: 清理遗留服务 (2天)
**工作目录**: `~/worktrees/video-dubbing/cleanup-legacy`

- [ ] 移除MinIO和TTS服务容器
- [ ] 清理遗留代码
- [ ] 更新所有文档
- [ ] 最终验证

---

## 🧪 质量保证

### 测试策略

- ✅ **单元测试**: 每个模块 > 80% 覆盖率
- ✅ **集成测试**: 完整流程端到端测试
- ✅ **性能测试**: 与旧系统对比验证
- ✅ **回归测试**: 确保现有功能不受影响

### Code Review

- ✅ 符合项目代码规范
- ✅ 无安全漏洞
- ✅ 错误处理健壮
- ✅ 日志记录完整

---

## ⚠️ 风险管理

| 风险 | 影响 | 缓解措施 |
|-----|------|---------|
| API配额不足 | 🔴 高 | 提前采购配额，实现限流 |
| 数据迁移失败 | 🔴 高 | 完整备份，渐进式迁移 |
| 性能下降 | 🟡 中 | 性能对比测试，优化批处理 |
| 音色不匹配 | 🟡 中 | 建立音色映射表，人工验证 |
| 成本超预算 | 🟡 中 | 生命周期规则，用量监控 |

**回滚策略**: 保留所有旧服务配置，可随时切换回原方案

---

## 🆘 获取帮助

### 文档不清楚？

1. 查看 [快速启动指南](docs/migration/QUICK_START_GUIDE.md) 的故障排除章节
2. 查看 [项目故障排除文档](docs/troubleshooting/README.md)
3. 搜索 GitHub Issues

### 技术问题？

1. 在团队群聊提问
2. 联系相关模块负责人
3. 提交 GitHub Issue（附上详细日志）

---

## 📊 当前进度

```
总体进度: [░░░░░░░░░░░░░░░░░░░░] 0% (0/40 完成)

✅ 完成: 迁移计划制定、文档准备、Worktree脚本
⏸️  待开始: 所有迁移任务
```

👉 [查看详细进度](docs/migration/MIGRATION_CHECKLIST.md)

---

## 🎉 预期成果

迁移完成后，你将获得：

1. ✅ **统一平台**: 所有服务使用阿里云百炼
2. ✅ **成本降低**: 移除GPU和自建服务成本
3. ✅ **性能提升**: TTS合成速度3x提升
4. ✅ **运维简化**: 无需维护本地服务
5. ✅ **可扩展性**: 轻松应对业务增长

---

## 🚀 立即行动

```bash
# 第一步: 初始化环境
cd /Users/micago/Desktop/index/video-auto-dubbing
./scripts/setup_migration_worktrees.sh

# 第二步: 查看文档
cat docs/migration/README.md

# 第三步: 选择任务开始开发
cd ~/worktrees/video-dubbing
source goto.sh asr    # 或 oss, llm, tts, cleanup
```

**祝迁移顺利！🎊**

---

## 📞 联系方式

| 角色 | 负责人 | 联系方式 |
|-----|-------|---------|
| 项目负责人 | TBD | ___ |
| 技术负责人 | TBD | ___ |
| QA 负责人 | TBD | ___ |

---

**文档版本**: v1.0
**最后更新**: 2026-02-02
**维护者**: 项目团队

---

**📖 更多详情，请查看**: [docs/migration/](docs/migration/)
