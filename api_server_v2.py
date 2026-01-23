#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
稳定版 api_server_v2.py 文件
移除 run_in_executor / 子线程事件循环，使用主事件循环串行推理
避免 uvloop kwargs 错误与潜在死锁
"""

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
import torch

from loguru import logger
logger.add("logs/api_server_v2.log", rotation="10 MB", retention=10, level="DEBUG", enqueue=True)

from indextts.infer_vllm_v2 import IndexTTS2

tts = None
UPLOAD_DIR = Path("/tmp/uploaded_prompts")

# 串行推理锁，避免并发导致的 PyTorch autograd 问题
inference_lock = asyncio.Lock()

# 确保上传目录存在
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)

# 原有的请求数据模型
class IndexTTS2RequestData(BaseModel):
    text: str
    spk_audio_path: str
    emo_control_method: int = 0
    emo_ref_path: Optional[str] = None
    emo_weight: float = 1.0
    emo_vec: List[float] = [0.0] * 8
    evo_text: Optional[str] = None
    emo_random: bool = False
    max_text_tokens_per_sentence: int = 120

    def to_dict(self):
        return self.dict()

# 扩展的请求数据模型 - 支持音色克隆
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

@asynccontextmanager
async def lifespan(app: FastAPI):
    global tts
    tts = IndexTTS2(
        model_dir=args.model_dir,
        is_fp16=args.is_fp16,
        gpu_memory_utilization=args.gpu_memory_utilization,
        qwenemo_gpu_memory_utilization=args.qwenemo_gpu_memory_utilization,
    )

    # 模型初始化后设置推理模式
    tts.gpt.eval()
    if hasattr(tts, 'bigvgan'):
        tts.bigvgan.eval()
    if hasattr(tts, 's2mel'):
        tts.s2mel.eval()

    # 禁用所有参数的梯度
    for param in tts.gpt.parameters():
        param.requires_grad_(False)
    if hasattr(tts, 'bigvgan'):
        for param in tts.bigvgan.parameters():
            param.requires_grad_(False)

    # 清理旧的上传文件
    try:
        current_time = time.time()
        for file_path in UPLOAD_DIR.glob("prompt_*"):
            if current_time - file_path.stat().st_mtime > 3600:  # 1小时
                file_path.unlink()
                logger.info(f"Cleaned up old file: {file_path}")
    except Exception as e:
        logger.warning(f"Cleanup failed: {e}")

    logger.info("TTS model loaded and server initialized (serial inference)")
    yield


app = FastAPI(lifespan=lifespan, title="IndexTTS VLLM API with Voice Cloning - Stable", version="2.0")

# Add CORS middleware configuration
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Allows all origins, change in production for security
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 音频上传接口
@app.post("/upload_audio")
async def upload_audio(file: UploadFile = File(...)):
    """上传音频文件供TTS使用 - 修复版本"""
    try:
        logger.info(f"Uploading audio file: {file.filename}")
        logger.info(f"Content type: {file.content_type}")

        # 从文件名中提取扩展名（处理URL参数）
        filename = file.filename if file.filename else "audio.wav"
        filename_clean = filename.split('?')[0]  # 移除URL参数
        file_ext = Path(filename_clean).suffix.lower()

        # 支持的音频扩展名
        allowed_extensions = {'.wav', '.mp3', '.m4a', '.flac', '.ogg', '.opus'}
        is_audio_by_extension = file_ext in allowed_extensions
        is_audio_by_content_type = file.content_type and file.content_type.startswith('audio/')

        logger.info(f"Clean filename: {filename_clean}, Extension: {file_ext}")
        logger.info(f"Audio by extension: {is_audio_by_extension}, Audio by content type: {is_audio_by_content_type}")

        # 宽松的文件类型验证
        if filename_clean.endswith(('.wav', '.mp3', '.m4a', '.flac', '.ogg', '.opus')) or \
           (file.content_type and file.content_type.startswith('audio/')) or \
           'prompt' in filename.lower():
            logger.info("File type validation passed")
        else:
            logger.warning(f"File type validation warning - but proceeding anyway. Content-Type: {file.content_type}, Filename: {filename}")

        # 生成唯一文件名
        timestamp = int(time.time() * 1000)
        final_extension = file_ext if file_ext in allowed_extensions else '.wav'
        unique_filename = f"prompt_{timestamp}{final_extension}"
        file_path = UPLOAD_DIR / unique_filename

        # 保存文件
        with open(file_path, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)

        # 验证文件大小
        file_size = file_path.stat().st_size
        if file_size == 0:
            file_path.unlink()
            raise HTTPException(status_code=400, detail="Uploaded file is empty")

        logger.info(f"Upload successful: {file_path} ({file_size} bytes)")

        return {
            "server_path": str(file_path),
            "filename": unique_filename,
            "size": file_size,
            "status": "success"
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Upload error: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Upload failed: {str(e)}")

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
            "message": "Service is running (serial inference)",
            "timestamp": time.time(),
            "upload_dir": str(UPLOAD_DIR),
            "upload_dir_exists": UPLOAD_DIR.exists(),
        }
    )

@app.post("/tts_url", responses={
    200: {"content": {"application/octet-stream": {}}},
    500: {"content": {"application/json": {}}}
})
async def tts_api_url(request: Request):
    """原有TTS接口 - 串行推理版本"""
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

        async with inference_lock:
            with torch.inference_mode():
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

        # 确保输出是numpy数组
        if torch.is_tensor(wav):
            wav = wav.detach().cpu().numpy()

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

# 音色克隆接口
@app.post("/tts_url_with_cloning", responses={
    200: {"content": {"application/octet-stream": {}}},
    500: {"content": {"application/json": {}}}
})
async def tts_with_voice_cloning(request: IndexTTS2RequestDataExtended):
    """使用音色和情感克隆的增强TTS接口 - 串行推理版本"""
    try:
        logger.info(f"Voice cloning request - text: {request.text[:50]}..., spk_path: {request.spk_audio_path}")

        global tts
        if tts is None:
            raise Exception("TTS model not initialized")

        # 准备参数
        emo_alpha = request.emo_alpha if request.emo_alpha else 0.8

        # 使用原音频作为音色和情感参考
        if request.emo_audio_path and request.emo_audio_path != request.spk_audio_path:
            emo_audio_prompt = request.emo_audio_path
            logger.info(f"Using separate emotion reference: {emo_audio_prompt}")
        else:
            emo_audio_prompt = request.spk_audio_path
            logger.info(f"Using same audio for voice and emotion: {emo_audio_prompt}")

        async with inference_lock:
            logger.info("Acquired inference lock for voice cloning")
            with torch.inference_mode():
                sr, wav = await tts.infer(
                    spk_audio_prompt=request.spk_audio_path,
                    text=request.text,
                    output_path=None,
                    emo_audio_prompt=emo_audio_prompt,
                    emo_alpha=emo_alpha,
                    emo_vector=None,
                    use_emo_text=False,
                    emo_text=None,
                    use_random=False,
                    max_text_tokens_per_sentence=request.max_text_tokens_per_sentence
                )
            logger.info("Released inference lock for voice cloning")

        # 确保输出张量不包含梯度信息
        if torch.is_tensor(wav):
            wav = wav.detach().cpu().numpy()

        # 返回音频数据
        with io.BytesIO() as wav_buffer:
            sf.write(wav_buffer, wav, sr, format='WAV')
            wav_bytes = wav_buffer.getvalue()

        logger.info(f"Voice cloning completed successfully, audio size: {len(wav_bytes)} bytes")
        return Response(content=wav_bytes, media_type="audio/wav")

    except Exception as ex:
        tb_str = ''.join(traceback.format_exception(type(ex), ex, ex.__traceback__))
        logger.error(f"Voice cloning error: {tb_str}")

        # 清理GPU缓存
        if torch.cuda.is_available():
            torch.cuda.empty_cache()

        return JSONResponse(
            status_code=500,
            content={
                "status": "error",
                "error": f"Voice cloning failed: {str(ex)}",
                "error_type": "voice_cloning_failure"
            }
        )

# API信息接口
@app.get("/")
async def root():
    """API根路径"""
    return {
        "message": "IndexTTS VLLM API with Voice Cloning - Stable",
        "version": "2.0 - Serial Inference",
        "endpoints": {
            "tts_url": "Original TTS endpoint (serial)",
            "tts_url_with_cloning": "Enhanced TTS with voice cloning (serial)",
            "upload_audio": "Upload audio for voice reference",
            "health": "Health check"
        },
        "stability": "Serial inference, no run_in_executor, no sub-loop"
    }


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

    logger.info(f"Starting IndexTTS VLLM server with serial inference")
    logger.info(f"Model dir: {args.model_dir}")
    logger.info(f"Upload dir: {UPLOAD_DIR}")
    logger.info(f"Server: http://{args.host}:{args.port}")

    uvicorn.run(app=app, host=args.host, port=args.port)
