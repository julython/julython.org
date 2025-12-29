import uuid
from datetime import datetime
from typing import Any, Optional

from sqlalchemy import String, UniqueConstraint
from sqlmodel import SQLModel, Relationship, Field

from july.db.fields import (
    Array,
    CreatedAt,
    Email,
    FK,
    ID,
    Identifier,
    JsonbData,
    PrimaryKey,
    ShortString,
    Timestamp,
    UpdatedAt,
)
from july.schema import (
    AnalysisStatus,
    IdentifierType,
    Leader,
    LeaderBoard,
    ReportStatus,
    ReportType,
    UserRole,
)


# Base Model
class Base(SQLModel, table=False):
    id: uuid.UUID = PrimaryKey
    created_at: datetime = CreatedAt
    updated_at: datetime = UpdatedAt


class User(Base, table=True):
    name: str = ShortString(nullable=False)
    username: str = ShortString(length=25, nullable=True)
    avatar_url: Optional[str] = None
    role: UserRole = Field(sa_type=String(20), default=UserRole.USER)
    is_active: bool = Field(default=True, index=True)
    is_banned: bool = Field(default=False)
    banned_reason: Optional[str] = None
    banned_at: Optional[datetime] = Timestamp(nullable=True)
    last_seen: Optional[datetime] = Timestamp(nullable=True)

    identifiers: list["UserIdentifier"] = Relationship(
        back_populates="user",
        cascade_delete=True,
    )
    players: list["Player"] = Relationship(
        back_populates="user",
        cascade_delete=True,
    )


class UserIdentifier(SQLModel, table=True):
    value: str = Field(primary_key=True)
    type: IdentifierType = ShortString(length=10, nullable=False, index=True)
    created_at: datetime = CreatedAt
    updated_at: datetime = UpdatedAt
    user_id: uuid.UUID = FK("user.id", ondelete="CASCADE")
    verified: bool = Field(default=False)
    primary: bool = Field(default=False)
    data: Optional[dict[str, Any]] = JsonbData()

    user: User = Relationship(back_populates="identifiers")


class Game(Base, table=True):
    """Competition period (e.g., Julython 2024)"""

    name: str = ShortString(length=25, nullable=False)
    start: datetime = Timestamp(nullable=False, description="Start of the Game UTC-12")
    end: datetime = Timestamp(nullable=False, description="End of the Game UTC+12")
    commit_points: int = Field(default=1, description="points per commit")
    project_points: int = Field(default=10, description="points per project")
    is_active: bool = Field(default=False)


class Project(Base, table=True):
    """A GitHub repository being tracked"""

    url: str = Field(unique=True, index=True)
    name: str = ShortString(nullable=False)
    slug: str = Field(unique=True, index=True)
    description: Optional[str] = None
    repo_id: Optional[int] = Field(default=None, index=True)
    service: str = Field(sa_type=String(20), default=IdentifierType.GITHUB)
    forked: bool = Field(default=False)
    forks: int = Field(default=0)
    watchers: int = Field(default=0)
    parent_url: Optional[str] = None
    is_active: bool = Field(default=True, index=True)

    boards: list["Board"] = Relationship(
        back_populates="project",
        cascade_delete=True,
    )
    commits: list["Commit"] = Relationship(
        back_populates="project",
        cascade_delete=True,
    )

    __table_args__ = (
        UniqueConstraint("service", "repo_id", name="uq_project_service_repo"),
    )


class Commit(Base, table=True):
    """Individual commit from webhook"""

    project_id: uuid.UUID = FK(
        "project.id",
        index=True,
        ondelete="CASCADE",
    )
    user_id: uuid.UUID | None = FK(
        "user.id",
        index=True,
        nullable=True,
        ondelete="CASCADE",
    )
    game_id: uuid.UUID | None = FK("game.id", index=True, nullable=True)

    hash: str = Identifier(unique=True, index=True)
    author: str = ShortString()
    email: str = Email()
    message: str
    url: str
    timestamp: datetime = Timestamp(nullable=False)
    languages: list[str] = Array(description="programming languages")
    files: dict[str, Any] = JsonbData()

    # AI verification status
    is_verified: bool = Field(default=False)
    is_flagged: bool = Field(default=False)
    flag_reason: Optional[str] = None

    project: Project = Relationship(back_populates="commits")


class Player(Base, table=True):
    """User's participation in a specific game"""

    game_id: uuid.UUID = FK("game.id", index=True)
    user_id: uuid.UUID = FK(
        "user.id",
        index=True,
        ondelete="CASCADE",
    )

    points: int = Field(default=0)
    potential_points: int = Field(default=0)
    verified_points: int = Field(default=0)

    commit_count: int = Field(default=0)
    project_count: int = Field(default=0)

    analysis_status: AnalysisStatus = Field(
        sa_type=String(20), default=AnalysisStatus.PENDING
    )
    last_analyzed_at: Optional[datetime] = Timestamp(nullable=True)

    user: User = Relationship(back_populates="players")

    def to_leader(self, rank: int) -> Leader:
        return Leader(
            rank=rank,
            user_id=self.user_id,
            name=self.user.name,
            avatar_url=self.user.avatar_url,
            points=self.points,
            verified_points=self.verified_points,
            commit_count=self.commit_count,
            project_count=self.project_count,
        )


class Board(Base, table=True):
    """Project leaderboard per game"""

    game_id: uuid.UUID = FK("game.id", index=True)
    project_id: uuid.UUID = FK(
        "project.id",
        index=True,
        ondelete="CASCADE",
    )

    points: int = Field(default=0)
    potential_points: int = Field(default=0)
    verified_points: int = Field(default=0)

    commit_count: int = Field(default=0)
    contributor_count: int = Field(default=0)

    project: Project = Relationship(back_populates="boards")

    def to_leader(self, rank: int) -> LeaderBoard:
        return LeaderBoard(
            rank=rank,
            project_id=self.project_id,
            name=self.project.name,
            slug=self.project.slug,
            url=self.project.url,
            points=self.points,
            verified_points=self.verified_points,
            commit_count=self.commit_count,
            contributor_count=self.contributor_count,
        )


class PlayerBoard(Base, table=True):
    """Many-to-many: which projects a player has contributed to"""

    player_id: uuid.UUID = FK("player.id", index=True, ondelete="CASCADE")
    board_id: uuid.UUID = FK("board.id", index=True, ondelete="CASCADE")
    commit_count: int = Field(default=0)


class Language(Base, table=True):
    """Programming languages"""

    name: str = Field(unique=True, index=True)


class LanguageBoard(Base, table=True):
    """Language leaderboard per game"""

    game_id: uuid.UUID = FK("game.id", index=True)
    language_id: uuid.UUID = FK("language.id", index=True)
    points: int = Field(default=0)
    commit_count: int = Field(default=0)


# class RepoAnalysis(Base, table=True):
#     """AI analysis results for a user's repo during a game"""

#     user_id: uuid.UUID = FK("user.id", index=True)
#     project_id: uuid.UUID = FK("project.id", index=True)
#     game_id: uuid.UUID = FK("game.id", index=True)

#     analyzed_at: datetime
#     ai_model: str = Field(default="gpt-4")
#     status: AnalysisStatus = Field(sa_type=String(20), default=AnalysisStatus.PENDING)

#     # Score impacts
#     quality_score: float
#     authenticity_score: float = Field(default=100.0)
#     points_adjustment: int = Field(default=0)

#     # AI insights
#     quality_reasoning: Optional[str] = None
#     authenticity_reasoning: Optional[str] = None
#     dev_style: Optional[str] = None
#     style_reasoning: Optional[str] = None
#     ai_insights: Optional[str] = None
#     red_flags: Optional[str] = None
#     key_strengths: Optional[str] = None
#     recommendations: Optional[str] = None

#     # Statistics
#     commit_count: int
#     date_range_start: datetime
#     date_range_end: datetime
#     streak_days: Optional[int] = None
#     avg_commits_per_day: Optional[float] = None
#     ghost_commit_ratio: Optional[float] = None

#     # Patterns
#     commit_patterns: Optional[str] = None
#     language_breakdown: Optional[str] = None
#     peak_activity_hours: Optional[str] = None
#     consistency_score: Optional[float] = None

#     # Visibility and moderation
#     is_public: bool = Field(default=True)
#     is_flagged: bool = Field(default=False)
#     is_removed: bool = Field(default=False)
#     removed_reason: Optional[str] = None
#     removed_at: Optional[datetime] = Timestamp(nullable=True)

#     # Social
#     likes_count: int = Field(default=0)
#     reports_count: int = Field(default=0)


# Teams
class Team(Base, table=True):
    name: str = Field(unique=True, index=True)
    slug: str = Field(unique=True, index=True)
    description: Optional[str] = None
    avatar_url: Optional[str] = None
    created_by: uuid.UUID = FK("user.id", ondelete="CASCADE")
    is_public: bool = Field(default=True)
    member_count: int = Field(default=0)


class TeamMember(Base, table=True):
    team_id: uuid.UUID = FK("team.id", index=True)
    user_id: uuid.UUID = FK("user.id", index=True, ondelete="CASCADE")
    role: str = Field(default="member")


class TeamBoard(Base, table=True):
    """Team leaderboard per game"""

    game_id: uuid.UUID = FK("game.id", index=True)
    team_id: uuid.UUID = FK("team.id", index=True, ondelete="CASCADE")
    points: int = Field(default=0)
    member_count: int = Field(default=0)


class Report(Base, table=True):
    reported_user_id: Optional[uuid.UUID] = FK("user.id", index=True, nullable=True)
    report_type: ReportType = Field(sa_type=String(20), default=ReportType.SPAM)
    reason: str = ShortString(description="Short reason for reporting 'other' type")
    status: ReportStatus = Field(sa_type=String(20), default=ReportStatus.PENDING)
    reviewed_by: Optional[uuid.UUID] = Field(default=None, foreign_key="user.id")
    reviewed_at: Optional[datetime] = Timestamp(nullable=True)
    moderator_notes: Optional[str] = None


class AuditLog(Base, table=True):
    moderator_id: uuid.UUID = FK("user.id", index=True)
    action: str
    target_type: str
    target_id: str
    reason: Optional[str] = None
