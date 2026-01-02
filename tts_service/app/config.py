"""Configuration management for TTS service."""

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    # Server configuration
    tts_host: str = "0.0.0.0"
    tts_port: int = 8000
    tts_workers: int = 1

    # ModelScope configuration
    modelscope_token: str = ""  # ModelScope API token (optional for public models)
    modelscope_model_id: str = "IndexTeam/IndexTTS-2"  # Model ID on ModelScope
    tts_backend: str = "index_tts2"  # Backend: "index_tts2", "modelscope", or "mock"
    strict_duration: bool = False  # Whether to strictly enforce target_duration_ms

    # IndexTTS2 local inference configuration
    indextts_model_dir: str = "/app/models/IndexTTS-2"
    indextts_cfg_path: str = "/app/models/IndexTTS-2/config.yaml"
    indextts_prompt_audio: str = "/app/assets/voice_01.wav"
    indextts_device: str = "auto"
    indextts_use_fp16: bool = True
    indextts_use_torch_compile: bool = False
    indextts_use_cuda_kernel: bool = False

    # Legacy model configuration (deprecated, kept for backward compatibility)
    model_path: str = "./models/index_tts2"
    speaker_embedding_path: str = "./models/speaker_embeddings"
    device: str = "cpu"  # cuda or cpu

    # Audio settings
    default_sample_rate: int = 22050
    default_format: str = "wav"

    # Storage configuration
    audio_temp_dir: str = "./temp/audio"
    audio_temp_retention_hours: int = 24

    # Concurrency and retry settings
    max_concurrent_requests: int = 10  # Max concurrent TTS requests
    max_retries: int = 3  # Max retries for ModelScope API calls
    retry_delay_seconds: float = 1.0  # Delay between retries

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )


# Global settings instance
settings = Settings()

