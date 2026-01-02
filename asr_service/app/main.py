"""FastAPI application entry point for ASR service."""

import logging
from pathlib import Path

from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse

from app.config import settings
from app.models import HealthResponse, TranscribeRequest, TranscribeResponse
from app.transcriber import ModelNotLoadedError, TranscriptionError, transcriber

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="ASR Service",
    description="Moonshine based ASR service for video dubbing",
    version="0.1.0",
)


@app.on_event("startup")
async def startup_event() -> None:
    logger.info("Starting ASR service...")
    Path(settings.audio_temp_dir).mkdir(parents=True, exist_ok=True)
    try:
        if settings.asr_backend == "mock":
            logger.warning("ASR running in mock mode")
        transcriber.load_model()
        logger.info("ASR service started successfully")
    except Exception as exc:
        logger.error("Failed to load ASR model: %s", exc, exc_info=True)


@app.get("/health", response_model=HealthResponse)
async def health_check() -> HealthResponse:
    model_loaded = getattr(transcriber, "_model_loaded", False)
    status = "healthy" if model_loaded else "degraded"
    return HealthResponse(status=status, model_loaded=model_loaded)


@app.post("/transcribe", response_model=TranscribeResponse)
async def transcribe(request: TranscribeRequest) -> TranscribeResponse:
    try:
        result = transcriber.transcribe(request.audio_url, request.language)
        return TranscribeResponse(
            segments=result["segments"],
            language=result["language"],
            duration_ms=result["duration_ms"],
        )
    except ModelNotLoadedError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except TranscriptionError as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc
    except Exception as exc:
        logger.error("Unexpected ASR error", exc_info=True)
        raise HTTPException(status_code=500, detail=f"internal_error: {exc}") from exc


@app.exception_handler(Exception)
async def default_exception_handler(request, exc: Exception):
    return JSONResponse(status_code=500, content={"error": "internal_error", "message": str(exc)})
