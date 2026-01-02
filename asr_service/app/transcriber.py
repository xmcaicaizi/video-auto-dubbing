"""Moonshine ASR transcriber implementation."""

from __future__ import annotations

import io
import threading
from typing import Any, Dict, List, Tuple

import numpy as np
import requests
import soundfile as sf
import torch
import torchaudio

from app.config import settings


class ModelNotLoadedError(RuntimeError):
    """Raised when ASR model is not loaded."""


class TranscriptionError(RuntimeError):
    """Raised when ASR transcription fails."""


def _normalize_segments(raw_segments: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    segments: List[Dict[str, Any]] = []
    for seg in raw_segments:
        text = str(seg.get("text", "")).strip()
        if not text:
            continue
        start = seg.get("start_ms")
        end = seg.get("end_ms")
        if start is None:
            start = seg.get("start_time", seg.get("start", 0))
        if end is None:
            end = seg.get("end_time", seg.get("end", 0))
        # If in seconds, convert to ms
        if isinstance(start, (int, float)) and start < 1000:
            start = int(start * 1000)
        else:
            start = int(start)
        if isinstance(end, (int, float)) and end < 1000:
            end = int(end * 1000)
        else:
            end = int(end)
        if end < start:
            end = start
        segments.append({"start_ms": start, "end_ms": end, "text": text})
    return segments


class MoonshineTranscriber:
    """Moonshine ASR wrapper."""

    def __init__(self) -> None:
        self._model = None
        self._lock = threading.Lock()

    def _resolve_device(self) -> str:
        if settings.asr_device == "cuda" and not torch.cuda.is_available():
            return "cpu"
        return settings.asr_device

    def load_model(self) -> None:
        if settings.asr_backend == "mock":
            self._model = "mock"
            return

        with self._lock:
            if self._model is not None:
                return
            try:
                import moonshine  # type: ignore
            except Exception as exc:
                raise ModelNotLoadedError(
                    "Moonshine package is not installed. Install with: pip install moonshine"
                ) from exc

            device = self._resolve_device()
            model_id = settings.asr_model_id
            compute_type = settings.asr_compute_type

            model = None
            if hasattr(moonshine, "load_model"):
                model = moonshine.load_model(model_id, device=device, compute_type=compute_type)
            elif hasattr(moonshine, "Moonshine"):
                model = moonshine.Moonshine.from_pretrained(
                    model_id, device=device, compute_type=compute_type
                )
            elif hasattr(moonshine, "Pipeline"):
                model = moonshine.Pipeline(model_id=model_id, device=device)

            if model is None:
                raise ModelNotLoadedError("Unsupported Moonshine API; update transcriber adapter.")

            self._model = model

    def _load_audio(self, data: bytes) -> Tuple[np.ndarray, int]:
        audio, sr = sf.read(io.BytesIO(data))
        if audio.ndim > 1:
            audio = np.mean(audio, axis=1)
        if sr != settings.target_sample_rate:
            audio_tensor = torch.tensor(audio, dtype=torch.float32)
            audio_tensor = torchaudio.functional.resample(
                audio_tensor, sr, settings.target_sample_rate
            )
            audio = audio_tensor.numpy()
            sr = settings.target_sample_rate
        return audio.astype(np.float32), sr

    def transcribe(self, audio_url: str, language: str | None) -> Dict[str, Any]:
        if self._model is None:
            raise ModelNotLoadedError("ASR model is not loaded")

        if settings.asr_backend == "mock":
            return {
                "segments": [
                    {"start_ms": 0, "end_ms": 1000, "text": "mock transcription"}
                ],
                "language": language or "unknown",
                "duration_ms": 1000,
            }

        resp = requests.get(audio_url, timeout=120)
        resp.raise_for_status()
        audio, sr = self._load_audio(resp.content)
        duration_ms = int(len(audio) / sr * 1000)

        result = None
        if hasattr(self._model, "transcribe"):
            result = self._model.transcribe(audio, sample_rate=sr, language=language)
        elif callable(self._model):
            result = self._model(audio, sample_rate=sr, language=language)
        else:
            raise TranscriptionError("Moonshine model does not support transcription")

        segments: List[Dict[str, Any]] = []
        detected_lang = language or "unknown"

        if isinstance(result, dict):
            if "segments" in result:
                segments = _normalize_segments(result.get("segments", []))
            if "language" in result:
                detected_lang = str(result["language"])
        elif hasattr(result, "segments"):
            segments = _normalize_segments(getattr(result, "segments"))
        elif isinstance(result, list):
            segments = _normalize_segments(result)
        else:
            text = str(result).strip()
            if text:
                segments = [{"start_ms": 0, "end_ms": duration_ms, "text": text}]

        if not segments:
            segments = [{"start_ms": 0, "end_ms": duration_ms, "text": ""}]

        return {
            "segments": segments,
            "language": detected_lang,
            "duration_ms": duration_ms,
        }


transcriber = MoonshineTranscriber()
