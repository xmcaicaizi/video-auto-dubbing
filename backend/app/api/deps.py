"""
API 依赖注入
"""

from typing import AsyncGenerator

from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_db as _get_db
from app.services import TaskService, StorageService


async def get_db() -> AsyncGenerator[AsyncSession, None]:
    """获取数据库会话"""
    async for session in _get_db():
        yield session


async def get_task_service(db: AsyncSession = None) -> TaskService:
    """获取任务服务（依赖注入）"""
    if db is None:
        async for session in get_db():
            yield TaskService(session)
    else:
        return TaskService(db)


def get_storage_service() -> StorageService:
    """获取存储服务（单例）"""
    return StorageService()
