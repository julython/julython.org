import re
from datetime import datetime
from os.path import splitext
from typing import Annotated, Optional
from urllib.parse import urlparse

import structlog
from pydantic import BaseModel, Field, computed_field
from pydantic.types import StringConstraints

from july.db.models import ReportType, ReportStatus

logger = structlog.stdlib.get_logger(__name__)


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
