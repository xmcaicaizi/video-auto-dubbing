# 标准IndexTTS 2.0的修改代码

# ============================================
# 修改位置1：第338行左右的startup_event函数
# ============================================

@app.on_event("startup")
async def startup_event():
    """应用启动时的初始化"""
    global tts_model, args  # 注意：需要添加args到global

    print("Starting IndexTTS VLLM API server...")
    print(f"Upload directory: {UPLOAD_DIR}")

    # 清理旧文件
    await cleanup_old_files()

    # 🔥 IndexTTS 2.0标准加载方式
    try:
        # 导入IndexTTS模块
        from indextts.infer_v2 import IndexTTS2

        # 加载模型
        tts_model = IndexTTS2(
            cfg_path=os.path.join(args.model_dir, "config.yaml"),
            model_dir=args.model_dir,
            use_fp16=True,  # 根据你的GPU情况调整
            use_cuda_kernel=False,
            use_deepspeed=False
        )

        print(f"✅ IndexTTS 2.0 model loaded from {args.model_dir}")

    except Exception as e:
        print(f"❌ Failed to load IndexTTS model: {e}")
        print(f"Model directory: {args.model_dir}")
        tts_model = None

    print("Server startup complete.")

# ============================================
# 修改位置2：第165行左右的tts_inference_same_audio_ref函数
# ============================================

async def tts_inference_same_audio_ref(text: str, spk_audio_path: str, emo_alpha: float, max_tokens: int):
    """使用同一音频文件作为音色和情感参考"""
    try:
        # 生成输出文件名
        output_filename = f"tts_output_{int(time.time() * 1000)}.wav"
        output_path = f"/tmp/{output_filename}"

        # 🔥 使用IndexTTS 2.0官方API
        if tts_model and hasattr(tts_model, 'infer'):
            result = tts_model.infer(
                spk_audio_prompt=spk_audio_path,    # 音色参考
                emo_audio_prompt=spk_audio_path,    # 情感参考（同一文件）
                emo_alpha=emo_alpha,               # 情感强度
                text=text,
                output_path=output_path,
                max_text_tokens_per_sentence=max_tokens,
                verbose=True
            )

            print(f"✅ TTS synthesis completed: {output_path}")
            return output_path
        else:
            raise Exception("TTS model not loaded or infer method not available")

    except Exception as e:
        print(f"❌ TTS inference error: {e}")
        raise Exception(f"TTS inference failed: {str(e)}")

# ============================================
# 修改位置3：第200行左右的tts_inference_with_emotion_ref函数
# ============================================

async def tts_inference_with_emotion_ref(text: str, spk_audio_path: str, emo_audio_path: str, emo_alpha: float, max_tokens: int):
    """使用独立的音色和情感参考音频"""
    try:
        output_filename = f"tts_output_{int(time.time() * 1000)}.wav"
        output_path = f"/tmp/{output_filename}"

        if tts_model and hasattr(tts_model, 'infer'):
            result = tts_model.infer(
                spk_audio_prompt=spk_audio_path,    # 音色参考
                emo_audio_prompt=emo_audio_path,    # 独立情感参考
                emo_alpha=emo_alpha,               # 情感强度
                text=text,
                output_path=output_path,
                max_text_tokens_per_sentence=max_tokens,
                verbose=True
            )

            print(f"✅ TTS synthesis with emotion completed: {output_path}")
            return output_path
        else:
            raise Exception("TTS model not loaded")

    except Exception as e:
        print(f"❌ TTS inference with emotion error: {e}")
        raise Exception(f"TTS inference with emotion failed: {str(e)}")

# ============================================
# 修改位置4：main函数中添加args为全局变量
# ============================================

def main():
    global args  # 添加这一行

    parser = argparse.ArgumentParser(description="IndexTTS VLLM API Server with Voice Cloning")
    parser.add_argument("--model_dir", type=str, required=True, help="Path to model directory")
    parser.add_argument("--host", type=str, default="0.0.0.0", help="Host to bind to")
    parser.add_argument("--port", type=int, default=6006, help="Port to bind to")
    parser.add_argument("--gpu_memory_utilization", type=float, default=0.25, help="GPU memory utilization")

    args = parser.parse_args()

    print(f"Starting server with model_dir: {args.model_dir}")
    print(f"Server will be available at http://{args.host}:{args.port}")
    print(f"GPU memory utilization: {args.gpu_memory_utilization}")

    # 启动服务器
    uvicorn.run(
        app,
        host=args.host,
        port=args.port,
        log_level="info"
    )