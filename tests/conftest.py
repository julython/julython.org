import pathlib
import uuid
from datetime import datetime
from typing import Any, AsyncGenerator, TypeVar

import httpx
import pytest
from alembic.config import Config
from sqlalchemy.ext.asyncio import (
    AsyncEngine,
    AsyncSession,
    async_sessionmaker,
)
from sqlalchemy.sql import text
from sqlmodel import SQLModel

from july.app import create_app
from july.db import ensure_database_exists, init_database, models, teardown_database
from july.globals import context
from july.services import game_service
from july.settings import Settings
from july.utils.logger import setup_logging

from tests.fixtures import INAGURAL_GAME

# Fake static
TEST_STATIC = pathlib.Path(__file__).parent / "static"

# For typing output of generators with yield statements
T = TypeVar("T")
YieldFixture = AsyncGenerator[T, None]

setup_logging("july")


@pytest.fixture(scope="session")
def database_name() -> str:
    return "july_test"


@pytest.fixture(scope="session")
def settings(database_name: str) -> Settings:
    """Resets the settings to the defaults before each test."""
    from july.globals import settings

    settings.database_name = database_name
    settings.database_available = True
    settings.static_dir = TEST_STATIC
    return settings


@pytest.fixture(scope="session")
async def database_uri(settings: Settings) -> str:
    """Construct a database connection string."""
    await ensure_database_exists(settings.database_uri)

    return settings.database_uri


@pytest.fixture(scope="session")
async def clean_database(database_uri: str) -> YieldFixture[AsyncEngine]:
    """Initializes the service database table schemas."""
    db = await init_database(database_uri)

    async with db.begin() as conn:
        await conn.run_sync(SQLModel.metadata.create_all)

    yield db

    async with db.begin() as conn:
        await conn.run_sync(SQLModel.metadata.drop_all)

    await teardown_database(db)


@pytest.fixture(scope="function")
async def db(
    clean_database: AsyncEngine,
    database_uri: str,
    settings: Settings,
) -> YieldFixture[async_sessionmaker]:
    """Test database, clears all database tables before every test."""
    await context.initialize(settings)

    yield context.db_session

    # Truncate all tables after each test
    async with context.db.begin() as conn:
        for table in reversed(SQLModel.metadata.sorted_tables):
            await conn.execute(text(f'DELETE FROM "{table.name}"'))

    await context.shutdown(settings)


@pytest.fixture()
def alembic_config(settings: Settings) -> Config:
    """This Fixture overrides the alembic config for `pytest-alembic` to point at the test database."""
    alembic_config = Config(str(settings.base_path / "alembic.ini"))
    alembic_config.set_main_option(
        "script_location", str(settings.base_path / "migrations")
    )
    alembic_config.set_main_option("sqlalchemy.url", settings.database_uri)
    return alembic_config


@pytest.fixture()
async def client(db: async_sessionmaker, settings: Settings) -> httpx.AsyncClient:
    """The test client for the main user, who belongs to an organization."""
    transport = httpx.ASGITransport(
        app=create_app(settings),
    )
    client = httpx.AsyncClient(transport=transport, base_url="http://test")
    return client


@pytest.fixture()
async def db_session(db) -> YieldFixture[AsyncSession]:
    async with db() as session:
        yield session


@pytest.fixture(scope="function")
async def active_game(db_session: AsyncSession) -> models.Game:
    games = game_service.GameService(db_session)
    return await games.create_julython_game(2012, is_active=True)


@pytest.fixture()
async def user(db_session: AsyncSession) -> models.User:
    user = models.User(id=uuid.uuid4(), name="test user", username="test")
    db_session.add(user)
    await db_session.commit()
    await db_session.refresh(user)
    return user


@pytest.fixture
async def project(db_session: AsyncSession) -> models.Project:
    project = models.Project(
        name="test-project",
        url="https://github.com/test/test-project",
        slug="gh-test/test-project",
    )
    db_session.add(project)
    await db_session.commit()
    return project


@pytest.fixture
def make_commit(db_session: AsyncSession, project: models.Project, user: models.User):
    async def _make(
        hash: str,
        timestamp: datetime | None = None,
        languages: list[str] | None = None,
        **kwargs,
    ) -> models.Commit:
        commit = models.Commit(
            hash=hash,
            message="fixing things",
            url="https://julython.org",
            author="timmy",
            email="timmy@jimmy.com",
            timestamp=timestamp or INAGURAL_GAME,
            project_id=kwargs.pop("project_id", project.id),
            user_id=kwargs.pop("user_id", user.id),
            languages=languages or [],
            **kwargs,
        )
        db_session.add(commit)
        await db_session.commit()
        return commit

    return _make


@pytest.fixture
async def player_factory(
    db_session: AsyncSession,
    active_game: models.Game,
    user: models.User,
):
    async def _create(**kwargs):
        player = models.Player(game_id=active_game.id, user_id=user.id, **kwargs)
        db_session.add(player)
        await db_session.commit()
        return player

    return _create


@pytest.fixture
def bitbucket_payload() -> dict:
    return {
        "canon_url": "https://bitbucket.org",
        "repository": {
            "absolute_url": "/team/bb-repo/",
            "name": "bb-repo",
            "owner": "team",
            "fork": False,
            "website": "https://example.com",
        },
        "commits": [
            {
                "raw_node": "fedcba987654",
                "node": "fedcba9",
                "message": "Update docs",
                "utctimestamp": "2024-07-15 12:00:00+00:00",
                "author": "bbuser",
                "raw_author": "BB User <bb@example.com>",
                "files": [
                    {"file": "README.md", "type": "modified"},
                    {"file": "app.go", "type": "added"},
                ],
            }
        ],
    }


@pytest.fixture
def gitlab_payload() -> dict:
    return {
        "object_kind": "push",
        "ref": "refs/heads/main",
        "user_username": "gitlabuser",
        "project": {
            "id": 98765,
            "name": "gitlab-project",
            "path_with_namespace": "team/gitlab-project",
            "web_url": "https://gitlab.com/team/gitlab-project",
            "namespace": "team",
            "description": "A GitLab project",
        },
        "commits": [
            {
                "id": "def789abc",
                "message": "Add feature",
                "timestamp": "2024-07-15T15:00:00Z",
                "url": "https://gitlab.com/team/gitlab-project/-/commit/def789",
                "author": {"name": "GL User", "email": "gl@example.com"},
                "added": ["feature.rs"],
                "modified": [],
                "removed": [],
            }
        ],
    }


@pytest.fixture
def github_payload() -> dict:
    return {
        "ref": "refs/heads/master",
        "repository": {
            "id": 12345,
            "name": "test-repo",
            "full_name": "user/test-repo",
            "html_url": "https://github.com/user/test-repo",
            "description": "A test repository",
            "owner": {"login": "user"},
            "forks": 5,
            "watchers": 10,
        },
        "commits": [
            {
                "id": "abc123def456",
                "message": "Fix bug",
                "timestamp": "2024-07-15T10:30:00-05:00",
                "url": "https://github.com/user/test-repo/commit/abc123",
                "author": {
                    "name": "Test User",
                    "email": "test@example.com",
                    "username": "testuser",
                },
                "added": ["new.py"],
                "modified": ["main.py"],
                "removed": ["old.py"],
            }
        ],
    }
