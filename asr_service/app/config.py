"""Configuration management for ASR service."""

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    asr_host: str = "0.0.0.0"
    asr_port: int = 8002

    # Moonshine model configuration
    asr_model_id: str = "moonshine-base"
    asr_device: str = "cuda"
    asr_compute_type: str = "float16"
    asr_backend: str = "moonshine"  # "moonshine" or "mock"

    # Audio settings
    target_sample_rate: int = 16000
    audio_temp_dir: str = "./temp/audio"

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )


settings = Settings()
