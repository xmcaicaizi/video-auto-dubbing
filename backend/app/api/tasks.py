"""
任务管理 API
"""

import math
from typing import Optional
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, UploadFile, File, Form, status
from loguru import logger

from app.models import TaskStatus
from app.schemas import (
    TaskCreate,
    TaskResponse,
    TaskDetail,
    TaskListResponse,
)
from app.services import TaskService, StorageService
from .deps import get_task_service, get_storage_service

router = APIRouter(prefix="/tasks", tags=["tasks"])


@router.post("", response_model=TaskResponse, status_code=status.HTTP_201_CREATED)
async def create_task(
    video: UploadFile = File(..., description="视频文件"),
    source_language: str = Form(..., description="源语言代码，如 zh, en"),
    target_language: str = Form(..., description="目标语言代码"),
    title: Optional[str] = Form(None, description="任务标题"),
    subtitle_mode: str = Form("burn", description="字幕模式: none/external/burn"),
    task_service: TaskService = Depends(get_task_service),
    storage_service: StorageService = Depends(get_storage_service),
):
    """
    创建配音任务

    - **video**: 视频文件（必需）
    - **source_language**: 源语言代码（必需）
    - **target_language**: 目标语言代码（必需）
    - **title**: 任务标题（可选）
    - **subtitle_mode**: 字幕模式（可选，默认 burn）
        - burn: 将字幕烧录到视频中（推荐，默认）
        - external: 生成外挂字幕文件
        - none: 不生成字幕
    """
    # 验证文件
    if not video.filename:
        raise HTTPException(status_code=400, detail="Invalid video file")

    # 验证语言代码
    valid_languages = {"zh", "en", "ja", "ko", "es", "fr", "de", "ru"}
    if source_language not in valid_languages:
        raise HTTPException(status_code=400, detail=f"Invalid source_language: {source_language}")
    if target_language not in valid_languages:
        raise HTTPException(status_code=400, detail=f"Invalid target_language: {target_language}")

    # 验证字幕模式
    from app.models import SubtitleMode
    try:
        # 将前端传来的小写值转为大写以匹配枚举
        subtitle_mode_enum = SubtitleMode(subtitle_mode.upper())
    except ValueError:
        raise HTTPException(
            status_code=400,
            detail=f"Invalid subtitle_mode: {subtitle_mode}. Must be one of: none, external, burn"
        )

    try:
        # 创建任务记录
        task_data = TaskCreate(
            title=title or video.filename,
            source_language=source_language,
            target_language=target_language,
            subtitle_mode=subtitle_mode_enum,
        )
        task = await task_service.create_task(task_data)

        # 上传视频到 OSS
        video_path = storage_service.upload_input_video(
            task.id, video.file, video.filename
        )

        # 更新任务视频路径
        task.input_video_path = video_path
        await task_service.db.commit()
        await task_service.db.refresh(task)

        # 提交 Celery 任务
        from app.workers.tasks import process_video_pipeline

        celery_task = process_video_pipeline.delay(str(task.id))
        task.celery_task_id = celery_task.id
        await task_service.db.commit()

        logger.info(
            f"Task created and queued: id={task.id}, celery_task_id={celery_task.id}"
        )

        return TaskResponse.model_validate(task)

    except Exception as e:
        logger.error(f"Failed to create task: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to create task: {str(e)}")


@router.get("", response_model=TaskListResponse)
async def list_tasks(
    page: int = 1,
    page_size: int = 20,
    status: Optional[TaskStatus] = None,
    task_service: TaskService = Depends(get_task_service),
):
    """
    获取任务列表

    - **page**: 页码（从 1 开始）
    - **page_size**: 每页数量
    - **status**: 状态过滤（可选）
    """
    if page < 1:
        raise HTTPException(status_code=400, detail="Page must be >= 1")
    if page_size < 1 or page_size > 100:
        raise HTTPException(status_code=400, detail="Page size must be between 1 and 100")

    tasks, total = await task_service.list_tasks(page, page_size, status)

    total_pages = math.ceil(total / page_size) if total > 0 else 0

    return TaskListResponse(
        items=[TaskResponse.model_validate(task) for task in tasks],
        total=total,
        page=page,
        page_size=page_size,
        total_pages=total_pages,
    )


@router.get("/{task_id}", response_model=TaskDetail)
async def get_task(
    task_id: UUID,
    task_service: TaskService = Depends(get_task_service),
):
    """
    获取任务详情

    - **task_id**: 任务 ID
    """
    task = await task_service.get_task(task_id, with_segments=True)

    if not task:
        raise HTTPException(status_code=404, detail="Task not found")

    return TaskDetail.model_validate(task)


@router.delete("/{task_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_task(
    task_id: UUID,
    task_service: TaskService = Depends(get_task_service),
    storage_service: StorageService = Depends(get_storage_service),
):
    """
    删除任务

    - **task_id**: 任务 ID
    """
    # 检查任务是否存在
    task = await task_service.get_task(task_id)
    if not task:
        raise HTTPException(status_code=404, detail="Task not found")

    # 删除 OSS 文件
    try:
        storage_service.delete_task_files(task_id)
    except Exception as e:
        logger.warning(f"Failed to delete OSS files for task {task_id}: {e}")

    # 删除数据库记录
    await task_service.delete_task(task_id)

    logger.info(f"Task deleted: id={task_id}")

    return None


@router.get("/{task_id}/result")
async def get_task_result(
    task_id: UUID,
    task_service: TaskService = Depends(get_task_service),
    storage_service: StorageService = Depends(get_storage_service),
):
    """
    获取任务结果下载链接

    - **task_id**: 任务 ID

    Returns:
        {
            "download_url": "https://...",
            "expires_in": 3600
        }
    """
    task = await task_service.get_task(task_id)

    if not task:
        raise HTTPException(status_code=404, detail="Task not found")

    if task.status != TaskStatus.COMPLETED:
        raise HTTPException(
            status_code=400,
            detail=f"Task not completed yet. Current status: {task.status.value}",
        )

    if not task.output_video_path:
        raise HTTPException(status_code=404, detail="Output video not found")

    # 生成下载链接（1 小时有效）
    download_url = storage_service.get_download_url(task.output_video_path, expires=3600)

    # 包含字幕下载链接（如果有）
    result = {"download_url": download_url, "expires_in": 3600}

    if task.subtitle_file_path:
        subtitle_url = storage_service.get_download_url(task.subtitle_file_path, expires=3600)
        result["subtitle_url"] = subtitle_url

    return result


@router.get("/{task_id}/subtitle")
async def get_task_subtitle(
    task_id: UUID,
    task_service: TaskService = Depends(get_task_service),
    storage_service: StorageService = Depends(get_storage_service),
):
    """
    获取字幕文件下载链接

    - **task_id**: 任务 ID

    Returns:
        {
            "subtitle_url": "https://...",
            "expires_in": 3600
        }
    """
    task = await task_service.get_task(task_id)

    if not task:
        raise HTTPException(status_code=404, detail="Task not found")

    if task.status != TaskStatus.COMPLETED:
        raise HTTPException(
            status_code=400,
            detail=f"Task not completed yet. Current status: {task.status.value}",
        )

    if not task.subtitle_file_path:
        raise HTTPException(status_code=404, detail="No subtitle file for this task")

    subtitle_url = storage_service.get_download_url(task.subtitle_file_path, expires=3600)

    return {"subtitle_url": subtitle_url, "expires_in": 3600}
