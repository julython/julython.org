from typing import AsyncGenerator

from sqlalchemy.ext.asyncio import AsyncSession

from july.globals import context


async def get_session() -> AsyncGenerator[AsyncSession, None]:
    async with context.db_session.begin() as session:
        yield session
