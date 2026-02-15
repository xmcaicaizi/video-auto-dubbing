"""
阿里云 OSS 客户端封装
提供文件上传、下载、签名 URL 生成等功能
"""

import os
from pathlib import Path
from typing import BinaryIO, Optional
from urllib.parse import urljoin

import oss2
from loguru import logger

from app.config import settings


class OSSClient:
    """阿里云 OSS 客户端"""

    def __init__(
        self,
        endpoint: Optional[str] = None,
        bucket_name: Optional[str] = None,
        access_key_id: Optional[str] = None,
        access_key_secret: Optional[str] = None,
        public_domain: Optional[str] = None,
        prefix: Optional[str] = None,
    ):
        """
        初始化 OSS 客户端

        Args:
            endpoint: OSS 节点地址，如 oss-cn-beijing.aliyuncs.com
            bucket_name: Bucket 名称
            access_key_id: AccessKey ID
            access_key_secret: AccessKey Secret
            public_domain: 公网访问域名（可选，用于生成公网 URL）
            prefix: 文件路径前缀，如 'videos/'
        """
        self.endpoint = endpoint or settings.oss_endpoint
        self.bucket_name = bucket_name or settings.oss_bucket
        self.access_key_id = access_key_id or settings.oss_access_key_id
        self.access_key_secret = access_key_secret or settings.oss_access_key_secret
        self.public_domain = public_domain or settings.oss_public_domain
        self.prefix = (prefix or settings.oss_prefix).rstrip("/")

        # 验证必要配置
        if not all([self.endpoint, self.bucket_name, self.access_key_id, self.access_key_secret]):
            raise ValueError("OSS configuration incomplete. Please check environment variables.")

        # 创建认证对象
        self.auth = oss2.Auth(self.access_key_id, self.access_key_secret)

        # 创建 Bucket 对象（确保使用 HTTPS，增加超时以支持跨区域大文件传输）
        endpoint = self.endpoint
        if not endpoint.startswith("http"):
            endpoint = f"https://{endpoint}" if settings.oss_use_ssl else f"http://{endpoint}"
        self.bucket = oss2.Bucket(
            self.auth, endpoint, self.bucket_name,
            connect_timeout=30,
        )
        # 设置全局超时（connect_timeout, read_timeout），大文件跨区域传输需要较长读取超时
        self.bucket.timeout = (30, 600)  # 连接 30s，读取 10 分钟

        logger.info(f"OSS Client initialized: endpoint={self.endpoint}, bucket={self.bucket_name}")

    def _build_key(self, path: str) -> str:
        """
        构建完整的 OSS key

        Args:
            path: 相对路径，如 'task_123/input.mp4'

        Returns:
            完整的 key，如 'videos/task_123/input.mp4'
        """
        path = path.lstrip("/")
        if self.prefix:
            return f"{self.prefix}/{path}"
        return path

    def upload_file(self, file_path: str, oss_path: str, content_type: Optional[str] = None) -> str:
        """
        上传本地文件到 OSS

        Args:
            file_path: 本地文件路径
            oss_path: OSS 中的目标路径（相对路径）
            content_type: 文件 MIME 类型（可选）

        Returns:
            OSS key

        Raises:
            FileNotFoundError: 文件不存在
            oss2.exceptions.OssError: 上传失败
        """
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"File not found: {file_path}")

        key = self._build_key(oss_path)

        # 自动检测 content_type
        if not content_type:
            content_type = self._guess_content_type(file_path)

        headers = {}
        if content_type:
            headers["Content-Type"] = content_type

        logger.info(f"Uploading file: {file_path} -> oss://{self.bucket_name}/{key}")

        try:
            result = self.bucket.put_object_from_file(key, file_path, headers=headers)
            logger.info(
                f"Upload success: {key}, "
                f"status={result.status}, "
                f"request_id={result.request_id}"
            )
            return key
        except oss2.exceptions.OssError as e:
            logger.error(f"Upload failed: {e}")
            raise

    def upload_bytes(
        self, data: bytes, oss_path: str, content_type: Optional[str] = None
    ) -> str:
        """
        上传字节数据到 OSS

        Args:
            data: 字节数据
            oss_path: OSS 中的目标路径（相对路径）
            content_type: 文件 MIME 类型（可选）

        Returns:
            OSS key
        """
        key = self._build_key(oss_path)

        headers = {}
        if content_type:
            headers["Content-Type"] = content_type

        logger.info(f"Uploading bytes: {len(data)} bytes -> oss://{self.bucket_name}/{key}")

        try:
            result = self.bucket.put_object(key, data, headers=headers)
            logger.info(
                f"Upload success: {key}, "
                f"status={result.status}, "
                f"request_id={result.request_id}"
            )
            return key
        except oss2.exceptions.OssError as e:
            logger.error(f"Upload failed: {e}")
            raise

    def upload_stream(
        self, stream: BinaryIO, oss_path: str, content_type: Optional[str] = None
    ) -> str:
        """
        上传文件流到 OSS

        Args:
            stream: 文件流对象
            oss_path: OSS 中的目标路径（相对路径）
            content_type: 文件 MIME 类型（可选）

        Returns:
            OSS key
        """
        key = self._build_key(oss_path)

        headers = {}
        if content_type:
            headers["Content-Type"] = content_type

        logger.info(f"Uploading stream -> oss://{self.bucket_name}/{key}")

        try:
            result = self.bucket.put_object(key, stream, headers=headers)
            logger.info(
                f"Upload success: {key}, "
                f"status={result.status}, "
                f"request_id={result.request_id}"
            )
            return key
        except oss2.exceptions.OssError as e:
            logger.error(f"Upload failed: {e}")
            raise

    def download_file(self, oss_path: str, local_path: str) -> str:
        """
        从 OSS 下载文件到本地

        Args:
            oss_path: OSS 中的文件路径（相对路径）
            local_path: 本地保存路径

        Returns:
            本地文件路径

        Raises:
            oss2.exceptions.NoSuchKey: 文件不存在
        """
        key = self._build_key(oss_path)

        # 确保本地目录存在
        local_dir = os.path.dirname(local_path)
        if local_dir:
            os.makedirs(local_dir, exist_ok=True)

        logger.info(f"Downloading file: oss://{self.bucket_name}/{key} -> {local_path}")

        try:
            result = self.bucket.get_object_to_file(key, local_path)
            logger.info(
                f"Download success: {local_path}, "
                f"status={result.status}, "
                f"size={os.path.getsize(local_path)} bytes"
            )
            return local_path
        except oss2.exceptions.NoSuchKey:
            logger.error(f"File not found in OSS: {key}")
            raise
        except oss2.exceptions.OssError as e:
            logger.error(f"Download failed: {e}")
            raise

    def download_bytes(self, oss_path: str) -> bytes:
        """
        从 OSS 下载文件内容（字节）

        Args:
            oss_path: OSS 中的文件路径（相对路径）

        Returns:
            文件内容（bytes）
        """
        key = self._build_key(oss_path)

        logger.info(f"Downloading bytes: oss://{self.bucket_name}/{key}")

        try:
            result = self.bucket.get_object(key)
            data = result.read()
            logger.info(f"Download success: {len(data)} bytes")
            return data
        except oss2.exceptions.NoSuchKey:
            logger.error(f"File not found in OSS: {key}")
            raise
        except oss2.exceptions.OssError as e:
            logger.error(f"Download failed: {e}")
            raise

    def delete_file(self, oss_path: str) -> None:
        """
        删除 OSS 中的文件

        Args:
            oss_path: OSS 中的文件路径（相对路径）
        """
        key = self._build_key(oss_path)

        logger.info(f"Deleting file: oss://{self.bucket_name}/{key}")

        try:
            result = self.bucket.delete_object(key)
            logger.info(f"Delete success: {key}, status={result.status}")
        except oss2.exceptions.OssError as e:
            logger.error(f"Delete failed: {e}")
            raise

    def file_exists(self, oss_path: str) -> bool:
        """
        检查文件是否存在

        Args:
            oss_path: OSS 中的文件路径（相对路径）

        Returns:
            文件是否存在
        """
        key = self._build_key(oss_path)

        try:
            return self.bucket.object_exists(key)
        except oss2.exceptions.OssError as e:
            logger.error(f"Check file existence failed: {e}")
            return False

    def get_file_size(self, oss_path: str) -> int:
        """
        获取文件大小

        Args:
            oss_path: OSS 中的文件路径（相对路径）

        Returns:
            文件大小（字节）

        Raises:
            oss2.exceptions.NoSuchKey: 文件不存在
        """
        key = self._build_key(oss_path)

        try:
            meta = self.bucket.head_object(key)
            return meta.content_length
        except oss2.exceptions.NoSuchKey:
            logger.error(f"File not found in OSS: {key}")
            raise

    def generate_presigned_url(
        self,
        oss_path: str,
        expires: int = 3600,
        method: str = "GET",
        filename: Optional[str] = None,
    ) -> str:
        """
        生成预签名 URL（临时访问链接）

        Args:
            oss_path: OSS 中的文件路径（相对路径）
            expires: 过期时间（秒），默认 1 小时
            method: HTTP 方法，默认 'GET'
            filename: 下载时显示的文件名（可选，设置后强制下载而非在浏览器中打开）

        Returns:
            预签名 URL
        """
        key = self._build_key(oss_path)

        try:
            # 如果指定了文件名，设置 Content-Disposition 响应头强制下载
            headers = None
            if filename:
                headers = {
                    "response-content-disposition": f'attachment; filename="{filename}"'
                }

            url = self.bucket.sign_url(
                method, key, expires, slash_safe=True, headers=headers
            )
            logger.info(f"Generated presigned URL: {key}, expires_in={expires}s, filename={filename}")
            return url
        except oss2.exceptions.OssError as e:
            logger.error(f"Generate presigned URL failed: {e}")
            raise

    def get_public_url(self, oss_path: str) -> str:
        """
        获取公网访问 URL（需要 Bucket 设置为公共读或配置 CDN）

        Args:
            oss_path: OSS 中的文件路径（相对路径）

        Returns:
            公网 URL
        """
        key = self._build_key(oss_path)

        # 优先使用自定义域名
        if self.public_domain:
            base_url = f"https://{self.public_domain}" if settings.oss_use_ssl else f"http://{self.public_domain}"
            return urljoin(base_url, f"/{key}")

        # 使用默认 OSS 域名
        protocol = "https" if settings.oss_use_ssl else "http"
        return f"{protocol}://{self.bucket_name}.{self.endpoint}/{key}"

    def list_files(self, prefix: str = "", max_keys: int = 100) -> list[str]:
        """
        列出指定前缀的文件

        Args:
            prefix: 文件路径前缀（相对于配置的 prefix）
            max_keys: 最大返回数量

        Returns:
            文件路径列表
        """
        full_prefix = self._build_key(prefix) if prefix else self.prefix

        try:
            result = self.bucket.list_objects(prefix=full_prefix, max_keys=max_keys)
            files = [obj.key for obj in result.object_list]
            logger.info(f"Listed {len(files)} files with prefix: {full_prefix}")
            return files
        except oss2.exceptions.OssError as e:
            logger.error(f"List files failed: {e}")
            raise

    @staticmethod
    def _guess_content_type(file_path: str) -> Optional[str]:
        """根据文件扩展名猜测 MIME 类型"""
        ext = Path(file_path).suffix.lower()
        content_types = {
            ".mp4": "video/mp4",
            ".avi": "video/x-msvideo",
            ".mov": "video/quicktime",
            ".mkv": "video/x-matroska",
            ".flv": "video/x-flv",
            ".mp3": "audio/mpeg",
            ".wav": "audio/wav",
            ".m4a": "audio/mp4",
            ".json": "application/json",
            ".txt": "text/plain",
            ".jpg": "image/jpeg",
            ".jpeg": "image/jpeg",
            ".png": "image/png",
        }
        return content_types.get(ext)


# 全局单例
_oss_client: Optional[OSSClient] = None


def get_oss_client() -> OSSClient:
    """获取 OSS 客户端单例"""
    global _oss_client
    if _oss_client is None:
        _oss_client = OSSClient()
    return _oss_client
