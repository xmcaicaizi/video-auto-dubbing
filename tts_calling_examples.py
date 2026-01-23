# TTS调用部分的具体调整示例

# ============================================
# 位置：api_server_v2.py 第165行左右
# 函数：tts_inference_same_audio_ref()
# ============================================

async def tts_inference_same_audio_ref(text: str, spk_audio_path: str, emo_alpha: float, max_tokens: int):
    """使用同一音频文件作为音色和情感参考"""
    try:
        # 生成输出文件名
        output_filename = f"tts_output_{int(time.time() * 1000)}.wav"
        output_path = f"/tmp/{output_filename}"

        # 🔥 这里需要根据你的实际TTS模型调用方式进行调整 🔥
        # ============================================
        # 方案A：标准IndexTTS 2.0调用（推荐）
        # ============================================
        if hasattr(tts_model, 'infer'):
            # 官方IndexTTS 2.0 API调用方式
            result = tts_model.infer(
                spk_audio_prompt=spk_audio_path,    # 音色参考
                emo_audio_prompt=spk_audio_path,    # 情感参考（同一文件）
                emo_alpha=emo_alpha,               # 情感强度
                text=text,
                output_path=output_path,
                max_text_tokens_per_sentence=max_tokens,
                verbose=True  # 可选：显示详细日志
            )

            print(f"✅ TTS inference completed: {output_path}")
            return output_path

        # ============================================
        # 方案B：如果你的模型有不同的调用方式
        # ============================================
        elif hasattr(tts_model, 'synthesize'):
            # 某些TTS库可能使用synthesize方法
            result = await tts_model.synthesize(
                text=text,
                reference_audio=spk_audio_path,
                emotion_audio=spk_audio_path,
                emotion_weight=emo_alpha,
                output_file=output_path,
                max_tokens_per_sentence=max_tokens
            )
            return result

        # ============================================
        # 方案C：如果是VLLM封装的调用方式
        # ============================================
        elif hasattr(tts_model, 'generate'):
            # VLLM风格的调用
            result = await tts_model.generate(
                prompt_audio=spk_audio_path,
                text=text,
                output_path=output_path,
                emotion_strength=emo_alpha
            )
            return result

        # ============================================
        # 方案D：如果是HTTP API调用
        # ============================================
        else:
            # 如果你的TTS是通过HTTP API调用的
            import aiohttp

            async with aiohttp.ClientSession() as session:
                data = {
                    "text": text,
                    "spk_audio_path": spk_audio_path,
                    "emo_audio_path": spk_audio_path,
                    "emo_alpha": emo_alpha,
                    "max_text_tokens_per_sentence": max_tokens
                }

                async with session.post("http://localhost:8000/tts", json=data) as resp:
                    result = await resp.json()
                    return result["output_path"]

    except Exception as e:
        raise Exception(f"TTS inference failed: {str(e)}")

# ============================================
# 同样需要修改的其他TTS调用函数
# ============================================

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
            return output_path
        else:
            # 其他调用方式...
            pass

    except Exception as e:
        raise Exception(f"TTS inference with emotion failed: {str(e)}")

# ============================================
# 如何确定使用哪种调用方式？
# ============================================

# 1. 检查你当前api_server_v2.py中的TTS调用代码
# 2. 查看你的TTS模型对象有哪些方法：
#    - print(dir(tts_model))  # 显示所有可用方法
# 3. 查看原有的工作代码是如何调用TTS的

# 常见的方法名称：
# - tts_model.infer()           # IndexTTS官方
# - tts_model.synthesize()      # 通用TTS库
# - tts_model.generate()        # VLLM风格
# - tts_model.text_to_speech()  # 其他库