from sqlalchemy.ext.asyncio import AsyncEngine, async_sessionmaker

from july.db import init_database, teardown_database
from july.settings import Settings


class Context:
    """Main global context object.

    The attributes listed/typed below are set in `july.app.startup()`.

    """

    db: AsyncEngine
    db_session: async_sessionmaker

    async def initialize(self, settings: Settings) -> None:
        """Initialize the global context with connections to other services."""
        if settings.database_available:
            self.db = await init_database(settings.database_uri)
            self.db_session = async_sessionmaker(
                self.db,
                # This allows us to access models that are detached from the session
                # https://docs.sqlalchemy.org/en/20/errors.html#error-bhk3
                expire_on_commit=False,
            )

    async def shutdown(self, settings: Settings) -> None:
        """Teardown all open clients or connection that need cleanup."""
        if settings.database_available and self.db:
            await teardown_database(self.db)
