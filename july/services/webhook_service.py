import re
from urllib.parse import urljoin

from sqlalchemy import select
from sqlalchemy.dialects.postgresql import insert
from sqlalchemy.ext.asyncio import AsyncSession
from sqlmodel import col
from structlog.stdlib import get_logger

from july.db.models import Project, User, Commit
from july.services import game_service, user_service
from july.schema import CommitData, EmailAddress, FileChange, RepoData, WebhookPayload
from july.utils.languages import detect_language, parse_files
from july.utils.times import parse_timestamp

logger = get_logger(__name__)

EMAIL_MATCH = re.compile(r"<(.+?)>")


def parse_email(raw: str) -> str:
    """Extract email from 'Name <email>' format."""
    if m := EMAIL_MATCH.search(raw):
        return m.group(1)
    return raw


def parse_github(data: dict) -> WebhookPayload:
    repo_data = data["repository"]
    repo = RepoData(
        url=repo_data.get("html_url") or repo_data.get("url", ""),
        name=repo_data.get("name", ""),
        description=repo_data.get("description") or "",
        service="github",
        repo_id=repo_data.get("id"),
        owner=repo_data.get("owner", {}).get("login", ""),
        forks=repo_data.get("forks", 0),
        watchers=repo_data.get("watchers", 0),
    )

    commits = []
    for c in data.get("commits", []):
        author = c.get("author", {})
        commits.append(
            CommitData(
                hash=c["id"],
                message=c.get("message", ""),
                timestamp=parse_timestamp(c["timestamp"]),
                url=c.get("url", ""),
                author_name=author.get("name", ""),
                author_email=author.get("email", ""),
                author_username=author.get("username"),
                files=parse_files(
                    c.get("added", []),
                    c.get("modified", []),
                    c.get("removed", []),
                ),
            )
        )

    return WebhookPayload(
        provider="github",
        before=data.get("before", ""),
        after=data.get("after", ""),
        ref=data.get("ref", ""),
        repo=repo,
        commits=commits,
        forced=data.get("forced", False),
    )


def parse_gitlab(data: dict) -> WebhookPayload:
    project = data.get("project", {})
    repo = RepoData(
        url=project.get("web_url", ""),
        name=project.get("name", ""),
        description=project.get("description") or "",
        service="gitlab",
        repo_id=project.get("id"),
        owner=data.get("user_username") or project.get("namespace", ""),
    )
    before = data.get("before", "")
    is_new = before == "0" * 40
    commits = []
    for c in data.get("commits", []):
        author = c.get("author", {})
        commits.append(
            CommitData(
                hash=c["id"],
                message=c.get("message", ""),
                timestamp=parse_timestamp(c["timestamp"]),
                url=c.get("url", ""),
                author_name=author.get("name", ""),
                author_email=author.get("email", ""),
                author_username=None,
                files=parse_files(
                    c.get("added", []),
                    c.get("modified", []),
                    c.get("removed", []),
                ),
            )
        )
    commit_ids = [commit.hash for commit in commits]
    is_normal_push = is_new or before in commit_ids

    return WebhookPayload(
        provider="gitlab",
        ref=data.get("ref", ""),
        repo=repo,
        commits=commits,
        before=before,
        after=data.get("after", ""),
        forced=not is_normal_push,
    )


def parse_bitbucket(data: dict) -> WebhookPayload:
    repo_data = data.get("repository", {})
    canon_url = data.get("canon_url", "https://bitbucket.org")
    abs_url = repo_data.get("absolute_url", "")
    if not abs_url.startswith("http"):  # pragma: no cover
        abs_url = urljoin(canon_url, abs_url)

    repo = RepoData(
        url=abs_url,
        name=repo_data.get("name", ""),
        description=repo_data.get("website") or "",
        service="bitbucket",
        repo_id=None,
        owner=repo_data.get("owner", ""),
        forked=repo_data.get("fork", False),
    )

    commits = []
    for c in data.get("commits", []):
        bb_files = c.get("files", [])
        files = [
            FileChange(
                file=f["file"],
                type=f.get("type", "modified"),
                language=detect_language(f["file"]),
            )
            for f in bb_files
        ]

        raw_node = c.get("raw_node", c.get("node", ""))
        commits.append(
            CommitData(
                hash=raw_node,
                message=c.get("message", ""),
                timestamp=parse_timestamp(
                    c.get("utctimestamp", c.get("timestamp", ""))
                ),
                url=urljoin(abs_url, f"commits/{raw_node}"),
                author_name=c.get("author", ""),
                author_email=parse_email(c.get("raw_author", "")),
                author_username=c.get("author"),
                files=files,
            )
        )

    return WebhookPayload(
        provider="bitbucket",
        ref=data.get("ref", ""),
        repo=repo,
        commits=commits,
        before="",
        after="",
        forced=False,
    )


class WebhookService:
    """Service for processing webhook payloads using PostgreSQL upserts."""

    def __init__(self, session: AsyncSession):
        self.session = session
        self.game_service = game_service.GameService(session)
        self.user_service = user_service.UserService(session)

    async def process_webhook(self, payload: WebhookPayload) -> list[Commit]:
        """Process a webhook payload, creating project and commits."""
        project = await self.upsert_project(payload.repo)

        if project is None or not project.is_active:
            logger.warning(f"Project disabled or missing: {payload.repo.slug}")
            return []

        created_commits = []
        for commit_data in payload.commits:
            commit, created = await self.upsert_commit(commit_data, project)
            if commit and created:
                created_commits.append(commit)

        await self.session.commit()
        return created_commits

    async def upsert_project(self, repo: RepoData) -> Project | None:
        """Upsert a project using PostgreSQL ON CONFLICT."""
        values = dict(
            url=repo.url,
            name=repo.name,
            slug=repo.slug,
            description=repo.description,
            service=repo.service,
            repo_id=repo.repo_id,
            forked=repo.forked,
            forks=repo.forks,
            watchers=repo.watchers,
        )

        update_values = dict(
            url=repo.url,
            name=repo.name,
            slug=repo.slug,
            description=repo.description,
            forks=repo.forks,
            watchers=repo.watchers,
        )

        insert_stmt = insert(Project).values(**values)

        if repo.repo_id:
            # GitHub/GitLab: conflict on service+repo_id (handles renames)
            stmt = insert_stmt.on_conflict_do_update(
                index_elements=["service", "repo_id"],
                set_=update_values,
            ).returning(Project)
        else:
            # Bitbucket: conflict on slug
            stmt = insert_stmt.on_conflict_do_update(
                index_elements=["slug"],
                set_=update_values,
            ).returning(Project)

        result = await self.session.execute(stmt)
        project = result.scalar_one_or_none()
        await self.session.flush()

        return project

    async def upsert_commit(
        self, commit_data: CommitData, project: Project
    ) -> tuple[Commit | None, bool]:
        """
        Insert a commit, ignoring if it already exists.

        Returns (commit, was_created). Orphan commits (no user) are
        claimed later when a user registers, not here.
        """
        user = await self.find_user_by_email(commit_data.author_email)

        if user and not user.is_active:
            return None, False

        files = [
            {"file": f.file, "type": f.type, "language": f.language}
            for f in commit_data.files
        ]

        stmt = (
            insert(Commit)
            .values(
                hash=commit_data.hash,
                message=commit_data.message[:2024],
                timestamp=commit_data.timestamp,
                url=commit_data.url,
                author=commit_data.author_name,
                email=commit_data.author_email,
                project_id=project.id,
                user_id=user.id if user else None,
                languages=commit_data.languages,
                files=files,
            )
            .on_conflict_do_nothing(
                index_elements=["hash"],
            )
            .returning(Commit)
        )

        result = await self.session.execute(stmt)
        commit = result.scalar_one_or_none()

        # If None, commit already existed - skip
        if commit is None:
            return None, False

        await self.session.flush()
        await self.game_service.add_commit(commit)

        return commit, True

    async def find_user_by_email(self, email: str) -> User | None:
        """Find a user by email address."""
        if not email:  # pragma: no cover
            return None

        return await self.user_service.find_by_email(EmailAddress(email=email))
