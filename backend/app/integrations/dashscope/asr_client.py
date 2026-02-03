"""
DashScope ASR 客户端封装
语音识别服务，支持情感检测和说话人分离
"""

import time
from typing import Optional

import dashscope
from dashscope.audio.asr import Transcription
from loguru import logger

from app.config import settings


class ASRSegment:
    """ASR 识别分段结果"""

    def __init__(
        self,
        text: str,
        start_time_ms: int,
        end_time_ms: int,
        speaker_id: Optional[str] = None,
        emotion: Optional[str] = None,
        confidence: Optional[float] = None,
    ):
        self.text = text
        self.start_time_ms = start_time_ms
        self.end_time_ms = end_time_ms
        self.speaker_id = speaker_id
        self.emotion = emotion
        self.confidence = confidence

    @property
    def duration_ms(self) -> int:
        """分段时长（毫秒）"""
        return self.end_time_ms - self.start_time_ms

    def __repr__(self) -> str:
        return (
            f"ASRSegment(text='{self.text[:20]}...', "
            f"time={self.start_time_ms}-{self.end_time_ms}ms, "
            f"speaker={self.speaker_id}, emotion={self.emotion})"
        )


class ASRResult:
    """ASR 识别结果"""

    def __init__(
        self,
        task_id: str,
        file_url: str,
        segments: list[ASRSegment],
        full_text: str,
        duration_ms: int,
    ):
        self.task_id = task_id
        self.file_url = file_url
        self.segments = segments
        self.full_text = full_text
        self.duration_ms = duration_ms

    def __repr__(self) -> str:
        return (
            f"ASRResult(task_id={self.task_id}, "
            f"segments={len(self.segments)}, "
            f"duration={self.duration_ms}ms)"
        )


class ASRClient:
    """DashScope ASR 客户端"""

    def __init__(
        self,
        api_key: Optional[str] = None,
        model: Optional[str] = None,
        language_hints: Optional[list[str]] = None,
    ):
        """
        初始化 ASR 客户端

        Args:
            api_key: DashScope API Key
            model: 模型名称，默认 sensevoice-v1
            language_hints: 语言提示，如 ['zh', 'en']
        """
        self.api_key = api_key or settings.dashscope_api_key
        self.model = model or settings.asr_model
        self.language_hints = language_hints or settings.asr_language_hints

        if not self.api_key:
            raise ValueError("DASHSCOPE_API_KEY is required")

        # 设置 API Key
        dashscope.api_key = self.api_key

        logger.info(f"ASR Client initialized: model={self.model}, languages={self.language_hints}")

    def transcribe(
        self,
        audio_url: str,
        timeout: int = 300,
        poll_interval: int = 2,
    ) -> ASRResult:
        """
        语音识别（异步模式）

        Args:
            audio_url: 音频文件 URL（公网可访问）
            timeout: 超时时间（秒），默认 5 分钟
            poll_interval: 轮询间隔（秒），默认 2 秒

        Returns:
            ASRResult 识别结果

        Raises:
            TimeoutError: 识别超时
            RuntimeError: 识别失败
        """
        logger.info(f"Submitting ASR task: url={audio_url}, model={self.model}")

        # 提交异步任务
        try:
            response = Transcription.async_call(
                model=self.model,
                file_urls=[audio_url],
                language_hints=self.language_hints,
                enable_speaker_diarization=True,  # 启用说话人分离
                disfluency_removal_enabled=True,  # 启用语气词过滤
            )

            if response.status_code != 200:
                raise RuntimeError(f"ASR task submission failed: {response.message}")

            task_id = response.output.task_id
            logger.info(f"ASR task submitted: task_id={task_id}")

        except Exception as e:
            logger.error(f"Failed to submit ASR task: {e}")
            raise RuntimeError(f"ASR task submission failed: {e}") from e

        # 轮询结果
        start_time = time.time()
        while True:
            elapsed = time.time() - start_time

            if elapsed > timeout:
                raise TimeoutError(f"ASR task timeout after {timeout}s: task_id={task_id}")

            try:
                result = Transcription.fetch(task=task_id)
                status = result.output.task_status

                if status == "SUCCEEDED":
                    logger.info(f"ASR task completed: task_id={task_id}, elapsed={elapsed:.1f}s")
                    return self._parse_result(task_id, audio_url, result)

                elif status == "FAILED":
                    error_msg = result.output.get("error_message", "Unknown error")
                    raise RuntimeError(f"ASR task failed: {error_msg}")

                else:
                    # PENDING or RUNNING
                    logger.debug(
                        f"ASR task status: {status}, elapsed={elapsed:.1f}s, "
                        f"task_id={task_id}"
                    )
                    time.sleep(poll_interval)

            except Exception as e:
                if isinstance(e, (TimeoutError, RuntimeError)):
                    raise
                logger.error(f"Failed to fetch ASR result: {e}")
                raise RuntimeError(f"Failed to fetch ASR result: {e}") from e

    def _parse_result(self, task_id: str, audio_url: str, response) -> ASRResult:
        """
        解析 ASR 识别结果

        Args:
            task_id: 任务 ID
            audio_url: 音频 URL
            response: API 响应

        Returns:
            ASRResult
        """
        try:
            # 获取识别结果 URL
            results = response.output.results
            if not results:
                raise ValueError("No transcription results")

            # 下载结果文件（通常是 JSON URL）
            transcription_data = results[0]

            # 如果有 transcription_url，需要下载
            if "transcription_url" in transcription_data:
                import requests

                transcription_url = transcription_data["transcription_url"]
                logger.info(f"Downloading transcription from: {transcription_url}")

                resp = requests.get(transcription_url, timeout=30)
                resp.raise_for_status()
                transcription_data = resp.json()

            # 解析分段
            segments = self._parse_segments(transcription_data)

            # 提取完整文本
            full_text = " ".join(seg.text for seg in segments)

            # 提取时长
            duration_ms = transcription_data.get("properties", {}).get(
                "original_duration_in_milliseconds", 0
            )

            return ASRResult(
                task_id=task_id,
                file_url=audio_url,
                segments=segments,
                full_text=full_text,
                duration_ms=duration_ms,
            )

        except Exception as e:
            logger.error(f"Failed to parse ASR result: {e}")
            raise RuntimeError(f"Failed to parse ASR result: {e}") from e

    def _parse_segments(self, data: dict) -> list[ASRSegment]:
        """
        解析识别分段

        Args:
            data: 转写结果 JSON

        Returns:
            分段列表
        """
        import re

        segments = []

        # 清理 fun-asr / sensevoice 标签的正则
        tag_pattern = re.compile(r'<\|[^|]+\|>')

        transcripts = data.get("transcripts", [])
        for transcript in transcripts:
            sentences = transcript.get("sentences", [])

            for sentence in sentences:
                raw_text = sentence.get("text", "").strip()

                # 清理标签（<|Speech|>, <|NEUTRAL|> 等）
                text = tag_pattern.sub('', raw_text).strip()

                # 过滤纯标点/空白分段
                if not text or text in ['.', '。', ',', '，', '!', '！', '?', '？']:
                    continue

                start_time = sentence.get("begin_time", 0)
                end_time = sentence.get("end_time", 0)

                # 提取元数据
                speaker_id = sentence.get("speaker_id") or "speaker_0"
                emotion = sentence.get("emotion")
                confidence = sentence.get("confidence")

                segment = ASRSegment(
                    text=text,
                    start_time_ms=start_time,
                    end_time_ms=end_time,
                    speaker_id=speaker_id,
                    emotion=emotion,
                    confidence=confidence,
                )

                segments.append(segment)

        logger.info(f"Parsed {len(segments)} valid segments (filtered out empty/punctuation-only)")
        return segments


# 全局单例
_asr_client: Optional[ASRClient] = None


def get_asr_client() -> ASRClient:
    """获取 ASR 客户端单例"""
    global _asr_client
    if _asr_client is None:
        _asr_client = ASRClient()
    return _asr_client
