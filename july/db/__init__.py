import sqlalchemy
import structlog
from sqlalchemy.engine import make_url
from sqlalchemy.exc import ProgrammingError
from sqlalchemy.ext.asyncio import AsyncEngine, create_async_engine

log = structlog.stdlib.get_logger(__name__)


async def ensure_database_exists(database_uri: str) -> None:
    database_uri_root, _, database_name = database_uri.rpartition("/")
    database_uri = f"{database_uri_root}/postgres"
    db = create_async_engine(make_url(database_uri), isolation_level="AUTOCOMMIT")
    async with db.connect() as conn:
        try:
            await conn.execute(sqlalchemy.text(f"create database {database_name}"))
        except ProgrammingError as e:
            if "already exists" in str(e):
                log.info("Database already exists. Nothing to do.")
            else:  # pragma: no cover
                log.error("Exception raised when creating database: %s", e)
                raise e

        log.info("Database created successfully")


async def init_database(database_uri: str) -> AsyncEngine:
    """Initialize the database engine and connect to it."""
    engine = create_async_engine(
        make_url(database_uri),
        # This isolation level allows us to use transactions that involve
        # more than one operation. For example if an exception is raised during
        # a transaction that creates a user and multiple usernames all items
        # are removed during the rollback. With autocommit only the failed operation
        # is rolled back leaving the database dirty.
        isolation_level="READ COMMITTED",
        pool_size=10,
        max_overflow=5,
        pool_recycle=60 * 60,  # 1 hour
        connect_args={
            # For AWS RDS Filtering
            "server_settings": {
                "application_name": "{{cookiecutter.service_name_dashed}}"
            }
        },
    )
    return engine


async def teardown_database(engine: AsyncEngine) -> None:
    """Disconnect from the database."""
    await engine.dispose()
