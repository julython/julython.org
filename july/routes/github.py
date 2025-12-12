import structlog
from fastapi import APIRouter, Depends, responses
from sqlalchemy.ext.asyncio import AsyncSession

from july.dependencies import get_session
from july.services.game_service import GameService

logger = structlog.stdlib.get_logger(__name__)

router = APIRouter(prefix="/api/github")


@router.post("")
async def webhook(payload: dict, session: AsyncSession = Depends(get_session)):
    """Handle GitHub webhook for new commits"""
    game_service = GameService(session)

    return responses.JSONResponse(payload)
