"""
Task 数据库模型
"""

import enum
from datetime import datetime
from typing import Optional
from uuid import uuid4

from sqlalchemy import String, Integer, Text, DateTime, Enum, Boolean
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base


class TaskStatus(str, enum.Enum):
    """任务状态枚举"""

    PENDING = "pending"  # 等待处理
    EXTRACTING = "extracting"  # 提取音频中
    TRANSCRIBING = "transcribing"  # 语音识别中
    TRANSLATING = "translating"  # 翻译中
    SYNTHESIZING = "synthesizing"  # 语音合成中
    MUXING = "muxing"  # 视频合成中
    COMPLETED = "completed"  # 已完成
    FAILED = "failed"  # 失败


class SubtitleMode(str, enum.Enum):
    """字幕模式"""

    NONE = "NONE"              # 不生成字幕
    EXTERNAL = "EXTERNAL"      # 外挂字幕文件
    BURN = "BURN"              # 烧录到视频中


class Task(Base):
    """视频配音任务"""

    __tablename__ = "tasks"

    # 主键
    id: Mapped[UUID] = mapped_column(
        UUID(as_uuid=True), primary_key=True, default=uuid4, index=True
    )

    # 基本信息
    title: Mapped[Optional[str]] = mapped_column(String(255), nullable=True)
    source_language: Mapped[str] = mapped_column(String(10), nullable=False)
    target_language: Mapped[str] = mapped_column(String(10), nullable=False)

    # 状态信息
    status: Mapped[TaskStatus] = mapped_column(
        Enum(TaskStatus), nullable=False, default=TaskStatus.PENDING, index=True
    )
    current_step: Mapped[Optional[str]] = mapped_column(String(20), nullable=True)
    progress: Mapped[int] = mapped_column(Integer, nullable=False, default=0)  # 0-100
    error_message: Mapped[Optional[str]] = mapped_column(Text, nullable=True)

    # 字幕配置
    subtitle_mode: Mapped[SubtitleMode] = mapped_column(
        Enum(SubtitleMode), nullable=False, default=SubtitleMode.BURN
    )
    burn_subtitles: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)

    # 文件路径 (OSS 相对路径)
    input_video_path: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    extracted_audio_path: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    output_video_path: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)
    subtitle_file_path: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)

    # 元数据
    video_duration_ms: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    segment_count: Mapped[int] = mapped_column(Integer, nullable=False, default=0)

    # 时间戳
    created_at: Mapped[datetime] = mapped_column(
        DateTime, nullable=False, default=datetime.utcnow, index=True
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, nullable=False, default=datetime.utcnow, onupdate=datetime.utcnow
    )
    completed_at: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)

    # Celery 任务 ID
    celery_task_id: Mapped[Optional[str]] = mapped_column(String(100), nullable=True)

    # 关联关系
    segments: Mapped[list["Segment"]] = relationship(
        "Segment", back_populates="task", cascade="all, delete-orphan"
    )

    def __repr__(self) -> str:
        return (
            f"<Task(id={self.id}, status={self.status.value}, "
            f"progress={self.progress}%, created_at={self.created_at})>"
        )

    @property
    def is_completed(self) -> bool:
        """任务是否完成"""
        return self.status == TaskStatus.COMPLETED

    @property
    def is_failed(self) -> bool:
        """任务是否失败"""
        return self.status == TaskStatus.FAILED

    @property
    def is_processing(self) -> bool:
        """任务是否正在处理"""
        return self.status not in (TaskStatus.PENDING, TaskStatus.COMPLETED, TaskStatus.FAILED)
