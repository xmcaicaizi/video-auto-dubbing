"""
FFmpeg 工具类
视频/音频处理
"""

import subprocess
import tempfile
from pathlib import Path
from typing import Optional

from loguru import logger


class FFmpegHelper:
    """FFmpeg 工具类"""

    @staticmethod
    def check_ffmpeg() -> bool:
        """检查 FFmpeg 是否已安装"""
        try:
            subprocess.run(
                ["ffmpeg", "-version"],
                check=True,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            return True
        except (subprocess.CalledProcessError, FileNotFoundError):
            return False

    def extract_audio(
        self,
        video_path: str,
        output_path: Optional[str] = None,
        sample_rate: int = 16000,
        channels: int = 1,
    ) -> str:
        """
        从视频中提取音频

        Args:
            video_path: 视频文件路径
            output_path: 输出音频路径（可选，默认自动生成）
            sample_rate: 采样率（Hz）
            channels: 声道数（1=单声道，2=立体声）

        Returns:
            输出音频文件路径

        Raises:
            RuntimeError: FFmpeg 执行失败
        """
        if not output_path:
            output_path = str(
                Path(video_path).parent / f"{Path(video_path).stem}_audio.wav"
            )

        logger.info(f"Extracting audio: {video_path} -> {output_path}")

        cmd = [
            "ffmpeg",
            "-i",
            video_path,
            "-vn",  # 不处理视频
            "-acodec",
            "pcm_s16le",  # PCM 16-bit
            "-ar",
            str(sample_rate),  # 采样率
            "-ac",
            str(channels),  # 声道数
            "-y",  # 覆盖输出文件
            output_path,
        ]

        try:
            subprocess.run(
                cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )
            logger.info(f"Audio extracted: {output_path}")
            return output_path
        except subprocess.CalledProcessError as e:
            logger.error(f"FFmpeg extraction failed: {e.stderr.decode()}")
            raise RuntimeError(f"Audio extraction failed: {e.stderr.decode()}")

    def get_duration_ms(self, media_path: str) -> int:
        """
        获取媒体文件时长（毫秒）

        Args:
            media_path: 媒体文件路径

        Returns:
            时长（毫秒）

        Raises:
            RuntimeError: FFprobe 执行失败
        """
        cmd = [
            "ffprobe",
            "-v",
            "error",
            "-show_entries",
            "format=duration",
            "-of",
            "default=noprint_wrappers=1:nokey=1",
            media_path,
        ]

        try:
            result = subprocess.run(
                cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )
            duration_sec = float(result.stdout.decode().strip())
            duration_ms = int(duration_sec * 1000)
            logger.info(f"Media duration: {duration_ms}ms ({media_path})")
            return duration_ms
        except subprocess.CalledProcessError as e:
            logger.error(f"FFprobe failed: {e.stderr.decode()}")
            raise RuntimeError(f"Failed to get duration: {e.stderr.decode()}")

    def merge_audio_segments(
        self, segments: list[dict], output_path: Optional[str] = None
    ) -> str:
        """
        合并多个音频分段（按时间轴）

        Args:
            segments: 分段列表，每个元素包含:
                - path: 音频文件路径
                - start_ms: 开始时间（毫秒）
                - end_ms: 结束时间（毫秒）
            output_path: 输出音频路径（可选）

        Returns:
            输出音频文件路径

        Raises:
            RuntimeError: FFmpeg 执行失败
        """
        if not segments:
            raise ValueError("No segments provided")

        if not output_path:
            output_path = tempfile.mktemp(suffix=".mp3")

        logger.info(f"Merging {len(segments)} audio segments -> {output_path}")

        # 创建 FFmpeg concat 文件列表
        concat_file = tempfile.mktemp(suffix=".txt")
        with open(concat_file, "w") as f:
            for seg in segments:
                # 格式: file 'path'
                f.write(f"file '{seg['path']}'\n")

        # 合并音频
        cmd = [
            "ffmpeg",
            "-f",
            "concat",
            "-safe",
            "0",
            "-i",
            concat_file,
            "-c",
            "copy",  # 复制编码（快速）
            "-y",
            output_path,
        ]

        try:
            subprocess.run(
                cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )
            logger.info(f"Audio segments merged: {output_path}")
            return output_path
        except subprocess.CalledProcessError as e:
            logger.error(f"FFmpeg merge failed: {e.stderr.decode()}")
            raise RuntimeError(f"Audio merge failed: {e.stderr.decode()}")
        finally:
            # 清理临时文件
            Path(concat_file).unlink(missing_ok=True)

    def replace_audio(
        self, video_path: str, audio_path: str, output_path: Optional[str] = None
    ) -> str:
        """
        替换视频的音轨

        Args:
            video_path: 原视频文件路径
            audio_path: 新音频文件路径
            output_path: 输出视频路径（可选）

        Returns:
            输出视频文件路径

        Raises:
            RuntimeError: FFmpeg 执行失败
        """
        if not output_path:
            output_path = str(
                Path(video_path).parent / f"{Path(video_path).stem}_dubbed.mp4"
            )

        logger.info(f"Replacing audio: video={video_path}, audio={audio_path}")

        cmd = [
            "ffmpeg",
            "-i",
            video_path,  # 输入视频
            "-i",
            audio_path,  # 输入音频
            "-c:v",
            "copy",  # 复制视频流（不重新编码）
            "-c:a",
            "aac",  # 音频编码为 AAC
            "-b:a",
            "192k",  # 音频比特率
            "-map",
            "0:v:0",  # 映射第一个输入的视频流
            "-map",
            "1:a:0",  # 映射第二个输入的音频流
            "-shortest",  # 使用最短流的长度
            "-y",
            output_path,
        ]

        try:
            subprocess.run(
                cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )
            logger.info(f"Audio replaced: {output_path}")
            return output_path
        except subprocess.CalledProcessError as e:
            logger.error(f"FFmpeg replace failed: {e.stderr.decode()}")
            raise RuntimeError(f"Audio replacement failed: {e.stderr.decode()}")

    def adjust_audio_speed(
        self, audio_path: str, speed_factor: float, output_path: Optional[str] = None
    ) -> str:
        """
        调整音频速度（不改变音调）

        Args:
            audio_path: 输入音频路径
            speed_factor: 速度倍数（0.5 = 慢一半，2.0 = 快一倍）
            output_path: 输出音频路径（可选）

        Returns:
            输出音频文件路径

        Raises:
            RuntimeError: FFmpeg 执行失败
        """
        if not output_path:
            output_path = tempfile.mktemp(suffix=".mp3")

        logger.info(f"Adjusting audio speed: {speed_factor}x")

        cmd = [
            "ffmpeg",
            "-i",
            audio_path,
            "-filter:a",
            f"atempo={speed_factor}",  # 时间拉伸
            "-vn",
            "-y",
            output_path,
        ]

        try:
            subprocess.run(
                cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )
            logger.info(f"Audio speed adjusted: {output_path}")
            return output_path
        except subprocess.CalledProcessError as e:
            logger.error(f"FFmpeg speed adjust failed: {e.stderr.decode()}")
            raise RuntimeError(f"Audio speed adjustment failed: {e.stderr.decode()}")

    def extract_segment(
        self,
        media_path: str,
        start_ms: int,
        end_ms: int,
        output_path: Optional[str] = None,
    ) -> str:
        """
        提取媒体片段

        Args:
            media_path: 媒体文件路径
            start_ms: 开始时间（毫秒）
            end_ms: 结束时间（毫秒）
            output_path: 输出文件路径（可选）

        Returns:
            输出文件路径

        Raises:
            RuntimeError: FFmpeg 执行失败
        """
        if not output_path:
            suffix = Path(media_path).suffix
            output_path = tempfile.mktemp(suffix=suffix)

        start_sec = start_ms / 1000
        duration_sec = (end_ms - start_ms) / 1000

        cmd = [
            "ffmpeg",
            "-ss",
            str(start_sec),
            "-i",
            media_path,
            "-t",
            str(duration_sec),
            "-c",
            "copy",  # 复制编码（快速）
            "-y",
            output_path,
        ]

        try:
            subprocess.run(
                cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )
            logger.info(
                f"Extracted segment: {start_ms}-{end_ms}ms -> {output_path}"
            )
            return output_path
        except subprocess.CalledProcessError as e:
            logger.error(f"FFmpeg extract failed: {e.stderr.decode()}")
            raise RuntimeError(f"Segment extraction failed: {e.stderr.decode()}")
