#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
完整的 api_server_v2.py 文件
在原有基础上添加音频上传和音色克隆功能
"""

from fastapi import FastAPI, UploadFile, File, HTTPException
from pydantic import BaseModel
import uvicorn
import argparse
import shutil
import os
import time
from pathlib import Path
from typing import Optional, List
import json
import asyncio

# 创建FastAPI应用
app = FastAPI(title="IndexTTS VLLM API with Voice Cloning", version="2.0")

# 全局变量
tts_model = None
UPLOAD_DIR = Path("/tmp/uploaded_prompts")

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

# 音频上传接口
@app.post("/upload_audio")
async def upload_audio(file: UploadFile = File(...)):
    """上传音频文件供TTS使用"""
    try:
        # 验证文件类型
        if file.content_type and not file.content_type.startswith('audio/'):
            # 如果content_type检测失败，通过文件扩展名验证
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
            file_path.unlink()  # 删除空文件
            raise HTTPException(status_code=400, detail="Uploaded file is empty")

        return {
            "server_path": str(file_path),
            "filename": unique_filename,
            "size": file_size,
            "status": "success"
        }

    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Upload failed: {str(e)}")

# 原有的TTS接口
@app.post("/tts_url")
async def tts_url(request: IndexTTS2RequestData):
    """原有的TTS接口"""
    try:
        # 这里调用你的TTS模型进行推理
        # 需要根据你实际的模型加载和推理方式调整

        # 示例调用（请根据实际情况修改）
        result = await tts_inference(
            text=request.text,
            spk_audio_path=request.spk_audio_path,
            emo_control_method=request.emo_control_method,
            emo_ref_path=request.emo_ref_path,
            emo_weight=request.emo_weight,
            emo_vec=request.emo_vec,
            max_tokens=request.max_text_tokens_per_sentence
        )

        return {"audio_url": result}

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"TTS synthesis failed: {str(e)}")

# 新增的音色克隆接口
@app.post("/tts_url_with_cloning")
async def tts_with_voice_cloning(request: IndexTTS2RequestDataExtended):
    """使用音色和情感克隆的增强TTS接口"""
    try:
        # 如果指定了独立的情感音频路径，使用它
        if request.emo_audio_path:
            # 使用独立的情感参考
            result = await tts_inference_with_emotion_ref(
                text=request.text,
                spk_audio_path=request.spk_audio_path,
                emo_audio_path=request.emo_audio_path,
                emo_alpha=request.emo_alpha,
                max_tokens=request.max_text_tokens_per_sentence
            )
        else:
            # 使用同一音频作为音色和情感参考
            result = await tts_inference_same_audio_ref(
                text=request.text,
                spk_audio_path=request.spk_audio_path,
                emo_alpha=request.emo_alpha,
                max_tokens=request.max_text_tokens_per_sentence
            )

        return {"audio_url": result, "status": "success"}

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"TTS synthesis failed: {str(e)}")

# TTS推理函数 - 使用同一音频作为音色和情感参考
async def tts_inference_same_audio_ref(text: str, spk_audio_path: str, emo_alpha: float, max_tokens: int):
    """使用同一音频文件作为音色和情感参考"""
    try:
        # 生成输出文件名
        output_filename = f"tts_output_{int(time.time() * 1000)}.wav"
        output_path = f"/tmp/{output_filename}"

        # 调用IndexTTS 2.0的推理方法
        # 这里需要根据你的实际TTS模型调用方式进行调整

        if hasattr(tts_model, 'infer'):
            # 官方IndexTTS 2.0 API调用方式
            result = tts_model.infer(
                spk_audio_prompt=spk_audio_path,    # 音色参考
                emo_audio_prompt=spk_audio_path,    # 情感参考（同一文件）
                emo_alpha=emo_alpha,               # 情感强度
                text=text,
                output_path=output_path,
                max_text_tokens_per_sentence=max_tokens
            )
        else:
            # 如果你使用的是其他调用方式，请在这里修改
            # 例如：
            result = await custom_tts_call(text, spk_audio_path, output_path)

        # 返回生成的音频文件URL或路径
        return output_path

    except Exception as e:
        raise Exception(f"TTS inference failed: {str(e)}")

# TTS推理函数 - 使用独立的音色和情感参考
async def tts_inference_with_emotion_ref(text: str, spk_audio_path: str, emo_audio_path: str, emo_alpha: float, max_tokens: int):
    """使用独立的音色和情感参考音频"""
    try:
        output_filename = f"tts_output_{int(time.time() * 1000)}.wav"
        output_path = f"/tmp/{output_filename}"

        if hasattr(tts_model, 'infer'):
            # 官方IndexTTS 2.0 API调用方式
            result = tts_model.infer(
                spk_audio_prompt=spk_audio_path,    # 音色参考
                emo_audio_prompt=emo_audio_path,    # 独立的情感参考
                emo_alpha=emo_alpha,               # 情感强度
                text=text,
                output_path=output_path,
                max_text_tokens_per_sentence=max_tokens
            )
        else:
            # 自定义调用方式
            result = await custom_tts_call_with_emotion(text, spk_audio_path, emo_audio_path, output_path)

        return result

    except Exception as e:
        raise Exception(f"TTS inference with emotion failed: {str(e)}")

# 原有的TTS推理函数（兼容性）
async def tts_inference(text: str, spk_audio_path: str, emo_control_method: int,
                       emo_ref_path: Optional[str], emo_weight: float,
                       emo_vec: List[float], max_tokens: int):
    """原有的TTS推理接口，保持兼容性"""
    try:
        output_filename = f"tts_output_{int(time.time() * 1000)}.wav"
        output_path = f"/tmp/{output_filename}"

        # 这里需要根据你的实际TTS模型实现进行调整
        # 示例实现：
        if hasattr(tts_model, 'infer'):
            if emo_control_method == 1 and emo_ref_path:
                # 使用情感参考音频
                result = tts_model.infer(
                    spk_audio_prompt=spk_audio_path,
                    emo_audio_prompt=emo_ref_path,
                    emo_alpha=emo_weight,
                    text=text,
                    output_path=output_path,
                    max_text_tokens_per_sentence=max_tokens
                )
            elif emo_control_method == 2:
                # 使用情感向量
                result = tts_model.infer(
                    spk_audio_prompt=spk_audio_path,
                    text=text,
                    output_path=output_path,
                    emo_vec=emo_vec,
                    max_text_tokens_per_sentence=max_tokens
                )
            else:
                # 默认：情感与音色参考音频相同
                result = tts_model.infer(
                    spk_audio_prompt=spk_audio_path,
                    text=text,
                    output_path=output_path,
                    max_text_tokens_per_sentence=max_tokens
                )
        else:
            # 兜底的自定义实现
            result = await custom_tts_call(text, spk_audio_path, output_path)

        return result

    except Exception as e:
        raise Exception(f"TTS inference failed: {str(e)}")

# 自定义TTS调用函数（需要根据实际情况实现）
async def custom_tts_call(text: str, audio_path: str, output_path: str):
    """
    自定义TTS调用函数
    请根据你的实际TTS模型调用方式进行实现
    """
    # 这里是示例实现，你需要根据实际情况修改
    # 例如，如果你使用的是特定的TTS库或API

    # 示例：
    # result = your_tts_model.synthesize(
    #     text=text,
    #     reference_audio=audio_path,
    #     output_file=output_path
    # )

    # 临时返回输出路径（实际应该返回TTS结果）
    return output_path

async def custom_tts_call_with_emotion(text: str, spk_audio_path: str, emo_audio_path: str, output_path: str):
    """
    带情感参考的自定义TTS调用
    """
    # 根据实际情况实现
    return output_path

# 健康检查接口
@app.get("/health")
async def health_check():
    """健康检查接口"""
    return {
        "status": "healthy",
        "upload_dir": str(UPLOAD_DIR),
        "upload_dir_exists": UPLOAD_DIR.exists(),
        "tts_model_loaded": tts_model is not None,
        "timestamp": int(time.time())
    }

# 获取API信息
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
        }
    }

# 清理旧文件的函数
async def cleanup_old_files():
    """清理超过1小时的旧上传文件"""
    try:
        current_time = time.time()
        for file_path in UPLOAD_DIR.glob("prompt_*"):
            if current_time - file_path.stat().st_mtime > 3600:  # 1小时
                file_path.unlink()
                print(f"Cleaned up old file: {file_path}")

    except Exception as e:
        print(f"Cleanup failed: {e}")

# 应用启动事件
@app.on_event("startup")
async def startup_event():
    """应用启动时的初始化"""
    global tts_model

    print("Starting IndexTTS VLLM API server...")
    print(f"Upload directory: {UPLOAD_DIR}")

    # 清理旧文件
    await cleanup_old_files()

    # 这里添加你的TTS模型加载逻辑
    # 例如：
    # from your_tts_module import load_model
    # tts_model = load_model(model_dir=args.model_dir)

    print("Server startup complete.")

# 应用关闭事件
@app.on_event("shutdown")
async def shutdown_event():
    """应用关闭时的清理"""
    print("Shutting down IndexTTS VLLM API server...")

    # 这里添加模型清理逻辑
    global tts_model
    if tts_model:
        # 清理模型资源
        # tts_model.cleanup() 或类似操作
        pass

    print("Server shutdown complete.")

# 主函数
def main():
    parser = argparse.ArgumentParser(description="IndexTTS VLLM API Server with Voice Cloning")
    parser.add_argument("--model_dir", type=str, required=True, help="Path to model directory")
    parser.add_argument("--host", type=str, default="0.0.0.0", help="Host to bind to")
    parser.add_argument("--port", type=int, default=6006, help="Port to bind to")
    parser.add_argument("--gpu_memory_utilization", type=float, default=0.25, help="GPU memory utilization")

    args = parser.parse_args()

    print(f"Starting server with model_dir: {args.model_dir}")
    print(f"Server will be available at http://{args.host}:{args.port}")
    print(f"GPU memory utilization: {args.gpu_memory_utilization}")

    # 这里添加你的模型加载逻辑
    global tts_model
    # 例如：
    # tts_model = YourTTSModel(
    #     model_dir=args.model_dir,
    #     gpu_memory_utilization=args.gpu_memory_utilization
    # )

    # 启动服务器
    uvicorn.run(
        app,
        host=args.host,
        port=args.port,
        log_level="info"
    )

if __name__ == "__main__":
    main()