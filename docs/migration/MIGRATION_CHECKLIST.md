# 阿里云统一平台迁移检查清单

> 🎯 **目标**: 快速追踪迁移进度，确保所有任务按计划完成
> 📅 **更新**: 每日更新进度
> ✅ **完成标准**: 所有复选框打勾，测试全部通过

---

## 📋 总体进度

```
进度: [░░░░░░░░░░░░░░░░░░░░] 0% (0/40 完成)

Phase 1: ASR 迁移       [░░░░░░░░░░] 0% (0/8)
Phase 2: OSS 迁移       [░░░░░░░░░░] 0% (0/10)
Phase 3: LLM 翻译迁移   [░░░░░░░░░░] 0% (0/8)
Phase 4: TTS 迁移       [░░░░░░░░░░] 0% (0/10)
Phase 5: 清理遗留服务   [░░░░░░░░░░] 0% (0/4)
```

---

## Phase 1: ASR 迁移 (Day 1-3)

### 配置文件修改
- [ ] `shared/config/config.go`: 修改 ASR_BACKEND 默认值为 "aliyun"
- [ ] `.env.example`: 更新 ASR 配置示例
- [ ] `docker-compose.yml`: 添加阿里云 ASR 环境变量注释

### 代码实现
- [ ] `worker/internal/asr/aliyun_client.go`: 优化错误处理
- [ ] `worker/internal/asr/aliyun_client.go`: 优化轮询逻辑
- [ ] `worker/internal/asr/client.go`: 优化客户端工厂逻辑
- [ ] 添加优雅降级到火山引擎的逻辑

### 测试
- [ ] 单元测试: ASR 客户端功能
- [ ] 集成测试: 完整视频转写流程
- [ ] 性能测试: 对比火山引擎和阿里云延迟
- [ ] 回归测试: 确保现有功能不受影响

### 文档
- [ ] `docs/quick-start.md`: 更新快速开始指南
- [ ] `docs/aliyun_asr_integration.md`: 完善集成文档
- [ ] `README.md`: 更新主文档 ASR 部分

**负责人**: ___________
**状态**: ⏸️ 未开始
**预计完成**: Day 3

---

## Phase 2: OSS 迁移 (Day 3-6)

### 配置文件修改
- [ ] `shared/config/config.go`: 修改 STORAGE_BACKEND 默认值为 "oss"
- [ ] `.env.example`: 添加完整的 OSS 配置示例
- [ ] `docker-compose.yml`: 移除 MinIO 服务定义

### 存储层优化
- [ ] `shared/storage/oss_storage.go`: 添加分片上传支持
- [ ] `shared/storage/oss_storage.go`: 实现 CDN 加速配置
- [ ] `shared/storage/oss_storage.go`: 添加生命周期管理接口
- [ ] `shared/storage/storage.go`: 优化工厂模式逻辑
- [ ] `shared/storage/storage.go`: 添加存储健康检查

### 迁移工具
- [ ] `scripts/migrate_minio_to_oss.go`: 实现数据迁移工具
- [ ] `scripts/migrate_minio_to_oss.go`: 添加数据完整性验证
- [ ] `scripts/migrate_minio_to_oss.go`: 实现 URL 更新逻辑
- [ ] 创建数据库迁移脚本
- [ ] 创建回滚脚本

### 测试
- [ ] 单元测试: OSS 客户端功能
- [ ] 集成测试: 文件上传、下载、删除
- [ ] 性能测试: 对比 MinIO 和 OSS 速度
- [ ] 迁移测试: 验证数据迁移正确性

### 文档
- [ ] `docs/guides/oss-migration-guide.md`: 创建迁移指南
- [ ] `docs/deployment-guide.md`: 更新部署文档
- [ ] `docs/troubleshooting/oss-issues.md`: 创建故障排除文档

**负责人**: ___________
**状态**: ⏸️ 未开始
**预计完成**: Day 6

---

## Phase 3: LLM 翻译迁移 (Day 6-8)

### 配置文件修改
- [ ] `worker/internal/translate/factory.go`: 修改默认翻译服务为 "dashscope"
- [ ] `.env.example`: 更新翻译服务配置示例

### 翻译客户端优化
- [ ] `worker/internal/translate/dashscope_client.go`: 优化批处理逻辑
- [ ] `worker/internal/translate/dashscope_client.go`: 添加术语表支持
- [ ] `worker/internal/translate/dashscope_client.go`: 实现翻译缓存机制
- [ ] `worker/internal/translate/dashscope_client.go`: 支持自定义 Prompt 模板

### 功能增强
- [ ] 实现翻译记忆库（避免重复翻译）
- [ ] 支持多模型对比（qwen-turbo vs qwen-plus）
- [ ] 添加翻译后处理逻辑
- [ ] 添加翻译质量评分接口

### 测试
- [ ] 单元测试: 各种语言对翻译
- [ ] 性能测试: 批处理 vs 单条请求
- [ ] 质量测试: 对比 GLM 翻译结果
- [ ] 回归测试: 确保翻译准确性

### 文档
- [ ] `docs/guides/DASHSCOPE_LLM_SETUP.md`: 更新配置指南
- [ ] 添加翻译质量监控文档

**负责人**: ___________
**状态**: ⏸️ 未开始
**预计完成**: Day 8

---

## Phase 4: TTS 迁移 (Day 8-11)

### 配置文件修改
- [ ] `shared/config/config.go`: 修改 TTS_BACKEND 默认值为 "aliyun"
- [ ] `.env.example`: 更新 TTS 配置示例
- [ ] `docker-compose.yml`: 移除 tts_service 服务定义

### TTS 客户端优化
- [ ] `worker/internal/tts/aliyun_client.go`: 添加流式合成支持
- [ ] `worker/internal/tts/aliyun_client.go`: 优化音色映射逻辑
- [ ] `worker/internal/tts/aliyun_client.go`: 支持情绪和语速控制
- [ ] `worker/internal/tts/aliyun_client.go`: 添加合成音频质量检测
- [ ] `worker/internal/tts/client.go`: 优化客户端工厂

### 音色系统增强
- [ ] 创建音色配置文件 `configs/voice_mapping.yaml`
- [ ] 实现动态音色加载和验证

### 性能优化
- [ ] 实现并发合成控制
- [ ] 添加合成结果缓存（相同文本+音色复用）
- [ ] 优化大段文本分片逻辑

### 降级和容错
- [ ] 实现 TTS 服务健康检查
- [ ] 添加自动降级机制
- [ ] 实现合成失败的详细日志
- [ ] 添加合成质量监控指标

### 测试
- [ ] 单元测试: TTS 客户端功能
- [ ] 集成测试: 完整合成流程
- [ ] 性能测试: 对比 IndexTTS 速度
- [ ] 质量测试: 音色准确性验证

### 文档
- [ ] `docs/aliyun_tts_integration.md`: 更新集成文档
- [ ] 添加 TTS 故障排除文档

**负责人**: ___________
**状态**: ⏸️ 未开始
**预计完成**: Day 11

---

## Phase 5: 清理遗留服务 (Day 11-14)

### Docker Compose 清理
- [ ] 移除 MinIO 服务定义
- [ ] 移除 tts_service 服务定义
- [ ] 精简环境变量配置

### 代码清理
- [ ] 移动 `tts_service/` 到 `backup/`
- [ ] 移动 `worker/internal/tts/gradio_client.go` 到 `backup/`
- [ ] 移动 `worker/internal/tts/vllm_client.go` 到 `backup/`
- [ ] 删除未使用的测试文件
- [ ] 清理注释掉的代码

### 依赖清理
- [ ] 移除 Python TTS 服务依赖
- [ ] 运行 `go mod tidy` 清理 Go 依赖

### 文档整理
- [ ] `README.md`: 更新架构图和快速开始
- [ ] `docs/ARCHITECTURE_OVERVIEW.md`: 更新架构说明
- [ ] `docs/quick-start.md`: 简化配置步骤
- [ ] 创建 `docs/guides/ALIYUN_UNIFIED_PLATFORM.md`
- [ ] 创建 `docs/migration/FROM_LEGACY_TO_ALIYUN.md`
- [ ] 归档遗留文档到 `docs/archive/legacy/`

### 测试
- [ ] 验证系统正常启动
- [ ] 验证所有功能正常工作
- [ ] 运行完整的回归测试套件

**负责人**: ___________
**状态**: ⏸️ 未开始
**预计完成**: Day 14

---

## 🧪 最终验证清单

### 功能测试
- [ ] 完整视频处理流程（上传 → ASR → 翻译 → TTS → 合成）
- [ ] 所有语言对翻译正确
- [ ] 所有音色合成正确
- [ ] 错误处理和重试机制正常
- [ ] 文件存储和访问正常

### 性能测试
- [ ] ASR 处理速度不低于火山引擎
- [ ] 翻译速度提升 20%+
- [ ] TTS 合成速度不低于 IndexTTS
- [ ] 文件上传/下载速度满足要求
- [ ] 系统整体处理速度对比测试

### 稳定性测试
- [ ] 长时间运行测试（24小时）
- [ ] 高并发测试（10+ 任务并行）
- [ ] 异常情况测试（网络中断、API 限流等）
- [ ] 资源使用监控（CPU、内存、网络）

### 成本评估
- [ ] OSS 存储成本评估
- [ ] API 调用成本评估
- [ ] 总成本对比（vs 自建服务）
- [ ] 成本优化建议

### 文档审查
- [ ] 所有文档更新完成
- [ ] 配置示例准确
- [ ] 故障排除指南完整
- [ ] 迁移指南清晰

### Code Review
- [ ] 代码符合项目规范
- [ ] 单元测试覆盖率 > 80%
- [ ] 无安全漏洞
- [ ] 性能优化到位
- [ ] 日志记录完整

---

## 📊 里程碑

| 里程碑 | 日期 | 状态 | 备注 |
|-------|------|------|------|
| M1: Worktree 环境初始化 | Day 1 | ⏸️ 未开始 | 运行 `setup_migration_worktrees.sh` |
| M2: ASR 迁移完成 | Day 3 | ⏸️ 未开始 | Phase 1 所有任务完成 |
| M3: OSS 迁移完成 | Day 6 | ⏸️ 未开始 | Phase 2 所有任务完成，数据迁移完成 |
| M4: LLM 迁移完成 | Day 8 | ⏸️ 未开始 | Phase 3 所有任务完成 |
| M5: TTS 迁移完成 | Day 11 | ⏸️ 未开始 | Phase 4 所有任务完成，IndexTTS 下线 |
| M6: 清理完成 | Day 13 | ⏸️ 未开始 | Phase 5 所有任务完成 |
| M7: 最终验证 | Day 14 | ⏸️ 未开始 | 所有测试通过 |
| M8: 合并到 main | Day 14 | ⏸️ 未开始 | PR 审核通过并合并 |

---

## 🚨 风险项

| 风险 | 影响 | 状态 | 缓解措施 | 负责人 |
|-----|------|------|---------|--------|
| API 配额不足 | 🔴 高 | ⚠️ 监控中 | 提前采购配额 | ___ |
| 数据迁移失败 | 🔴 高 | ⏸️ 待评估 | 完整备份+回滚方案 | ___ |
| 性能下降 | 🟡 中 | ⏸️ 待测试 | 性能对比测试 | ___ |
| 音色不匹配 | 🟡 中 | ⏸️ 待验证 | 人工验证+映射表 | ___ |
| OSS 成本超预算 | 🟡 中 | ⏸️ 待评估 | 生命周期规则 | ___ |

---

## 📝 每日站会记录

### Day 1 - YYYY-MM-DD
- **完成**:
- **问题**:
- **计划**:
- **备注**:

### Day 2 - YYYY-MM-DD
- **完成**:
- **问题**:
- **计划**:
- **备注**:

### Day 3 - YYYY-MM-DD
- **完成**:
- **问题**:
- **计划**:
- **备注**:

*(继续添加...)*

---

## 📞 联系人

| 角色 | 姓名 | 联系方式 | 负责模块 |
|-----|------|---------|---------|
| 项目负责人 | ___ | ___ | 整体协调 |
| ASR 负责人 | ___ | ___ | Phase 1 |
| 存储负责人 | ___ | ___ | Phase 2 |
| 翻译负责人 | ___ | ___ | Phase 3 |
| TTS 负责人 | ___ | ___ | Phase 4 |
| QA 负责人 | ___ | ___ | 测试验证 |

---

**最后更新**: 2026-02-02
**版本**: v1.0
**下次更新**: 每日 18:00
