"""
change table names

Revision ID: b7c249366b7a
Revises: 76f88c250081
Create Date: 2026-02-19 08:45:35.213001

"""

import sqlalchemy as sa
import sqlmodel.sql.sqltypes
from alembic import op
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision = "b7c249366b7a"
down_revision = "76f88c250081"
branch_labels = None
depends_on = None


RENAMES = [
    ("user", "users"),
    ("useridentifier", "user_identifiers"),
    ("game", "games"),
    ("project", "projects"),
    ("commit", "commits"),
    ("player", "players"),
    ("board", "boards"),
    ("language", "languages"),
    ("languageboard", "language_boards"),
    ("team", "teams"),
    ("teammember", "team_members"),
    ("teamboard", "team_boards"),
    ("report", "reports"),
    ("auditlog", "audit_logs"),
]


def upgrade():
    for old, new in RENAMES:
        op.rename_table(old, new)

    op.alter_column("user_identifiers", "primary", new_column_name="is_primary")
    op.alter_column("games", "end", new_column_name="ends_at")
    op.alter_column("games", "start", new_column_name="starts_at")


def downgrade():
    for old, new in reversed(RENAMES):
        op.rename_table(new, old)

    op.alter_column("useridentifier", "is_primary", new_column_name="primary")
    op.alter_column("game", "ends_at", new_column_name="end")
    op.alter_column("game", "starts_at", new_column_name="start")
