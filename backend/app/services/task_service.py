"""
任务服务层
"""

from typing import Optional
from uuid import UUID

from loguru import logger
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models import Task, TaskStatus, Segment
from app.schemas import TaskCreate, TaskUpdate


class TaskService:
    """任务服务"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def create_task(self, task_data: TaskCreate) -> Task:
        """
        创建任务

        Args:
            task_data: 任务创建数据

        Returns:
            创建的任务对象
        """
        task = Task(
            title=task_data.title,
            source_language=task_data.source_language,
            target_language=task_data.target_language,
            status=TaskStatus.PENDING,
            progress=0,
        )

        self.db.add(task)
        await self.db.commit()
        await self.db.refresh(task)

        logger.info(f"Task created: id={task.id}, languages={task.source_language}->{task.target_language}")

        return task

    async def get_task(self, task_id: UUID, with_segments: bool = False) -> Optional[Task]:
        """
        获取任务

        Args:
            task_id: 任务 ID
            with_segments: 是否加载分段数据

        Returns:
            任务对象，不存在则返回 None
        """
        query = select(Task).where(Task.id == task_id)

        if with_segments:
            query = query.options(selectinload(Task.segments))

        result = await self.db.execute(query)
        task = result.scalar_one_or_none()

        return task

    async def list_tasks(
        self,
        page: int = 1,
        page_size: int = 20,
        status: Optional[TaskStatus] = None,
    ) -> tuple[list[Task], int]:
        """
        获取任务列表

        Args:
            page: 页码（从 1 开始）
            page_size: 每页数量
            status: 状态过滤（可选）

        Returns:
            (任务列表, 总数量)
        """
        # 构建查询
        query = select(Task)

        if status:
            query = query.where(Task.status == status)

        # 排序
        query = query.order_by(Task.created_at.desc())

        # 获取总数
        count_query = select(func.count()).select_from(Task)
        if status:
            count_query = count_query.where(Task.status == status)

        total = await self.db.scalar(count_query) or 0

        # 分页
        offset = (page - 1) * page_size
        query = query.offset(offset).limit(page_size)

        result = await self.db.execute(query)
        tasks = list(result.scalars().all())

        logger.info(f"Listed {len(tasks)} tasks (page={page}, total={total})")

        return tasks, total

    async def update_task(
        self, task_id: UUID, task_update: TaskUpdate
    ) -> Optional[Task]:
        """
        更新任务

        Args:
            task_id: 任务 ID
            task_update: 更新数据

        Returns:
            更新后的任务对象，不存在则返回 None
        """
        task = await self.get_task(task_id)
        if not task:
            return None

        # 更新字段
        update_data = task_update.model_dump(exclude_unset=True)
        for field, value in update_data.items():
            setattr(task, field, value)

        await self.db.commit()
        await self.db.refresh(task)

        logger.info(f"Task updated: id={task_id}, fields={list(update_data.keys())}")

        return task

    async def update_task_status(
        self,
        task_id: UUID,
        status: TaskStatus,
        current_step: Optional[str] = None,
        progress: Optional[int] = None,
        error_message: Optional[str] = None,
    ) -> Optional[Task]:
        """
        更新任务状态

        Args:
            task_id: 任务 ID
            status: 新状态
            current_step: 当前步骤
            progress: 进度
            error_message: 错误信息

        Returns:
            更新后的任务对象
        """
        task = await self.get_task(task_id)
        if not task:
            return None

        task.status = status

        if current_step is not None:
            task.current_step = current_step

        if progress is not None:
            task.progress = progress

        if error_message is not None:
            task.error_message = error_message

        # 如果完成或失败，记录完成时间
        if status in (TaskStatus.COMPLETED, TaskStatus.FAILED):
            from datetime import datetime

            task.completed_at = datetime.utcnow()

        await self.db.commit()
        await self.db.refresh(task)

        logger.info(f"Task status updated: id={task_id}, status={status.value}, progress={progress}%")

        return task

    async def delete_task(self, task_id: UUID) -> bool:
        """
        删除任务

        Args:
            task_id: 任务 ID

        Returns:
            是否删除成功
        """
        task = await self.get_task(task_id)
        if not task:
            return False

        await self.db.delete(task)
        await self.db.commit()

        logger.info(f"Task deleted: id={task_id}")

        return True

    async def create_segment(
        self,
        task_id: UUID,
        segment_index: int,
        start_time_ms: int,
        end_time_ms: int,
        original_text: Optional[str] = None,
        speaker_id: Optional[str] = None,
        emotion: Optional[str] = None,
        confidence: Optional[float] = None,
    ) -> Segment:
        """
        创建分段

        Args:
            task_id: 任务 ID
            segment_index: 分段索引
            start_time_ms: 开始时间
            end_time_ms: 结束时间
            original_text: 原始文本
            speaker_id: 说话人 ID
            emotion: 情感
            confidence: 置信度

        Returns:
            创建的分段对象
        """
        segment = Segment(
            task_id=task_id,
            segment_index=segment_index,
            start_time_ms=start_time_ms,
            end_time_ms=end_time_ms,
            original_text=original_text,
            speaker_id=speaker_id,
            emotion=emotion,
            confidence=confidence,
        )

        self.db.add(segment)
        await self.db.commit()
        await self.db.refresh(segment)

        return segment

    async def update_segment_translation(
        self, segment_id: UUID, translated_text: str
    ) -> Optional[Segment]:
        """
        更新分段翻译

        Args:
            segment_id: 分段 ID
            translated_text: 翻译文本

        Returns:
            更新后的分段对象
        """
        query = select(Segment).where(Segment.id == segment_id)
        result = await self.db.execute(query)
        segment = result.scalar_one_or_none()

        if not segment:
            return None

        segment.translated_text = translated_text
        await self.db.commit()
        await self.db.refresh(segment)

        return segment

    async def update_segment_audio(
        self, segment_id: UUID, audio_path: str
    ) -> Optional[Segment]:
        """
        更新分段音频路径

        Args:
            segment_id: 分段 ID
            audio_path: 音频文件路径

        Returns:
            更新后的分段对象
        """
        query = select(Segment).where(Segment.id == segment_id)
        result = await self.db.execute(query)
        segment = result.scalar_one_or_none()

        if not segment:
            return None

        segment.audio_path = audio_path
        await self.db.commit()
        await self.db.refresh(segment)

        return segment
