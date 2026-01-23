"""TTS synthesizer supporting local IndexTTS2 and remote Gradio backends."""

import io
import logging
import time
import tempfile
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Dict, Optional
import threading

import numpy as np
import soundfile as sf

from app.config import settings
from app.exceptions import InvalidParameterError, ModelNotLoadedError, SynthesisError
from app.models import ProsodyControl

logger = logging.getLogger(__name__)


class TTSSynthesizer:
    """TTS synthesizer wrapper for local IndexTTS2 inference and optional Gradio backend."""

    def __init__(self):
        """Initialize the TTS synthesizer."""
        self._index_tts = None
        self._index_tts_lock = threading.Lock()
        self._model_loaded = False  # indicates IndexTTS2 is ready
        self._index_tts_loaded = False
        self._executor: Optional[ThreadPoolExecutor] = None
        self._prompt_cache: Dict[str, Path] = {}
        self._prompt_cache_lock = threading.Lock()
        self._gradio_clients: Dict[str, tuple[object, threading.Lock]] = {}
        self._gradio_clients_lock = threading.Lock()
        if settings.max_concurrent_requests > 0:
            self._executor = ThreadPoolExecutor(max_workers=settings.max_concurrent_requests)

    def load_model(self) -> None:
        """Load the IndexTTS2 model.

        Raises:
            ModelNotLoadedError: If model loading fails.
        """
        if self._model_loaded:
            logger.info("Model already loaded")
            self._index_tts_loaded = self._index_tts is not None or self._index_tts_loaded
            return

        try:
            self._load_index_tts()
            self._model_loaded = True
            self._index_tts_loaded = True
            logger.info("IndexTTS2 model loaded successfully")
        except Exception as e:
            logger.error("Failed to load IndexTTS2 model", extra={"error": str(e)}, exc_info=True)
            raise ModelNotLoadedError(f"Failed to load IndexTTS2 model: {e}") from e

    def is_model_loaded(self) -> bool:
        """Check if the model is loaded.

        Returns:
            True if model is loaded, False otherwise.
        """
        return self._model_loaded

    def is_index_tts_loaded(self) -> bool:
        """Check if IndexTTS2 is loaded."""
        return self._index_tts is not None or self._index_tts_loaded

    def _get_index_tts_device(self) -> Optional[str]:
        device = settings.indextts_device.strip().lower()
        if device in ("", "auto", "none"):
            return None
        return settings.indextts_device

    def _load_index_tts(self) -> None:
        if self._index_tts is not None:
            return
        with self._index_tts_lock:
            if self._index_tts is not None:
                return
            # Use external index-tts-vllm package or HTTP API
            try:
                from indextts_vllm import IndexTTSVLLM as IndexTTS2
            except ImportError:
                # Fallback to HTTP API mode
                raise ModelNotLoadedError("Neither indextts nor index-tts-vllm is available")

            self._index_tts = IndexTTS2(
                cfg_path=settings.indextts_cfg_path,
                model_dir=settings.indextts_model_dir,
                use_fp16=settings.indextts_use_fp16,
                device=self._get_index_tts_device(),
                use_cuda_kernel=settings.indextts_use_cuda_kernel,
                use_torch_compile=settings.indextts_use_torch_compile,
            )
            self._index_tts_loaded = True

    def _call_index_tts(
        self,
        text: str,
        prompt_audio: str,
    ) -> tuple[np.ndarray, int]:
        if self._index_tts is None:
            self._load_index_tts()
        if self._index_tts is None:
            raise ModelNotLoadedError("IndexTTS2 model is not loaded")

        prompt_path = self._prepare_prompt_audio(prompt_audio)

        with tempfile.NamedTemporaryFile(
            suffix=".wav",
            delete=False,
            dir=settings.audio_temp_dir,
        ) as tmp:
            output_path = tmp.name

        try:
            result = self._index_tts.infer(
                spk_audio_prompt=str(prompt_path),
                text=text,
                output_path=output_path,
                verbose=False,
            )
            if not output_path or not Path(output_path).exists():
                raise SynthesisError(f"IndexTTS2 did not produce audio: {result}")

            audio_data, sr = sf.read(output_path)
            if not isinstance(audio_data, np.ndarray):
                audio_data = np.array(audio_data, dtype=np.float32)
            else:
                audio_data = audio_data.astype(np.float32)
            if audio_data.ndim > 1:
                audio_data = np.mean(audio_data, axis=1)
            return audio_data, int(sr)
        finally:
            try:
                Path(output_path).unlink(missing_ok=True)
            except Exception:
                pass

    def _get_cached_prompt(self, prompt_audio: str) -> Optional[Path]:
        with self._prompt_cache_lock:
            cached = self._prompt_cache.get(prompt_audio)
        if cached and cached.exists():
            return cached
        return None

    def _store_cached_prompt(self, prompt_audio: str, path: Path) -> None:
        with self._prompt_cache_lock:
            self._prompt_cache[prompt_audio] = path

    def _prepare_prompt_audio(self, prompt_audio: str) -> Path:
        prompt_path = Path(prompt_audio)
        if prompt_audio.startswith(("http://", "https://")):
            cached = self._get_cached_prompt(prompt_audio)
            if cached:
                logger.debug(
                    "Using cached prompt audio",
                    extra={"prompt_audio": prompt_audio, "path": str(cached)},
                )
                return cached

            last_error: Optional[Exception] = None
            for attempt in range(settings.max_retries):
                try:
                    import requests

                    logger.info(
                        "Downloading prompt audio",
                        extra={"prompt_audio": prompt_audio, "attempt": attempt + 1},
                    )
                    resp = requests.get(prompt_audio, timeout=30)
                    resp.raise_for_status()
                    with tempfile.NamedTemporaryFile(
                        suffix=".wav", delete=False, dir=settings.audio_temp_dir
                    ) as tmp_prompt:
                        tmp_prompt.write(resp.content)
                        downloaded_path = Path(tmp_prompt.name)
                    self._store_cached_prompt(prompt_audio, downloaded_path)
                    return downloaded_path
                except Exception as e:  # pragma: no cover - network related
                    last_error = e
                    logger.warning(
                        "Failed to download prompt audio, will retry",
                        extra={
                            "prompt_audio": prompt_audio,
                            "attempt": attempt + 1,
                            "error": str(e),
                        },
                    )
                    time.sleep(settings.retry_delay_seconds * (attempt + 1))

            raise SynthesisError(
                f"Failed to download prompt audio after {settings.max_retries} attempts: {last_error}"
            ) from last_error

        if prompt_path.exists():
            return prompt_path

        raise ModelNotLoadedError(
            f"IndexTTS2 prompt audio not found or unreachable: {prompt_audio}"
        )

    def _normalize_backend(self, tts_backend: Optional[str]) -> str:
        value = (tts_backend or settings.tts_backend or "").strip()
        if value in ("", "index_tts2"):
            return "index_tts2"
        if value in ("index_tts2_gradio", "gradio"):
            return "index_tts2_gradio"
        raise InvalidParameterError(f"Unsupported TTS backend: {value}")

    def _get_gradio_client(self, base_url: str) -> tuple[object, threading.Lock]:
        normalized = base_url.strip().rstrip("/")
        with self._gradio_clients_lock:
            cached = self._gradio_clients.get(normalized)
            if cached:
                return cached
            try:
                from gradio_client import Client
            except Exception as e:  # pragma: no cover - optional dependency
                raise ModelNotLoadedError(f"gradio_client is not installed: {e}") from e
            client = Client(normalized)
            lock = threading.Lock()
            self._gradio_clients[normalized] = (client, lock)
            return client, lock

    def _extract_gradio_filepath(self, output: object) -> Path:
        if isinstance(output, str):
            return Path(output)
        if isinstance(output, dict):
            for key in ("path", "name", "filepath"):
                value = output.get(key)
                if isinstance(value, str) and value:
                    return Path(value)
        if isinstance(output, (list, tuple)) and output:
            return self._extract_gradio_filepath(output[0])
        raise SynthesisError(f"Unexpected gradio output type: {type(output)}")

    def _call_gradio_index_tts2(
        self,
        text: str,
        prompt_audio: str,
        gradio_url: str,
    ) -> tuple[np.ndarray, int]:
        try:
            from gradio_client import handle_file
        except Exception as e:  # pragma: no cover - optional dependency
            raise ModelNotLoadedError(f"gradio_client is not installed: {e}") from e

        client, lock = self._get_gradio_client(gradio_url)
        prompt_path = self._prepare_prompt_audio(prompt_audio)

        with lock:
            result = client.predict(
                emo_control_method="与音色参考音频相同",
                prompt=handle_file(str(prompt_path)),
                text=text,
                emo_ref_path=handle_file(str(prompt_path)),
                emo_weight=0.8,
                vec1=0,
                vec2=0,
                vec3=0,
                vec4=0,
                vec5=0,
                vec6=0,
                vec7=0,
                vec8=0,
                emo_text="",
                emo_random=False,
                max_text_tokens_per_segment=120,
                param_16=True,
                param_17=0.8,
                param_18=30,
                param_19=0.8,
                param_20=0,
                param_21=3,
                param_22=10,
                param_23=1500,
                api_name="/gen_single",
            )

        output_path = self._extract_gradio_filepath(result)
        if not output_path.exists():
            raise SynthesisError(f"Gradio did not return a file: {output_path}")

        audio_data, sr = sf.read(output_path)
        if not isinstance(audio_data, np.ndarray):
            audio_data = np.array(audio_data, dtype=np.float32)
        else:
            audio_data = audio_data.astype(np.float32)
        if audio_data.ndim > 1:
            audio_data = np.mean(audio_data, axis=1)
        return audio_data, int(sr)

    def _call_tts(
        self,
        text: str,
        prompt_audio_url: Optional[str],
        tts_backend: Optional[str] = None,
        indextts_gradio_url: Optional[str] = None,
    ) -> tuple[np.ndarray, int]:
        backend = self._normalize_backend(tts_backend)
        prompt_audio = prompt_audio_url or settings.indextts_prompt_audio

        if backend == "index_tts2":
            if not self._model_loaded:
                self.load_model()
            return self._call_index_tts(text=text, prompt_audio=prompt_audio)

        gradio_url = (indextts_gradio_url or settings.indextts_gradio_url).strip()
        if not gradio_url:
            raise InvalidParameterError(
                "indextts_gradio_url is required when tts_backend=index_tts2_gradio"
            )
        if not gradio_url.startswith(("http://", "https://")):
            raise InvalidParameterError("indextts_gradio_url must start with http:// or https://")

        return self._call_gradio_index_tts2(text=text, prompt_audio=prompt_audio, gradio_url=gradio_url)

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
        prompt_audio_url: Optional[str] = None,
        tts_backend: Optional[str] = None,
        indextts_gradio_url: Optional[str] = None,
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
        resolved_backend = self._normalize_backend(tts_backend)
        logger.info(
            "Synthesizing audio",
            extra={
                "text_length": len(text),
                "target_duration_ms": target_duration_ms,
                "speaker_id": speaker_id,
                "language": language,
                "strict_duration": settings.strict_duration,
                "backend": resolved_backend,
            },
        )

        try:
            audio_data, produced_sr = self._call_tts(
                text=text,
                prompt_audio_url=prompt_audio_url,
                tts_backend=resolved_backend,
                indextts_gradio_url=indextts_gradio_url,
            )

            # Resample if needed
            if sample_rate != produced_sr:
                try:
                    import librosa

                    audio_data = librosa.resample(
                        audio_data,
                        orig_sr=produced_sr,
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
        prompt_audio_url: Optional[str] = None,
        tts_backend: Optional[str] = None,
        indextts_gradio_url: Optional[str] = None,
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
        resolved_backend = self._normalize_backend(tts_backend)
        logger.info(
            "Synthesizing audio with segments",
            extra={
                "segment_count": len(segments),
                "speaker_id": speaker_id,
                "language": language,
                "backend": resolved_backend,
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
                        prompt_audio_url=prompt_audio_url,
                        tts_backend=resolved_backend,
                        indextts_gradio_url=indextts_gradio_url,
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
                        prompt_audio_url=prompt_audio_url,
                        tts_backend=resolved_backend,
                        indextts_gradio_url=indextts_gradio_url,
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
