"""Data models for ASR service API."""

from typing import List, Optional

from pydantic import BaseModel, Field


class TranscribeRequest(BaseModel):
    """ASR transcription request."""

    audio_url: str = Field(..., description="Presigned audio URL")
    language: Optional[str] = Field(None, description="Language hint, e.g. zh or en")


class Segment(BaseModel):
    """ASR segment."""

    start_ms: int
    end_ms: int
    text: str


class TranscribeResponse(BaseModel):
    """ASR transcription response."""

    segments: List[Segment]
    language: str
    duration_ms: int


class HealthResponse(BaseModel):
    """Health check response."""

    status: str
    model_loaded: bool
