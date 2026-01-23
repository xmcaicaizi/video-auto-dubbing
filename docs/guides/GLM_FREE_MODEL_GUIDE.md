# 🆓 GLM-4-Flash 免费模型使用指南

## 📋 模型信息

### GLM-4-Flash-250414 特性
- **💰 智谱AI首个免费大模型**: 完全免费使用
- **🌐 超长上下文**: 128K 上下文长度，相当于 300 页书籍
- **🗣️ 多语言支持**: 支持多达 26 种语言翻译
- **⚡ 实时响应**: 支持流式输出，提升交互体验
- **🎯 智能翻译**: 专业翻译能力，理解文化背景和语义细微差别
- **📄 官方文档**: https://bigmodel.cn/dev/activities/free/glm-4-flash?gad_source=1&gad_campaignid=23375921191&gbraid=0AAAABBUMtgQOaMp5WGF7jZkWzAgboub1Q&gclid=Cj0KCQiA1czLBhDhARIsAIEc7ug0vfMFqoZfH3brlpsjH6CWuigCv-twBihDyyEI1x7HK9u9Bva9zTsaAvwLEALw_wcB

### 模型参数
- **模型名称**: `glm-4-flash`
- **API 接口**: `https://open.bigmodel.cn/api/paas/v4/chat/completions`
- **费用**: 完全免费 🆓
- **上下文长度**: 128K tokens
- **支持语言**: 26 种主流语言
- **特色能力**: 流式输出、结构化输出、Function Call

## 🔧 系统配置更新

### 默认配置已更新
系统已将 GLM-4-Flash 设置为默认翻译模型：

1. **前端界面**: GLM-4-Flash 已设为默认选项
2. **后端配置**: 所有配置文件已更新
3. **数据库默认值**: 新安装将使用 GLM-4-Flash
4. **API 服务**: 测试接口已更新

### 配置位置
```
📁 前端配置: web/index.html
📁 共享配置: shared/config/config.go
📁 API 配置: api/internal/service/settings.go
📁 Worker 配置: worker/internal/settings/loader.go
```

## 🎨 前端界面更新

### 翻译服务设置
现在翻译服务配置页面显示：

```
模型: [GLM-4-Flash (免费模型，推荐) ▼]
      └── GLM-4
      └── GLM-4-Air

💡 GLM-4-Flash 是智谱 AI 提供的免费模型，适合翻译任务使用
```

### 用户提示
- **免费标识**: 明确标注免费模型
- **推荐指示**: 设为推荐选项
- **使用说明**: 提供简单的使用提示

## 🚀 快速开始使用

### 1. 获取 API Key
```
1. 访问智谱 AI 官网: https://open.bigmodel.cn/
2. 注册/登录账号
3. 在控制台获取 API Key
4. 免费模型无需付费，直接可用
```

### 2. 配置系统
```
1. 打开设置面板 (http://localhost:3000)
2. 切换到 "翻译服务" 标签页
3. 填写 GLM API Key
4. 确认模型选择 "GLM-4-Flash"
5. 点击 "测试连接" 验证
6. 保存设置
```

### 3. 使用流程
```
视频上传 → ASR识别 → GLM-4-Flash翻译 → TTS合成 → 下载结果
```

## 💡 使用建议

### 翻译质量优化
1. **分句处理**: 系统会自动按句子分割文本
2. **上下文保持**: GLM-4-Flash 会保持翻译一致性
3. **多语言支持**: 支持中英日韩等多种语言翻译

### 成本控制
- **完全免费**: 无需担心 API 调用费用
- **无限制**: 适合个人和小团队使用
- **高效率**: 快速响应，节省时间

### 最佳实践
1. **批量翻译**: 一次处理多个视频文件
2. **质量检查**: 翻译完成后可人工校对
3. **备份保存**: 重要翻译结果及时保存

## 📊 性能对比

| 模型 | 费用 | 速度 | 质量 | 推荐场景 |
|------|------|------|------|----------|
| GLM-4-Flash | 🆓 免费 | ⚡ 很快 | ⭐⭐⭐⭐ 优秀 | ✅ 视频翻译 |
| GLM-4 | 💰 付费 | 🚶 中等 | ⭐⭐⭐⭐ 良好 | 通用翻译 |
| GLM-4-Air | 💰 付费 | ⚡ 较快 | ⭐⭐⭐⭐ 良好 | 轻量翻译 |

## 🔍 测试验证

### API 连接测试
```bash
curl -X POST http://localhost:8080/api/v1/settings/test \
  -H "Content-Type: application/json" \
  -d '{"type": "translate"}'
```

### 期望响应
```json
{
  "code": 0,
  "data": {
    "status": "connected",
    "message": "GLM 翻译服务连接成功",
    "latency_ms": 200
  }
}
```

## 🎯 完整配置示例

### 推荐设置
```json
{
  "translate": {
    "glm_api_key": "your-api-key-here",
    "glm_api_url": "https://open.bigmodel.cn/api/paas/v4/chat/completions",
    "glm_model": "glm-4-flash"
  }
}
```

## ❓ 常见问题

### Q: GLM-4-Flash 有使用限制吗？
A: 免费模型有合理的使用限制，但足够个人和小团队使用。

### Q: 翻译质量如何？
A: GLM-4-Flash 专门为翻译任务优化，质量优秀，满足大部分视频翻译需求。

### Q: 如何获取 API Key？
A: 访问 https://open.bigmodel.cn/ 注册账号即可免费获取。

### Q: 是否支持其他免费模型？
A: 目前重点支持 GLM-4-Flash，未来可根据需要扩展其他免费模型。

---

## ✅ 更新完成

系统已完全配置为使用 GLM-4-Flash 免费模型作为默认翻译服务。现在你可以：

1. 刷新前端页面查看新的界面
2. 在翻译服务设置中看到免费模型标识
3. 配置你的智谱 AI API Key
4. 开始免费使用高质量的翻译服务！

🎉 **现在你有了完全免费的 ASR + 翻译 + TTS 视频配音解决方案！**