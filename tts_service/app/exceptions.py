"""Custom exceptions for TTS service."""


class TTSServiceException(Exception):
    """Base exception for TTS service errors."""

    pass


class InvalidParameterError(TTSServiceException):
    """Raised when request parameters are invalid."""

    pass


class TextTooLongError(TTSServiceException):
    """Raised when text exceeds maximum length."""

    pass


class DurationMismatchError(TTSServiceException):
    """Raised when duration constraints cannot be met."""

    pass


class ModelNotLoadedError(TTSServiceException):
    """Raised when model is not loaded."""

    pass


class SynthesisError(TTSServiceException):
    """Raised when audio synthesis fails."""

    pass


class InternalError(TTSServiceException):
    """Raised when an internal error occurs."""

    pass

