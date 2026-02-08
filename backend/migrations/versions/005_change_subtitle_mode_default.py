"""change subtitle_mode default to BURN

Revision ID: 005
Revises: 004
Create Date: 2026-02-08 19:20:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '005'
down_revision = '004'
branch_labels = None
depends_on = None


def upgrade():
    # 修改 subtitle_mode 默认值为 BURN
    op.execute("ALTER TABLE tasks ALTER COLUMN subtitle_mode DROP DEFAULT")
    op.execute("ALTER TABLE tasks ALTER COLUMN subtitle_mode SET DEFAULT 'BURN'::subtitlemode")


def downgrade():
    # 恢复为 EXTERNAL
    op.execute("ALTER TABLE tasks ALTER COLUMN subtitle_mode DROP DEFAULT")
    op.execute("ALTER TABLE tasks ALTER COLUMN subtitle_mode SET DEFAULT 'EXTERNAL'::subtitlemode")
