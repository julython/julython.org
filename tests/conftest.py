from datetime import datetime
from typing import Any, AsyncGenerator, TypeVar
import uuid

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
from july.globals import context, settings
from july.services import game_service
from july.utils.logger import setup_logging

from tests.fixtures import INAGURAL_GAME

# For typing output of generators with yield statements
T = TypeVar("T")
YieldFixture = AsyncGenerator[T, None]

setup_logging("july")


@pytest.fixture(scope="session")
def database_name() -> str:
    return "july_test"


@pytest.fixture(scope="function", autouse=True)
def init_settings_and_mocks(monkeypatch: Any, database_name: str) -> None:
    """Resets the settings to the defaults before each test."""
    monkeypatch.setattr(settings, "database_name", database_name)


@pytest.fixture(scope="session")
async def database_uri(database_name: str) -> str:
    """Construct a database connection string."""
    database_uri_root, _, _ = settings.database_uri.rpartition("/")
    database_uri = f"{database_uri_root}/{database_name}"
    await ensure_database_exists(database_uri)

    return database_uri


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
    clean_database: AsyncEngine, database_uri: str
) -> YieldFixture[async_sessionmaker]:
    """Test database, clears all database tables before every test."""
    await context.initialize(settings)

    yield context.db_session

    # Truncate all tables after each test
    async with context.db.begin() as conn:
        for table in reversed(SQLModel.metadata.sorted_tables):
            await conn.execute(text(f'DELETE FROM "{table.name}"'))

    await context.shutdown()


@pytest.fixture()
def alembic_config(database_uri: str) -> Config:
    """This Fixture overrides the alembic config for `pytest-alembic` to point at the test database."""
    alembic_config = Config(str(settings.base_path / "alembic.ini"))
    alembic_config.set_main_option(
        "script_location", str(settings.base_path / "migrations")
    )
    alembic_config.set_main_option("sqlalchemy.url", database_uri)
    return alembic_config


@pytest.fixture()
async def client() -> httpx.AsyncClient:
    """The test client for the main user, who belongs to an organization."""
    transport = httpx.ASGITransport(
        app=create_app(),
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
