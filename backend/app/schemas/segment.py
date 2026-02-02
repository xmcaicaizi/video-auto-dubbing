"""
Segment Pydantic schemas
"""

from datetime import datetime
from typing import Optional
from uuid import UUID

from pydantic import BaseModel, Field, ConfigDict


class SegmentBase(BaseModel):
    """Segment 基础 schema"""

    segment_index: int = Field(..., ge=0, description="分段索引")
    start_time_ms: int = Field(..., ge=0, description="开始时间（毫秒）")
    end_time_ms: int = Field(..., gt=0, description="结束时间（毫秒）")
    original_text: Optional[str] = Field(None, description="原始文本")
    translated_text: Optional[str] = Field(None, description="翻译文本")
    speaker_id: Optional[str] = Field(None, max_length=50, description="说话人 ID")
    emotion: Optional[str] = Field(None, max_length=20, description="情感标签")
    confidence: Optional[float] = Field(None, ge=0, le=1, description="置信度")


class SegmentCreate(SegmentBase):
    """创建分段请求"""

    task_id: UUID


class SegmentUpdate(BaseModel):
    """更新分段请求"""

    translated_text: Optional[str] = None
    audio_path: Optional[str] = None


class SegmentResponse(SegmentBase):
    """分段响应"""

    model_config = ConfigDict(from_attributes=True)

    id: UUID
    task_id: UUID
    audio_path: Optional[str] = None
    created_at: datetime
    updated_at: datetime

    @property
    def duration_ms(self) -> int:
        """分段时长（毫秒）"""
        return self.end_time_ms - self.start_time_ms
