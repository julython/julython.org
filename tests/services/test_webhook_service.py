import pytest

from july.db.models import Project, User
from july.schema import IdentifierType
from july.services.game_service import GameService
from july.services.user_service import UserService
from july.services.webhook_service import (
    parse_github,
    parse_gitlab,
    parse_bitbucket,
    parse_email,
    WebhookService,
    WebhookPayload,
)


class TestParseEmail:

    @pytest.mark.parametrize(
        "raw,expected",
        [
            pytest.param(
                "John Doe <john@example.com>",
                "john@example.com",
                id="with-name",
            ),
            pytest.param("plain@example.com", "plain@example.com", id="plain"),
            pytest.param(
                "Multiple <first@ex.com> Names <second@ex.com>",
                "first@ex.com",
                id="multiple-takes-first",
            ),
        ],
    )
    def test_parse_email(self, raw: str, expected: str):
        assert parse_email(raw) == expected


class TestParseGithub:

    def test_parses_repo(self, github_payload):
        result = parse_github(github_payload)

        assert result.provider == "github"
        assert result.ref == "refs/heads/master"
        assert result.repo.name == "test-repo"
        assert result.repo.repo_id == 12345
        assert result.repo.service == "github"
        assert result.repo.owner == "user"
        assert result.repo.slug == "gh-user-test-repo"

    def test_parses_commits(self, github_payload):
        result = parse_github(github_payload)

        assert len(result.commits) == 1
        commit = result.commits[0]
        assert commit.hash == "abc123def456"
        assert commit.message == "Fix bug"
        assert commit.author_email == "test@example.com"
        assert commit.author_username == "testuser"

    def test_parses_files_with_languages(self, github_payload):
        result = parse_github(github_payload)
        commit = result.commits[0]

        assert len(commit.files) == 3
        assert commit.languages == ["Python"]

        added = [f for f in commit.files if f.type == "added"]
        assert len(added) == 1
        assert added[0].file == "new.py"


class TestParseGitlab:

    def test_parses_repo(self, gitlab_payload):
        result = parse_gitlab(gitlab_payload)

        assert result.provider == "gitlab"
        assert result.repo.name == "gitlab-project"
        assert result.repo.repo_id == 98765
        assert result.repo.service == "gitlab"
        assert result.repo.owner == "gitlabuser"
        assert result.repo.slug == "gl-team-gitlab-project"

    def test_parses_commits(self, gitlab_payload):
        result = parse_gitlab(gitlab_payload)
        commit = result.commits[0]

        assert commit.hash == "def789abc"
        assert commit.author_email == "gl@example.com"
        assert commit.author_username is None  # GitLab doesn't include this
        assert commit.languages == ["Rust"]


class TestParseBitbucket:

    def test_parses_repo(self, bitbucket_payload):
        result = parse_bitbucket(bitbucket_payload)

        assert result.provider == "bitbucket"
        assert result.repo.name == "bb-repo"
        assert result.repo.service == "bitbucket"
        assert result.repo.owner == "team"
        assert result.repo.url == "https://bitbucket.org/team/bb-repo/"

    def test_parses_commits(self, bitbucket_payload):
        result = parse_bitbucket(bitbucket_payload)
        commit = result.commits[0]

        assert commit.hash == "fedcba987654"
        assert commit.author_email == "bb@example.com"
        assert commit.author_username == "bbuser"
        assert sorted(commit.languages) == ["Documentation", "Golang"]


class TestProcessWebhook:

    @pytest.fixture
    def webhook_service(self, db_session) -> WebhookService:
        return WebhookService(db_session)

    @pytest.fixture
    def sample_payload(self, github_payload) -> WebhookPayload:
        return parse_github(github_payload)

    async def test_skips_inactive_project(
        self,
        webhook_service: WebhookService,
        sample_payload: WebhookPayload,
        db_session,
    ):
        # Create inactive project first
        project = Project(
            name=sample_payload.repo.name,
            url=sample_payload.repo.url,
            slug=sample_payload.repo.slug,
            service=sample_payload.repo.service,
            repo_id=sample_payload.repo.repo_id,
            is_active=False,
        )
        db_session.add(project)
        await db_session.commit()

        commits = await webhook_service.process_webhook(sample_payload)

        assert len(commits) == 0

    async def test_skips_commits_from_inactive_user(
        self,
        webhook_service: WebhookService,
        sample_payload: WebhookPayload,
        user: User,
        db_session,
    ):
        # Create inactive user with matching email
        user_service = UserService(db_session)
        await user_service.upsert_identifier(
            user, IdentifierType.EMAIL, "test@example.com", data={}
        )
        game_service = GameService(db_session)
        await game_service.deactivate_user(user.id)
        await db_session.commit()

        commits = await webhook_service.process_webhook(sample_payload)

        assert len(commits) == 0

    async def test_skips_duplicate_commits(
        self,
        webhook_service: WebhookService,
        sample_payload: WebhookPayload,
    ):
        # Process same payload twice
        first_commits = await webhook_service.process_webhook(sample_payload)
        second_commits = await webhook_service.process_webhook(sample_payload)

        assert len(first_commits) == 1
        assert len(second_commits) == 0
