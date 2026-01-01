"""FastAPI application entry point for TTS service."""

import logging
from pathlib import Path

from fastapi import FastAPI, Header, HTTPException, Request, Response
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse

from app.config import settings
from app.exceptions import (
    AuthenticationError,
    DurationMismatchError,
    InternalError,
    InvalidParameterError,
    ModelNotLoadedError,
    ModelScopeAPIError,
    RateLimitError,
    SynthesisError,
    TextTooLongError,
)
from app.models import (
    BatchSynthesisRequest,
    BatchSynthesisResponse,
    BatchSynthesisSegment,
    HealthResponse,
    SpeakerInfo,
    SpeakersResponse,
    SynthesisRequest,
    SynthesisResponse,
)
from app.synthesizer import synthesizer

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="TTS Service",
    description="IndexTTS2 based text-to-speech service for video dubbing",
    version="0.1.0",
)


@app.on_event("startup")
async def startup_event():
    """Initialize services on startup."""
    logger.info("Starting TTS service...")

    # Create necessary directories
    temp_dir = Path(settings.audio_temp_dir)
    temp_dir.mkdir(parents=True, exist_ok=True)

    # Load TTS model
    try:
        synthesizer.load_model()
        logger.info("TTS service started successfully")
    except Exception as e:
        logger.error(f"Failed to start TTS service: {e}", exc_info=True)
        # Don't raise - allow service to start in degraded mode
        # Health check will report model not loaded


@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup on shutdown."""
    logger.info("Shutting down TTS service...")


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    """Handle request validation errors."""
    return JSONResponse(
        status_code=400,
        content={
            "error": "invalid_parameter",
            "message": "Request validation failed",
            "details": exc.errors(),
        },
    )


@app.exception_handler(InvalidParameterError)
async def invalid_parameter_handler(request: Request, exc: InvalidParameterError):
    """Handle invalid parameter errors."""
    return JSONResponse(
        status_code=400,
        content={
            "error": "invalid_parameter",
            "message": str(exc),
            "details": {},
        },
    )


@app.exception_handler(TextTooLongError)
async def text_too_long_handler(request: Request, exc: TextTooLongError):
    """Handle text too long errors."""
    return JSONResponse(
        status_code=400,
        content={
            "error": "text_too_long",
            "message": str(exc),
            "details": {},
        },
    )


@app.exception_handler(DurationMismatchError)
async def duration_mismatch_handler(request: Request, exc: DurationMismatchError):
    """Handle duration mismatch errors."""
    return JSONResponse(
        status_code=400,
        content={
            "error": "duration_mismatch",
            "message": str(exc),
            "details": {},
        },
    )


@app.exception_handler(ModelNotLoadedError)
async def model_not_loaded_handler(request: Request, exc: ModelNotLoadedError):
    """Handle model not loaded errors."""
    return JSONResponse(
        status_code=503,
        content={
            "error": "model_not_loaded",
            "message": str(exc),
            "details": {},
        },
    )


@app.exception_handler(SynthesisError)
async def synthesis_error_handler(request: Request, exc: SynthesisError):
    """Handle synthesis errors."""
    return JSONResponse(
        status_code=500,
        content={
            "error": "synthesis_failed",
            "message": str(exc),
            "details": {},
        },
    )


@app.exception_handler(InternalError)
async def internal_error_handler(request: Request, exc: InternalError):
    """Handle internal errors."""
    return JSONResponse(
        status_code=500,
        content={
            "error": "internal_error",
            "message": str(exc),
            "details": {},
        },
    )


@app.exception_handler(AuthenticationError)
async def authentication_error_handler(request: Request, exc: AuthenticationError):
    """Handle authentication errors."""
    return JSONResponse(
        status_code=401,
        content={
            "error": "authentication_error",
            "message": str(exc),
            "details": {},
        },
    )


@app.exception_handler(ModelScopeAPIError)
async def modelscope_api_error_handler(request: Request, exc: ModelScopeAPIError):
    """Handle ModelScope API errors."""
    return JSONResponse(
        status_code=502,
        content={
            "error": "modelscope_api_error",
            "message": str(exc),
            "details": {},
        },
    )


@app.exception_handler(RateLimitError)
async def rate_limit_error_handler(request: Request, exc: RateLimitError):
    """Handle rate limit errors."""
    return JSONResponse(
        status_code=429,
        content={
            "error": "rate_limit_error",
            "message": str(exc),
            "details": {},
        },
    )


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    model_loaded = synthesizer.is_model_loaded()
    status = "healthy" if model_loaded else "degraded"
    return HealthResponse(status=status, model_loaded=model_loaded)


@app.post("/synthesize", response_model=SynthesisResponse)
async def synthesize(
    request: SynthesisRequest,
    http_request: Request,
    x_modelscope_token: str | None = Header(default=None, alias="X-ModelScope-Token"),
) -> Response:
    """Synthesize audio from text with time constraints."""
    # Validate request
    if len(request.text) > 10000:  # Max text length
        raise TextTooLongError("Text exceeds maximum length of 10000 characters")

    # Synthesize audio
    try:
        if request.time_constraints and request.time_constraints.segments:
            # Segment-based synthesis
            segments = [
                {
                    "text": seg.text,
                    "target_duration_ms": seg.target_duration_ms,
                }
                for seg in request.time_constraints.segments
            ]
            audio_bytes = synthesizer.synthesize_with_segments(
                segments=segments,
                speaker_id=request.speaker_id,
                language=request.language,
                prosody_control=request.prosody_control,
                sample_rate=request.sample_rate,
                modelscope_token=x_modelscope_token,
            )
        else:
            # Whole text synthesis
            audio_bytes = synthesizer.synthesize(
                text=request.text,
                target_duration_ms=request.target_duration_ms,
                speaker_id=request.speaker_id,
                language=request.language,
                prosody_control=request.prosody_control,
                sample_rate=request.sample_rate,
                modelscope_token=x_modelscope_token,
            )

        # Calculate actual duration
        import soundfile as sf
        import io

        buffer = io.BytesIO(audio_bytes)
        data, sr = sf.read(buffer)
        actual_duration_ms = int((len(data) / sr) * 1000)

        # Save to temporary file
        import uuid

        audio_id = str(uuid.uuid4())
        audio_filename = f"{audio_id}.{request.output_format}"
        audio_path = Path(settings.audio_temp_dir) / audio_filename
        audio_path.write_bytes(audio_bytes)

        # Generate URL (in production, this would be a proper URL)
        # Absolute URL so worker can download reliably
        audio_url = str(http_request.base_url).rstrip("/") + f"/audio/{audio_filename}"

        # Check Accept header
        accept = http_request.headers.get("Accept", "")
        if "audio" in accept:
            # Return audio directly
            media_type = f"audio/{request.output_format}"
            return Response(
                content=audio_bytes,
                media_type=media_type,
                headers={
                    "Content-Disposition": f'attachment; filename="{audio_filename}"',
                },
            )

        # Return JSON response
        return SynthesisResponse(
            audio_url=audio_url,
            duration_ms=actual_duration_ms,
            sample_rate=request.sample_rate,
            format=request.output_format,
            file_size=len(audio_bytes),
        )

    except (ModelNotLoadedError, SynthesisError):
        raise
    except Exception as e:
        logger.error("Unexpected error in synthesis", exc_info=True)
        raise InternalError(f"Internal error: {e}") from e


@app.post("/synthesize/batch", response_model=BatchSynthesisResponse)
async def synthesize_batch(request: BatchSynthesisRequest):
    """Synthesize multiple audio segments in batch."""
    # Synthesize each request
    segment_audios = []
    segment_responses = []

    for idx, req in enumerate(request.requests):
        try:
            if req.time_constraints and req.time_constraints.segments:
                segments = [
                    {
                        "text": seg.text,
                        "target_duration_ms": seg.target_duration_ms,
                    }
                    for seg in req.time_constraints.segments
                ]
                audio_bytes = synthesizer.synthesize_with_segments(
                    segments=segments,
                    speaker_id=req.speaker_id,
                    language=req.language,
                    prosody_control=req.prosody_control,
                    sample_rate=req.sample_rate,
                )
            else:
                audio_bytes = synthesizer.synthesize(
                    text=req.text,
                    target_duration_ms=req.target_duration_ms,
                    speaker_id=req.speaker_id,
                    language=req.language,
                    prosody_control=req.prosody_control,
                    sample_rate=req.sample_rate,
                )

            # Save segment audio
            import uuid

            segment_id = str(uuid.uuid4())
            segment_filename = f"seg_{idx}_{segment_id}.{req.output_format}"
            segment_path = Path(settings.audio_temp_dir) / segment_filename
            segment_path.write_bytes(audio_bytes)

            # Calculate duration
            import soundfile as sf
            import io

            buffer = io.BytesIO(audio_bytes)
            data, sr = sf.read(buffer)
            duration_ms = int((len(data) / sr) * 1000)

            segment_audios.append(audio_bytes)
            segment_responses.append(
                BatchSynthesisSegment(
                    idx=idx,
                    duration_ms=duration_ms,
                    audio_url=f"/audio/{segment_filename}",
                )
            )

        except Exception as e:
            logger.error(f"Failed to synthesize segment {idx}", exc_info=True)
            raise SynthesisError(f"Failed to synthesize segment {idx}: {e}") from e

    # Merge if requested
    merged_audio_url = None
    total_duration_ms = sum(seg.duration_ms for seg in segment_responses)

    if request.merge and segment_audios:
        import uuid

        merged_audio = synthesizer._merge_audio_segments(
            segment_audios, request.requests[0].sample_rate
        )
        merged_id = str(uuid.uuid4())
        merged_filename = f"batch_{merged_id}.{request.requests[0].output_format}"
        merged_path = Path(settings.audio_temp_dir) / merged_filename
        merged_path.write_bytes(merged_audio)
        merged_audio_url = f"/audio/{merged_filename}"

    return BatchSynthesisResponse(
        audio_url=merged_audio_url or "",
        duration_ms=total_duration_ms,
        segments=segment_responses,
    )


@app.get("/speakers", response_model=SpeakersResponse)
async def get_speakers():
    """Get list of available speakers."""
    # TODO: Load actual speakers from model
    # For now, return default speaker
    speakers = [
        SpeakerInfo(
            id="default",
            name="默认说话人",
            language="en",
            gender="neutral",
        )
    ]
    return SpeakersResponse(speakers=speakers)


@app.get("/audio/{filename}")
async def get_audio(filename: str):
    """Serve audio files."""
    audio_path = Path(settings.audio_temp_dir) / filename
    if not audio_path.exists():
        raise HTTPException(status_code=404, detail="Audio file not found")

    # Determine content type
    ext = audio_path.suffix.lower()
    content_type_map = {
        ".wav": "audio/wav",
        ".mp3": "audio/mpeg",
    }
    content_type = content_type_map.get(ext, "application/octet-stream")

    return Response(
        content=audio_path.read_bytes(),
        media_type=content_type,
        headers={
            "Content-Disposition": f'attachment; filename="{filename}"',
        },
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "app.main:app",
        host=settings.tts_host,
        port=settings.tts_port,
        reload=True,
    )

