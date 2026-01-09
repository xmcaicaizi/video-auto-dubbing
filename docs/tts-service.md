# TTS 服务规范

## 服务概述

TTS 服务基于本地部署的 IndexTTS2 提供语音合成能力，模型权重从 HuggingFace 拉取，提供 HTTP REST API 接口，支持时间轴约束的可控语音合成。

## 技术栈

- **框架**: FastAPI
- **运行时**: Python 3.11+
- **依赖管理**: uv
- **模型**: IndexTTS-2（本地推理，权重来自 HuggingFace）

## 工程结构

```
tts_service/
├── pyproject.toml          # uv 项目配置
├── uv.lock                 # 依赖锁定文件
├── README.md
├── app/
│   ├── __init__.py
│   ├── main.py             # FastAPI 应用入口
│   ├── models.py           # 数据模型
│   ├── synthesizer.py      # IndexTTS2 本地推理封装
│   ├── config.py           # 配置管理
│   └── exceptions.py       # 异常定义
└── Dockerfile
```

## API 接口规范

### Base URL

- **开发环境**: `http://localhost:8000`
- **生产环境**: `http://tts_service:8000`

### 1. 健康检查

**接口**: `GET /health`

**响应**:
```json
{
  "status": "healthy",
  "model_loaded": true
}
```

### 2. 语音合成

**接口**: `POST /synthesize`

**请求头**:
- `Content-Type: application/json`

**请求体**:
```json
{
  "text": "Hello, world",
  "speaker_id": "default",
  "target_duration_ms": 1500,
  "language": "en",
  "prosody_control": {
    "speed": 1.0,
    "pitch": 1.0,
    "energy": 1.0
  },
  "time_constraints": {
    "segments": [
      {
        "text": "Hello",
        "target_duration_ms": 500,
        "start_time_ms": 0
      },
      {
        "text": "world",
        "target_duration_ms": 1000,
        "start_time_ms": 500
      }
    ]
  },
  "output_format": "wav",
  "sample_rate": 22050
}
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `text` | string | 是 | 要合成的文本 |
| `speaker_id` | string | 否 | 说话人 ID（默认: "default"） |
| `target_duration_ms` | integer | 是 | 目标时长（毫秒），用于时间轴约束 |
| `language` | string | 否 | 语言代码（默认: "en"） |
| `prosody_control` | object | 否 | 韵律控制参数 |
| `prosody_control.speed` | float | 否 | 语速（0.5-2.0，默认: 1.0） |
| `prosody_control.pitch` | float | 否 | 音调（0.5-2.0，默认: 1.0） |
| `prosody_control.energy` | float | 否 | 能量（0.5-2.0，默认: 1.0） |
| `time_constraints` | object | 否 | 时间约束（分段对齐） |
| `time_constraints.segments` | array | 否 | 分段信息，用于精细控制 |
| `output_format` | string | 否 | 输出格式（wav/mp3，默认: wav） |
| `sample_rate` | integer | 否 | 采样率（默认: 22050） |

**响应**:

**成功响应** (200):
- Content-Type: `audio/wav` 或 `application/json`
- 如果 `Accept: audio/*`，直接返回音频二进制流
- 如果 `Accept: application/json`，返回 JSON：

```json
{
  "audio_url": "http://tts_service:8000/audio/temp_123456.wav",
  "duration_ms": 1500,
  "sample_rate": 22050,
  "format": "wav",
  "file_size": 132300
}
```

**错误响应** (400/500):
```json
{
  "error": "invalid_parameter",
  "message": "target_duration_ms must be positive",
  "details": {}
}
```

### 3. 批量合成

**接口**: `POST /synthesize/batch`

**请求体**:
```json
{
  "requests": [
    {
      "text": "Hello",
      "target_duration_ms": 500,
      "speaker_id": "default"
    },
    {
      "text": "world",
      "target_duration_ms": 1000,
      "speaker_id": "default"
    }
  ],
  "merge": true
}
```

**响应**:
```json
{
  "audio_url": "http://tts_service:8000/audio/batch_123456.wav",
  "duration_ms": 1500,
  "segments": [
    {
      "idx": 0,
      "duration_ms": 500,
      "audio_url": "http://tts_service:8000/audio/seg_0.wav"
    },
    {
      "idx": 1,
      "duration_ms": 1000,
      "audio_url": "http://tts_service:8000/audio/seg_1.wav"
    }
  ]
}
```

### 4. 获取说话人列表

**接口**: `GET /speakers`

**响应**:
```json
{
  "speakers": [
    {
      "id": "default",
      "name": "默认说话人",
      "language": "en",
      "gender": "neutral"
    },
    {
      "id": "female_1",
      "name": "女性声音1",
      "language": "en",
      "gender": "female"
    }
  ]
}
```

## 时间轴约束实现

### 核心原理

IndexTTS2 支持通过以下方式控制时长：

1. **目标时长参数**: `target_duration_ms`
   - 模型会调整语速和停顿来匹配目标时长
   - 适用于整段文本的时长控制

2. **分段对齐**: `time_constraints.segments`
   - 将文本分段，每段指定目标时长
   - 模型分别合成每段，然后拼接
   - 适用于精确的时间轴对齐

3. **韵律控制**: `prosody_control`
   - 通过调整 speed/pitch/energy 间接影响时长
   - 作为辅助手段

### 实现策略

```python
def synthesize_with_time_constraint(
    text: str,
    target_duration_ms: int,
    segments: Optional[List[Segment]] = None
) -> Audio:
    if segments:
        # 分段合成模式
        audio_segments = []
        for seg in segments:
            audio = synthesize_segment(
                text=seg.text,
                target_duration_ms=seg.target_duration_ms,
                prosody_control=calculate_prosody(seg)
            )
            audio_segments.append(audio)
        return merge_audio(audio_segments)
    else:
        # 整体合成模式
        return synthesize_whole(
            text=text,
            target_duration_ms=target_duration_ms
        )
```

### 时长对齐算法

```python
def calculate_prosody_for_duration(
    text: str,
    target_duration_ms: int,
    base_duration_ms: int
) -> ProsodyControl:
    """
    根据目标时长和基准时长计算韵律参数
    """
    ratio = target_duration_ms / base_duration_ms
    
    if ratio < 0.8:
        # 需要加快
        speed = 1.0 / ratio
        speed = min(speed, 2.0)  # 限制最大语速
    elif ratio > 1.2:
        # 需要放慢
        speed = 1.0 / ratio
        speed = max(speed, 0.5)  # 限制最慢语速
    else:
        speed = 1.0
    
    return ProsodyControl(speed=speed, pitch=1.0, energy=1.0)
```

## uv 工程配置

### pyproject.toml

```toml
[project]
name = "tts-service"
version = "0.1.0"
description = "IndexTTS2 based TTS service"
requires-python = ">=3.11"
dependencies = [
    "fastapi>=0.104.0",
    "uvicorn[standard]>=0.24.0",
    "pydantic>=2.5.0",
    "pydantic-settings>=2.1.0",
    "numpy>=1.24.0",
    "torch>=2.1.0",
    "torchaudio>=2.1.0",
    "librosa>=0.10.0",
    "soundfile>=0.12.0",
    "scipy>=1.11.0",
    "python-multipart>=0.0.6",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.4.0",
    "pytest-asyncio>=0.21.0",
    "black>=23.11.0",
    "ruff>=0.1.6",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.uv]
dev-dependencies = [
    "pytest>=7.4.0",
    "pytest-asyncio>=0.21.0",
    "black>=23.11.0",
    "ruff>=0.1.6",
]

[tool.black]
line-length = 100
target-version = ['py311']

[tool.ruff]
line-length = 100
target-version = "py311"
```

### 使用 uv 管理依赖

**初始化项目**:
```bash
cd tts_service
uv venv
source .venv/bin/activate  # Windows: .venv\Scripts\activate
uv pip install -e .
```

**安装依赖**:
```bash
uv sync
```

**添加新依赖**:
```bash
uv add package-name
```

**更新依赖**:
```bash
uv lock --upgrade
uv sync
```

**生产环境安装**:
```bash
uv sync --frozen  # 使用 uv.lock 锁定版本
```

## 配置管理

### 环境变量配置

**必需配置**:
- `INDEXTTS_MODEL_DIR`: IndexTTS2 模型目录（需提前下载权重）
- `INDEXTTS_CFG_PATH`: IndexTTS2 配置文件路径（默认 `config.yaml`）

**可选配置**:
- `TTS_BACKEND`: 后端模式（默认: `index_tts2`）
- `INDEXTTS_PROMPT_AUDIO`: 默认提示音频路径（用于无 prompt_audio_url 时兜底）
- `INDEXTTS_DEVICE`: 推理设备（`auto`/`cpu`/`cuda`）
- `INDEXTTS_USE_FP16`: 是否启用 FP16
- `INDEXTTS_USE_TORCH_COMPILE`: 是否启用 torch.compile
- `INDEXTTS_USE_CUDA_KERNEL`: 是否启用自定义 CUDA kernel
- `HF_ENDPOINT`: HuggingFace 访问地址（可选镜像）
- `HF_HUB_CACHE`: HuggingFace 缓存目录
- `STRICT_DURATION`: 是否严格对齐目标时长（默认: `false`）
  - `true`: 使用音频时间拉伸强制对齐，可能影响音质
  - `false`: 返回自然时长，质量优先
- `MAX_CONCURRENT_REQUESTS`: 最大并发请求数（默认: `10`）
- `MAX_RETRIES`: 最大重试次数（默认: `3`）
- `RETRY_DELAY_SECONDS`: 重试延迟（秒，默认: `1.0`）

**服务配置**:
- `TTS_HOST`: 服务监听地址（默认: `0.0.0.0`）
- `TTS_PORT`: 服务端口（默认: `8000`）
- `TTS_WORKERS`: Worker 数量（默认: `1`）
- `DEFAULT_SAMPLE_RATE`: 默认采样率（默认: `22050`）
- `DEFAULT_FORMAT`: 默认音频格式（默认: `wav`）
- `AUDIO_TEMP_DIR`: 临时音频文件目录（默认: `./temp/audio`）
- `AUDIO_TEMP_RETENTION_HOURS`: 临时文件保留时间（小时，默认: `24`）

### 配置示例

```env
# IndexTTS2 模型配置（必需）
INDEXTTS_MODEL_DIR=/app/models/IndexTTS-2
INDEXTTS_CFG_PATH=/app/models/IndexTTS-2/config.yaml

# TTS 服务配置
TTS_HOST=0.0.0.0
TTS_PORT=8000
TTS_BACKEND=index_tts2
INDEXTTS_PROMPT_AUDIO=/app/assets/voice_01.wav
INDEXTTS_DEVICE=auto
INDEXTTS_USE_FP16=true
INDEXTTS_USE_TORCH_COMPILE=false
INDEXTTS_USE_CUDA_KERNEL=false
HF_ENDPOINT=https://hf-mirror.com
HF_HUB_CACHE=/app/models/IndexTTS-2/hf_cache
STRICT_DURATION=false

# 并发和重试配置
MAX_CONCURRENT_REQUESTS=10
MAX_RETRIES=3
RETRY_DELAY_SECONDS=1.0

# 音频设置
DEFAULT_SAMPLE_RATE=22050
DEFAULT_FORMAT=wav

# 存储配置
AUDIO_TEMP_DIR=./temp/audio
AUDIO_TEMP_RETENTION_HOURS=24
```

## 错误处理

### 错误码定义

| 错误码 | HTTP 状态码 | 说明 |
|--------|------------|------|
| `invalid_parameter` | 400 | 参数错误 |
| `text_too_long` | 400 | 文本过长 |
| `duration_mismatch` | 400 | 时长不匹配 |
| `model_not_loaded` | 503 | 模型未加载 |
| `synthesis_failed` | 500 | 合成失败 |
| `internal_error` | 500 | 内部错误 |

### 错误响应格式

```json
{
  "error": "error_code",
  "message": "Human readable message",
  "details": {
    "field": "additional info"
  }
}
```

## 性能优化建议

1. **并发控制**: 通过 `MAX_CONCURRENT_REQUESTS` 限制并发请求数，避免 GPU/CPU 过载
2. **批处理**: 支持批量合成减少调用次数
3. **缓存**: 相同文本和参数的合成结果可以缓存（待实现）
4. **重试机制**: 自动重试失败的合成请求，支持指数退避
5. **时长策略**: 根据需求选择 `STRICT_DURATION` 模式
   - 质量优先：`STRICT_DURATION=false`（推荐）
   - 时长精确：`STRICT_DURATION=true`（可能影响音质）

## 部署注意事项

1. **模型权重**: 确保 IndexTTS2 权重已下载到 `INDEXTTS_MODEL_DIR`
2. **HuggingFace 访问**: 如需镜像或代理，请配置 `HF_ENDPOINT`；私有模型需配置 HF Token
3. **存储与缓存**: 为模型目录与 `HF_HUB_CACHE` 预留足够磁盘空间
4. **资源限制**: 设置合适的内存、CPU 与 GPU 限制
5. **健康检查**: 实现 `/health` 接口用于容器健康检查
6. **日志脱敏**: 确保日志中不包含敏感信息

