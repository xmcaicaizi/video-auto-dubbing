"""
DashScope TTS 客户端封装
支持两种模式：
1. 系统音色模式（cosyvoice-v1）- 使用预置音色
2. 声音复刻模式（qwen3-tts-vc-realtime-2026-01-15）- 先复刻，后合成
"""

from typing import Optional

import dashscope
from dashscope.audio.tts_v2 import SpeechSynthesizer, AudioFormat
from loguru import logger

from app.config import settings


class VoiceCloneService:
    """声音复刻服务"""

    def __init__(self, api_key: str):
        self.api_key = api_key
        dashscope.api_key = api_key

    def enroll_voice(
        self,
        audio_path: str,
        target_model: str = "qwen3-tts-vc-realtime-2026-01-15",
        prefix: str = "custom_voice",
    ) -> Optional[str]:
        """
        复刻音色

        Args:
            audio_path: 音频文件路径（本地或 URL），建议 10-20 秒
            target_model: 目标模型
            prefix: 音色前缀

        Returns:
            voice_id（如 vc_xxx），失败返回 None
        """
        try:
            from dashscope.audio.tts_v2 import VoiceEnrollmentService

            logger.info(f"Enrolling voice: audio={audio_path}, model={target_model}")

            service = VoiceEnrollmentService()
            result = service.create_voice(
                target_model=target_model, prefix=prefix, audio_url=audio_path
            )

            if result.status_code == 200:
                voice_id = result.output.voice_id
                logger.info(f"Voice enrolled successfully: voice_id={voice_id}")
                return voice_id
            else:
                logger.error(f"Voice enrollment failed: {result.message}")
                return None

        except Exception as e:
            logger.error(f"Voice enrollment error: {e}")
            return None


class TTSClient:
    """DashScope TTS 客户端"""

    # 支持的模型
    MODEL_COSYVOICE = "cosyvoice-v1"  # 系统音色
    MODEL_QWEN3_VC = "qwen3-tts-vc-realtime-2026-01-15"  # 声音复刻

    def __init__(
        self,
        api_key: Optional[str] = None,
        model: Optional[str] = None,
        voice: Optional[str] = None,
        format: Optional[str] = None,
    ):
        """
        初始化 TTS 客户端

        Args:
            api_key: DashScope API Key
            model: 模型名称
                - cosyvoice-v1: 系统音色（默认）
                - qwen3-tts-vc-realtime-2026-01-15: 声音复刻
            voice: 音色名称或 voice_id
                - 系统音色（cosyvoice-v1）：longxiaochun, longyuan 等
                - 复刻音色（qwen3-tts-vc）：vc_xxx 格式
            format: 音频格式，如 mp3, wav, pcm
        """
        self.api_key = api_key or settings.dashscope_api_key
        self.model = model or settings.tts_model
        self.voice = voice or settings.tts_voice
        self.format = format or settings.tts_format

        if not self.api_key:
            raise ValueError("DASHSCOPE_API_KEY is required")

        # 设置 API Key
        dashscope.api_key = self.api_key

        # 声音复刻服务（仅在使用复刻模型时初始化）
        if self.model == self.MODEL_QWEN3_VC:
            self.clone_service = VoiceCloneService(self.api_key)
        else:
            self.clone_service = None

        logger.info(
            f"TTS Client initialized: model={self.model}, "
            f"voice={self.voice}, format={self.format}"
        )

    def enroll_voice(
        self, audio_path: str, prefix: str = "custom_voice"
    ) -> Optional[str]:
        """
        复刻音色（仅适用于声音复刻模型）

        Args:
            audio_path: 音频文件路径（本地或 URL），建议 10-20 秒
            prefix: 音色前缀

        Returns:
            voice_id（如 vc_xxx），失败返回 None

        Example:
            >>> client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")
            >>> voice_id = client.enroll_voice("sample_audio.wav")
            >>> audio = client.synthesize("你好", voice=voice_id)
        """
        if self.model != self.MODEL_QWEN3_VC:
            logger.warning(
                f"Voice cloning only supported for {self.MODEL_QWEN3_VC}, "
                f"current model: {self.model}"
            )
            return None

        if not self.clone_service:
            self.clone_service = VoiceCloneService(self.api_key)

        return self.clone_service.enroll_voice(
            audio_path=audio_path, target_model=self.model, prefix=prefix
        )

    def synthesize(
        self,
        text: str,
        voice: Optional[str] = None,
        format: Optional[str] = None,
        auto_clone: bool = False,
        clone_audio_path: Optional[str] = None,
    ) -> bytes:
        """
        语音合成

        Args:
            text: 待合成文本
            voice: 音色名称或 voice_id（可选，覆盖初始化参数）
                - 系统音色（cosyvoice-v1）：longxiaochun, longyuan 等
                - 复刻音色（qwen3-tts-vc）：vc_xxx 格式
            format: 音频格式（可选，覆盖初始化参数）
            auto_clone: 是否自动复刻（仅限声音复刻模型）
            clone_audio_path: 复刻音频路径（仅在 auto_clone=True 时使用）

        Returns:
            音频数据（bytes）

        Raises:
            RuntimeError: 合成失败
            ValueError: 参数错误

        Example:
            # 系统音色模式
            >>> client = TTSClient(model="cosyvoice-v1")
            >>> audio = client.synthesize("你好", voice="longxiaochun")

            # 声音复刻模式（手动复刻）
            >>> client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")
            >>> voice_id = client.enroll_voice("sample.wav")
            >>> audio = client.synthesize("你好", voice=voice_id)

            # 声音复刻模式（自动复刻）
            >>> client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")
            >>> audio = client.synthesize(
            ...     "你好",
            ...     auto_clone=True,
            ...     clone_audio_path="sample.wav"
            ... )
        """
        voice = voice or self.voice
        format = format or self.format

        # 如果是声音复刻模型 + auto_clone
        if (
            self.model == self.MODEL_QWEN3_VC
            and auto_clone
            and clone_audio_path
            and (not voice or not voice.startswith("vc_"))
        ):
            logger.info("Auto-cloning voice before synthesis...")
            voice = self.enroll_voice(clone_audio_path)
            if not voice:
                raise RuntimeError("Auto-clone failed, cannot synthesize")

        # 验证复刻模型必须使用 voice_id
        if self.model == self.MODEL_QWEN3_VC and (not voice or not voice.startswith("vc_")):
            raise ValueError(
                f"Model {self.MODEL_QWEN3_VC} requires voice_id (vc_xxx format). "
                f"Got: {voice}. Please call enroll_voice() first or use auto_clone=True."
            )

        logger.info(
            f"Synthesizing: text_len={len(text)}, model={self.model}, "
            f"voice={voice}, format={format}"
        )

        try:
            synthesizer = SpeechSynthesizer(
                model=self.model,
                voice=voice,
            )

            # 调用合成
            audio_data = synthesizer.call(text)

            if not audio_data:
                raise RuntimeError("TTS returned empty audio")

            logger.info(f"Synthesis completed: {len(audio_data)} bytes")
            return audio_data

        except Exception as e:
            logger.error(f"Synthesis failed: {e}")
            raise RuntimeError(f"Synthesis failed: {e}") from e

    def synthesize_with_duration(
        self,
        text: str,
        target_duration_ms: int,
        voice: Optional[str] = None,
        format: Optional[str] = None,
        tolerance: float = 0.1,
    ) -> bytes:
        """
        合成指定时长的音频（通过调整语速）

        Args:
            text: 待合成文本
            target_duration_ms: 目标时长（毫秒）
            voice: 音色名称（可选）
            format: 音频格式（可选）
            tolerance: 时长容差（0.1 表示 ±10%）

        Returns:
            音频数据（bytes）

        Note:
            当前 DashScope TTS 不直接支持时长控制
            这是一个占位实现，实际需要通过语速调整或后处理实现
        """
        # TODO: 实现时长控制
        # 可能的方案：
        # 1. 使用 qwen3-tts-vc-realtime 的实时模式
        # 2. 通过 ffmpeg 后处理调整语速
        # 3. 使用 librosa 等库进行音频拉伸

        logger.warning(
            f"Duration control not fully implemented. "
            f"Target: {target_duration_ms}ms, tolerance: {tolerance}"
        )

        # 暂时使用基础合成
        return self.synthesize(text, voice, format)

    def synthesize_batch(
        self,
        texts: list[str],
        voice: Optional[str] = None,
        format: Optional[str] = None,
    ) -> list[bytes]:
        """
        批量合成

        Args:
            texts: 待合成文本列表
            voice: 音色名称（可选）
            format: 音频格式（可选）

        Returns:
            音频数据列表
        """
        logger.info(f"Batch synthesizing {len(texts)} texts")

        results = []
        for i, text in enumerate(texts):
            try:
                audio = self.synthesize(text, voice, format)
                results.append(audio)
                logger.debug(f"Synthesized {i+1}/{len(texts)}")
            except Exception as e:
                logger.error(f"Failed to synthesize text {i+1}: {e}")
                # 失败时返回空音频
                results.append(b"")

        return results

    def list_voices(self) -> list[str]:
        """
        获取可用音色列表

        Returns:
            音色名称列表（系统音色）或空列表（复刻音色需要先注册）

        Note:
            - cosyvoice-v1: 返回预定义的系统音色列表
            - qwen3-tts-vc-realtime: 返回空列表（需要调用 enroll_voice）
        """
        if self.model == self.MODEL_COSYVOICE:
            # 预定义的系统音色（cosyvoice-v1）
            return [
                "longxiaochun",  # 龙小春（女声）
                "longyunshu",  # 龙韵书（女声）
                "longhua",  # 龙华（男声）
                "longsiqian",  # 龙思谦（男声）
                "longwan",  # 龙婉（女声）
                "longxiaoxia",  # 龙小夏（女声）
                "longjing",  # 龙靖（男声）
                "longmengchi",  # 龙梦驰（男声）
                "longshushu",  # 龙姝姝（女声）
            ]
        else:
            # 声音复刻模型没有预置音色
            logger.info(
                f"Model {self.model} uses voice cloning. "
                "Call enroll_voice() to create custom voices."
            )
            return []


# 全局单例
_tts_client: Optional[TTSClient] = None


def get_tts_client() -> TTSClient:
    """获取 TTS 客户端单例"""
    global _tts_client
    if _tts_client is None:
        _tts_client = TTSClient()
    return _tts_client
