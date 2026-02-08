"""
Celery 应用配置
"""

from celery import Celery

from app.config import settings

# 创建 Celery 应用
celery_app = Celery(
    "video_dubbing",
    broker=settings.get_celery_broker_url,
    backend=settings.get_celery_result_backend,
)

# Celery 配置
celery_app.conf.update(
    # 任务配置
    task_serializer="json",
    accept_content=["json"],
    result_serializer="json",
    timezone="Asia/Shanghai",
    enable_utc=True,
    # 结果过期时间
    result_expires=3600,
    # 任务超时
    task_time_limit=settings.task_timeout,
    task_soft_time_limit=settings.task_timeout - 60,
    # 重试配置
    task_acks_late=True,
    task_reject_on_worker_lost=True,
    # Worker 配置
    worker_prefetch_multiplier=1,
    worker_max_tasks_per_child=100,
)

# 自动发现任务
celery_app.autodiscover_tasks(["workers"])


# 任务路由配置
celery_app.conf.task_routes = {
    "process_video_pipeline": {"queue": "default"},
    "extract_audio": {"queue": "media"},
    "transcribe_audio": {"queue": "ai"},
    "translate_segments": {"queue": "ai"},
    "synthesize_audio": {"queue": "ai"},
    "mux_video": {"queue": "media"},
    "workers.tasks.*": {"queue": "default"},
    "workers.steps.extract_audio.*": {"queue": "media"},
    "workers.steps.asr.*": {"queue": "ai"},
    "workers.steps.translate.*": {"queue": "ai"},
    "workers.steps.tts.*": {"queue": "ai"},
    "workers.steps.mux_video.*": {"queue": "media"},
}


if __name__ == "__main__":
    celery_app.start()
