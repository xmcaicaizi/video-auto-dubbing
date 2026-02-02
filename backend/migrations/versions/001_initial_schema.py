"""Initial schema

Revision ID: 001
Revises:
Create Date: 2026-02-02 16:00:00.000000

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision: str = '001'
down_revision: Union[str, None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # Create tasks table
    op.create_table('tasks',
    sa.Column('id', postgresql.UUID(as_uuid=True), nullable=False),
    sa.Column('title', sa.String(length=255), nullable=True),
    sa.Column('source_language', sa.String(length=10), nullable=False),
    sa.Column('target_language', sa.String(length=10), nullable=False),
    sa.Column('status', sa.Enum('PENDING', 'EXTRACTING', 'TRANSCRIBING', 'TRANSLATING', 'SYNTHESIZING', 'MUXING', 'COMPLETED', 'FAILED', name='taskstatus'), nullable=False),
    sa.Column('current_step', sa.String(length=20), nullable=True),
    sa.Column('progress', sa.Integer(), nullable=False),
    sa.Column('error_message', sa.Text(), nullable=True),
    sa.Column('input_video_path', sa.String(length=500), nullable=True),
    sa.Column('extracted_audio_path', sa.String(length=500), nullable=True),
    sa.Column('output_video_path', sa.String(length=500), nullable=True),
    sa.Column('video_duration_ms', sa.Integer(), nullable=True),
    sa.Column('segment_count', sa.Integer(), nullable=False),
    sa.Column('created_at', sa.DateTime(), nullable=False),
    sa.Column('updated_at', sa.DateTime(), nullable=False),
    sa.Column('completed_at', sa.DateTime(), nullable=True),
    sa.Column('celery_task_id', sa.String(length=100), nullable=True),
    sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_tasks_id'), 'tasks', ['id'], unique=False)
    op.create_index(op.f('ix_tasks_status'), 'tasks', ['status'], unique=False)
    op.create_index(op.f('ix_tasks_created_at'), 'tasks', ['created_at'], unique=False)

    # Create segments table
    op.create_table('segments',
    sa.Column('id', postgresql.UUID(as_uuid=True), nullable=False),
    sa.Column('task_id', postgresql.UUID(as_uuid=True), nullable=False),
    sa.Column('segment_index', sa.Integer(), nullable=False),
    sa.Column('start_time_ms', sa.Integer(), nullable=False),
    sa.Column('end_time_ms', sa.Integer(), nullable=False),
    sa.Column('original_text', sa.Text(), nullable=True),
    sa.Column('translated_text', sa.Text(), nullable=True),
    sa.Column('speaker_id', sa.String(length=50), nullable=True),
    sa.Column('emotion', sa.String(length=20), nullable=True),
    sa.Column('confidence', sa.Float(), nullable=True),
    sa.Column('audio_path', sa.String(length=500), nullable=True),
    sa.Column('created_at', sa.DateTime(), nullable=False),
    sa.Column('updated_at', sa.DateTime(), nullable=False),
    sa.ForeignKeyConstraint(['task_id'], ['tasks.id'], ondelete='CASCADE'),
    sa.PrimaryKeyConstraint('id')
    )
    op.create_index(op.f('ix_segments_id'), 'segments', ['id'], unique=False)
    op.create_index(op.f('ix_segments_task_id'), 'segments', ['task_id'], unique=False)
    op.create_index('idx_task_segment', 'segments', ['task_id', 'segment_index'], unique=True)


def downgrade() -> None:
    op.drop_index('idx_task_segment', table_name='segments')
    op.drop_index(op.f('ix_segments_task_id'), table_name='segments')
    op.drop_index(op.f('ix_segments_id'), table_name='segments')
    op.drop_table('segments')
    op.drop_index(op.f('ix_tasks_created_at'), table_name='tasks')
    op.drop_index(op.f('ix_tasks_status'), table_name='tasks')
    op.drop_index(op.f('ix_tasks_id'), table_name='tasks')
    op.drop_table('tasks')
    op.execute('DROP TYPE taskstatus')
