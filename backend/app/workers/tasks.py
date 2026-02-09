"""
Celery 任务定义
视频配音处理流程
"""

import asyncio
import tempfile
from pathlib import Path
from typing import Optional
from uuid import UUID

from celery import chain, group
from loguru import logger

from app.config import settings
from app.database import get_db_context
from app.integrations.dashscope import ASRClient, LLMClient, TTSClient
from app.integrations.oss import OSSClient
from app.models import TaskStatus, SubtitleMode
from app.services import TaskService, StorageService, TranslationChunker
from app.utils.ffmpeg import FFmpegHelper
from .celery_app import celery_app


def _run_async(coro):
    """在 Celery worker 进程内复用事件循环执行异步协程。"""
    loop = getattr(_run_async, "_loop", None)
    if loop is None or loop.is_closed():
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        _run_async._loop = loop
    return loop.run_until_complete(coro)


# ==================== 任务链主入口 ====================


@celery_app.task(name="process_video_pipeline", bind=True)
def process_video_pipeline(self, task_id: str):
    """
    视频配音处理主流程

    流程:
    1. extract_audio - 提取音频
    2. transcribe_audio - 语音识别
    3. translate_segments - 翻译文本
    4. synthesize_audio - 语音合成（并行）
    5. mux_video - 合成最终视频

    Args:
        task_id: 任务 ID（字符串格式）
    """
    logger.info(f"Starting video processing pipeline: task_id={task_id}")

    try:
        # 构建任务链
        pipeline = chain(
            extract_audio_task.s(task_id),
            transcribe_audio_task.s(task_id),
            translate_segments_task.s(task_id),
            synthesize_audio_task.s(task_id),
            mux_video_task.s(task_id),
        )

        # 执行任务链
        result = pipeline.apply_async()

        logger.info(f"Pipeline started: task_id={task_id}, chain_id={result.id}")

        return {"task_id": task_id, "chain_id": result.id, "status": "started"}

    except Exception as e:
        logger.error(f"Failed to start pipeline: task_id={task_id}, error={e}")
        # 标记任务失败
        _update_task_status(task_id, TaskStatus.FAILED, error_message=str(e))
        raise


# ==================== Step 1: 提取音频 ====================


@celery_app.task(name="extract_audio", bind=True)
def extract_audio_task(self, task_id: str):
    """
    从视频中提取音频

    Args:
        task_id: 任务 ID

    Returns:
        task_id（用于链式传递）
    """
    logger.info(f"[Step 1] Extracting audio: task_id={task_id}")

    try:
        # 更新任务状态
        _update_task_status(
            task_id, TaskStatus.EXTRACTING, current_step="extract_audio", progress=10
        )

        async def _extract():
            async with get_db_context() as db:
                task_service = TaskService(db)
                storage_service = StorageService()

                # 获取任务
                task = await task_service.get_task(UUID(task_id))
                if not task or not task.input_video_path:
                    raise ValueError(f"Task {task_id} not found or missing input video")

                # 下载视频到临时目录
                temp_dir = tempfile.mkdtemp(prefix=f"task_{task_id}_")
                local_video = storage_service.download_file(
                    task.input_video_path, temp_dir
                )

                logger.info(f"Downloaded video: {local_video}")

                # 提取音频
                ffmpeg = FFmpegHelper()
                audio_file = ffmpeg.extract_audio(
                    local_video, output_path=f"{temp_dir}/audio.wav"
                )

                logger.info(f"Extracted audio: {audio_file}")

                # 获取视频时长
                duration_ms = ffmpeg.get_duration_ms(local_video)
                task.video_duration_ms = duration_ms

                # 上传音频到 OSS
                audio_path = storage_service.upload_extracted_audio(
                    UUID(task_id), audio_file
                )
                task.extracted_audio_path = audio_path

                await db.commit()
                await db.refresh(task)

                logger.info(
                    f"Audio extracted and uploaded: duration={duration_ms}ms, path={audio_path}"
                )

                # 清理临时文件
                import shutil
                shutil.rmtree(temp_dir)

        _run_async(_extract())

        # 更新进度
        _update_task_status(task_id, TaskStatus.EXTRACTING, progress=20)

        return task_id

    except Exception as e:
        logger.error(f"Audio extraction failed: task_id={task_id}, error={e}")
        _update_task_status(task_id, TaskStatus.FAILED, error_message=str(e))
        raise


# ==================== Step 2: 语音识别 ====================


@celery_app.task(name="transcribe_audio", bind=True)
def transcribe_audio_task(self, previous_result, task_id: str):
    """
    语音识别（ASR）

    Args:
        previous_result: 上一步结果（task_id）
        task_id: 任务 ID

    Returns:
        task_id
    """
    logger.info(f"[Step 2] Transcribing audio: task_id={task_id}")

    try:
        # 更新任务状态
        _update_task_status(
            task_id, TaskStatus.TRANSCRIBING, current_step="transcribe", progress=30
        )

        async def _transcribe():
            async with get_db_context() as db:
                task_service = TaskService(db)
                storage_service = StorageService()

                # 获取任务
                task = await task_service.get_task(UUID(task_id))
                if not task or not task.extracted_audio_path:
                    raise ValueError(f"Task {task_id} missing extracted audio")

                # 下载音频
                temp_dir = tempfile.mkdtemp(prefix=f"task_{task_id}_asr_")
                local_audio = storage_service.download_file(
                    task.extracted_audio_path, temp_dir
                )

                logger.info(f"Downloaded audio for ASR: {local_audio}")

                # 语音识别
                asr_client = ASRClient()
                audio_url = storage_service.get_download_url(
                    task.extracted_audio_path, expires=3600
                )
                result = asr_client.transcribe(audio_url)

                logger.info(
                    f"ASR completed: {len(result.segments)} segments, "
                    f"duration={result.duration_ms}ms"
                )

                # 创建分段
                for i, segment in enumerate(result.segments):
                    await task_service.create_segment(
                        task_id=UUID(task_id),
                        segment_index=i,
                        start_time_ms=segment.start_time_ms,
                        end_time_ms=segment.end_time_ms,
                        original_text=segment.text,
                        speaker_id=getattr(segment, "speaker_id", None),
                        confidence=getattr(segment, "confidence", None),
                        emotion=getattr(segment, "emotion", None),
                    )

                # 更新分段数量
                task.segment_count = len(result.segments)
                await db.commit()

                logger.info(f"Created {len(result.segments)} segments")

                # 清理临时文件
                import shutil
                shutil.rmtree(temp_dir)

        _run_async(_transcribe())

        # 更新进度
        _update_task_status(task_id, TaskStatus.TRANSCRIBING, progress=40)

        return task_id

    except Exception as e:
        logger.error(f"Transcription failed: task_id={task_id}, error={e}")
        _update_task_status(task_id, TaskStatus.FAILED, error_message=str(e))
        raise


# ==================== Step 3: 翻译 ====================


@celery_app.task(name="translate_segments", bind=True)
def translate_segments_task(self, previous_result, task_id: str):
    """
    翻译所有分段

    Args:
        previous_result: 上一步结果（task_id）
        task_id: 任务 ID

    Returns:
        task_id
    """
    logger.info(f"[Step 3] Translating segments: task_id={task_id}")

    try:
        # 更新任务状态
        _update_task_status(
            task_id, TaskStatus.TRANSLATING, current_step="translate", progress=50
        )

        async def _translate():
            async with get_db_context() as db:
                task_service = TaskService(db)

                # 获取任务和分段
                task = await task_service.get_task(UUID(task_id), with_segments=True)
                if not task:
                    raise ValueError(f"Task {task_id} not found")

                segments = task.segments
                logger.info(f"Translating {len(segments)} segments using chunked translation")

                # 翻译客户端
                llm_client = LLMClient()

                try:
                    # ========== 分块翻译（使用 TranslationChunker） ==========
                    logger.info("Starting chunked translation with overlap context")

                    # Step 1: 智能分块
                    chunks = TranslationChunker.chunk_segments(segments)
                    logger.info(
                        f"Segmentation complete: {len(segments)} segments -> {len(chunks)} chunks, "
                        f"max_chars={TranslationChunker.MAX_CHARS_PER_CHUNK}, "
                        f"overlap={TranslationChunker.OVERLAP_SEGMENTS}"
                    )

                    # 存储所有翻译结果（使用 segment_index 作为 key）
                    all_translations = {}

                    # Step 2: 逐块翻译
                    for chunk_idx, chunk in enumerate(chunks, start=1):
                        chunk_size = len(chunk)
                        chunk_indices = [seg.segment_index for seg in chunk]

                        logger.info(
                            f"Processing chunk {chunk_idx}/{len(chunks)}: "
                            f"{chunk_size} segments, indices={chunk_indices}"
                        )

                        # 构建输入文本（带 segment_index 标记）
                        chunk_text = TranslationChunker.build_chunk_text(chunk)

                        logger.debug(
                            f"Chunk {chunk_idx} input ({len(chunk_text)} chars): "
                            f"{chunk_text[:100]}..."
                        )

                        # 调用 LLM 翻译
                        translated_chunk = llm_client.translate(
                            text=chunk_text,
                            source_lang=task.source_language,
                            target_lang=task.target_language,
                        )

                        logger.debug(
                            f"Chunk {chunk_idx} output: {translated_chunk[:100]}..."
                        )

                        # 解析翻译结果
                        chunk_translations = TranslationChunker.parse_translation_result(
                            translated_chunk
                        )

                        logger.info(
                            f"Chunk {chunk_idx} parsed: {len(chunk_translations)} translations"
                        )

                        # 合并到总结果（处理重叠：使用 update 覆盖之前的版本）
                        overlap_count = len(set(chunk_translations.keys()) & set(all_translations.keys()))
                        if overlap_count > 0:
                            logger.debug(
                                f"Chunk {chunk_idx} has {overlap_count} overlapping segments, "
                                "updating with newer translations"
                            )

                        all_translations.update(chunk_translations)

                    # Step 3: 更新所有分段的翻译
                    logger.info(f"Updating {len(all_translations)} segment translations in database")

                    for segment in segments:
                        if not segment.original_text:
                            continue

                        # 使用 segment.segment_index（不是 enumerate 的 i）
                        translated = all_translations.get(
                            segment.segment_index,
                            segment.original_text  # 降级：未翻译则保留原文
                        )

                        await task_service.update_segment_translation(
                            segment.id, translated
                        )

                        logger.debug(
                            f"Segment {segment.segment_index}: "
                            f"{segment.original_text[:30]} -> {translated[:30]}"
                        )

                    logger.info(
                        f"Chunked translation completed successfully: "
                        f"{len(segments)} segments processed via {len(chunks)} chunks"
                    )

                except Exception as e:
                    logger.error(
                        f"Chunked translation failed: {e}, falling back to segment-by-segment translation"
                    )

                    # 降级：逐段翻译
                    for segment in segments:
                        if not segment.original_text:
                            continue

                        try:
                            translated = llm_client.translate(
                                text=segment.original_text,
                                source_lang=task.source_language,
                                target_lang=task.target_language,
                            )
                            await task_service.update_segment_translation(segment.id, translated)

                            logger.debug(
                                f"Fallback: Segment {segment.segment_index} translated individually"
                            )

                        except Exception as seg_err:
                            logger.error(
                                f"Segment {segment.segment_index} translation failed: {seg_err}, "
                                "using original text"
                            )
                            await task_service.update_segment_translation(
                                segment.id, segment.original_text
                            )

        _run_async(_translate())

        # 更新进度
        _update_task_status(task_id, TaskStatus.TRANSLATING, progress=60)

        return task_id

    except Exception as e:
        logger.error(f"Translation failed: task_id={task_id}, error={e}")
        _update_task_status(task_id, TaskStatus.FAILED, error_message=str(e))
        raise


# ==================== Step 4: 语音合成 ====================


@celery_app.task(name="synthesize_audio", bind=True)
def synthesize_audio_task(self, previous_result, task_id: str):
    """
    为每个分段合成音频（支持多说话人声音复刻）

    流程:
    1. 按 speaker_id 分组分段
    2. 为每个 speaker 复刻声音（获得 voice_id）
    3. 使用对应的 voice_id 合成每个分段的音频

    Args:
        previous_result: 上一步结果（task_id）
        task_id: 任务 ID

    Returns:
        task_id
    """
    logger.info(f"[Step 4] Synthesizing audio: task_id={task_id}")

    try:
        # 更新任务状态
        _update_task_status(
            task_id, TaskStatus.SYNTHESIZING, current_step="synthesize", progress=70
        )

        async def _synthesize():
            async with get_db_context() as db:
                task_service = TaskService(db)
                storage_service = StorageService()

                # 获取任务和分段
                task = await task_service.get_task(UUID(task_id), with_segments=True)
                if not task:
                    raise ValueError(f"Task {task_id} not found")

                if not task.extracted_audio_path:
                    raise ValueError(f"Task {task_id} missing extracted audio")

                segments = task.segments
                logger.info(f"Synthesizing {len(segments)} segments")

                # 下载原始音频（用于声音复刻）
                temp_dir = tempfile.mkdtemp(prefix=f"task_{task_id}_tts_")
                local_audio = storage_service.download_file(
                    task.extracted_audio_path, temp_dir
                )

                # TTS 客户端
                tts_client = TTSClient()

                # 检查是否使用声音复刻模型
                from app.config import settings

                use_voice_cloning = settings.tts_model in tts_client.VOICE_CLONE_MODELS

                # voice_id 缓存（speaker_id -> voice_id）
                voice_cache = {}

                if use_voice_cloning:
                    logger.info("Using voice cloning model, enrolling speakers...")

                    # 按说话人分组
                    from collections import defaultdict

                    segments_by_speaker = defaultdict(list)
                    for seg in segments:
                        speaker_id = seg.speaker_id or "default"
                        segments_by_speaker[speaker_id].append(
                            {
                                "start_time_ms": seg.start_time_ms,
                                "end_time_ms": seg.end_time_ms,
                            }
                        )

                    logger.info(
                        f"Found {len(segments_by_speaker)} speakers: "
                        f"{list(segments_by_speaker.keys())}"
                    )

                    # 为每个说话人复刻声音
                    from app.services.voice_service import VoiceService

                    voice_service = VoiceService()

                    for speaker_id, speaker_segments in segments_by_speaker.items():
                        voice_id = voice_service.get_or_create_voice_id(
                            task_id=UUID(task_id),
                            speaker_id=speaker_id,
                            audio_path=local_audio,
                            segments=speaker_segments,
                            cache=voice_cache,
                        )

                        if not voice_id:
                            logger.error(
                                f"Failed to enroll speaker {speaker_id}, "
                                "using default voice"
                            )

                    logger.info(f"Voice enrollment completed: {voice_cache}")

                # 为每个分段合成音频
                for i, segment in enumerate(segments):
                    if not segment.translated_text:
                        logger.warning(f"Segment {segment.id} has no translated text")
                        continue

                    try:
                        # 确定使用的 voice
                        if use_voice_cloning:
                            speaker_id = segment.speaker_id or "default"
                            voice_id = voice_cache.get(speaker_id)

                            if not voice_id:
                                logger.warning(
                                    f"No voice_id for speaker {speaker_id}, "
                                    f"falling back to system voice for segment {i+1}"
                                )
                                # 降级：使用系统音色（必须指定voice）
                                fallback_tts = TTSClient(
                                    model="cosyvoice-v1",
                                    voice="longxiaochun"  # 系统默认音色
                                )
                                audio_data = fallback_tts.synthesize(segment.translated_text)
                            else:
                                # 保存 voice_id 到分段
                                segment.voice_id = voice_id
                                await db.commit()

                                # 合成音频
                                audio_data = tts_client.synthesize(
                                    segment.translated_text, voice=voice_id
                                )
                        else:
                            # 使用系统音色
                            audio_data = tts_client.synthesize(segment.translated_text)

                        # 上传到 OSS
                        audio_path = storage_service.upload_segment_audio(
                            UUID(task_id), segment.segment_index, audio_data
                        )

                        # 更新分段
                        await task_service.update_segment_audio(segment.id, audio_path)

                        logger.debug(
                            f"Synthesized segment {i+1}/{len(segments)}: "
                            f"{len(audio_data)} bytes -> {audio_path}"
                        )

                    except Exception as e:
                        logger.error(f"Failed to synthesize segment {segment.id}: {e}")
                        # 继续处理下一个分段

                logger.info(f"Synthesis completed: {len(segments)} segments")

                # 清理临时文件
                import shutil

                shutil.rmtree(temp_dir)

        _run_async(_synthesize())

        # 更新进度
        _update_task_status(task_id, TaskStatus.SYNTHESIZING, progress=80)

        return task_id

    except Exception as e:
        logger.error(f"Synthesis failed: task_id={task_id}, error={e}")
        _update_task_status(task_id, TaskStatus.FAILED, error_message=str(e))
        raise


# ==================== Step 5: 视频合成 ====================


@celery_app.task(name="mux_video", bind=True)
def mux_video_task(self, previous_result, task_id: str):
    """
    合成最终视频

    Args:
        previous_result: 上一步结果（task_id）
        task_id: 任务 ID

    Returns:
        task_id
    """
    logger.info(f"[Step 5] Muxing video: task_id={task_id}")

    try:
        # 更新任务状态
        _update_task_status(
            task_id, TaskStatus.MUXING, current_step="mux", progress=90
        )

        async def _mux():
            async with get_db_context() as db:
                task_service = TaskService(db)
                storage_service = StorageService()

                # 获取任务和分段
                task = await task_service.get_task(UUID(task_id), with_segments=True)
                if not task:
                    raise ValueError(f"Task {task_id} not found")

                # 创建临时目录
                temp_dir = tempfile.mkdtemp(prefix=f"task_{task_id}_mux_")

                # 下载原视频
                local_video = storage_service.download_file(
                    task.input_video_path, temp_dir
                )

                # 下载所有音频分段
                audio_files = []
                for segment in sorted(task.segments, key=lambda s: s.segment_index):
                    if segment.audio_path:
                        local_audio = storage_service.download_file(
                            segment.audio_path, temp_dir
                        )
                        audio_files.append(
                            {
                                "path": local_audio,
                                "start_ms": segment.start_time_ms,
                                "end_ms": segment.end_time_ms,
                            }
                        )

                logger.info(f"Downloaded {len(audio_files)} audio segments")

                # 合成音频
                ffmpeg = FFmpegHelper()
                merged_audio = ffmpeg.merge_audio_segments(
                    audio_files,
                    output_path=f"{temp_dir}/merged_audio.mp3",
                    total_duration_ms=task.video_duration_ms,
                )

                # ========== 字幕生成 ==========
                subtitle_mode = task.subtitle_mode or SubtitleMode.BURN
                subtitle_path = None

                if subtitle_mode != SubtitleMode.NONE:
                    # 准备分段数据
                    sorted_segments = sorted(task.segments, key=lambda s: s.segment_index)
                    subtitle_segments = [
                        {
                            "start_time_ms": seg.start_time_ms,
                            "end_time_ms": seg.end_time_ms,
                            "original_text": seg.original_text or "",
                            "translated_text": seg.translated_text or "",
                        }
                        for seg in sorted_segments
                        if seg.original_text or seg.translated_text
                    ]

                    if subtitle_segments:
                        # 获取视频分辨率用于字幕布局
                        video_width, video_height = ffmpeg.get_video_resolution(local_video)

                        # 生成 ASS 字幕文件
                        local_subtitle = ffmpeg.generate_ass_subtitle(
                            segments=subtitle_segments,
                            output_path=f"{temp_dir}/subtitle.ass",
                            subtitle_type="bilingual",
                            video_width=video_width,
                            video_height=video_height,
                        )
                        subtitle_path = local_subtitle

                        # 上传字幕文件到 OSS（无论是否烧录都上传，用于下载）
                        oss_subtitle_path = storage_service.upload_subtitle_file(
                            UUID(task_id), local_subtitle
                        )
                        task.subtitle_file_path = oss_subtitle_path

                        logger.info(f"Subtitle file uploaded: {oss_subtitle_path}")

                # ========== 视频合成 ==========
                if subtitle_mode == SubtitleMode.BURN and subtitle_path:
                    # 烧录模式：替换音轨 + 烧录字幕（单次 FFmpeg 调用）
                    output_video = ffmpeg.replace_audio_and_burn_subtitles(
                        video_path=local_video,
                        audio_path=merged_audio,
                        subtitle_path=subtitle_path,
                        output_path=f"{temp_dir}/output.mp4",
                    )
                    logger.info(f"Video muxed with burned subtitles: {output_video}")
                else:
                    # 外挂模式 / 无字幕：仅替换音轨（视频流 copy，速度快）
                    output_video = ffmpeg.replace_audio(
                        video_path=local_video,
                        audio_path=merged_audio,
                        output_path=f"{temp_dir}/output.mp4",
                    )
                    logger.info(f"Video muxed: {output_video}")

                # 上传最终视频
                video_path = storage_service.upload_output_video(
                    UUID(task_id), output_video
                )

                task.output_video_path = video_path
                await db.commit()

                logger.info(f"Output video uploaded: {video_path}")

                # 清理临时文件
                import shutil
                shutil.rmtree(temp_dir)

        _run_async(_mux())

        # 标记任务完成
        _update_task_status(
            task_id, TaskStatus.COMPLETED, current_step="completed", progress=100
        )

        logger.info(f"Video processing completed: task_id={task_id}")

        return task_id

    except Exception as e:
        logger.error(f"Video muxing failed: task_id={task_id}, error={e}")
        _update_task_status(task_id, TaskStatus.FAILED, error_message=str(e))
        raise


# ==================== 辅助函数 ====================


def _update_task_status(
    task_id: str,
    status: TaskStatus,
    current_step: Optional[str] = None,
    progress: Optional[int] = None,
    error_message: Optional[str] = None,
):
    """
    更新任务状态（同步包装）

    Args:
        task_id: 任务 ID
        status: 新状态
        current_step: 当前步骤
        progress: 进度
        error_message: 错误信息
    """

    async def _update():
        async with get_db_context() as db:
            task_service = TaskService(db)
            await task_service.update_task_status(
                UUID(task_id), status, current_step, progress, error_message
            )

    _run_async(_update())
