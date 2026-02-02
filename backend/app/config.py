"""
应用配置管理
使用 pydantic-settings 从环境变量加载配置
"""

from functools import lru_cache
from typing import Literal

from pydantic import Field, PostgresDsn, RedisDsn
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """应用配置"""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # ==================== 应用配置 ====================
    app_name: str = "Video Dubbing API"
    app_version: str = "2.0.0"
    debug: bool = False
    api_prefix: str = "/api/v1"

    # ==================== 数据库配置 ====================
    db_host: str = Field(default="localhost", alias="DB_HOST")
    db_port: int = Field(default=5432, alias="DB_PORT")
    db_name: str = Field(default="dubbing", alias="DB_NAME")
    db_user: str = Field(default="dubbing", alias="DB_USER")
    db_password: str = Field(default="dubbing123", alias="DB_PASSWORD")

    @property
    def database_url(self) -> str:
        """数据库连接 URL"""
        return f"postgresql://{self.db_user}:{self.db_password}@{self.db_host}:{self.db_port}/{self.db_name}"

    # ==================== Redis 配置 ====================
    redis_host: str = Field(default="localhost", alias="REDIS_HOST")
    redis_port: int = Field(default=6379, alias="REDIS_PORT")
    redis_db: int = Field(default=0, alias="REDIS_DB")
    redis_password: str | None = Field(default=None, alias="REDIS_PASSWORD")

    @property
    def redis_url(self) -> str:
        """Redis 连接 URL"""
        if self.redis_password:
            return f"redis://:{self.redis_password}@{self.redis_host}:{self.redis_port}/{self.redis_db}"
        return f"redis://{self.redis_host}:{self.redis_port}/{self.redis_db}"

    # ==================== Celery 配置 ====================
    celery_broker_url: str | None = None
    celery_result_backend: str | None = None

    @property
    def get_celery_broker_url(self) -> str:
        """Celery Broker URL (默认使用 Redis)"""
        return self.celery_broker_url or self.redis_url

    @property
    def get_celery_result_backend(self) -> str:
        """Celery Result Backend URL (默认使用 Redis)"""
        return self.celery_result_backend or self.redis_url

    # ==================== 阿里云 OSS ====================
    oss_endpoint: str = Field(default="", alias="OSS_ENDPOINT")
    oss_bucket: str = Field(default="", alias="OSS_BUCKET")
    oss_access_key_id: str = Field(default="", alias="OSS_ACCESS_KEY_ID")
    oss_access_key_secret: str = Field(default="", alias="OSS_ACCESS_KEY_SECRET")
    oss_public_domain: str = Field(default="", alias="OSS_PUBLIC_DOMAIN")
    oss_prefix: str = Field(default="videos/", alias="OSS_PREFIX")
    oss_use_ssl: bool = Field(default=True, alias="OSS_USE_SSL")

    # ==================== 阿里百炼 DashScope ====================
    dashscope_api_key: str = Field(default="", alias="DASHSCOPE_API_KEY")

    # ASR 配置
    asr_model: str = Field(default="sensevoice-v1", alias="ASR_MODEL")
    asr_language_hints: list[str] = Field(default=["zh", "en"], alias="ASR_LANGUAGE_HINTS")

    # LLM 配置
    llm_base_url: str = Field(
        default="https://dashscope.aliyuncs.com/compatible-mode/v1",
        alias="DASHSCOPE_LLM_BASE_URL",
    )
    llm_model: str = Field(default="qwen-turbo", alias="DASHSCOPE_LLM_MODEL")
    llm_max_tokens: int = Field(default=2000, alias="LLM_MAX_TOKENS")

    # TTS 配置
    tts_model: Literal[
        "cosyvoice-v1",  # 系统音色模式
        "qwen3-tts-vc-realtime-2026-01-15",  # 声音复刻模式
    ] = Field(default="cosyvoice-v1", alias="TTS_MODEL")
    tts_voice: str = Field(
        default="longxiaochun",
        alias="TTS_VOICE",
        description="音色名称（系统音色）或 voice_id（复刻音色，如 vc_xxx）",
    )
    tts_format: str = Field(default="mp3", alias="TTS_FORMAT")

    # ==================== 处理配置 ====================
    # 上传限制
    max_upload_size: int = Field(default=500 * 1024 * 1024, alias="MAX_UPLOAD_SIZE")  # 500MB
    allowed_video_formats: list[str] = Field(
        default=["mp4", "avi", "mov", "mkv", "flv"], alias="ALLOWED_VIDEO_FORMATS"
    )

    # Worker 配置
    worker_concurrency: int = Field(default=4, alias="WORKER_CONCURRENCY")
    task_timeout: int = Field(default=3600, alias="TASK_TIMEOUT")  # 1小时

    # ==================== CORS 配置 ====================
    cors_origins: list[str] = Field(
        default=["http://localhost:3000", "http://localhost"], alias="CORS_ORIGINS"
    )
    cors_allow_credentials: bool = True
    cors_allow_methods: list[str] = ["*"]
    cors_allow_headers: list[str] = ["*"]

    # ==================== 日志配置 ====================
    log_level: Literal["DEBUG", "INFO", "WARNING", "ERROR"] = Field(default="INFO", alias="LOG_LEVEL")
    log_file: str = Field(default="logs/app.log", alias="LOG_FILE")


@lru_cache
def get_settings() -> Settings:
    """获取配置单例"""
    return Settings()


# 全局配置实例
settings = get_settings()
