"""Configuration management for TTS service."""

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    # Server configuration
    tts_host: str = "0.0.0.0"
    tts_port: int = 8000
    tts_workers: int = 1

    # Model configuration
    model_path: str = "./models/index_tts2"
    speaker_embedding_path: str = "./models/speaker_embeddings"
    device: str = "cpu"  # cuda or cpu

    # Audio settings
    default_sample_rate: int = 22050
    default_format: str = "wav"

    # Storage configuration
    audio_temp_dir: str = "./temp/audio"
    audio_temp_retention_hours: int = 24

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )


# Global settings instance
settings = Settings()

