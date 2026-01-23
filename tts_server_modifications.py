# TTS服务器修改 - 添加到 api_server_v2.py

from fastapi import FastAPI, UploadFile, File, HTTPException
from pydantic import BaseModel
import shutil
import os
import time
from pathlib import Path
from typing import Optional

# 在现有的app定义后添加以下代码：

# 创建上传目录
UPLOAD_DIR = Path("/tmp/uploaded_prompts")
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)

@app.post("/upload_audio")
async def upload_audio(file: UploadFile = File(...)):
    """上传音频文件供TTS使用"""
    try:
        # 验证文件类型
        if not file.content_type or not file.content_type.startswith('audio/'):
            # 如果content_type检测失败，通过文件扩展名验证
            allowed_extensions = {'.wav', '.mp3', '.m4a', '.flac', '.ogg'}
            file_ext = Path(file.filename).suffix.lower()
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

# 更新现有的IndexTTS2RequestData类以支持情感参考音频
class IndexTTS2RequestDataExtended(BaseModel):
    text: str
    spk_audio_path: str
    emo_control_method: int = 0
    emo_ref_path: Optional[str] = None
    emo_weight: float = 1.0
    emo_vec: list = [0.0] * 8
    emo_text: Optional[str] = None
    emo_random: bool = False
    max_text_tokens_per_sentence: int = 120
    # 新增：支持独立的情感音频路径
    emo_audio_path: Optional[str] = None
    emo_alpha: float = 0.8

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
            # 使用原有的方法，但确保使用音色音频作为情感参考
            result = await tts_inference_same_audio_ref(
                text=request.text,
                spk_audio_path=request.spk_audio_path,
                emo_alpha=request.emo_alpha,
                max_tokens=request.max_text_tokens_per_sentence
            )

        return {"audio_url": result, "status": "success"}

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"TTS synthesis failed: {str(e)}")

async def tts_inference_same_audio_ref(text: str, spk_audio_path: str, emo_alpha: float, max_tokens: int):
    """使用同一音频文件作为音色和情感参考"""
    # 这里需要根据具体的IndexTTS集成方式调用
    # 示例实现（需要根据实际的TTS模型调用方式调整）:

    # 假设你的TTS模型实例叫做 tts_model
    output_path = f"/tmp/tts_output_{int(time.time() * 1000)}.wav"

    # 调用IndexTTS 2.0的推理方法
    # 这里需要根据你实际的模型加载和推理方式调整
    result = tts_model.infer(
        spk_audio_prompt=spk_audio_path,    # 音色参考
        emo_audio_prompt=spk_audio_path,    # 情感参考（同一文件）
        emo_alpha=emo_alpha,               # 情感强度
        text=text,
        output_path=output_path,
        max_text_tokens_per_sentence=max_tokens
    )

    return result

async def tts_inference_with_emotion_ref(text: str, spk_audio_path: str, emo_audio_path: str, emo_alpha: float, max_tokens: int):
    """使用独立的音色和情感参考音频"""
    output_path = f"/tmp/tts_output_{int(time.time() * 1000)}.wav"

    # 调用IndexTTS 2.0的推理方法，使用独立的情感参考
    result = tts_model.infer(
        spk_audio_prompt=spk_audio_path,    # 音色参考
        emo_audio_prompt=emo_audio_path,    # 独立的情感参考
        emo_alpha=emo_alpha,               # 情感强度
        text=text,
        output_path=output_path,
        max_text_tokens_per_sentence=max_tokens
    )

    return result

# 清理旧的上传文件的定期任务
@app.on_event("startup")
async def cleanup_old_files():
    """启动时清理超过1小时的旧上传文件"""
    try:
        import glob
        current_time = time.time()
        pattern = str(UPLOAD_DIR / "prompt_*")

        for file_path in glob.glob(pattern):
            file_stat = os.stat(file_path)
            if current_time - file_stat.st_mtime > 3600:  # 1小时
                os.remove(file_path)
                print(f"Cleaned up old file: {file_path}")

    except Exception as e:
        print(f"Cleanup failed: {e}")

# 健康检查接口
@app.get("/health")
async def health_check():
    """健康检查接口"""
    return {
        "status": "healthy",
        "upload_dir": str(UPLOAD_DIR),
        "upload_dir_exists": UPLOAD_DIR.exists(),
        "timestamp": int(time.time())
    }