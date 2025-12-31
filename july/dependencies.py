from typing import AsyncGenerator

from fastapi import HTTPException, Request
from sqlalchemy.ext.asyncio import AsyncSession

from july.globals import context
from july.schema import SessionData


async def get_session() -> AsyncGenerator[AsyncSession, None]:
    async with context.db_session.begin() as session:
        yield session


async def current_user(request: Request) -> SessionData:
    if not request.session:
        raise HTTPException(401, "User not authenticated")
    return SessionData.model_validate(request.session)
