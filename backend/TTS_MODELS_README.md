# TTS 模型使用指南

## 📋 支持的模型

### 1. CosyVoice-v1（系统音色模式）

**特点:**
- ✅ 无需注册，开箱即用
- ✅ 提供 9 种预置音色
- ✅ 适合快速集成
- ❌ 不支持自定义音色

**使用场景:**
- 标准配音需求
- 快速原型开发
- 多角色配音（使用不同预置音色）

**配置:**
```bash
TTS_MODEL=cosyvoice-v1
TTS_VOICE=longxiaochun  # 可选音色见下表
```

**可用音色:**

| 音色代码 | 说明 | 性别 |
|---------|------|------|
| `longxiaochun` | 龙小春 | 女声 |
| `longyunshu` | 龙韵书 | 女声 |
| `longhua` | 龙华 | 男声 |
| `longsiqian` | 龙思谦 | 男声 |
| `longwan` | 龙婉 | 女声 |
| `longxiaoxia` | 龙小夏 | 女声 |
| `longjing` | 龙靖 | 男声 |
| `longmengchi` | 龙梦驰 | 男声 |
| `longshushu` | 龙姝姝 | 女声 |

---

### 2. Qwen3-TTS-VC-Realtime（声音复刻模式）

**特点:**
- ✅ 支持自定义音色
- ✅ 高度还原原声
- ✅ 实时合成能力
- ❌ 需要先注册音色（10-20秒样本）
- ❌ 相对复杂

**使用场景:**
- 需要还原特定说话人声音
- 个性化配音需求
- 品牌声音定制

**配置:**
```bash
TTS_MODEL=qwen3-tts-vc-realtime-2026-01-15
# 注意：此模型需要使用 voice_id（vc_xxx 格式）
# 不能直接使用系统音色名称
```

---

## 🚀 快速开始

### 模式 1: 系统音色（推荐新手）

```python
from app.integrations.dashscope import TTSClient

# 初始化客户端
client = TTSClient(
    model="cosyvoice-v1",
    voice="longxiaochun",  # 女声
)

# 合成语音
audio = client.synthesize("你好，我是通义千问。")

# 保存音频
with open("output.mp3", "wb") as f:
    f.write(audio)
```

### 模式 2: 声音复刻（手动注册）

```python
from app.integrations.dashscope import TTSClient

# 初始化客户端
client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")

# Step 1: 复刻音色（只需执行一次）
voice_id = client.enroll_voice(
    audio_path="sample_voice.wav",  # 10-20 秒样本
    prefix="my_voice"
)
print(f"Voice ID: {voice_id}")  # 输出: vc_xxx

# Step 2: 使用复刻的音色合成
audio = client.synthesize(
    "这是使用我的声音合成的音频。",
    voice=voice_id  # 使用 voice_id
)

with open("output.mp3", "wb") as f:
    f.write(audio)
```

### 模式 3: 声音复刻（自动注册）

```python
from app.integrations.dashscope import TTSClient

# 初始化客户端
client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")

# 自动复刻 + 合成（一步完成）
audio = client.synthesize(
    "这是使用自动复刻功能合成的音频。",
    auto_clone=True,
    clone_audio_path="sample_voice.wav"
)

with open("output.mp3", "wb") as f:
    f.write(audio)
```

---

## 📊 模型对比

| 特性 | CosyVoice-v1 | Qwen3-TTS-VC |
|-----|-------------|-------------|
| **音色数量** | 9 种预置 | 无限（自定义） |
| **使用难度** | ⭐ 简单 | ⭐⭐⭐ 中等 |
| **音色质量** | ⭐⭐⭐⭐ 优秀 | ⭐⭐⭐⭐⭐ 卓越 |
| **初始化时间** | 即时 | 需复刻（10-30秒） |
| **适用场景** | 通用配音 | 个性化配音 |
| **价格** | 标准 | 较高 |

---

## ⚠️ 常见错误

### 错误 1: 复刻模型使用系统音色名称

```python
client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")
audio = client.synthesize("测试", voice="longxiaochun")  # ❌ 错误
```

**错误信息:**
```
ValueError: Model qwen3-tts-vc-realtime-2026-01-15 requires voice_id (vc_xxx format).
Got: longxiaochun. Please call enroll_voice() first or use auto_clone=True.
```

**解决方案:**
```python
# 方案 1: 先复刻音色
voice_id = client.enroll_voice("sample.wav")
audio = client.synthesize("测试", voice=voice_id)  # ✅ 正确

# 方案 2: 使用自动复刻
audio = client.synthesize(
    "测试",
    auto_clone=True,
    clone_audio_path="sample.wav"
)  # ✅ 正确
```

### 错误 2: 系统音色模型调用复刻功能

```python
client = TTSClient(model="cosyvoice-v1")
voice_id = client.enroll_voice("sample.wav")  # ⚠️ 返回 None
```

**警告信息:**
```
Voice cloning only supported for qwen3-tts-vc-realtime-2026-01-15, current model: cosyvoice-v1
```

---

## 📝 最佳实践

### 1. 选择合适的模型

```python
# 场景 1: 标准配音，快速上线
client = TTSClient(model="cosyvoice-v1")

# 场景 2: 需要还原特定说话人
client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")
```

### 2. 复刻音色建议

- **音频时长:** 10-20 秒（不要太短或太长）
- **音频质量:** 清晰、无噪音、单人说话
- **音频格式:** WAV（推荐）或 MP3
- **音频内容:** 自然语音，包含多种音节

### 3. 缓存 voice_id

```python
# ❌ 不好的做法：每次都复刻
for text in texts:
    voice_id = client.enroll_voice("sample.wav")  # 浪费资源
    audio = client.synthesize(text, voice=voice_id)

# ✅ 好的做法：复刻一次，重复使用
voice_id = client.enroll_voice("sample.wav")  # 只复刻一次
for text in texts:
    audio = client.synthesize(text, voice=voice_id)  # 重复使用
```

### 4. 批量处理

```python
client = TTSClient(model="cosyvoice-v1")

texts = ["第一句", "第二句", "第三句"]
audios = client.synthesize_batch(texts)  # 批量合成
```

---

## 🔗 相关文档

- [阿里云模型服务平台](https://help.aliyun.com/zh/model-studio/)
- [DashScope API 文档](https://help.aliyun.com/zh/model-studio/developer-reference/api-details)
- [API Key 获取指南](https://help.aliyun.com/zh/model-studio/get-api-key)

---

## 📞 技术支持

如有问题，请查看:
- 示例代码: `backend/examples/tts_usage_examples.py`
- API 文档: `http://localhost:8000/api/v1/docs`
- 错误日志: `logs/app.log`
