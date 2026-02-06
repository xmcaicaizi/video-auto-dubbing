"""Add subtitle fields to tasks

Revision ID: 003
Revises: 002
Create Date: 2026-02-03 10:00:00.000000

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision: str = '003'
down_revision: Union[str, None] = '002'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None

subtitle_mode_enum = sa.Enum('none', 'external', 'burn', name='subtitlemode')


def upgrade() -> None:
    subtitle_mode_enum.create(op.get_bind(), checkfirst=True)
    op.add_column('tasks', sa.Column(
        'subtitle_mode', subtitle_mode_enum,
        nullable=False, server_default='external'
    ))
    op.add_column('tasks', sa.Column(
        'burn_subtitles', sa.Boolean(),
        nullable=False, server_default=sa.text('false')
    ))
    op.add_column('tasks', sa.Column(
        'subtitle_file_path', sa.String(length=500),
        nullable=True
    ))


def downgrade() -> None:
    op.drop_column('tasks', 'subtitle_file_path')
    op.drop_column('tasks', 'burn_subtitles')
    op.drop_column('tasks', 'subtitle_mode')
    subtitle_mode_enum.drop(op.get_bind(), checkfirst=True)
