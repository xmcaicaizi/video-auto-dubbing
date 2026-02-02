# 统一阿里云百炼平台迁移计划

> **目标**: 将所有外部服务（ASR/OSS/LLM/TTS）统一迁移到阿里云百炼平台（DashScope）
> **创建时间**: 2026-02-02
> **负责人**: 待定
> **预计完成**: 2周

---

## 📋 迁移概览

### 当前状态

| 服务类型 | 当前使用 | 迁移目标 | 优先级 |
|---------|---------|---------|--------|
| **ASR** (语音识别) | 火山引擎 | ✅ **Qwen-Audio** (支持时间戳+说话人) | P1 - 高 |
| **OSS** (对象存储) | MinIO (自建) | ✅ **阿里云 OSS** (已实现) | P1 - 高 |
| **LLM** (翻译) | 智谱GLM | ✅ **DashScope** (已实现) | P1 - 高 |
| **TTS** (语音合成) | IndexTTS-vLLM | ✅ **Qwen3-TTS 自部署** (全新方案) | P1 - 高 |

### 迁移收益

#### 💰 成本优化
- ❌ 移除本地 GPU 服务器依赖 (IndexTTS)
- ❌ 移除 MinIO 自建存储维护成本
- ✅ 按需付费，无闲置资源成本
- ✅ 统一阿里云账单，便于成本管理

#### 🚀 性能提升
- ⚡ qwen-tts-flash 低延迟合成
- ⚡ OSS CDN 加速文件分发
- ⚡ 多区域部署支持

#### 🛠️ 运维简化
- 📦 无需维护 GPU 环境和 Docker 镜像
- 🔧 统一的 API Key 管理
- 📊 统一的监控和日志平台
- 🔄 更简单的扩容和降级

---

## 🗺️ Git Worktree 开发策略

### 分支规划

```
main (生产主分支)
├── feature/aliyun-unified-migration (总分支)
│   ├── feature/aliyun-asr-default (子任务1)
│   ├── feature/aliyun-oss-default (子任务2)
│   ├── feature/aliyun-llm-default (子任务3)
│   ├── feature/aliyun-tts-default (子任务4)
│   └── feature/cleanup-legacy-services (子任务5)
```

### Worktree 创建命令

```bash
# 创建工作目录
mkdir -p ~/worktrees/video-dubbing

# 创建主迁移分支
git worktree add ~/worktrees/video-dubbing/aliyun-migration -b feature/aliyun-unified-migration

# 创建子任务分支 (从主迁移分支创建)
cd ~/worktrees/video-dubbing/aliyun-migration

git worktree add ~/worktrees/video-dubbing/asr-default -b feature/aliyun-asr-default
git worktree add ~/worktrees/video-dubbing/oss-default -b feature/aliyun-oss-default
git worktree add ~/worktrees/video-dubbing/llm-default -b feature/aliyun-llm-default
git worktree add ~/worktrees/video-dubbing/tts-default -b feature/aliyun-tts-default
git worktree add ~/worktrees/video-dubbing/cleanup-legacy -b feature/cleanup-legacy-services
```

### 工作流程

1. **并行开发**: 每个子任务在独立 worktree 中开发
2. **定期同步**: 每日将子任务合并到主迁移分支
3. **集成测试**: 在主迁移分支进行完整测试
4. **最终合并**: 测试通过后合并到 main

---

## 📊 任务分解

### Phase 1: ASR 迁移 (3天)

#### 任务1.1: 设置阿里云 ASR 为默认
**分支**: `feature/aliyun-asr-default`
**优先级**: P1 - 高
**预计工时**: 1天

**详细步骤**:

1. **配置文件修改**
   - [ ] `shared/config/config.go`: 将 `ASR_BACKEND` 默认值改为 `"aliyun"`
   - [ ] `.env.example`: 更新 ASR 配置示例
   - [ ] `docker-compose.yml`: 添加阿里云 ASR 环境变量

2. **代码优化**
   - [ ] `worker/internal/asr/client.go`: 优化客户端工厂逻辑
   - [ ] `worker/internal/asr/aliyun_client.go`:
     - 添加更详细的错误处理
     - 优化轮询逻辑
     - 支持批量处理
   - [ ] 添加优雅降级到火山引擎的逻辑（API Key 未配置时）

3. **文档更新**
   - [ ] `docs/quick-start.md`: 更新快速开始指南
   - [ ] `docs/aliyun_asr_integration.md`: 完善集成文档
   - [ ] `README.md`: 更新主文档

4. **测试**
   - [ ] 单元测试: `tests/test_aliyun_asr.go`
   - [ ] 集成测试: 完整视频转写流程
   - [ ] 性能测试: 对比火山引擎和阿里云延迟

**验收标准**:
- ✅ 新项目默认使用阿里云 ASR
- ✅ 火山引擎作为备选方案可正常切换
- ✅ 所有测试通过
- ✅ 文档完整且准确

---

### Phase 2: OSS 迁移 (3天)

#### 任务2.1: 设置阿里云 OSS 为默认
**分支**: `feature/aliyun-oss-default`
**优先级**: P1 - 高
**预计工时**: 1.5天

**详细步骤**:

1. **配置文件修改**
   - [ ] `shared/config/config.go`: 将 `STORAGE_BACKEND` 默认值改为 `"oss"`
   - [ ] `.env.example`:
     ```bash
     # 对象存储配置
     STORAGE_BACKEND=oss  # 默认使用阿里云 OSS

     # 阿里云 OSS 配置
     OSS_ENDPOINT=oss-cn-beijing.aliyuncs.com
     OSS_BUCKET=video-dubbing
     OSS_ACCESS_KEY_ID=LTAI5...
     OSS_ACCESS_KEY_SECRET=xxx
     OSS_PUBLIC_DOMAIN=cdn.example.com  # 可选CDN域名
     OSS_PREFIX=dubbing/  # 可选路径前缀
     OSS_USE_SSL=true
     ```
   - [ ] `docker-compose.yml`: 移除 MinIO 服务，添加 OSS 配置注释

2. **存储层优化**
   - [ ] `shared/storage/oss_storage.go`:
     - 添加分片上传支持（大文件优化）
     - 实现 CDN 加速配置
     - 添加生命周期管理接口
   - [ ] `shared/storage/storage.go`:
     - 优化工厂模式逻辑
     - 添加存储健康检查
   - [ ] 添加 OSS 到 OSS 的对象复制功能（无需下载再上传）

3. **迁移工具开发**
   - [ ] `scripts/migrate_minio_to_oss.go`:
     ```go
     // MinIO -> OSS 数据迁移工具
     // 功能:
     // 1. 列举 MinIO 所有对象
     // 2. 批量复制到 OSS
     // 3. 验证数据完整性
     // 4. 更新数据库中的 URL
     // 5. 生成迁移报告
     ```

4. **数据库迁移**
   - [ ] 创建迁移脚本更新现有任务的存储 URL
   - [ ] 添加回滚脚本

5. **文档更新**
   - [ ] `docs/guides/oss-migration-guide.md`: 新建迁移指南
   - [ ] `docs/deployment-guide.md`: 更新部署文档
   - [ ] `docs/troubleshooting/oss-issues.md`: 新建故障排除文档

**验收标准**:
- ✅ 新上传文件使用 OSS
- ✅ 提供 MinIO 到 OSS 的迁移工具
- ✅ 支持 CDN 加速（可选）
- ✅ 所有存储操作测试通过

#### 任务2.2: MinIO 遗留数据迁移
**分支**: `feature/aliyun-oss-default`
**优先级**: P2 - 中
**预计工时**: 1.5天

**详细步骤**:
- [ ] 执行数据迁移脚本
- [ ] 验证数据完整性
- [ ] 更新数据库引用
- [ ] 生成迁移报告
- [ ] (可选) 保留 MinIO 作为冷备份

---

### Phase 3: LLM 翻译迁移 (2天)

#### 任务3.1: 设置阿里云百炼为默认翻译服务
**分支**: `feature/aliyun-llm-default`
**优先级**: P1 - 高
**预计工时**: 1天

**详细步骤**:

1. **配置文件修改**
   - [ ] `worker/internal/translate/factory.go`:
     ```go
     // 修改默认翻译服务
     func NewTranslateClient(...) (TranslateClient, error) {
         // 优先使用 DashScope
         provider := settings.GetString("translate.provider", "dashscope")
         // ...
     }
     ```
   - [ ] `.env.example`: 移除 GLM 配置，保留作为注释

2. **翻译客户端优化**
   - [ ] `worker/internal/translate/dashscope_client.go`:
     - 优化批处理逻辑
     - 添加术语表支持
     - 实现翻译缓存机制
     - 支持自定义 Prompt 模板
   - [ ] 添加翻译质量评分接口

3. **功能增强**
   - [ ] 实现翻译记忆库（避免重复翻译）
   - [ ] 支持多模型对比（qwen-turbo vs qwen-plus）
   - [ ] 添加翻译后处理（格式保持、特殊字符处理）

4. **测试**
   - [ ] 单元测试: 各种语言对翻译
   - [ ] 性能测试: 批处理vs单条请求
   - [ ] 质量测试: 对比GLM翻译结果

**验收标准**:
- ✅ 默认使用 DashScope 翻译
- ✅ GLM 作为备选可切换
- ✅ 翻译质量不低于 GLM
- ✅ 性能提升 20%+

#### 任务3.2: 翻译质量监控
**分支**: `feature/aliyun-llm-default`
**优先级**: P2 - 中
**预计工时**: 1天

**详细步骤**:
- [ ] 添加翻译质量指标收集
- [ ] 实现 A/B 测试框架
- [ ] 生成翻译质量报告
- [ ] 添加异常翻译告警

---

### Phase 4: TTS 迁移 (3天)

#### 任务4.1: 设置阿里云 TTS 为默认
**分支**: `feature/aliyun-tts-default`
**优先级**: P1 - 高
**预计工时**: 2天

**详细步骤**:

1. **配置文件修改**
   - [ ] `shared/config/config.go`: 将 `TTS_BACKEND` 默认值改为 `"aliyun"`
   - [ ] `.env.example`:
     ```bash
     # TTS 服务配置
     TTS_BACKEND=aliyun  # 默认使用阿里云百炼

     # 阿里云百炼 TTS 配置
     DASHSCOPE_API_KEY=sk-xxx
     DASHSCOPE_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
     DASHSCOPE_MODEL=qwen-tts-flash
     ```

2. **TTS 客户端优化**
   - [ ] `worker/internal/tts/aliyun_client.go`:
     - 添加流式合成支持（降低首字节延迟）
     - 实现音色映射优化
     - 支持情绪和语速控制
     - 添加合成音频质量检测
   - [ ] `worker/internal/tts/client.go`: 优化客户端工厂

3. **音色系统增强**
   - [ ] 创建音色配置文件 `configs/voice_mapping.yaml`:
     ```yaml
     voices:
       default: longxiaochun
       female_young: Cherry
       male_young: Alex
       female_mature: Serena
       male_mature: Ethan
       # 支持自定义音色映射
     ```
   - [ ] 实现动态音色加载和验证

4. **性能优化**
   - [ ] 实现并发合成控制
   - [ ] 添加合成结果缓存（相同文本+音色复用）
   - [ ] 优化大段文本分片逻辑

5. **迁移 TTS 服务依赖**
   - [ ] `docker-compose.yml`: 移除 tts_service 容器
   - [ ] 清理相关的 Python 代码

**验收标准**:
- ✅ 新项目默认使用阿里云 TTS
- ✅ 移除 IndexTTS 服务依赖
- ✅ 支持流式合成
- ✅ 音色映射准确
- ✅ 合成速度不低于 IndexTTS

#### 任务4.2: TTS 降级和容错
**分支**: `feature/aliyun-tts-default`
**优先级**: P1 - 高
**预计工时**: 1天

**详细步骤**:
- [ ] 实现 TTS 服务健康检查
- [ ] 添加自动降级机制（阿里云失败 -> 重试 -> 告警）
- [ ] 实现合成失败的详细日志
- [ ] 添加合成质量监控指标

---

### Phase 5: 清理遗留服务 (2天)

#### 任务5.1: 移除不必要的服务和代码
**分支**: `feature/cleanup-legacy-services`
**优先级**: P2 - 中
**预计工时**: 1天

**详细步骤**:

1. **Docker Compose 清理**
   - [ ] 移除 MinIO 服务定义
   - [ ] 移除 tts_service 服务定义
   - [ ] 精简环境变量配置

2. **代码清理**
   - [ ] 移动到 backup/:
     - `tts_service/` 目录
     - `worker/internal/tts/gradio_client.go`
     - `worker/internal/tts/vllm_client.go`
   - [ ] 删除未使用的测试文件
   - [ ] 清理注释掉的代码

3. **依赖清理**
   - [ ] 移除 Python TTS 服务的依赖
   - [ ] 清理 Go 依赖: `go mod tidy`

**验收标准**:
- ✅ docker-compose.yml 精简至最小
- ✅ 遗留代码移动到 backup/ 目录
- ✅ 依赖清理完成
- ✅ 系统正常启动

#### 任务5.2: 文档整理和更新
**分支**: `feature/cleanup-legacy-services`
**优先级**: P2 - 中
**预计工时**: 1天

**详细步骤**:

1. **主文档更新**
   - [ ] `README.md`:
     - 更新架构图
     - 更新快速开始指南
     - 强调阿里云百炼统一平台
   - [ ] `docs/ARCHITECTURE_OVERVIEW.md`: 更新架构说明
   - [ ] `docs/quick-start.md`: 简化配置步骤

2. **新增文档**
   - [ ] `docs/guides/ALIYUN_UNIFIED_PLATFORM.md`: 统一平台配置指南
   - [ ] `docs/migration/FROM_LEGACY_TO_ALIYUN.md`: 遗留系统迁移指南
   - [ ] `docs/cost-analysis.md`: 成本分析文档

3. **遗留文档归档**
   - [ ] 移动到 `docs/archive/legacy/`:
     - 火山引擎相关文档
     - GLM 翻译文档
     - IndexTTS 文档
     - MinIO 配置文档

**验收标准**:
- ✅ 文档准确反映新架构
- ✅ 遗留文档归档但可查阅
- ✅ 新用户能快速上手

---

## 🧪 测试策略

### 单元测试
```bash
# 每个子任务的测试
cd ~/worktrees/video-dubbing/asr-default
go test ./worker/internal/asr/... -v

cd ~/worktrees/video-dubbing/oss-default
go test ./shared/storage/... -v

cd ~/worktrees/video-dubbing/llm-default
go test ./worker/internal/translate/... -v

cd ~/worktrees/video-dubbing/tts-default
go test ./worker/internal/tts/... -v
```

### 集成测试
```bash
# 在主迁移分支运行完整测试
cd ~/worktrees/video-dubbing/aliyun-migration
./scripts/integration_test.sh
```

### 性能测试
```bash
# 对比测试: 旧系统 vs 新系统
./scripts/performance_comparison.sh
```

### 回归测试
```bash
# 确保现有功能不受影响
./scripts/regression_test.sh
```

---

## 📈 进度追踪

### Week 1 (Day 1-5)
- [x] 制定迁移计划
- [ ] 创建 worktree 环境
- [ ] Phase 1: ASR 迁移
- [ ] Phase 2: OSS 迁移 (开始)

### Week 2 (Day 6-10)
- [ ] Phase 2: OSS 迁移 (完成)
- [ ] Phase 3: LLM 翻译迁移
- [ ] Phase 4: TTS 迁移 (开始)

### Week 3 (Day 11-14)
- [ ] Phase 4: TTS 迁移 (完成)
- [ ] Phase 5: 清理遗留服务
- [ ] 集成测试和性能验证
- [ ] 文档更新和 Code Review
- [ ] 合并到 main 分支

---

## 🔄 合并策略

### 每日同步
```bash
# 每个子任务完成后
cd ~/worktrees/video-dubbing/asr-default
git add .
git commit -m "feat(asr): implement aliyun asr as default"
git push origin feature/aliyun-asr-default

# 合并到主迁移分支
cd ~/worktrees/video-dubbing/aliyun-migration
git merge feature/aliyun-asr-default
git push origin feature/aliyun-unified-migration
```

### 最终合并
```bash
# 所有测试通过后
cd /Users/micago/Desktop/index/video-auto-dubbing
git merge --no-ff feature/aliyun-unified-migration
git push origin main
```

---

## ⚠️ 风险管理

### 潜在风险

| 风险 | 影响 | 缓解措施 |
|-----|------|---------|
| API 配额不足 | 高 | 提前采购配额，实现限流和排队 |
| 数据迁移失败 | 高 | 完整的备份策略，渐进式迁移 |
| 性能下降 | 中 | 性能对比测试，优化批处理逻辑 |
| 音色不匹配 | 中 | 建立音色映射表，人工验证 |
| OSS 成本超预算 | 中 | 设置生命周期规则，监控用量 |

### 回滚计划
- 保留所有旧服务配置
- 数据库迁移脚本可逆
- 使用 feature flag 控制新旧切换

---

## 📞 联系人

| 角色 | 负责人 | 职责 |
|-----|-------|------|
| 项目负责人 | TBD | 整体协调和进度管理 |
| ASR 负责人 | TBD | ASR 迁移和测试 |
| 存储负责人 | TBD | OSS 迁移和数据验证 |
| 翻译负责人 | TBD | LLM 迁移和质量验证 |
| TTS 负责人 | TBD | TTS 迁移和音色验证 |
| QA 负责人 | TBD | 测试和质量保证 |

---

## 📚 相关文档

- [阿里云百炼平台文档](https://help.aliyun.com/zh/model-studio/)
- [DashScope API 文档](https://help.aliyun.com/zh/model-studio/developer-reference/api-details)
- [阿里云 OSS 文档](https://help.aliyun.com/zh/oss/)
- [项目架构文档](../ARCHITECTURE_OVERVIEW.md)

---

**最后更新**: 2026-02-02
**版本**: v1.0
**状态**: Draft - 待审核
