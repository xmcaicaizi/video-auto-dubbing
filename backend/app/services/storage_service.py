"""
存储服务层
封装 OSS 操作，提供业务级接口
"""

import os
import tempfile
from pathlib import Path
from typing import BinaryIO, Optional
from uuid import UUID

from loguru import logger

from app.integrations.oss import get_oss_client


class StorageService:
    """存储服务"""

    def __init__(self):
        self.oss = get_oss_client()

    def build_task_path(self, task_id: UUID, filename: str) -> str:
        """
        构建任务文件路径

        Args:
            task_id: 任务 ID
            filename: 文件名

        Returns:
            相对路径，如 'task_xxx/input.mp4'
        """
        return f"task_{task_id}/{filename}"

    def upload_input_video(
        self, task_id: UUID, file_data: BinaryIO, filename: str
    ) -> str:
        """
        上传输入视频

        Args:
            task_id: 任务 ID
            file_data: 文件数据流
            filename: 原始文件名

        Returns:
            OSS 相对路径
        """
        # 保留原始文件扩展名
        ext = Path(filename).suffix
        oss_path = self.build_task_path(task_id, f"input{ext}")

        # 上传
        self.oss.upload_stream(file_data, oss_path, content_type=self._get_video_content_type(ext))

        logger.info(f"Uploaded input video: task_id={task_id}, path={oss_path}")

        return oss_path

    def upload_extracted_audio(
        self, task_id: UUID, audio_file: str
    ) -> str:
        """
        上传提取的音频

        Args:
            task_id: 任务 ID
            audio_file: 本地音频文件路径

        Returns:
            OSS 相对路径
        """
        oss_path = self.build_task_path(task_id, "extracted_audio.wav")
        self.oss.upload_file(audio_file, oss_path, content_type="audio/wav")

        logger.info(f"Uploaded extracted audio: task_id={task_id}, path={oss_path}")

        return oss_path

    def upload_segment_audio(
        self, task_id: UUID, segment_index: int, audio_data: bytes
    ) -> str:
        """
        上传分段音频

        Args:
            task_id: 任务 ID
            segment_index: 分段索引
            audio_data: 音频数据

        Returns:
            OSS 相对路径
        """
        oss_path = self.build_task_path(task_id, f"segments/segment_{segment_index:04d}.mp3")
        self.oss.upload_bytes(audio_data, oss_path, content_type="audio/mpeg")

        logger.info(f"Uploaded segment audio: task_id={task_id}, index={segment_index}")

        return oss_path

    def upload_output_video(
        self, task_id: UUID, video_file: str
    ) -> str:
        """
        上传输出视频

        Args:
            task_id: 任务 ID
            video_file: 本地视频文件路径

        Returns:
            OSS 相对路径
        """
        oss_path = self.build_task_path(task_id, "output.mp4")
        self.oss.upload_file(video_file, oss_path, content_type="video/mp4")

        logger.info(f"Uploaded output video: task_id={task_id}, path={oss_path}")

        return oss_path

    def download_file(self, oss_path: str, local_dir: Optional[str] = None) -> str:
        """
        下载文件到本地

        Args:
            oss_path: OSS 路径
            local_dir: 本地目录（可选，默认使用临时目录）

        Returns:
            本地文件路径
        """
        if local_dir:
            os.makedirs(local_dir, exist_ok=True)
            local_path = os.path.join(local_dir, Path(oss_path).name)
        else:
            # 使用临时目录
            temp_dir = tempfile.gettempdir()
            local_path = os.path.join(temp_dir, Path(oss_path).name)

        self.oss.download_file(oss_path, local_path)

        return local_path

    def get_download_url(self, oss_path: str, expires: int = 3600) -> str:
        """
        生成下载链接

        Args:
            oss_path: OSS 路径
            expires: 过期时间（秒）

        Returns:
            下载 URL
        """
        return self.oss.generate_presigned_url(oss_path, expires)

    def get_public_url(self, oss_path: str) -> str:
        """
        获取公网 URL

        Args:
            oss_path: OSS 路径

        Returns:
            公网 URL
        """
        return self.oss.get_public_url(oss_path)

    def delete_task_files(self, task_id: UUID) -> None:
        """
        删除任务所有文件

        Args:
            task_id: 任务 ID
        """
        prefix = f"task_{task_id}/"
        files = self.oss.list_files(prefix)

        for file_path in files:
            try:
                self.oss.delete_file(file_path)
            except Exception as e:
                logger.error(f"Failed to delete file {file_path}: {e}")

        logger.info(f"Deleted {len(files)} files for task_id={task_id}")

    @staticmethod
    def _get_video_content_type(ext: str) -> str:
        """根据扩展名获取视频 MIME 类型"""
        content_types = {
            ".mp4": "video/mp4",
            ".avi": "video/x-msvideo",
            ".mov": "video/quicktime",
            ".mkv": "video/x-matroska",
            ".flv": "video/x-flv",
        }
        return content_types.get(ext.lower(), "video/mp4")
