"""
数据库连接和会话管理
"""

from contextlib import asynccontextmanager
from typing import AsyncGenerator

from sqlalchemy import create_engine
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.orm import DeclarativeBase, Session, sessionmaker

from app.config import settings


# 同步引擎 (用于 Alembic 迁移)
sync_engine = create_engine(
    settings.database_url,
    echo=settings.debug,
    pool_pre_ping=True,
    pool_size=10,
    max_overflow=20,
)

# 同步会话工厂
SessionLocal = sessionmaker(
    autocommit=False,
    autoflush=False,
    bind=sync_engine,
)

# 异步引擎 (用于 FastAPI)
async_database_url = settings.database_url.replace("postgresql://", "postgresql+asyncpg://")
async_engine = create_async_engine(
    async_database_url,
    echo=settings.debug,
    pool_pre_ping=True,
    pool_size=10,
    max_overflow=20,
)

# 异步会话工厂
AsyncSessionLocal = async_sessionmaker(
    async_engine,
    class_=AsyncSession,
    expire_on_commit=False,
)


# Base 模型
class Base(DeclarativeBase):
    """所有模型的基类"""

    pass


# 依赖注入：获取数据库会话
async def get_db() -> AsyncGenerator[AsyncSession, None]:
    """
    FastAPI 依赖项：提供异步数据库会话

    Usage:
        @app.get("/items")
        async def read_items(db: AsyncSession = Depends(get_db)):
            ...
    """
    async with AsyncSessionLocal() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()


def get_sync_db() -> Session:
    """
    获取同步数据库会话 (用于 Celery Worker)

    Usage:
        db = get_sync_db()
        try:
            # 操作数据库
            db.commit()
        finally:
            db.close()
    """
    return SessionLocal()


# 数据库初始化
async def init_db() -> None:
    """创建所有表 (仅开发环境使用，生产环境用 Alembic)"""
    async with async_engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)


async def close_db() -> None:
    """关闭数据库连接"""
    await async_engine.dispose()


@asynccontextmanager
async def get_db_context() -> AsyncGenerator[AsyncSession, None]:
    """
    异步上下文管理器：用于 Celery 任务中获取数据库会话

    Usage:
        async with get_db_context() as db:
            # 操作数据库
            await db.commit()
    """
    async with AsyncSessionLocal() as session:
        try:
            yield session
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()
