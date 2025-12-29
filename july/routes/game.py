from typing import Annotated, Optional

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.ext.asyncio import AsyncSession

from july.dependencies import get_session
from july.schema import LeaderBoard, ListResponse, Leader
from july.services.game_service import GameService
from july.utils import times

router = APIRouter(prefix="/api/v1/game", tags=["Game"])


@router.get("/leaders")
async def get_leaders(
    db_session: Annotated[AsyncSession, Depends(get_session)],
    date: Annotated[Optional[str], Query(...)] = None,
) -> ListResponse[Leader]:
    game_service = GameService(db_session)
    now = times.parse_timestamp(date)
    game = await game_service.get_active_or_latest_game(now)
    if game is None:
        raise HTTPException(
            status_code=404, detail=f"Active game not found for date:{now}"
        )

    players = await game_service.get_leaderboard(game.id)
    return ListResponse(data=[p.to_leader(i + 1) for i, p in enumerate(players)])


@router.get("/boards")
async def get_boards(
    db_session: Annotated[AsyncSession, Depends(get_session)],
    date: Annotated[Optional[str], Query(...)] = None,
) -> ListResponse[LeaderBoard]:
    game_service = GameService(db_session)
    now = times.parse_timestamp(date)
    game = await game_service.get_active_or_latest_game(now)
    if game is None:
        raise HTTPException(
            status_code=404, detail=f"Active game not found for date:{now}"
        )

    boards = await game_service.get_project_leaderboard(game.id)
    return ListResponse(data=[b.to_leader(i + 1) for i, b in enumerate(boards)])
