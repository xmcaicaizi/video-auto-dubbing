"""Data models for TTS service API."""

from typing import List, Optional

from pydantic import BaseModel, Field


class ProsodyControl(BaseModel):
    """Prosody control parameters for speech synthesis."""

    speed: float = Field(default=1.0, ge=0.5, le=2.0, description="Speech speed (0.5-2.0)")
    pitch: float = Field(default=1.0, ge=0.5, le=2.0, description="Pitch (0.5-2.0)")
    energy: float = Field(default=1.0, ge=0.5, le=2.0, description="Energy (0.5-2.0)")


class TimeSegment(BaseModel):
    """Time segment for fine-grained control."""

    text: str = Field(description="Text for this segment")
    target_duration_ms: int = Field(gt=0, description="Target duration in milliseconds")
    start_time_ms: int = Field(ge=0, description="Start time in milliseconds")


class TimeConstraints(BaseModel):
    """Time constraints for segment alignment."""

    segments: List[TimeSegment] = Field(default_factory=list, description="Segment information")


class SynthesisRequest(BaseModel):
    """Request model for speech synthesis."""

    text: str = Field(min_length=1, description="Text to synthesize")
    speaker_id: str = Field(default="default", description="Speaker ID")
    prompt_audio_url: Optional[str] = Field(
        default=None, description="Prompt audio URL for voice cloning"
    )
    target_duration_ms: int = Field(gt=0, description="Target duration in milliseconds")
    language: str = Field(default="en", description="Language code (ISO 639-1)")
    prosody_control: Optional[ProsodyControl] = Field(
        default=None, description="Prosody control parameters"
    )
    time_constraints: Optional[TimeConstraints] = Field(
        default=None, description="Time constraints for segment alignment"
    )
    output_format: str = Field(default="wav", pattern="^(wav|mp3)$", description="Output format")
    sample_rate: int = Field(default=22050, description="Sample rate")


class BatchSynthesisRequest(BaseModel):
    """Request model for batch synthesis."""

    requests: List[SynthesisRequest] = Field(min_length=1, description="List of synthesis requests")
    merge: bool = Field(default=True, description="Whether to merge all audio segments")


class SynthesisResponse(BaseModel):
    """Response model for speech synthesis."""

    audio_url: str = Field(description="URL to the generated audio file")
    duration_ms: int = Field(description="Actual duration in milliseconds")
    sample_rate: int = Field(description="Sample rate")
    format: str = Field(description="Audio format")
    file_size: int = Field(description="File size in bytes")


class BatchSynthesisSegment(BaseModel):
    """Segment information in batch synthesis response."""

    idx: int = Field(description="Segment index")
    duration_ms: int = Field(description="Duration in milliseconds")
    audio_url: str = Field(description="URL to the segment audio file")


class BatchSynthesisResponse(BaseModel):
    """Response model for batch synthesis."""

    audio_url: str = Field(description="URL to the merged audio file (if merge=True)")
    duration_ms: int = Field(description="Total duration in milliseconds")
    segments: List[BatchSynthesisSegment] = Field(description="Individual segment information")


class SpeakerInfo(BaseModel):
    """Speaker information."""

    id: str = Field(description="Speaker ID")
    name: str = Field(description="Speaker name")
    language: str = Field(description="Language code")
    gender: str = Field(description="Gender (male/female/neutral)")


class SpeakersResponse(BaseModel):
    """Response model for speakers list."""

    speakers: List[SpeakerInfo] = Field(description="List of available speakers")


class HealthResponse(BaseModel):
    """Response model for health check."""

    status: str = Field(description="Service status")
    model_loaded: bool = Field(description="Whether the model is loaded")
    backend: str = Field(description="Active TTS backend")
    index_tts2_loaded: bool = Field(description="Whether IndexTTS2 is loaded")
    message: str = Field(description="Additional health information")


class ErrorResponse(BaseModel):
    """Error response model."""

    error: str = Field(description="Error code")
    message: str = Field(description="Human readable error message")
    details: dict = Field(default_factory=dict, description="Additional error details")
