"""TTS synthesizer using ModelScope IndexTTS-2 API."""

import io
import logging
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Any, Dict, Optional
import threading

import numpy as np
import soundfile as sf

from app.config import settings
from app.exceptions import (
    AuthenticationError,
    ModelNotLoadedError,
    ModelScopeAPIError,
    RateLimitError,
    SynthesisError,
)
from app.models import ProsodyControl

logger = logging.getLogger(__name__)


class TTSSynthesizer:
    """TTS synthesizer wrapper for ModelScope IndexTTS-2 API."""

    def __init__(self):
        """Initialize the TTS synthesizer."""
        self._pipelines: Dict[str, Any] = {}
        self._pipeline_lock = threading.Lock()
        self._model_loaded = False  # indicates at least one pipeline is ready (or mock mode)
        self._executor: Optional[ThreadPoolExecutor] = None
        if settings.max_concurrent_requests > 0:
            self._executor = ThreadPoolExecutor(max_workers=settings.max_concurrent_requests)

    def load_model(self) -> None:
        """Load the ModelScope TTS pipeline.

        Raises:
            ModelNotLoadedError: If model loading fails.
            AuthenticationError: If authentication fails.
        """
        if self._model_loaded:
            logger.info("Model already loaded")
            return

        # Check backend mode
        if settings.tts_backend == "mock":
            logger.warning("Using mock mode for TTS synthesis")
            self._model_loaded = True
            return

        # Validate default token
        if not settings.modelscope_token:
            raise AuthenticationError(
                "MODELSCOPE_TOKEN is required but not set. "
                "Please set it in environment variables."
            )

        logger.info(
            "Loading ModelScope TTS pipeline",
            extra={
                "model_id": settings.modelscope_model_id,
                "backend": settings.tts_backend,
            },
        )

        try:
            # Warm up default pipeline in cache
            _ = self._get_pipeline(settings.modelscope_token)
            self._model_loaded = True
            logger.info("ModelScope TTS pipeline loaded successfully (default token)")

        except ImportError as e:
            logger.error("Failed to import ModelScope", extra={"error": str(e)}, exc_info=True)
            raise ModelNotLoadedError(
                "ModelScope package not installed. Install with: pip install modelscope"
            ) from e
        except Exception as e:
            error_msg = str(e).lower()
            if "token" in error_msg or "auth" in error_msg or "401" in error_msg:
                logger.error("Authentication failed", extra={"error": str(e)}, exc_info=True)
                raise AuthenticationError(f"ModelScope authentication failed: {e}") from e
            else:
                logger.error("Failed to load model", extra={"error": str(e)}, exc_info=True)
                raise ModelNotLoadedError(f"Failed to load TTS model: {e}") from e

    def is_model_loaded(self) -> bool:
        """Check if the model is loaded.

        Returns:
            True if model is loaded, False otherwise.
        """
        return self._model_loaded

    def _get_pipeline(self, token: str):
        """Get or create a ModelScope pipeline for a specific token."""
        if settings.tts_backend == "mock":
            return None
        if not token:
            raise AuthenticationError("ModelScope token is required")

        with self._pipeline_lock:
            if token in self._pipelines:
                return self._pipelines[token]

            from modelscope.pipelines import pipeline
            from modelscope.utils.constant import Tasks
            import os

            # ModelScope reads token from env. Create pipeline under lock to avoid token races.
            os.environ["MODELSCOPE_API_TOKEN"] = token
            p = pipeline(
                task=Tasks.text_to_speech,
                model=settings.modelscope_model_id,
            )
            self._pipelines[token] = p
            # If we successfully created one pipeline, mark loaded
            self._model_loaded = True
            return p

    def _call_modelscope_api(
        self,
        text: str,
        speaker_id: str = "default",
        language: str = "en",
        prosody_control: Optional[ProsodyControl] = None,
        modelscope_token: Optional[str] = None,
    ) -> np.ndarray:
        """Call ModelScope API to synthesize audio.

        Args:
            text: Text to synthesize.
            speaker_id: Speaker ID (may not be supported by all models).
            language: Language code.
            prosody_control: Optional prosody control parameters.

        Returns:
            Audio data as numpy array.

        Raises:
            ModelScopeAPIError: If API call fails.
            RateLimitError: If rate limit is exceeded.
        """
        if settings.tts_backend == "mock":
            # Mock mode: return silent audio
            duration_sec = 1.0  # Default 1 second
            samples = int(settings.default_sample_rate * duration_sec)
            return np.zeros(samples, dtype=np.float32)

        token = modelscope_token or settings.modelscope_token
        if not token:
            raise AuthenticationError("MODELSCOPE_TOKEN is required but not set")

        pipeline_obj = self._get_pipeline(token)
        if pipeline_obj is None:
            raise ModelNotLoadedError("TTS pipeline is not loaded")

        # Prepare input for ModelScope pipeline
        # Note: IndexTTS-2 may have specific input format requirements
        # This is a generic implementation that may need adjustment
        input_data = {"text": text}

        # Add optional parameters if supported
        # Note: These may not be supported by all ModelScope TTS models
        if speaker_id and speaker_id != "default":
            input_data["speaker"] = speaker_id

        if language:
            input_data["language"] = language

        # Retry logic
        last_error = None
        for attempt in range(settings.max_retries):
            try:
                result = pipeline_obj(input_data)

                # Extract audio from result
                # ModelScope TTS pipeline typically returns dict with 'output_wav' or 'audio'
                if isinstance(result, dict):
                    if "output_wav" in result:
                        audio = result["output_wav"]
                    elif "audio" in result:
                        audio = result["audio"]
                    elif "output" in result:
                        audio = result["output"]
                    else:
                        # Try to get first value that looks like audio
                        audio = next(iter(result.values()))
                elif isinstance(result, (np.ndarray, list)):
                    audio = np.array(result, dtype=np.float32)
                else:
                    raise ModelScopeAPIError(f"Unexpected result type: {type(result)}")

                # Ensure audio is numpy array
                if not isinstance(audio, np.ndarray):
                    audio = np.array(audio, dtype=np.float32)

                # Normalize audio to [-1, 1] range if needed
                if audio.dtype != np.float32:
                    audio = audio.astype(np.float32)
                if np.abs(audio).max() > 1.0:
                    audio = audio / np.abs(audio).max()

                return audio

            except Exception as e:
                last_error = e
                error_msg = str(e).lower()

                # Check for rate limit
                if "rate limit" in error_msg or "429" in error_msg or "quota" in error_msg:
                    if attempt < settings.max_retries - 1:
                        wait_time = settings.retry_delay_seconds * (2**attempt)
                        logger.warning(
                            "Rate limit hit, retrying",
                            extra={"attempt": attempt + 1, "wait_time": wait_time},
                        )
                        time.sleep(wait_time)
                        continue
                    raise RateLimitError(f"ModelScope API rate limit exceeded: {e}") from e

                # Check for authentication errors
                if "token" in error_msg or "auth" in error_msg or "401" in error_msg:
                    raise AuthenticationError(f"ModelScope authentication failed: {e}") from e

                # Other errors: retry with exponential backoff
                if attempt < settings.max_retries - 1:
                    wait_time = settings.retry_delay_seconds * (2**attempt)
                    logger.warning(
                        "API call failed, retrying",
                        extra={
                            "attempt": attempt + 1,
                            "wait_time": wait_time,
                            "error": str(e)[:100],
                        },
                    )
                    time.sleep(wait_time)
                else:
                    logger.error(
                        "ModelScope API call failed after retries",
                        extra={"error": str(e), "text_length": len(text)},
                        exc_info=True,
                    )

        raise ModelScopeAPIError(f"Failed to call ModelScope API after {settings.max_retries} attempts: {last_error}") from last_error

    def _adjust_duration(self, audio: np.ndarray, target_duration_ms: int, sample_rate: int) -> np.ndarray:
        """Adjust audio duration to match target duration.

        Args:
            audio: Input audio data.
            target_duration_ms: Target duration in milliseconds.
            sample_rate: Sample rate.

        Returns:
            Adjusted audio data.
        """
        if not settings.strict_duration:
            return audio

        current_duration_ms = (len(audio) / sample_rate) * 1000
        if abs(current_duration_ms - target_duration_ms) < 50:  # Within 50ms, no adjustment
            return audio

        # Use librosa for time-stretching
        try:
            import librosa

            target_samples = int(sample_rate * target_duration_ms / 1000.0)
            if target_samples == 0:
                return audio

            # Time-stretch to target duration
            rate = len(audio) / target_samples
            adjusted = librosa.effects.time_stretch(audio, rate=rate)

            # Trim or pad to exact length
            if len(adjusted) > target_samples:
                adjusted = adjusted[:target_samples]
            elif len(adjusted) < target_samples:
                padding = np.zeros(target_samples - len(adjusted), dtype=np.float32)
                adjusted = np.concatenate([adjusted, padding])

            logger.debug(
                "Duration adjusted",
                extra={
                    "original_ms": int(current_duration_ms),
                    "target_ms": target_duration_ms,
                    "adjusted_ms": int((len(adjusted) / sample_rate) * 1000),
                },
            )

            return adjusted

        except ImportError:
            logger.warning("librosa not available, cannot adjust duration strictly")
            return audio
        except Exception as e:
            logger.warning(f"Failed to adjust duration: {e}, returning original audio")
            return audio

    def synthesize(
        self,
        text: str,
        target_duration_ms: int,
        speaker_id: str = "default",
        language: str = "en",
        prosody_control: Optional[ProsodyControl] = None,
        sample_rate: int = 22050,
        modelscope_token: Optional[str] = None,
    ) -> bytes:
        """Synthesize audio from text with time constraints.

        Args:
            text: Text to synthesize.
            target_duration_ms: Target duration in milliseconds.
            speaker_id: Speaker ID for voice cloning.
            language: Language code.
            prosody_control: Optional prosody control parameters.
            sample_rate: Audio sample rate.

        Returns:
            Synthesized audio data as bytes (WAV format).

        Raises:
            ModelNotLoadedError: If model is not loaded.
            SynthesisError: If synthesis fails.
        """
        if not self._model_loaded:
            raise ModelNotLoadedError("TTS model is not loaded")

        logger.info(
            "Synthesizing audio",
            extra={
                "text_length": len(text),
                "target_duration_ms": target_duration_ms,
                "speaker_id": speaker_id,
                "language": language,
                "strict_duration": settings.strict_duration,
            },
        )

        try:
            # Call ModelScope API
            audio_data = self._call_modelscope_api(
                text=text,
                speaker_id=speaker_id,
                language=language,
                prosody_control=prosody_control,
                modelscope_token=modelscope_token,
            )

            # Resample if needed
            if sample_rate != settings.default_sample_rate:
                try:
                    import librosa

                    audio_data = librosa.resample(
                        audio_data,
                        orig_sr=settings.default_sample_rate,
                        target_sr=sample_rate,
                    )
                except ImportError:
                    logger.warning(
                        "librosa not available, cannot resample. Using original sample rate."
                    )
                except Exception as e:
                    logger.warning(f"Resampling failed: {e}, using original audio")

            # Adjust duration if strict mode is enabled
            audio_data = self._adjust_duration(audio_data, target_duration_ms, sample_rate)

            # Convert to WAV bytes
            buffer = io.BytesIO()
            sf.write(buffer, audio_data, sample_rate, format="WAV")
            audio_bytes = buffer.getvalue()

            logger.info(
                "Audio synthesized successfully",
                extra={
                    "duration_ms": int((len(audio_data) / sample_rate) * 1000),
                    "target_duration_ms": target_duration_ms,
                    "audio_size": len(audio_bytes),
                },
            )

            return audio_bytes

        except (ModelScopeAPIError, AuthenticationError, RateLimitError):
            raise
        except Exception as e:
            logger.error(
                "Synthesis failed",
                extra={"error": str(e), "text": text[:50]},
                exc_info=True,
            )
            raise SynthesisError(f"Failed to synthesize audio: {e}") from e

    def synthesize_with_segments(
        self,
        segments: list,
        speaker_id: str = "default",
        language: str = "en",
        prosody_control: Optional[ProsodyControl] = None,
        sample_rate: int = 22050,
        modelscope_token: Optional[str] = None,
    ) -> bytes:
        """Synthesize audio with segment-level time constraints.

        Args:
            segments: List of segment dictionaries with 'text' and 'target_duration_ms'.
            speaker_id: Speaker ID for voice cloning.
            language: Language code.
            prosody_control: Optional prosody control parameters.
            sample_rate: Audio sample rate.

        Returns:
            Merged audio data as bytes (WAV format).

        Raises:
            ModelNotLoadedError: If model is not loaded.
            SynthesisError: If synthesis fails.
        """
        if not self._model_loaded:
            raise ModelNotLoadedError("TTS model is not loaded")

        logger.info(
            "Synthesizing audio with segments",
            extra={
                "segment_count": len(segments),
                "speaker_id": speaker_id,
                "language": language,
            },
        )

        try:
            # Synthesize each segment
            segment_audios = []

            # Use thread pool for concurrent synthesis if executor is available
            if self._executor and len(segments) > 1:
                futures = {}
                for idx, seg in enumerate(segments):
                    future = self._executor.submit(
                        self.synthesize,
                        text=seg["text"],
                        target_duration_ms=seg["target_duration_ms"],
                        speaker_id=speaker_id,
                        language=language,
                        prosody_control=prosody_control,
                        sample_rate=sample_rate,
                        modelscope_token=modelscope_token,
                    )
                    futures[future] = idx

                # Collect results in order
                results = [None] * len(segments)
                for future in as_completed(futures):
                    idx = futures[future]
                    try:
                        results[idx] = future.result()
                    except Exception as e:
                        logger.error(f"Failed to synthesize segment {idx}: {e}", exc_info=True)
                        raise SynthesisError(f"Failed to synthesize segment {idx}: {e}") from e

                segment_audios = results
            else:
                # Sequential synthesis
                for seg in segments:
                    audio = self.synthesize(
                        text=seg["text"],
                        target_duration_ms=seg["target_duration_ms"],
                        speaker_id=speaker_id,
                        language=language,
                        prosody_control=prosody_control,
                        sample_rate=sample_rate,
                        modelscope_token=modelscope_token,
                    )
                    segment_audios.append(audio)

            # Merge segments
            merged_audio = self._merge_audio_segments(segment_audios, sample_rate)

            logger.info(
                "Segmented audio synthesized successfully",
                extra={"total_segments": len(segments), "audio_size": len(merged_audio)},
            )

            return merged_audio

        except Exception as e:
            logger.error("Segmented synthesis failed", extra={"error": str(e)}, exc_info=True)
            raise SynthesisError(f"Failed to synthesize segmented audio: {e}") from e

    def _merge_audio_segments(self, audio_segments: list, sample_rate: int) -> bytes:
        """Merge multiple audio segments into a single audio file.

        Args:
            audio_segments: List of audio data as bytes.
            sample_rate: Audio sample rate.

        Returns:
            Merged audio data as bytes (WAV format).
        """
        # Load all segments
        all_audio = []
        for audio_bytes in audio_segments:
            buffer = io.BytesIO(audio_bytes)
            data, _ = sf.read(buffer)
            all_audio.append(data)

        # Concatenate
        merged = np.concatenate(all_audio)

        # Convert back to WAV bytes
        buffer = io.BytesIO()
        sf.write(buffer, merged, sample_rate, format="WAV")
        return buffer.getvalue()

    def __del__(self):
        """Cleanup resources."""
        if self._executor:
            self._executor.shutdown(wait=False)


# Global synthesizer instance
synthesizer = TTSSynthesizer()
