"""
Task Pydantic schemas
"""

from datetime import datetime
from typing import Optional
from uuid import UUID

from pydantic import BaseModel, Field, ConfigDict

from app.models.task import TaskStatus, SubtitleMode
from .segment import SegmentResponse


class TaskBase(BaseModel):
    """Task 基础 schema"""

    title: Optional[str] = Field(None, max_length=255, description="任务标题")
    source_language: str = Field(..., max_length=10, description="源语言代码，如 zh, en")
    target_language: str = Field(..., max_length=10, description="目标语言代码")


class TaskCreate(TaskBase):
    """创建任务请求"""

    subtitle_mode: SubtitleMode = Field(
        default=SubtitleMode.BURN,
        description="字幕模式: burn=烧录到视频(推荐,默认), external=外挂字幕文件, none=不生成"
    )


class TaskUpdate(BaseModel):
    """更新任务请求"""

    title: Optional[str] = Field(None, max_length=255)
    status: Optional[TaskStatus] = None
    current_step: Optional[str] = None
    progress: Optional[int] = Field(None, ge=0, le=100)
    error_message: Optional[str] = None


class TaskResponse(TaskBase):
    """任务响应（列表）"""

    model_config = ConfigDict(from_attributes=True)

    id: UUID
    status: TaskStatus
    subtitle_mode: SubtitleMode = Field(default=SubtitleMode.BURN, description="字幕模式")
    progress: int = Field(..., ge=0, le=100, description="进度百分比")
    current_step: Optional[str] = None
    error_message: Optional[str] = None
    segment_count: int = Field(default=0, description="分段数量")
    created_at: datetime
    updated_at: datetime
    completed_at: Optional[datetime] = None


class TaskDetail(TaskResponse):
    """任务详情（包含分段）"""

    video_duration_ms: Optional[int] = Field(None, description="视频时长（毫秒）")
    input_video_path: Optional[str] = None
    extracted_audio_path: Optional[str] = None
    output_video_path: Optional[str] = None
    subtitle_file_path: Optional[str] = Field(None, description="字幕文件 OSS 路径")
    celery_task_id: Optional[str] = None
    segments: list[SegmentResponse] = Field(default_factory=list, description="分段列表")


class TaskListResponse(BaseModel):
    """任务列表响应"""

    items: list[TaskResponse]
    total: int
    page: int
    page_size: int
    total_pages: int
