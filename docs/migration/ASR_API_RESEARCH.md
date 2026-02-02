# 阿里云百炼 ASR API 调研

> **目标**: 调研并选择支持时间戳和说话人识别的阿里云百炼 ASR API

---

## 📋 需求确认

根据项目需求，ASR 服务必须支持：

- ✅ **时间戳**: 精确的开始和结束时间
- ✅ **说话人识别**: 区分不同说话人（多人对话场景）
- ✅ **高质量识别**: 准确的文字转写
- ✅ **异步处理**: 支持长音频文件

---

## 🔍 候选 API 方案

### 方案A: Qwen-Audio (语音理解)

**API 地址**: https://bailian.console.aliyun.com/cn-beijing/#/api/?type=model&url=2978300

**核心功能**:
- ✅ 支持时间戳
- ✅ 支持说话人识别（Speaker Diarization）
- ✅ 支持多轮对话理解
- ✅ 支持音频问答

**API 类型**:
- Multimodal Generation API
- 模型名称: `qwen-audio-turbo` 或 `qwen2-audio-instruct`

**优势**:
- 🎯 原生支持说话人分离
- 🎯 支持时间戳输出
- 🎯 可以一次性完成转写和说话人识别
- 🎯 统一在百炼平台管理

**调用方式**:

```bash
POST https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation
Authorization: Bearer <YOUR_API_KEY>
Content-Type: application/json

{
  "model": "qwen-audio-turbo",
  "input": {
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "audio": "https://your-audio-url.wav"
          },
          {
            "text": "请转写这段音频，并识别不同的说话人，输出时间戳"
          }
        ]
      }
    ]
  },
  "parameters": {
    "result_format": "message",
    "incremental_output": false
  }
}
```

---

### 方案B: Qwen-ASR-Flash (快速转写)

**API 地址**: 原有的 `qwen3-asr-flash-filetrans`

**核心功能**:
- ✅ 支持时间戳（句级和词级）
- ❌ 不支持说话人识别
- ✅ 异步文件转写
- ✅ 快速响应

**限制**:
- ⚠️ **无说话人分离功能** - 这是致命缺陷

**结论**: ❌ 不满足需求，需要说话人识别

---

## 🎯 推荐方案: Qwen-Audio (方案A)

### 为什么选择 Qwen-Audio？

1. **功能完整性**
   - ✅ 原生支持说话人识别
   - ✅ 精确的时间戳输出
   - ✅ 高质量转写
   - ✅ 支持长音频

2. **统一平台**
   - ✅ 与 LLM、TTS 同在百炼平台
   - ✅ 统一的 API Key 管理
   - ✅ 统一的计费和配额

3. **技术先进性**
   - ✅ 基于 Qwen2-Audio 大模型
   - ✅ 多模态理解能力
   - ✅ 持续迭代更新

---

## 📝 需要补充的信息

请提供以下信息以完善集成方案：

### 1. API 响应格式

请提供一个实际的 API 响应示例，特别是：
- 说话人信息的字段名称
- 时间戳的格式（毫秒？秒？）
- 是否支持词级时间戳
- 最大支持几个说话人

**示例问题**:
```json
{
  "output": {
    "choices": [{
      "message": {
        "content": "转写结果格式是什么？"
      }
    }]
  }
}
```

### 2. 调用方式确认

- [ ] 同步调用还是异步轮询？
- [ ] 音频文件如何上传（URL？Base64？）
- [ ] 最大音频时长限制
- [ ] 并发限制

### 3. 定价信息

- [ ] 按时长计费还是按次计费？
- [ ] 费用是多少？
- [ ] 有免费额度吗？

### 4. 特殊配置

- [ ] 是否需要特殊的 prompt 来激活说话人识别？
- [ ] 是否有参数控制输出格式？
- [ ] 是否支持自定义词汇表？

---

## 🔄 与现有实现的对比

| 功能 | 火山引擎 ASR | Qwen-ASR-Flash | **Qwen-Audio** (推荐) |
|-----|-------------|----------------|---------------------|
| **时间戳** | ✅ 句级+词级 | ✅ 句级+词级 | ✅ 待确认 |
| **说话人识别** | ✅ 最多10人 | ❌ 不支持 | ✅ **支持** |
| **情绪检测** | ✅ 支持 | ✅ 支持 | ⚠️ 待确认 |
| **性别检测** | ✅ 支持 | ❌ 不支持 | ⚠️ 待确认 |
| **语言检测** | ⚠️ 需指定 | ✅ 自动检测 | ✅ 自动检测 |
| **调用方式** | 异步轮询 | 异步轮询 | ⚠️ 待确认 |
| **平台** | 火山引擎 | 阿里云百炼 | **阿里云百炼** |

---

## 💻 实现方案草案

### 新建客户端: `qwen_audio_client.go`

```go
package asr

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// QwenAudioClient implements ASR using Qwen-Audio API
type QwenAudioClient struct {
    apiKey  string
    baseURL string
    model   string
    client  *http.Client
    logger  *zap.Logger
}

// QwenAudioConfig holds configuration for Qwen-Audio client
type QwenAudioConfig struct {
    APIKey         string
    BaseURL        string // Default: https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation
    Model          string // qwen-audio-turbo or qwen2-audio-instruct
    RequestTimeout int    // seconds
}

// QwenAudioRequest represents the API request structure
type QwenAudioRequest struct {
    Model  string `json:"model"`
    Input  Input  `json:"input"`
    Parameters Parameters `json:"parameters,omitempty"`
}

type Input struct {
    Messages []Message `json:"messages"`
}

type Message struct {
    Role    string    `json:"role"`
    Content []Content `json:"content"`
}

type Content struct {
    Audio string `json:"audio,omitempty"` // Audio URL
    Text  string `json:"text,omitempty"`  // Instruction text
}

type Parameters struct {
    ResultFormat     string `json:"result_format,omitempty"`
    IncrementalOutput bool   `json:"incremental_output,omitempty"`
}

// QwenAudioResponse represents the API response
// TODO: Update based on actual API response format
type QwenAudioResponse struct {
    RequestID string `json:"request_id"`
    Output    struct {
        Choices []struct {
            Message struct {
                Role    string `json:"role"`
                Content string `json:"content"` // 可能是 JSON 字符串
            } `json:"message"`
        } `json:"choices"`
    } `json:"output"`
}

// NewQwenAudioClient creates a new Qwen-Audio ASR client
func NewQwenAudioClient(cfg QwenAudioConfig, logger *zap.Logger) *QwenAudioClient {
    if cfg.BaseURL == "" {
        cfg.BaseURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
    }
    if cfg.Model == "" {
        cfg.Model = "qwen-audio-turbo"
    }
    if cfg.RequestTimeout == 0 {
        cfg.RequestTimeout = 60
    }

    return &QwenAudioClient{
        apiKey:  cfg.APIKey,
        baseURL: cfg.BaseURL,
        model:   cfg.Model,
        client: &http.Client{
            Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
        },
        logger: logger,
    }
}

// Recognize performs speech recognition with speaker diarization
func (c *QwenAudioClient) Recognize(ctx context.Context, audioURL string, language string) (*ASRResult, error) {
    c.logger.Info("Starting Qwen-Audio recognition",
        zap.String("audio_url", audioURL),
        zap.String("language", language),
    )

    // Construct request
    req := QwenAudioRequest{
        Model: c.model,
        Input: Input{
            Messages: []Message{
                {
                    Role: "user",
                    Content: []Content{
                        {Audio: audioURL},
                        {Text: "请转写这段音频，识别不同的说话人，并为每个句子标注时间戳。输出格式为JSON，包含说话人ID、开始时间、结束时间和文本内容。"},
                    },
                },
            },
        },
        Parameters: Parameters{
            ResultFormat:     "message",
            IncrementalOutput: false,
        },
    }

    // TODO: Implement HTTP request
    // TODO: Parse response and convert to ASRResult
    // TODO: Handle errors and retries

    return nil, fmt.Errorf("not implemented yet - waiting for API details")
}

// parseResponse converts Qwen-Audio response to standard ASRResult
func (c *QwenAudioClient) parseResponse(resp *QwenAudioResponse) (*ASRResult, error) {
    // TODO: Parse the actual response format
    // Expected format (需要根据实际响应调整):
    // {
    //   "segments": [
    //     {
    //       "speaker_id": "speaker_1",
    //       "start_time": 0.0,
    //       "end_time": 2.5,
    //       "text": "你好，欢迎使用阿里云"
    //     }
    //   ]
    // }

    return &ASRResult{
        Language: "zh", // TODO: Detect from response
        Segments: []Segment{
            // TODO: Parse segments from response
        },
    }, nil
}
```

---

## 🎯 下一步行动

### 立即需要做的事情：

1. **获取 API 详细信息**
   ```bash
   # 请提供：
   # 1. 实际的 API 响应示例（JSON 格式）
   # 2. 说话人识别的触发方式（自动？还是需要 prompt？）
   # 3. 时间戳的精度和格式
   # 4. 最大支持的说话人数量
   ```

2. **测试 API 调用**
   ```bash
   # 使用 curl 测试
   curl -X POST https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation \
     -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "model": "qwen-audio-turbo",
       "input": {
         "messages": [{
           "role": "user",
           "content": [
             {"audio": "https://test-audio-url.wav"},
             {"text": "转写音频并识别说话人"}
           ]
         }]
       }
     }'
   ```

3. **更新迁移计划**
   - 将 ASR 迁移方案改为使用 Qwen-Audio
   - 更新配置文件和环境变量
   - 编写完整的客户端实现

---

## 📝 临时问题清单

请帮助回答以下问题：

- [ ] Qwen-Audio API 的完整响应格式是什么？
- [ ] 说话人识别是否需要特殊 prompt 或参数？
- [ ] 时间戳精度如何（秒？毫秒？）？
- [ ] 是否支持词级时间戳？
- [ ] 最多支持几个说话人？
- [ ] API 是同步还是异步？
- [ ] 音频文件上传方式（URL？OSS？Base64？）
- [ ] 计费方式和价格？
- [ ] 并发限制和配额？
- [ ] 是否支持流式输出？

---

**最后更新**: 2026-02-02
**状态**: 🔍 调研中，等待 API 详细信息
**负责人**: 待定
