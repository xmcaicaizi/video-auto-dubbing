"""
监控和健康检查 API
"""

from typing import Any

from fastapi import APIRouter
from loguru import logger

from app.config import settings
from app.utils.ffmpeg import FFmpegHelper

router = APIRouter(prefix="/monitoring", tags=["monitoring"])


@router.get("/health")
async def health_check() -> dict[str, Any]:
    """
    健康检查

    Returns:
        {
            "status": "healthy",
            "services": {
                "database": true,
                "redis": true,
                "ffmpeg": true
            }
        }
    """
    services = {}

    # 检查 FFmpeg
    try:
        services["ffmpeg"] = FFmpegHelper.check_ffmpeg()
    except Exception:
        services["ffmpeg"] = False

    # 检查 Redis
    try:
        from app.workers.celery_app import celery_app

        celery_app.backend.ping()
        services["redis"] = True
    except Exception:
        services["redis"] = False

    # 检查数据库
    try:
        from app.database import async_engine
        from sqlalchemy import text

        async with async_engine.connect() as conn:
            await conn.execute(text("SELECT 1"))
        services["database"] = True
    except Exception:
        services["database"] = False

    all_healthy = all(services.values())

    return {
        "status": "healthy" if all_healthy else "unhealthy",
        "services": services,
        "version": settings.app_version,
    }


@router.get("/stats")
async def get_stats() -> dict[str, Any]:
    """
    获取系统统计信息

    Returns:
        {
            "tasks": {
                "total": 100,
                "pending": 10,
                "processing": 5,
                "completed": 80,
                "failed": 5
            },
            "workers": {
                "active": 4,
                "registered": ["worker1", "worker2"]
            }
        }
    """
    stats = {"tasks": {}, "workers": {}}

    # 获取任务统计
    try:
        from app.database import get_db_context
        from app.models import Task, TaskStatus
        from sqlalchemy import func, select

        async with get_db_context() as db:
            # 总数
            total = await db.scalar(select(func.count()).select_from(Task))
            stats["tasks"]["total"] = total or 0

            # 按状态统计
            for status in TaskStatus:
                count = await db.scalar(
                    select(func.count()).select_from(Task).where(Task.status == status)
                )
                stats["tasks"][status.value] = count or 0

    except Exception as e:
        logger.error(f"Failed to get task stats: {e}")
        stats["tasks"]["error"] = str(e)

    # 获取 Worker 信息
    try:
        from app.workers.celery_app import celery_app

        inspect = celery_app.control.inspect()
        active_workers = inspect.active()

        if active_workers:
            stats["workers"]["active"] = len(active_workers)
            stats["workers"]["registered"] = list(active_workers.keys())
        else:
            stats["workers"]["active"] = 0
            stats["workers"]["registered"] = []

    except Exception as e:
        logger.error(f"Failed to get worker stats: {e}")
        stats["workers"]["error"] = str(e)

    return stats


@router.get("/celery/inspect")
async def inspect_celery() -> dict[str, Any]:
    """
    检查 Celery 状态

    Returns:
        {
            "active": {...},
            "scheduled": {...},
            "reserved": {...}
        }
    """
    try:
        from app.workers.celery_app import celery_app

        inspect = celery_app.control.inspect()

        return {
            "active": inspect.active() or {},
            "scheduled": inspect.scheduled() or {},
            "reserved": inspect.reserved() or {},
            "stats": inspect.stats() or {},
        }
    except Exception as e:
        logger.error(f"Failed to inspect Celery: {e}")
        return {"error": str(e)}
