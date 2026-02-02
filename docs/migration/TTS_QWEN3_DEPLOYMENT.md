# Qwen3-TTS 自部署方案

> 🎯 **目标**: 使用 Qwen3-TTS 自建 TTS 服务，替代 IndexTTS
> 📅 **创建时间**: 2026-02-02
> 🔗 **官方仓库**: https://github.com/QwenLM/Qwen3-TTS

---

## 📊 Qwen3-TTS vs IndexTTS 对比

| 特性 | IndexTTS (旧方案) | Qwen3-TTS (新方案) | 优势 |
|-----|------------------|-------------------|------|
| **模型架构** | - | Dual-Track 流式架构 | ✅ 极低延迟 |
| **首字节延迟** | ~500ms | **97ms** | ✅ 5x 提升 |
| **多语言支持** | 中英日 | **10种语言** | ✅ 更全面 |
| **音色克隆** | ❌ 不支持 | ✅ **3秒快速克隆** | ✅ 新功能 |
| **自然语言控制** | ❌ 不支持 | ✅ 情绪、语速控制 | ✅ 新功能 |
| **流式输出** | ❌ | ✅ 实时流式 | ✅ 更好体验 |
| **说话人情绪** | 基础 | **自适应情绪** | ✅ 更自然 |
| **开源支持** | 一般 | **官方维护** | ✅ 持续更新 |

**结论**: Qwen3-TTS 全面领先 IndexTTS！

---

## 🎯 方案选择

### 推荐模型: Qwen3-TTS-12Hz-1.7B-CustomVoice

**为什么选择这个模型？**

1. **预设音色丰富** - 9种高质量预设音色
2. **自然语言控制** - 支持情绪、语速等指令控制
3. **性能平衡** - 1.7B 参数，GPU 需求适中
4. **多语言支持** - 覆盖项目所需的所有语言

**支持的9种预设音色**:

| 音色名称 | 特点 | 语言 | 适用场景 |
|---------|------|------|---------|
| **Vivian** | 明亮、略带锋芒的年轻女性 | 中文 | 活力、专业解说 |
| **Serena** | 温暖、温柔的年轻女性 | 中文 | 温馨、情感内容 |
| **Uncle_Fu** | 资深男性、低沉浑厚 | 中文 | 严肃、纪录片 |
| **Dylan** | 年轻北京男性 | 中文(北京话) | 日常对话 |
| **Eric** | 活力成都男性 | 中文(四川话) | 幽默、轻松 |
| **Ryan** | 动感男性 | 英文 | 动态内容 |
| **Aiden** | 阳光美国男性 | 英文 | 年轻、活泼 |
| **Ono_Anna** | 活泼日本女性 | 日文 | 动漫、可爱 |
| **Sohee** | 温暖韩国女性 | 韩文 | 温柔、亲切 |

---

## 🏗️ 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────┐
│                   API Gateway (Nginx)                │
└──────────────────────┬──────────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │                             │
┌───────▼────────┐           ┌────────▼────────┐
│ Worker Service │           │  Web Frontend   │
│  (Go Client)   │           │                 │
└───────┬────────┘           └─────────────────┘
        │
        │ HTTP POST /api/tts/synthesize
        │
┌───────▼──────────────────────────────────────────────┐
│         TTS Service (FastAPI)                         │
│  - 端口: 8000                                         │
│  - 路由: /synthesize, /health, /voices               │
│  - 并发控制: AsyncIO + Semaphore                     │
└──────────────────────┬────────────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │                             │
┌───────▼────────┐           ┌────────▼────────┐
│ Qwen3-TTS      │           │ GPU Memory      │
│ 1.7B Model     │           │ Cache           │
│ (VRAM: ~6GB)   │           │                 │
└────────────────┘           └─────────────────┘
```

### FastAPI 服务设计

**目录结构**:
```
tts_service/
├── app/
│   ├── __init__.py
│   ├── main.py              # FastAPI 应用入口
│   ├── config.py            # 配置管理
│   ├── models/
│   │   ├── __init__.py
│   │   ├── request.py       # 请求模型
│   │   └── response.py      # 响应模型
│   ├── services/
│   │   ├── __init__.py
│   │   └── tts_service.py   # TTS 核心逻辑
│   ├── routers/
│   │   ├── __init__.py
│   │   ├── synthesize.py    # 合成路由
│   │   └── health.py        # 健康检查
│   └── utils/
│       ├── __init__.py
│       ├── audio.py         # 音频处理
│       └── cache.py         # 缓存管理
├── requirements.txt
├── Dockerfile
└── README.md
```

---

## 💻 代码实现

### 1. FastAPI 服务端

#### `app/main.py`

```python
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import logging

from app.config import settings
from app.routers import synthesize, health
from app.services.tts_service import TTSService

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# 全局 TTS 服务实例
tts_service: TTSService | None = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    global tts_service

    # 启动时加载模型
    logger.info("Loading Qwen3-TTS model...")
    tts_service = TTSService(
        model_id=settings.MODEL_ID,
        device=settings.DEVICE,
        max_concurrent=settings.MAX_CONCURRENT
    )
    await tts_service.initialize()
    logger.info(f"Model loaded successfully on {settings.DEVICE}")

    yield  # 应用运行

    # 关闭时清理资源
    logger.info("Shutting down TTS service...")
    await tts_service.cleanup()
    logger.info("Shutdown complete")

# 创建 FastAPI 应用
app = FastAPI(
    title="Qwen3-TTS API",
    description="High-performance TTS service powered by Qwen3-TTS",
    version="1.0.0",
    lifespan=lifespan
)

# CORS 中间件
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 注册路由
app.include_router(synthesize.router, prefix="/api", tags=["TTS"])
app.include_router(health.router, prefix="/api", tags=["Health"])

@app.get("/")
async def root():
    """根路径"""
    return {
        "service": "Qwen3-TTS API",
        "version": "1.0.0",
        "status": "running",
        "model": settings.MODEL_ID
    }

# 依赖注入: 获取 TTS 服务实例
def get_tts_service() -> TTSService:
    if tts_service is None:
        raise HTTPException(status_code=503, detail="TTS service not initialized")
    return tts_service
```

#### `app/config.py`

```python
from pydantic_settings import BaseSettings
from functools import lru_cache

class Settings(BaseSettings):
    """应用配置"""

    # 模型配置
    MODEL_ID: str = "Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice"
    DEVICE: str = "cuda:0"  # 或 "cpu"
    TORCH_DTYPE: str = "bfloat16"  # 或 "float16"

    # 服务配置
    HOST: str = "0.0.0.0"
    PORT: int = 8000
    MAX_CONCURRENT: int = 4  # 最大并发合成数

    # 性能配置
    ENABLE_FLASH_ATTENTION: bool = True
    ENABLE_CACHE: bool = True
    CACHE_SIZE: int = 100  # 缓存最近100个合成结果

    # 音频配置
    SAMPLE_RATE: int = 24000
    OUTPUT_FORMAT: str = "wav"  # wav 或 mp3

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"

@lru_cache()
def get_settings() -> Settings:
    return Settings()

settings = get_settings()
```

#### `app/models/request.py`

```python
from pydantic import BaseModel, Field
from typing import Optional

class SynthesizeRequest(BaseModel):
    """TTS 合成请求"""

    text: str = Field(..., description="要合成的文本", min_length=1, max_length=5000)
    language: str = Field("zh", description="语言代码: zh, en, ja, ko 等")
    speaker: str = Field("Vivian", description="音色名称")
    instruct: Optional[str] = Field(None, description="可选的自然语言指令，如：更快一点、更温柔")

    # 高级选项
    speed: Optional[float] = Field(1.0, description="语速倍率 (0.5-2.0)", ge=0.5, le=2.0)
    pitch: Optional[float] = Field(1.0, description="音高倍率 (0.5-2.0)", ge=0.5, le=2.0)
    enable_streaming: bool = Field(False, description="是否启用流式输出")

    class Config:
        json_schema_extra = {
            "example": {
                "text": "你好，欢迎使用 Qwen3-TTS 语音合成服务。",
                "language": "zh",
                "speaker": "Vivian",
                "instruct": "用活泼的语气说",
                "speed": 1.0,
                "pitch": 1.0,
                "enable_streaming": False
            }
        }

class VoiceCloneRequest(BaseModel):
    """音色克隆请求（可选功能）"""

    text: str = Field(..., description="要合成的文本")
    language: str = Field("zh", description="语言代码")
    ref_audio: str = Field(..., description="参考音频 URL 或 Base64")
    ref_text: str = Field(..., description="参考音频对应的文本")
```

#### `app/models/response.py`

```python
from pydantic import BaseModel, Field
from typing import Optional

class SynthesizeResponse(BaseModel):
    """TTS 合成响应"""

    success: bool = Field(..., description="是否成功")
    audio_url: Optional[str] = Field(None, description="合成音频的 URL")
    audio_base64: Optional[str] = Field(None, description="Base64 编码的音频数据")
    duration: float = Field(..., description="音频时长（秒）")
    sample_rate: int = Field(24000, description="采样率")
    processing_time: float = Field(..., description="处理耗时（秒）")
    message: Optional[str] = Field(None, description="错误或提示信息")

    class Config:
        json_schema_extra = {
            "example": {
                "success": True,
                "audio_url": "https://storage.example.com/audio/xxx.wav",
                "audio_base64": None,
                "duration": 2.5,
                "sample_rate": 24000,
                "processing_time": 0.15,
                "message": None
            }
        }

class HealthResponse(BaseModel):
    """健康检查响应"""

    status: str = Field(..., description="服务状态: healthy, unhealthy")
    model_loaded: bool = Field(..., description="模型是否已加载")
    device: str = Field(..., description="运行设备")
    concurrent_tasks: int = Field(..., description="当前并发任务数")
    uptime_seconds: float = Field(..., description="服务运行时长（秒）")

class VoicesResponse(BaseModel):
    """可用音色列表"""

    voices: list[dict] = Field(..., description="音色列表")

    class Config:
        json_schema_extra = {
            "example": {
                "voices": [
                    {"id": "Vivian", "name": "Vivian", "language": "zh", "gender": "female", "description": "明亮年轻女性"},
                    {"id": "Serena", "name": "Serena", "language": "zh", "gender": "female", "description": "温柔女性"},
                ]
            }
        }
```

#### `app/services/tts_service.py`

```python
import torch
import asyncio
import time
import base64
from typing import Optional
from qwen_tts import Qwen3TTSModel
from app.config import settings
import logging

logger = logging.getLogger(__name__)

class TTSService:
    """TTS 核心服务"""

    def __init__(self, model_id: str, device: str, max_concurrent: int = 4):
        self.model_id = model_id
        self.device = device
        self.max_concurrent = max_concurrent
        self.model: Optional[Qwen3TTSModel] = None
        self.semaphore = asyncio.Semaphore(max_concurrent)
        self.start_time = time.time()
        self.concurrent_tasks = 0

    async def initialize(self):
        """异步初始化模型"""
        loop = asyncio.get_event_loop()

        # 在线程池中加载模型（避免阻塞事件循环）
        self.model = await loop.run_in_executor(
            None,
            self._load_model
        )
        logger.info("Model initialization complete")

    def _load_model(self) -> Qwen3TTSModel:
        """同步加载模型"""
        dtype = torch.bfloat16 if settings.TORCH_DTYPE == "bfloat16" else torch.float16

        model = Qwen3TTSModel.from_pretrained(
            self.model_id,
            device=self.device,
            torch_dtype=dtype,
            attn_implementation="flash_attention_2" if settings.ENABLE_FLASH_ATTENTION else None
        )

        return model

    async def synthesize(
        self,
        text: str,
        language: str,
        speaker: str,
        instruct: Optional[str] = None,
        speed: float = 1.0,
        pitch: float = 1.0
    ) -> tuple[bytes, float]:
        """
        合成语音

        Returns:
            (audio_bytes, duration)
        """
        async with self.semaphore:  # 并发控制
            self.concurrent_tasks += 1
            try:
                start_time = time.time()

                # 在线程池中执行合成（CPU/GPU 密集操作）
                loop = asyncio.get_event_loop()
                result = await loop.run_in_executor(
                    None,
                    self._synthesize_sync,
                    text,
                    language,
                    speaker,
                    instruct
                )

                audio_array, sample_rate = result
                duration = len(audio_array) / sample_rate

                # 转换为 WAV 字节流
                audio_bytes = self._array_to_wav(audio_array, sample_rate)

                processing_time = time.time() - start_time
                logger.info(f"Synthesis completed in {processing_time:.2f}s, audio duration: {duration:.2f}s")

                return audio_bytes, duration

            finally:
                self.concurrent_tasks -= 1

    def _synthesize_sync(
        self,
        text: str,
        language: str,
        speaker: str,
        instruct: Optional[str]
    ) -> tuple:
        """同步合成（在线程池中执行）"""
        if self.model is None:
            raise RuntimeError("Model not initialized")

        # 调用 Qwen3-TTS
        result = self.model.generate_custom_voice(
            text=text,
            language=language,
            speaker=speaker,
            instruct=instruct
        )

        # result 是 (audio_array, sample_rate) 的元组
        return result

    def _array_to_wav(self, audio_array, sample_rate: int) -> bytes:
        """将 NumPy 数组转换为 WAV 字节流"""
        import io
        import scipy.io.wavfile as wavfile

        buffer = io.BytesIO()
        wavfile.write(buffer, sample_rate, audio_array)
        buffer.seek(0)
        return buffer.read()

    async def cleanup(self):
        """清理资源"""
        if self.model is not None:
            del self.model
            self.model = None
            if torch.cuda.is_available():
                torch.cuda.empty_cache()
            logger.info("Model cleaned up")

    def get_uptime(self) -> float:
        """获取服务运行时长"""
        return time.time() - self.start_time
```

#### `app/routers/synthesize.py`

```python
from fastapi import APIRouter, Depends, HTTPException
from app.models.request import SynthesizeRequest
from app.models.response import SynthesizeResponse
from app.services.tts_service import TTSService
from app.main import get_tts_service
import base64
import time
import logging

logger = logging.getLogger(__name__)
router = APIRouter()

@router.post("/tts/synthesize", response_model=SynthesizeResponse)
async def synthesize_speech(
    request: SynthesizeRequest,
    tts_service: TTSService = Depends(get_tts_service)
):
    """
    合成语音

    - **text**: 要合成的文本（1-5000字符）
    - **language**: 语言代码（zh, en, ja, ko 等）
    - **speaker**: 音色名称（Vivian, Serena, Uncle_Fu 等）
    - **instruct**: 可选的自然语言指令（如：更快一点、更温柔）
    """
    try:
        start_time = time.time()

        # 调用 TTS 服务
        audio_bytes, duration = await tts_service.synthesize(
            text=request.text,
            language=request.language,
            speaker=request.speaker,
            instruct=request.instruct,
            speed=request.speed or 1.0,
            pitch=request.pitch or 1.0
        )

        processing_time = time.time() - start_time

        # 返回 Base64 编码的音频
        audio_base64 = base64.b64encode(audio_bytes).decode('utf-8')

        return SynthesizeResponse(
            success=True,
            audio_base64=audio_base64,
            audio_url=None,  # 如果需要，可以上传到 OSS 并返回 URL
            duration=duration,
            sample_rate=24000,
            processing_time=processing_time
        )

    except Exception as e:
        logger.error(f"Synthesis failed: {str(e)}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"Synthesis failed: {str(e)}")
```

#### `app/routers/health.py`

```python
from fastapi import APIRouter, Depends
from app.models.response import HealthResponse, VoicesResponse
from app.services.tts_service import TTSService
from app.main import get_tts_service

router = APIRouter()

@router.get("/health", response_model=HealthResponse)
async def health_check(tts_service: TTSService = Depends(get_tts_service)):
    """健康检查"""
    return HealthResponse(
        status="healthy" if tts_service.model is not None else "unhealthy",
        model_loaded=tts_service.model is not None,
        device=tts_service.device,
        concurrent_tasks=tts_service.concurrent_tasks,
        uptime_seconds=tts_service.get_uptime()
    )

@router.get("/voices", response_model=VoicesResponse)
async def list_voices():
    """获取可用音色列表"""
    voices = [
        {"id": "Vivian", "name": "Vivian", "language": "zh", "gender": "female", "description": "明亮年轻女性"},
        {"id": "Serena", "name": "Serena", "language": "zh", "gender": "female", "description": "温柔女性"},
        {"id": "Uncle_Fu", "name": "Uncle_Fu", "language": "zh", "gender": "male", "description": "资深低沉男性"},
        {"id": "Dylan", "name": "Dylan", "language": "zh", "gender": "male", "description": "年轻北京男性"},
        {"id": "Eric", "name": "Eric", "language": "zh", "gender": "male", "description": "活力成都男性"},
        {"id": "Ryan", "name": "Ryan", "language": "en", "gender": "male", "description": "动感英文男性"},
        {"id": "Aiden", "name": "Aiden", "language": "en", "gender": "male", "description": "阳光美国男性"},
        {"id": "Ono_Anna", "name": "Ono_Anna", "language": "ja", "gender": "female", "description": "活泼日本女性"},
        {"id": "Sohee", "name": "Sohee", "language": "ko", "gender": "female", "description": "温暖韩国女性"},
    ]

    return VoicesResponse(voices=voices)
```

---

### 2. Go 客户端 (Worker 调用)

#### `worker/internal/tts/qwen3_client.go`

```go
package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Qwen3Client implements TTS using self-hosted Qwen3-TTS FastAPI service
type Qwen3Client struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

// Qwen3Config holds configuration for Qwen3 client
type Qwen3Config struct {
	BaseURL string // FastAPI service URL, e.g., http://localhost:8000
	Timeout int    // Request timeout in seconds
}

// Qwen3Request represents the FastAPI synthesis request
type Qwen3Request struct {
	Text            string   `json:"text"`
	Language        string   `json:"language"`
	Speaker         string   `json:"speaker"`
	Instruct        *string  `json:"instruct,omitempty"`
	Speed           *float64 `json:"speed,omitempty"`
	Pitch           *float64 `json:"pitch,omitempty"`
	EnableStreaming bool     `json:"enable_streaming"`
}

// Qwen3Response represents the FastAPI synthesis response
type Qwen3Response struct {
	Success        bool    `json:"success"`
	AudioURL       *string `json:"audio_url"`
	AudioBase64    *string `json:"audio_base64"`
	Duration       float64 `json:"duration"`
	SampleRate     int     `json:"sample_rate"`
	ProcessingTime float64 `json:"processing_time"`
	Message        *string `json:"message"`
}

// NewQwen3Client creates a new Qwen3-TTS client
func NewQwen3Client(cfg Qwen3Config, logger *zap.Logger) *Qwen3Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60
	}

	return &Qwen3Client{
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		logger: logger,
	}
}

// Synthesize generates speech from text
func (c *Qwen3Client) Synthesize(ctx context.Context, params TTSParams) ([]byte, error) {
	c.logger.Info("Starting Qwen3-TTS synthesis",
		zap.String("text", params.Text[:min(50, len(params.Text))]),
		zap.String("speaker", params.SpeakerID),
		zap.String("language", params.Language),
	)

	// Map internal speaker IDs to Qwen3 speaker names
	speakerName := c.mapSpeaker(params.SpeakerID, params.Gender)

	// Construct request
	req := Qwen3Request{
		Text:            params.Text,
		Language:        params.Language,
		Speaker:         speakerName,
		EnableStreaming: false,
	}

	// Add optional natural language instruction based on emotion
	if params.Emotion != "" {
		instruct := c.buildInstruct(params.Emotion, params.Speed)
		req.Instruct = &instruct
	}

	// Make HTTP request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/api/tts/synthesize",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ttsResp Qwen3Response
	if err := json.NewDecoder(resp.Body).Decode(&ttsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !ttsResp.Success {
		msg := "unknown error"
		if ttsResp.Message != nil {
			msg = *ttsResp.Message
		}
		return nil, fmt.Errorf("synthesis failed: %s", msg)
	}

	// Decode Base64 audio
	if ttsResp.AudioBase64 == nil {
		return nil, fmt.Errorf("no audio data in response")
	}

	audioBytes, err := base64.StdEncoding.DecodeString(*ttsResp.AudioBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio base64: %w", err)
	}

	c.logger.Info("Qwen3-TTS synthesis completed",
		zap.Float64("duration", ttsResp.Duration),
		zap.Float64("processing_time", ttsResp.ProcessingTime),
		zap.Int("audio_size", len(audioBytes)),
	)

	return audioBytes, nil
}

// mapSpeaker maps internal speaker IDs to Qwen3 speaker names
func (c *Qwen3Client) mapSpeaker(speakerID string, gender string) string {
	// Default mapping
	speakerMap := map[string]string{
		"default":       "Vivian",
		"female_young":  "Vivian",
		"female_mature": "Serena",
		"male_young":    "Dylan",
		"male_mature":   "Uncle_Fu",
	}

	// Gender-based fallback
	if mapped, ok := speakerMap[speakerID]; ok {
		return mapped
	}

	// Fallback based on gender
	if gender == "male" {
		return "Dylan"
	}
	return "Vivian"
}

// buildInstruct constructs natural language instruction from emotion and speed
func (c *Qwen3Client) buildInstruct(emotion string, speed float64) string {
	instruct := ""

	// Emotion mapping
	emotionMap := map[string]string{
		"happy":   "用开心愉快的语气说",
		"sad":     "用悲伤低沉的语气说",
		"angry":   "用生气愤怒的语气说",
		"neutral": "用平静自然的语气说",
	}

	if text, ok := emotionMap[emotion]; ok {
		instruct = text
	}

	// Speed adjustment
	if speed > 1.2 {
		if instruct != "" {
			instruct += "，并且说得快一点"
		} else {
			instruct = "说得快一点"
		}
	} else if speed < 0.8 {
		if instruct != "" {
			instruct += "，并且说得慢一点"
		} else {
			instruct = "说得慢一点"
		}
	}

	return instruct
}

// HealthCheck checks if the TTS service is healthy
func (c *Qwen3Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service unhealthy, status: %d", resp.StatusCode)
	}

	return nil
}
```

---

## 📦 部署方案

### Docker 部署

#### `tts_service/Dockerfile`

```dockerfile
FROM nvidia/cuda:12.1.0-cudnn8-runtime-ubuntu22.04

# 安装 Python 3.11
RUN apt-get update && apt-get install -y \
    python3.11 \
    python3.11-pip \
    python3.11-dev \
    git \
    && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 复制依赖文件
COPY requirements.txt .

# 安装 Python 依赖
RUN pip3 install --no-cache-dir -r requirements.txt

# 安装 FlashAttention 2 (可选，加速推理)
RUN pip3 install flash-attn --no-build-isolation

# 复制应用代码
COPY app/ ./app/

# 暴露端口
EXPOSE 8000

# 启动命令
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]
```

#### `tts_service/requirements.txt`

```txt
fastapi==0.115.0
uvicorn[standard]==0.32.0
pydantic==2.9.0
pydantic-settings==2.5.0
qwen-tts>=1.0.0
torch>=2.0.0
scipy>=1.10.0
numpy>=1.24.0
```

#### `docker-compose.yml` 更新

```yaml
services:
  # ... 其他服务 ...

  tts:
    build:
      context: ./tts_service
      dockerfile: Dockerfile
    container_name: qwen3-tts-service
    environment:
      - MODEL_ID=Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice
      - DEVICE=cuda:0
      - TORCH_DTYPE=bfloat16
      - MAX_CONCURRENT=4
      - ENABLE_FLASH_ATTENTION=true
    ports:
      - "8000:8000"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    volumes:
      - ./tts_service/app:/app/app:ro
      - tts_cache:/root/.cache/huggingface
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

volumes:
  tts_cache:
    driver: local
```

---

## 🚀 使用指南

### 1. 部署服务

```bash
# 启动 TTS 服务
docker-compose up -d tts

# 查看日志
docker-compose logs -f tts

# 等待模型加载完成（首次启动需要下载模型，约3-5分钟）
# 日志中看到 "Model loaded successfully" 表示就绪
```

### 2. 测试 API

```bash
# 健康检查
curl http://localhost:8000/api/health

# 获取音色列表
curl http://localhost:8000/api/voices | jq

# 测试合成
curl -X POST http://localhost:8000/api/tts/synthesize \
  -H "Content-Type: application/json" \
  -d '{
    "text": "你好，欢迎使用 Qwen3-TTS 语音合成服务。",
    "language": "zh",
    "speaker": "Vivian",
    "instruct": "用开心的语气说"
  }' | jq '.audio_base64' -r | base64 -d > output.wav

# 播放音频
ffplay output.wav
```

### 3. Worker 配置

```bash
# .env
TTS_BACKEND=qwen3
TTS_SERVICE_URL=http://tts:8000
TTS_MAX_CONCURRENCY=4
TTS_REQUEST_TIMEOUT=60
```

---

## 📊 性能优化

### GPU 内存优化

| 配置 | VRAM 使用 | 性能 |
|-----|----------|------|
| **float16 + FlashAttention** | ~5GB | 最佳 |
| **bfloat16 + FlashAttention** | ~6GB | 推荐 |
| **float32 无优化** | ~12GB | 不推荐 |

### 并发优化

```python
# app/config.py
MAX_CONCURRENT = 4  # 根据 GPU 内存调整

# 单个 RTX 4090 (24GB):
# - 1.7B 模型: 可支持 4-6 并发
# - 0.6B 模型: 可支持 8-12 并发
```

### 缓存优化

```python
# 启用结果缓存（相同文本+音色复用）
ENABLE_CACHE = True
CACHE_SIZE = 100  # 缓存最近 100 个结果
```

---

## 🎯 优势总结

| 特性 | Qwen3-TTS | 阿里云 TTS API | IndexTTS |
|-----|-----------|---------------|----------|
| **首字节延迟** | **97ms** | ~200ms | ~500ms |
| **音色克隆** | ✅ 3秒克隆 | ❌ | ❌ |
| **自然语言控制** | ✅ | ❌ | ❌ |
| **多语言** | ✅ 10种 | ✅ | ✅ 3种 |
| **成本** | 仅GPU成本 | 按量计费 | GPU成本 |
| **可控性** | 完全自主 | 依赖API | 完全自主 |
| **维护** | 需要维护 | 无需维护 | 需要维护 |

**结论**: Qwen3-TTS 在性能和功能上全面领先，且保持了自主可控的优势！

---

**最后更新**: 2026-02-02
**状态**: ✅ 方案确定
**下一步**: 开始实现 FastAPI 服务
