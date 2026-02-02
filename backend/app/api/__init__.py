"""
API 路由模块
"""

from .deps import get_db, get_task_service, get_storage_service
from .tasks import router as tasks_router

__all__ = [
    "get_db",
    "get_task_service",
    "get_storage_service",
    "tasks_router",
]
