# 阿里云百炼 Qwen-TTS 集成指南

本文档说明如何在 `video-auto-dubbing` 项目中配置和使用阿里云百炼平台的 **Qwen-TTS** (通义千问语音合成) 服务，特别是 **qwen-tts-flash** 模型。

## 简介

Qwen-TTS 是阿里云推出的高性价比、低延迟语音合成服务。本项目通过兼容 OpenAI API 格式 (`/compatible-mode/v1`) 集成了该服务，旨在替代本地部署的 IndexTTS，以降低运维成本并提高生成速度。

**主要优势：**
*   **无需 GPU**：完全基于 SaaS，无需维护本地显卡服务器。
*   **极速响应**：`qwen-tts-flash` 专为实时交互设计，首包延迟极低。
*   **多语言支持**：支持中、英、日、韩等多种语言。
*   **高性价比**：按字符计费，成本远低于自建 GPU 实例。

## 快速开始

### 1. 获取 API Key
访问 [阿里云百炼控制台](https://bailian.console.aliyun.com/)，创建并获取您的 API Key。

### 2. 配置环境 (.env)
编辑项目根目录下的 `.env` 文件，启用阿里云 TTS 后端：

```bash
# TTS 服务配置
# 设置后端为 'aliyun'
TTS_BACKEND=aliyun

# 阿里云百炼配置
DASHSCOPE_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxx  # 您的 API Key
DASHSCOPE_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
DASHSCOPE_MODEL=qwen-tts-flash  # 推荐使用 flash 模型
```

### 3. 重启服务
```bash
docker compose up -d --build worker
```

## 音色映射 (Voice Mapping)

系统内部的通用角色 ID 会自动映射到 Qwen-TTS 的特定音色。如果未找到映射，系统会将 `speaker_id` 直接透传给阿里云 API，因此您也可以直接使用阿里云支持的音色 ID。

| 系统 Speaker ID | 映射到的 Qwen 音色 | 说明 |
| :--- | :--- | :--- |
| `default` / `longxiaochun` | `longxiaochun` | 龙小淳 (默认亲和女声) |
| `female_young` | `Cherry` | 知性女声 |
| `male_young` | `Alex` | 亲切男声 |
| `female_mature` | `Serena` | 悦耳女声 |
| `male_mature` | `Ethan` | 沉稳男声 |

> **提示**：更多可用音色（如 `Nofish`, `Momo` 等）请参考[阿里云官方文档](https://help.aliyun.com/zh/model-studio/developer-reference/text-to-speech-api-details)。您可以直接在请求中传入这些 ID。

## 功能限制与注意事项

### 1. 音色克隆 (Voice Cloning)
*   **不支持**：`qwen-tts-flash` 和标准版 `qwen-tts` **不支持** Zero-shot 音色克隆（即通过上传一段音频来复刻声音）。
*   **行为**：如果您的请求中包含了参考音频 (`prompt_audio_url`)，系统会记录一条警告日志，但**不会报错**，而是忽略该音频，使用指定的预置音色（如 `longxiaochun`）继续合成。

### 2. 文本长度
*   虽然接口支持长文本，但为了最佳稳定性和字幕对齐效果，建议单次合成文本保持在合理范围内（Worker 会自动按句子切分）。

### 3. 并发限制
*   请根据您在阿里云购买的并发配额（QPS）调整 `worker` 的并发设置。默认情况下 `qwen-tts-flash` 的配额较为宽裕。

## 参考资料

*   [Qwen-TTS 产品介绍](https://help.aliyun.com/zh/model-studio/qwen-tts)
*   [Qwen-TTS API 详情与音色列表](https://help.aliyun.com/zh/model-studio/developer-reference/text-to-speech-api-details)
*   [Qwen 语音识别 (ASR) 文档](https://help.aliyun.com/zh/model-studio/qwen-speech-recognition) *(注：本项目 ASR 目前使用的是火山引擎，未来可考虑迁移)*
