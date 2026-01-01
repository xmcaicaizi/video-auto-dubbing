"""TTS synthesizer using IndexTTS2 model."""

import logging
from pathlib import Path
from typing import Optional

from app.config import settings
from app.exceptions import ModelNotLoadedError, SynthesisError
from app.models import ProsodyControl, SynthesisRequest

logger = logging.getLogger(__name__)


class TTSSynthesizer:
    """TTS synthesizer wrapper for IndexTTS2 model."""

    def __init__(self):
        """Initialize the TTS synthesizer."""
        self._model = None
        self._model_loaded = False
        self._model_path = Path(settings.model_path)
        self._speaker_embedding_path = Path(settings.speaker_embedding_path)
        self._device = settings.device

    def load_model(self) -> None:
        """Load the IndexTTS2 model.

        Raises:
            ModelNotLoadedError: If model loading fails.
        """
        if self._model_loaded:
            logger.info("Model already loaded")
            return

        logger.info("Loading TTS model", extra={"model_path": str(self._model_path)})

        try:
            # TODO: Implement actual IndexTTS2 model loading
            # This is a placeholder for the actual implementation
            # For now, we'll just mark the model as loaded
            if not self._model_path.exists():
                logger.warning(
                    "Model path does not exist, using mock mode",
                    extra={"model_path": str(self._model_path)},
                )
                self._model_loaded = True  # Mock mode for development
                return

            # Placeholder for actual model loading
            # from index_tts2 import IndexTTS2
            # self._model = IndexTTS2(
            #     model_path=str(self._model_path),
            #     speaker_embedding_path=str(self._speaker_embedding_path),
            #     device=self._device,
            # )
            self._model_loaded = True
            logger.info("Model loaded successfully")

        except Exception as e:
            logger.error("Failed to load model", extra={"error": str(e)}, exc_info=True)
            raise ModelNotLoadedError(f"Failed to load TTS model: {e}") from e

    def is_model_loaded(self) -> bool:
        """Check if the model is loaded.

        Returns:
            True if model is loaded, False otherwise.
        """
        return self._model_loaded

    def synthesize(
        self,
        text: str,
        target_duration_ms: int,
        speaker_id: str = "default",
        language: str = "en",
        prosody_control: Optional[ProsodyControl] = None,
        sample_rate: int = 22050,
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
            },
        )

        try:
            # TODO: Implement actual IndexTTS2 synthesis
            # This is a placeholder that returns empty audio
            # For development, we'll generate a silent audio file
            import numpy as np
            import soundfile as sf
            import io

            # Generate silent audio with target duration
            duration_sec = target_duration_ms / 1000.0
            samples = int(sample_rate * duration_sec)
            audio_data = np.zeros(samples, dtype=np.float32)

            # Convert to WAV bytes
            buffer = io.BytesIO()
            sf.write(buffer, audio_data, sample_rate, format="WAV")
            audio_bytes = buffer.getvalue()

            logger.info(
                "Audio synthesized successfully",
                extra={
                    "duration_ms": target_duration_ms,
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
            for seg in segments:
                audio = self.synthesize(
                    text=seg["text"],
                    target_duration_ms=seg["target_duration_ms"],
                    speaker_id=speaker_id,
                    language=language,
                    prosody_control=prosody_control,
                    sample_rate=sample_rate,
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
        import numpy as np
        import soundfile as sf
        import io

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


# Global synthesizer instance
synthesizer = TTSSynthesizer()

