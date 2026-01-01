from typing import Annotated, Optional

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from july.dependencies import get_session
from july.schema import LeaderBoard, ListResponse, Leader
from july.services.game_service import GameService
from july.services.user_service import UserService
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
    players = await game_service.get_leaderboard(game.id)
    return ListResponse(data=[p.to_leader(i + 1) for i, p in enumerate(players)])


@router.get("/leaders/{username}")
async def get_leader(
    username: str,
    db_session: Annotated[AsyncSession, Depends(get_session)],
    date: Annotated[Optional[str], Query(...)] = None,
) -> Leader:
    game_service = GameService(db_session)
    user_service = UserService(db_session)
    now = times.parse_timestamp(date)
    game = await game_service.get_active_or_latest_game(now)
    user = await user_service.find_by_username(username)
    player = await game_service.upsert_player(game, user.id)
    return player.to_leader(1)


@router.get("/boards")
async def get_boards(
    db_session: Annotated[AsyncSession, Depends(get_session)],
    date: Annotated[Optional[str], Query(...)] = None,
) -> ListResponse[LeaderBoard]:
    game_service = GameService(db_session)
    now = times.parse_timestamp(date)
    game = await game_service.get_active_or_latest_game(now)
    boards = await game_service.get_project_leaderboard(game.id)
    return ListResponse(data=[b.to_leader(i + 1) for i, b in enumerate(boards)])
