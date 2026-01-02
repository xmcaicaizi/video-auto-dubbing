"""Moonshine ASR transcriber implementation."""

from __future__ import annotations

import io
import math
import tempfile
import threading
from pathlib import Path
from typing import Any, Dict, List, Tuple

import numpy as np
import requests
import soundfile as sf

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


def _segments_with_estimated_timestamps(texts: List[str], duration_ms: int) -> List[Dict[str, Any]]:
    clean = [t.strip() for t in texts if t.strip()]
    if not clean:
        return [{"start_ms": 0, "end_ms": duration_ms, "text": ""}]
    n = len(clean)
    step = int(math.ceil(duration_ms / n)) if n else duration_ms
    out = []
    for idx, text in enumerate(clean):
        start = idx * step
        end = min(duration_ms, (idx + 1) * step)
        out.append({"start_ms": start, "end_ms": end, "text": text})
    return out


class MoonshineONNXTranscriber:
    """Moonshine ONNX wrapper (moonshine-onnx)."""

    def __init__(self) -> None:
        self._model_loaded = False
        self._lock = threading.Lock()

    def load_model(self) -> None:
        if settings.asr_backend == "mock":
            self._model_loaded = True
            return
        with self._lock:
            if self._model_loaded:
                return
            try:
                import moonshine_onnx  # noqa: F401
            except Exception as exc:
                raise ModelNotLoadedError(
                    "moonshine_onnx package is not installed. Install useful-moonshine-onnx."
                ) from exc
            self._model_loaded = True

    def _load_audio(self, data: bytes) -> Tuple[np.ndarray, int]:
        audio, sr = sf.read(io.BytesIO(data))
        if audio.ndim > 1:
            audio = np.mean(audio, axis=1)
        return audio.astype(np.float32), sr

    def _resolve_model_id(self, language: str | None) -> str:
        model_id = settings.asr_model_id
        if language and language != "en" and "-" not in model_id:
            # Use language-tagged model if caller provides language hint
            return f"{model_id}-{language}"
        return model_id

    def transcribe(self, audio_url: str, language: str | None) -> Dict[str, Any]:
        if not self._model_loaded:
            raise ModelNotLoadedError("ASR model is not loaded")

        if settings.asr_backend == "mock":
            return {
                "segments": [{"start_ms": 0, "end_ms": 1000, "text": "mock transcription"}],
                "language": language or "unknown",
                "duration_ms": 1000,
            }

        import moonshine_onnx

        resp = requests.get(audio_url, timeout=120)
        resp.raise_for_status()
        audio, sr = self._load_audio(resp.content)
        duration_ms = int(len(audio) / sr * 1000) if sr > 0 else 0

        with tempfile.TemporaryDirectory() as tmpdir:
            audio_path = Path(tmpdir) / "audio.wav"
            sf.write(audio_path, audio, sr)
            model_id = self._resolve_model_id(language)
            result = moonshine_onnx.transcribe(audio_path, model_id)

        segments: List[Dict[str, Any]] = []
        detected_lang = language or "unknown"

        if isinstance(result, dict) and "segments" in result:
            segments = _normalize_segments(result.get("segments", []))
            if "language" in result:
                detected_lang = str(result["language"])
        elif isinstance(result, list):
            if result and isinstance(result[0], dict):
                segments = _normalize_segments(result)
            elif result and isinstance(result[0], (list, tuple)) and len(result[0]) >= 3:
                segments = _normalize_segments(
                    [{"start_ms": r[0], "end_ms": r[1], "text": r[2]} for r in result]
                )
            else:
                segments = _segments_with_estimated_timestamps(result, duration_ms)
        else:
            text = str(result).strip()
            segments = _segments_with_estimated_timestamps([text], duration_ms)

        return {
            "segments": segments,
            "language": detected_lang,
            "duration_ms": duration_ms,
        }


transcriber = MoonshineONNXTranscriber()
