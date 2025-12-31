from datetime import datetime
from enum import Enum
from re import I
from typing import Annotated, Any, Generic, Optional, TypeVar
from urllib.parse import urlparse

import structlog
from pydantic import BaseModel, ConfigDict, Field, computed_field
from pydantic.types import StringConstraints

from july.types import Identifier

logger = structlog.stdlib.get_logger(__name__)

DataType = TypeVar("DataType")

HOST_ABBR = {
    "github.com": "gh",
    "gitlab.com": "gl",
    "bitbucket.org": "bb",
}


def parse_project_slug(url: str) -> str:
    """Parse a project url and return a slug.

    Example: http://github.com/julython/julython.org -> gh-julython-julython_org
    """
    if not url:
        return ""
    parsed = urlparse(url)
    path = parsed.path.strip("/")
    host_abbr = HOST_ABBR.get(parsed.netloc, parsed.netloc.replace(".", "-"))
    name = path.replace("/", "-").replace(".", "_")
    return f"{host_abbr}-{name}"


# Enums
class AnalysisStatus(str, Enum):
    PENDING = "pending"
    IN_PROGRESS = "in_progress"
    COMPLETED = "completed"
    FLAGGED = "flagged"


class IdentifierType(str, Enum):
    EMAIL = "email"
    GITHUB = "github"
    GITLAB = "gitlab"
    BITBUCKET = "bitbucket"


class OAuthProvider(str, Enum):
    GITHUB = "github"
    GITLAB = "gitlab"


class ReportStatus(str, Enum):
    PENDING = "pending"
    REVIEWED = "reviewed"
    RESOLVED = "resolved"
    REJECTED = "rejected"


class ReportType(str, Enum):
    FAKE_DATA = "fake_data"
    SPAM = "spam"
    INAPPROPRIATE = "inappropriate"
    CHEATING = "cheating"
    OTHER = "other"


class UserRole(str, Enum):
    USER = "user"
    MODERATOR = "moderator"
    ADMIN = "admin"


class FileChange(BaseModel):
    file: str
    type: str  # added, modified, removed
    language: str | None = None


class CommitData(BaseModel):
    hash: str
    message: str = ""
    timestamp: datetime
    url: str = ""
    author_name: str = ""
    author_email: str = ""
    author_username: str | None = None
    files: list[FileChange] = Field(default_factory=list)

    @computed_field
    @property
    def languages(self) -> list[str]:
        return list({f.language for f in self.files if f.language})


class RepoData(BaseModel):
    url: str
    name: str
    description: str = ""
    service: str
    repo_id: int | None = None
    owner: str = ""
    forked: bool = False
    forks: int = 0
    watchers: int = 0

    @computed_field
    @property
    def slug(self) -> str:
        return parse_project_slug(self.url)


class WebhookPayload(BaseModel):
    provider: str
    before: str
    after: str
    ref: str = ""
    repo: RepoData
    commits: list[CommitData] = Field(default_factory=list)
    forced: bool = False


class TeamCreate(BaseModel):
    name: str
    description: Optional[str] = None
    avatar_url: Optional[str] = None
    is_public: bool = True


class ReportCreate(BaseModel):
    analysis_id: Optional[int] = None
    reported_user_id: Optional[int] = None
    report_type: ReportType
    reason: str


class ReportAction(BaseModel):
    status: ReportStatus
    moderator_notes: Optional[str] = None
    ban_user: bool = False
    remove_analysis: bool = False
    ban_reason: Optional[str] = None


class EmailAddress(BaseModel):
    email: Annotated[str, StringConstraints(to_lower=True)]
    primary: bool = False
    verified: bool = False

    @property
    def key(self) -> str:
        return f"email:{self.email}"


class OAuthTokens(BaseModel):
    access_token: str
    refresh_token: str | None = None
    expires_in: int | None = None  # seconds


class OAuthUser(BaseModel):
    id: str
    provider: OAuthProvider
    username: str
    emails: list[EmailAddress]
    name: str | None
    avatar_url: str | None
    data: dict[str, Any]

    @property
    def key(self) -> str:
        return f"{self.provider.value}:{self.id}"


class Leader(BaseModel):
    rank: int
    user_id: Identifier
    name: str
    avatar_url: str | None
    points: int
    verified_points: int
    commit_count: int
    project_count: int


class LeaderBoard(BaseModel):
    rank: int
    project_id: Identifier
    name: str
    slug: str
    url: str
    points: int
    verified_points: int
    commit_count: int
    contributor_count: int


class ListResponse(BaseModel, Generic[DataType]):
    data: list[DataType]
    limit: int = 100
    offset: int = 0


class SessionUser(BaseModel):
    id: str
    name: str
    username: Optional[str]
    avatar_url: Optional[str] = None
    role: UserRole
    is_active: bool
    is_banned: bool


class SessionData(BaseModel):
    oauth_provider: Optional[str] = None
    oauth_state: Optional[str] = None
    oauth_verifier: Optional[str] = None
    identity_key: Optional[str] = None
    user: Optional[SessionUser] = None
