"""
Celery 任务定义
视频配音处理流程
"""

import tempfile
from pathlib import Path
from typing import Optional
from uuid import UUID

from celery import chain, group
from loguru import logger

from app.config import settings
from app.database import get_db_context
from app.integrations.dashscope import get_asr_client, get_llm_client, get_tts_client
from app.integrations.oss import get_oss_client
from app.models import TaskStatus
from app.services import TaskService, StorageService
from app.utils.ffmpeg import FFmpegHelper
from .celery_app import celery_app


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

        import asyncio

        asyncio.run(_extract())

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
                asr_client = get_asr_client()
                result = asr_client.transcribe(
                    local_audio, language_hints=[task.source_language]
                )

                logger.info(
                    f"ASR completed: {len(result.sentences)} sentences, "
                    f"{len(result.words)} words"
                )

                # 创建分段
                for i, sentence in enumerate(result.sentences):
                    await task_service.create_segment(
                        task_id=UUID(task_id),
                        segment_index=i,
                        start_time_ms=sentence.start_time_ms,
                        end_time_ms=sentence.end_time_ms,
                        original_text=sentence.text,
                        speaker_id=getattr(sentence, "speaker_id", None),
                        confidence=getattr(sentence, "confidence", None),
                    )

                # 更新分段数量
                task.segment_count = len(result.sentences)
                await db.commit()

                logger.info(f"Created {len(result.sentences)} segments")

                # 清理临时文件
                import shutil
                shutil.rmtree(temp_dir)

        import asyncio

        asyncio.run(_transcribe())

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
                logger.info(f"Translating {len(segments)} segments")

                # 翻译客户端
                llm_client = get_llm_client()

                # 批量翻译
                for i, segment in enumerate(segments):
                    if not segment.original_text:
                        continue

                    try:
                        translated = llm_client.translate(
                            text=segment.original_text,
                            source_lang=task.source_language,
                            target_lang=task.target_language,
                        )

                        await task_service.update_segment_translation(
                            segment.id, translated
                        )

                        logger.debug(
                            f"Translated segment {i+1}/{len(segments)}: "
                            f"{segment.original_text[:30]} -> {translated[:30]}"
                        )

                    except Exception as e:
                        logger.error(f"Failed to translate segment {segment.id}: {e}")
                        # 翻译失败时保留原文
                        await task_service.update_segment_translation(
                            segment.id, segment.original_text
                        )

                logger.info(f"Translation completed: {len(segments)} segments")

        import asyncio

        asyncio.run(_translate())

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
    为每个分段合成音频

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

                segments = task.segments
                logger.info(f"Synthesizing {len(segments)} segments")

                # TTS 客户端
                tts_client = get_tts_client()

                # 为每个分段合成音频
                for i, segment in enumerate(segments):
                    if not segment.translated_text:
                        logger.warning(f"Segment {segment.id} has no translated text")
                        continue

                    try:
                        # 合成音频
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

        import asyncio

        asyncio.run(_synthesize())

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
                    audio_files, output_path=f"{temp_dir}/merged_audio.mp3"
                )

                # 替换视频音轨
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

        import asyncio

        asyncio.run(_mux())

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

    import asyncio

    asyncio.run(_update())
