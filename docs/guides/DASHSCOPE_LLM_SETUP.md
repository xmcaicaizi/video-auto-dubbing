# 阿里百炼 DashScope LLM 翻译服务配置指南

## 🎯 概述

本系统已统一迁移至**阿里百炼平台（DashScope）**作为默认的LLM翻译服务提供商。阿里百炼提供强大的多语言翻译能力，支持视频字幕的高质量翻译。

### 为什么选择阿里百炼？

- 🚀 **高性能**：qwen-turbo 等模型响应速度快
- 🌍 **多语言支持**：支持100+种语言的专业翻译
- 💰 **性价比高**：按量计费，成本可控
- 🔄 **兼容性好**：Compatible-mode API 兼容 OpenAI 格式
- 🛡️ **稳定可靠**：企业级服务保障

## 📋 配置步骤

### 第一步：获取 API Key

1. 访问 [阿里云百炼控制台](https://bailian.console.aliyun.com/)
2. 登录您的阿里云账号（如果没有账号，需要先注册）
3. 进入「API-KEY管理」页面
4. 点击「创建新的API-KEY」
5. **重要**：立即复制并保存 API Key，关闭后将无法再次查看

### 第二步：配置系统

#### 方式1：环境变量配置（推荐）

编辑项目根目录的 `.env` 文件：

```bash
# 翻译服务配置 - 阿里百炼 DashScope
DASHSCOPE_LLM_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxx  # 你的 API Key
DASHSCOPE_LLM_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
DASHSCOPE_LLM_MODEL=qwen-turbo  # 推荐模型
DASHSCOPE_LLM_RPS=5.0  # 请求速率限制（每秒请求数）
```

#### 方式2：数据库配置（通过Web界面）

1. 访问系统设置页面（如 `http://localhost:3000/settings`）
2. 找到「翻译服务配置」部分
3. 选择翻译服务提供商：`dashscope`
4. 填写以下信息：
   - **API Key**：你的阿里百炼 API Key
   - **Base URL**：`https://dashscope.aliyuncs.com/compatible-mode/v1`
   - **Model**：`qwen-turbo`（或其他支持的模型）
5. 点击「测试连接」验证配置
6. 点击「保存设置」

### 第三步：重启服务

```bash
# 使用 Docker Compose
docker-compose restart worker

# 或重启所有服务
docker-compose restart
```

## 🔧 支持的模型

阿里百炼支持多种大语言模型，根据需求选择：

| 模型名称 | 特点 | 推荐场景 |
|---------|------|----------|
| `qwen-turbo` | 快速响应，性价比高 | **默认推荐**，适合大量翻译任务 |
| `qwen-plus` | 更强的理解能力 | 复杂文本翻译 |
| `qwen-max` | 最强性能 | 专业文档翻译 |
| `qwen-long` | 超长上下文 | 长篇内容翻译 |

配置示例：
```bash
DASHSCOPE_LLM_MODEL=qwen-plus  # 切换到 qwen-plus
```

## 📊 配置验证

### 检查配置是否生效

```bash
# 查看 Worker 日志
docker-compose logs worker | grep -i dashscope

# 应该看到类似输出：
# "Creating DashScope translator" model="qwen-turbo" base_url="https://dashscope.aliyuncs.com/compatible-mode/v1"
```

### 测试翻译功能

通过API测试翻译：

```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "video_url": "https://example.com/test.mp4",
    "source_language": "zh",
    "target_language": "en"
  }'
```

查看任务日志，确认使用了 DashScope 进行翻译。

## 🔄 从智谱GLM迁移

如果你之前使用的是智谱GLM，迁移到阿里百炼非常简单：

### 保留GLM作为备选（可选）

系统支持同时配置多个翻译服务，可以通过数据库设置切换：

```bash
# 保留 GLM 配置
GLM_API_KEY=your_glm_api_key
GLM_API_URL=https://open.bigmodel.cn/api/paas/v4/chat/completions
GLM_MODEL=glm-4-flash

# 同时配置 DashScope
DASHSCOPE_LLM_API_KEY=your_dashscope_api_key
DASHSCOPE_LLM_MODEL=qwen-turbo
```

在数据库 `settings` 表中设置翻译服务提供商：

```sql
-- 切换到 DashScope
INSERT INTO settings (category, key, value)
VALUES ('translate', 'provider', 'dashscope')
ON CONFLICT (category, key)
DO UPDATE SET value = 'dashscope';

-- 或切换回 GLM
UPDATE settings
SET value = 'glm'
WHERE category = 'translate' AND key = 'provider';
```

## 🎛️ 高级配置

### 速率限制

根据你的API配额调整请求速率：

```bash
# 每秒5个请求（默认）
DASHSCOPE_LLM_RPS=5.0

# 如果有更高配额，可以增加
DASHSCOPE_LLM_RPS=10.0
```

### 批处理优化

在 worker 配置中调整批处理大小：

```bash
# worker/internal/config/config.go 中的默认值
TRANSLATE_BATCH_SIZE=20        # 每批翻译的句子数
TRANSLATE_ITEM_MAX_RETRIES=2   # 单个句子的最大重试次数
TRANSLATE_MAX_TEXT_LENGTH=4000 # 单个句子最大长度
```

## 🔍 故障排除

### 问题1：API Key 无效

**症状**：
```
error: DASHSCOPE_LLM_API_KEY is required
```

**解决方案**：
- 确认 API Key 正确复制，没有多余空格
- 检查 `.env` 文件中的配置
- 重启 worker 服务使配置生效

### 问题2：请求速率限制

**症状**：
```
DashScope API returned status 429: Too Many Requests
```

**解决方案**：
- 降低 `DASHSCOPE_LLM_RPS` 值
- 检查阿里云控制台的配额使用情况
- 考虑升级API配额

### 问题3：翻译质量不佳

**解决方案**：
- 尝试使用 `qwen-plus` 或 `qwen-max` 模型
- 调整翻译prompt（在 `dashscope_client.go` 中）
- 检查源语言和目标语言设置是否正确

### 问题4：连接超时

**症状**：
```
failed to call DashScope translation API: context deadline exceeded
```

**解决方案**：
- 检查网络连接
- 确认防火墙允许访问 `dashscope.aliyuncs.com`
- 增加超时时间（在 `dashscope_client.go` 中修改 `http.Client.Timeout`）

## 💡 最佳实践

1. **监控配额使用**：定期查看阿里云控制台，避免超出配额
2. **合理设置速率**：根据实际任务量调整 RPS
3. **日志监控**：通过日志监控翻译质量和成功率
4. **成本优化**：大量任务使用 qwen-turbo，重要任务使用 qwen-plus

## 📚 相关文档

- [阿里百炼官方文档](https://help.aliyun.com/zh/model-studio/)
- [DashScope API 文档](https://help.aliyun.com/zh/model-studio/developer-reference/api-details)
- [Qwen 模型介绍](https://qwenlm.github.io/)

## 🆘 获取帮助

如果遇到问题：
1. 查看 [故障排除文档](../troubleshooting/README.md)
2. 检查 GitHub Issues
3. 提交新的 Issue 并附上详细日志

---

**配置完成后，建议进行完整的翻译测试以确保功能正常！**
