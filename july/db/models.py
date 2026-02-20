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
    __tablename__ = "users"  # type: ignore

    name: str = ShortString(nullable=False)
    username: str = ShortString(length=25, nullable=False, unique=True)
    avatar_url: Optional[str] = None
    role: str = Field(sa_type=String(20), default=UserRole.USER)
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
    __tablename__ = "user_identifiers"  # type: ignore

    value: str = Field(primary_key=True)
    type: IdentifierType = ShortString(length=10, nullable=False, index=True)
    created_at: datetime = CreatedAt
    updated_at: datetime = UpdatedAt
    user_id: uuid.UUID = FK("users.id", ondelete="CASCADE")
    verified: bool = Field(default=False)
    primary: bool = Field(default=False)
    data: Optional[dict[str, Any]] = JsonbData()

    user: User = Relationship(back_populates="identifiers")


class Game(Base, table=True):
    __tablename__ = "games"  # type: ignore

    name: str = ShortString(length=25, nullable=False)
    start: datetime = Timestamp(nullable=False, description="Start of the Game UTC-12")
    end: datetime = Timestamp(nullable=False, description="End of the Game UTC+12")
    commit_points: int = Field(default=1, description="points per commit")
    project_points: int = Field(default=10, description="points per project")
    is_active: bool = Field(default=False)


class Project(Base, table=True):
    __tablename__ = "projects"  # type: ignore

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
    __tablename__ = "commits"  # type: ignore

    project_id: uuid.UUID = FK(
        "projects.id",
        index=True,
        ondelete="CASCADE",
    )
    user_id: uuid.UUID | None = FK(
        "users.id",
        index=True,
        nullable=True,
        ondelete="CASCADE",
    )
    game_id: uuid.UUID | None = FK("games.id", index=True, nullable=True)

    hash: str = Identifier(unique=True, index=True)
    author: str = ShortString()
    email: str = Email()
    message: str
    url: str
    timestamp: datetime = Timestamp(nullable=False)
    languages: list[str] = Array(description="programming languages")
    files: dict[str, Any] = JsonbData()

    is_verified: bool = Field(default=False)
    is_flagged: bool = Field(default=False)
    flag_reason: Optional[str] = None

    project: Project = Relationship(back_populates="commits")


class Player(Base, table=True):
    __tablename__ = "players"  # type: ignore

    game_id: uuid.UUID = FK("games.id", index=True)
    user_id: uuid.UUID = FK(
        "users.id",
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

    __table_args__ = (
        UniqueConstraint("game_id", "user_id", name="uq_player_user_game"),
    )

    def to_leader(self, rank: int) -> Leader:
        return Leader(
            rank=rank,
            user_id=self.user_id,
            name=self.user.name,
            username=self.user.username,
            avatar_url=self.user.avatar_url,
            points=self.points,
            verified_points=self.verified_points,
            commit_count=self.commit_count,
            project_count=self.project_count,
        )


class Board(Base, table=True):
    __tablename__ = "boards"  # type: ignore

    game_id: uuid.UUID = FK("games.id", index=True)
    project_id: uuid.UUID = FK(
        "projects.id",
        index=True,
        ondelete="CASCADE",
    )

    points: int = Field(default=0)
    potential_points: int = Field(default=0)
    verified_points: int = Field(default=0)

    commit_count: int = Field(default=0)
    contributor_count: int = Field(default=0)

    project: Project = Relationship(back_populates="boards")

    __table_args__ = (
        UniqueConstraint("game_id", "project_id", name="uq_board_project_game"),
    )

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


class Language(Base, table=True):
    __tablename__ = "languages"  # type: ignore

    name: str = Field(unique=True, index=True)


class LanguageBoard(Base, table=True):
    __tablename__ = "language_boards"  # type: ignore

    game_id: uuid.UUID = FK("games.id", index=True)
    language_id: uuid.UUID = FK("languages.id", index=True)
    points: int = Field(default=0)
    commit_count: int = Field(default=0)

    __table_args__ = (
        UniqueConstraint("game_id", "language_id", name="uq_language_game"),
    )


class Team(Base, table=True):
    __tablename__ = "teams"  # type: ignore

    name: str = Field(unique=True, index=True)
    slug: str = Field(unique=True, index=True)
    description: Optional[str] = None
    avatar_url: Optional[str] = None
    created_by: uuid.UUID = FK("users.id", ondelete="CASCADE")
    is_public: bool = Field(default=True)
    member_count: int = Field(default=0)


class TeamMember(Base, table=True):
    __tablename__ = "team_members"  # type: ignore

    team_id: uuid.UUID = FK("teams.id", index=True)
    user_id: uuid.UUID = FK("users.id", index=True, ondelete="CASCADE")
    role: str = Field(default="member")


class TeamBoard(Base, table=True):
    __tablename__ = "team_boards"  # type: ignore

    game_id: uuid.UUID = FK("games.id", index=True)
    team_id: uuid.UUID = FK("teams.id", index=True, ondelete="CASCADE")
    points: int = Field(default=0)
    member_count: int = Field(default=0)

    __table_args__ = (
        UniqueConstraint("game_id", "team_id", name="uq_teamboard_team_game"),
    )


class Report(Base, table=True):
    __tablename__ = "reports"  # type: ignore

    reported_user_id: Optional[uuid.UUID] = FK("users.id", index=True, nullable=True)
    report_type: ReportType = Field(sa_type=String(20), default=ReportType.SPAM)
    reason: str = ShortString(description="Short reason for reporting 'other' type")
    status: ReportStatus = Field(sa_type=String(20), default=ReportStatus.PENDING)
    reviewed_by: Optional[uuid.UUID] = Field(default=None, foreign_key="users.id")
    reviewed_at: Optional[datetime] = Timestamp(nullable=True)
    moderator_notes: Optional[str] = None


class AuditLog(Base, table=True):
    __tablename__ = "audit_logs"  # type: ignore

    moderator_id: uuid.UUID = FK("users.id", index=True)
    action: str
    target_type: str
    target_id: str
    reason: Optional[str] = None
