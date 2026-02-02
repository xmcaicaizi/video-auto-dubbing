"""
FastAPI 应用入口
"""

from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.config import settings
from app.database import close_db, init_db


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    # 启动时
    print("🚀 Starting Video Dubbing API...")
    if settings.debug:
        await init_db()
        print("✅ Database initialized")

    yield

    # 关闭时
    print("🛑 Shutting down...")
    await close_db()
    print("✅ Database connections closed")


# 创建 FastAPI 应用
app = FastAPI(
    title=settings.app_name,
    version=settings.app_version,
    description="视频自动配音系统 - 基于阿里百炼平台",
    docs_url=f"{settings.api_prefix}/docs",
    redoc_url=f"{settings.api_prefix}/redoc",
    openapi_url=f"{settings.api_prefix}/openapi.json",
    lifespan=lifespan,
)

# CORS 中间件
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=settings.cors_allow_credentials,
    allow_methods=settings.cors_allow_methods,
    allow_headers=settings.cors_allow_headers,
)


# ==================== 路由注册 ====================
from app.api import tasks_router
from app.api.monitoring import router as monitoring_router

app.include_router(tasks_router, prefix=settings.api_prefix)
app.include_router(monitoring_router, prefix=settings.api_prefix)


# ==================== 基础路由 ====================
@app.get("/")
async def root():
    """根路径"""
    return {
        "message": "Video Dubbing API",
        "version": settings.app_version,
        "docs": f"{settings.api_prefix}/docs",
    }


@app.get("/health")
async def health_check():
    """健康检查"""
    return JSONResponse(
        content={
            "status": "healthy",
            "version": settings.app_version,
        }
    )


@app.get(f"{settings.api_prefix}/")
async def api_root():
    """API 根路径"""
    return {
        "message": "Video Dubbing API v2",
        "endpoints": {
            "docs": f"{settings.api_prefix}/docs",
            "health": "/health",
            "tasks": f"{settings.api_prefix}/tasks",
        },
    }


# ==================== 错误处理 ====================
@app.exception_handler(Exception)
async def global_exception_handler(request, exc):
    """全局异常处理"""
    return JSONResponse(
        status_code=500,
        content={
            "error": "Internal Server Error",
            "detail": str(exc) if settings.debug else "An error occurred",
        },
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=8000,
        reload=settings.debug,
        log_level=settings.log_level.lower(),
    )
