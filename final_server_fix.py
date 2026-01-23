# 基于你原始文件的精确修改代码

# ============================================
# 1. 修改导入部分（文件顶部）
# ============================================

import os
import asyncio
import io
import traceback
from fastapi import FastAPI, Request, Response, File, UploadFile, Form, HTTPException
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager
from fastapi.middleware.cors import CORSMiddleware
import uvicorn
import argparse
import json
import time
import soundfile as sf
from typing import List, Optional, Union
import shutil
from pathlib import Path
from pydantic import BaseModel

from loguru import logger
logger.add("logs/api_server_v2.log", rotation="10 MB", retention=10, level="DEBUG", enqueue=True)

from indextts.infer_vllm_v2 import IndexTTS2

# ============================================
# 2. 全局变量和初始化
# ============================================

tts = None
UPLOAD_DIR = Path("/tmp/uploaded_prompts")
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)

# ============================================
# 3. 数据模型（新增）
# ============================================

class IndexTTS2RequestDataExtended(BaseModel):
    text: str
    spk_audio_path: str
    emo_control_method: int = 0
    emo_ref_path: Optional[str] = None
    emo_weight: float = 1.0
    emo_vec: List[float] = [0.0] * 8
    emo_text: Optional[str] = None
    emo_random: bool = False
    max_text_tokens_per_sentence: int = 120
    # 新增：支持独立的情感音频路径
    emo_audio_path: Optional[str] = None
    emo_alpha: float = 0.8

# ============================================
# 4. 修改lifespan管理器
# ============================================

@asynccontextmanager
async def lifespan(app: FastAPI):
    global tts
    tts = IndexTTS2(
        model_dir=args.model_dir,
        is_fp16=args.is_fp16,
        gpu_memory_utilization=args.gpu_memory_utilization,
        qwenemo_gpu_memory_utilization=args.qwenemo_gpu_memory_utilization,
    )

    # 清理旧的上传文件
    try:
        current_time = time.time()
        for file_path in UPLOAD_DIR.glob("prompt_*"):
            if current_time - file_path.stat().st_mtime > 3600:  # 1小时
                file_path.unlink()
                logger.info(f"Cleaned up old file: {file_path}")
    except Exception as e:
        logger.warning(f"Cleanup failed: {e}")

    logger.info("TTS model loaded and server initialized")
    yield

# ============================================
# 5. 创建FastAPI应用
# ============================================

app = FastAPI(lifespan=lifespan, title="IndexTTS VLLM API with Voice Cloning", version="2.0")

# Add CORS middleware configuration
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# ============================================
# 6. 音频上传接口（新增）
# ============================================

@app.post("/upload_audio")
async def upload_audio(file: UploadFile = File(...)):
    """上传音频文件供TTS使用"""
    try:
        logger.info(f"Uploading audio file: {file.filename}")

        # 验证文件类型
        if file.content_type and not file.content_type.startswith('audio/'):
            allowed_extensions = {'.wav', '.mp3', '.m4a', '.flac', '.ogg'}
            file_ext = Path(file.filename).suffix.lower() if file.filename else ''
            if file_ext not in allowed_extensions:
                raise HTTPException(status_code=400, detail="File must be audio format")

        # 生成唯一文件名
        timestamp = int(time.time() * 1000)
        file_extension = Path(file.filename).suffix if file.filename else '.wav'
        unique_filename = f"prompt_{timestamp}{file_extension}"
        file_path = UPLOAD_DIR / unique_filename

        # 保存文件
        with open(file_path, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)

        # 验证文件大小
        file_size = file_path.stat().st_size
        if file_size == 0:
            file_path.unlink()
            raise HTTPException(status_code=400, detail="Uploaded file is empty")

        logger.info(f"Audio uploaded successfully: {file_path} ({file_size} bytes)")

        return {
            "server_path": str(file_path),
            "filename": unique_filename,
            "size": file_size,
            "status": "success"
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Upload failed: {e}")
        raise HTTPException(status_code=500, detail=f"Upload failed: {str(e)}")

# ============================================
# 7. 保持原有的health检查接口
# ============================================

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    if tts is None:
        return JSONResponse(
            status_code=503,
            content={
                "status": "unhealthy",
                "message": "TTS model not initialized",
                "upload_dir": str(UPLOAD_DIR),
                "upload_dir_exists": UPLOAD_DIR.exists(),
            }
        )

    return JSONResponse(
        status_code=200,
        content={
            "status": "healthy",
            "message": "Service is running",
            "timestamp": time.time(),
            "upload_dir": str(UPLOAD_DIR),
            "upload_dir_exists": UPLOAD_DIR.exists(),
        }
    )

# ============================================
# 8. 保持原有的tts_url接口不变
# ============================================

@app.post("/tts_url", responses={
    200: {"content": {"application/octet-stream": {}}},
    500: {"content": {"application/json": {}}}
})
async def tts_api_url(request: Request):
    try:
        data = await request.json()
        emo_control_method = data.get("emo_control_method", 0)
        text = data["text"]
        spk_audio_path = data["spk_audio_path"]
        emo_ref_path = data.get("emo_ref_path", None)
        emo_weight = data.get("emo_weight", 1.0)
        emo_vec = data.get("emo_vec", [0] * 8)
        emo_text = data.get("emo_text", None)
        emo_random = data.get("emo_random", False)
        max_text_tokens_per_sentence = data.get("max_text_tokens_per_sentence", 120)

        global tts
        if type(emo_control_method) is not int:
            emo_control_method = emo_control_method.value
        if emo_control_method == 0:
            emo_ref_path = None
            emo_weight = 1.0
        if emo_control_method == 1:
            emo_weight = emo_weight
        if emo_control_method == 2:
            vec = emo_vec
            vec_sum = sum(vec)
            if vec_sum > 1.5:
                return JSONResponse(
                    status_code=500,
                    content={
                        "status": "error",
                        "error": "情感向量之和不能超过1.5，请调整后重试。"
                    }
                )
        else:
            vec = None

        logger.info(f"TTS request - text length: {len(text)}, spk_audio_path: {spk_audio_path}")

        sr, wav = await tts.infer(
            spk_audio_prompt=spk_audio_path,
            text=text,
            output_path=None,
            emo_audio_prompt=emo_ref_path,
            emo_alpha=emo_weight,
            emo_vector=vec,
            use_emo_text=(emo_control_method==3),
            emo_text=emo_text,
            use_random=emo_random,
            max_text_tokens_per_sentence=int(max_text_tokens_per_sentence)
        )

        with io.BytesIO() as wav_buffer:
            sf.write(wav_buffer, wav, sr, format='WAV')
            wav_bytes = wav_buffer.getvalue()

        return Response(content=wav_bytes, media_type="audio/wav")

    except Exception as ex:
        tb_str = ''.join(traceback.format_exception(type(ex), ex, ex.__traceback__))
        logger.error(f"TTS API error: {tb_str}")
        return JSONResponse(
            status_code=500,
            content={
                "status": "error",
                "error": str(tb_str)
            }
        )

# ============================================
# 9. 新增音色克隆接口
# ============================================

@app.post("/tts_url_with_cloning", responses={
    200: {"content": {"application/octet-stream": {}}},
    500: {"content": {"application/json": {}}}
})
async def tts_with_voice_cloning(request: IndexTTS2RequestDataExtended):
    """使用音色和情感克隆的增强TTS接口"""
    try:
        logger.info(f"Voice cloning request - text: {request.text[:50]}..., spk_path: {request.spk_audio_path}")

        global tts
        if tts is None:
            raise Exception("TTS model not initialized")

        # 准备参数
        emo_alpha = request.emo_alpha if request.emo_alpha else 0.8

        # 🔥 关键：使用原音频作为音色和情感参考
        if request.emo_audio_path and request.emo_audio_path != request.spk_audio_path:
            # 使用独立的情感参考音频
            emo_audio_prompt = request.emo_audio_path
            logger.info(f"Using separate emotion reference: {emo_audio_prompt}")
        else:
            # 使用同一音频作为音色和情感参考
            emo_audio_prompt = request.spk_audio_path
            logger.info(f"Using same audio for voice and emotion: {emo_audio_prompt}")

        # 调用TTS推理
        sr, wav = await tts.infer(
            spk_audio_prompt=request.spk_audio_path,    # 音色参考
            text=request.text,
            output_path=None,
            emo_audio_prompt=emo_audio_prompt,          # 情感参考
            emo_alpha=emo_alpha,                        # 情感强度
            emo_vector=None,                            # 不使用情感向量
            use_emo_text=False,                         # 不使用情感文本
            emo_text=None,
            use_random=False,
            max_text_tokens_per_sentence=request.max_text_tokens_per_sentence
        )

        # 返回音频数据
        with io.BytesIO() as wav_buffer:
            sf.write(wav_buffer, wav, sr, format='WAV')
            wav_bytes = wav_buffer.getvalue()

        logger.info(f"Voice cloning completed successfully, audio size: {len(wav_bytes)} bytes")
        return Response(content=wav_bytes, media_type="audio/wav")

    except Exception as ex:
        tb_str = ''.join(traceback.format_exception(type(ex), ex, ex.__traceback__))
        logger.error(f"Voice cloning error: {tb_str}")
        return JSONResponse(
            status_code=500,
            content={
                "status": "error",
                "error": str(tb_str)
            }
        )

# ============================================
# 10. API信息接口（新增）
# ============================================

@app.get("/")
async def root():
    """API根路径"""
    return {
        "message": "IndexTTS VLLM API with Voice Cloning",
        "version": "2.0",
        "endpoints": {
            "tts_url": "Original TTS endpoint",
            "tts_url_with_cloning": "Enhanced TTS with voice cloning",
            "upload_audio": "Upload audio for voice reference",
            "health": "Health check"
        },
        "voice_cloning_features": {
            "upload_audio": "Upload reference audio for voice cloning",
            "same_audio_emotion": "Use same audio for voice and emotion reference",
            "separate_emotion": "Use separate audio for emotion reference",
            "emotion_strength": "Control emotion strength with emo_alpha (0.0-1.0)"
        }
    }

# ============================================
# 11. 保持原有的main函数，但添加参数
# ============================================

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", type=str, default="0.0.0.0")
    parser.add_argument("--port", type=int, default=6006)
    parser.add_argument("--model_dir", type=str, default="checkpoints/IndexTTS-2-vLLM", help="Model checkpoints directory")
    parser.add_argument("--is_fp16", action="store_true", default=False, help="Fp16 infer")
    parser.add_argument("--gpu_memory_utilization", type=float, default=0.25)
    parser.add_argument("--qwenemo_gpu_memory_utilization", type=float, default=0.10)
    parser.add_argument("--verbose", action="store_true", default=False, help="Enable verbose mode")
    args = parser.parse_args()

    if not os.path.exists("outputs"):
        os.makedirs("outputs")

    # 确保日志目录存在
    if not os.path.exists("logs"):
        os.makedirs("logs")

    logger.info(f"Starting IndexTTS VLLM server with voice cloning")
    logger.info(f"Model dir: {args.model_dir}")
    logger.info(f"Upload dir: {UPLOAD_DIR}")
    logger.info(f"Server: http://{args.host}:{args.port}")

    uvicorn.run(app=app, host=args.host, port=args.port)