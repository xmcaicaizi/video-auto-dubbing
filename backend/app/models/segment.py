"""
Segment 数据库模型
"""

from datetime import datetime
from typing import Optional
from uuid import uuid4

from sqlalchemy import String, Integer, Text, DateTime, Float, ForeignKey, Index
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base


class Segment(Base):
    """语音分段数据"""

    __tablename__ = "segments"

    # 主键
    id: Mapped[UUID] = mapped_column(
        UUID(as_uuid=True), primary_key=True, default=uuid4, index=True
    )

    # 外键
    task_id: Mapped[UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("tasks.id", ondelete="CASCADE"), nullable=False, index=True
    )

    # 顺序
    segment_index: Mapped[int] = mapped_column(Integer, nullable=False)

    # 时间信息（毫秒）
    start_time_ms: Mapped[int] = mapped_column(Integer, nullable=False)
    end_time_ms: Mapped[int] = mapped_column(Integer, nullable=False)

    # 文本内容
    original_text: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    translated_text: Mapped[Optional[str]] = mapped_column(Text, nullable=True)

    # ASR 元数据
    speaker_id: Mapped[Optional[str]] = mapped_column(String(50), nullable=True)
    emotion: Mapped[Optional[str]] = mapped_column(String(20), nullable=True)
    confidence: Mapped[Optional[float]] = mapped_column(Float, nullable=True)

    # TTS 音频路径 (OSS 相对路径)
    audio_path: Mapped[Optional[str]] = mapped_column(String(500), nullable=True)

    # 时间戳
    created_at: Mapped[datetime] = mapped_column(
        DateTime, nullable=False, default=datetime.utcnow
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, nullable=False, default=datetime.utcnow, onupdate=datetime.utcnow
    )

    # 关联关系
    task: Mapped["Task"] = relationship("Task", back_populates="segments")

    # 索引
    __table_args__ = (
        Index("idx_task_segment", "task_id", "segment_index", unique=True),
    )

    def __repr__(self) -> str:
        return (
            f"<Segment(id={self.id}, task_id={self.task_id}, "
            f"index={self.segment_index}, time={self.start_time_ms}-{self.end_time_ms}ms)>"
        )

    @property
    def duration_ms(self) -> int:
        """分段时长（毫秒）"""
        return self.end_time_ms - self.start_time_ms
