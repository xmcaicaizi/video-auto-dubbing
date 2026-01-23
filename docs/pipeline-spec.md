# Pipeline 配置与 JSON 结构规范

> 更新日期：2026-01-10｜适用版本：main 分支

本文定义可复用的 pipeline 配置模板与中间数据 JSON 结构规范，用于对齐“ASR → 翻译 → TTS → 合成”的全流程实现与数据落地。

## 1. JSON 结构规范

### 1.1 `segments.jsonl`（推荐落地格式）

每行一个 JSON 对象，便于增量处理与恢复。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 否 | 业务层面段 ID（可选） |
| `idx` | int | 是 | 分段序号（从 0 开始） |
| `start_ms` | int | 是 | 片段开始时间（毫秒） |
| `end_ms` | int | 是 | 片段结束时间（毫秒） |
| `duration_ms` | int | 是 | 时长（毫秒） |
| `speaker_id` | string | 否 | 说话人 ID（可选） |
| `src_text` | string | 是 | 源语言识别文本 |
| `mt_text` | string | 否 | 翻译结果 |
| `mt_text_constrained` | string | 否 | 时长约束改写后的译文 |
| `tts_params` | object | 否 | TTS 参数（见下） |
| `tts_audio_key` | string | 否 | TTS 音频对象存储 Key |
| `status` | string | 否 | `pending`/`done`/`failed` |

`tts_params` 推荐字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `speaker_id` | string | 否 | 目标语音色 ID |
| `target_duration_ms` | int | 是 | 目标时长（毫秒） |
| `style` | string | 否 | 语气/风格（如 `neutral`） |
| `prosody` | object | 否 | 语速/音高等控制参数 |

**示例：**
```json
{"idx":0,"start_ms":0,"end_ms":1500,"duration_ms":1500,"speaker_id":"S1","src_text":"你好，世界","mt_text":"Hello, world","mt_text_constrained":"Hello world","tts_params":{"speaker_id":"S1","target_duration_ms":1500,"style":"neutral"},"tts_audio_key":"tts/550e8400/segment_0.wav","status":"done"}
```

### 1.2 `pipeline_config.json`（可复用模板）

用于将参数从任务层、服务层解耦，支持多任务复用。

**JSON 模板：**
```json
{
  "pipeline_id": "default-v1",
  "steps": ["extract_audio", "asr", "translate", "tts", "mux_video"],
  "translate": {
    "batch_size": 20,
    "max_text_length": 4000,
    "glm": {
      "model": "glm-4-flash",
      "temperature": 0.2,
      "rps": 0
    }
  },
  "tts": {
    "batch_size": 20,
    "strict_duration": true,
    "voice_profiles": {
      "S1": {"ref_audio_keys": ["voices/S1/ref.wav"]}
    }
  },
  "mux": {
    "audio_codec": "aac",
    "audio_bitrate": "192k"
  }
}
```

## 2. GLM 翻译规范（可直接用于 prompt 约束）

为保证翻译质量和 TTS 可用性，建议在翻译阶段遵循以下约束：

1. **只输出 JSON 数组**：按输入数组顺序输出 string[]，不输出解释或额外字符。
2. **保持数量与顺序一致**：不得新增或删除条目。
3. **保留关键实体**：专有名词、品牌、地名、人名尽量保持原文或规范译名。
4. **数字与单位准确**：数字、日期、货币、度量单位不改写其数值含义。
5. **标点与格式尽量保留**：避免打乱时间轴中的停顿节奏。
6. **不添加语气词或无关补充**：避免拉长或缩短内容造成时间轴偏移。

如需“等时长约束改写”，建议在翻译后单独执行 `mt_text_constrained` 生成，以免污染直译结果。
