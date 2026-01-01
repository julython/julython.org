import httpx

from tests.fixtures import INAUGURAL_GAME

from july.services.game_service import GameService
from july.services.user_service import UserService
from july.schema import IdentifierType, Leader, LeaderBoard


async def test_game_records_commits(
    client: httpx.AsyncClient,
    github_payload: dict,
    db_session,
    user,
    active_game,
):
    # Update the user to set the email to match fixture
    user_service = UserService(db_session)
    ua, created = await user_service.upsert_identifier(
        user, IdentifierType.EMAIL, "test@example.com", data={}
    )
    assert ua.user_id == user.id
    assert created
    await db_session.commit()

    response = await client.post("/api/v1/github", json=github_payload)
    assert response.status_code == 201, response.text

    leaders = await client.get("/api/v1/game/leaders")

    assert leaders.status_code == 200, leaders.text
    leader_data = leaders.json()
    assert len(leader_data["data"]) == 1, leader_data
    leader = Leader.model_validate(leader_data["data"][0])
    assert leader.rank == 1, leader
    assert leader.name == user.name, leader
    assert leader.avatar_url == user.avatar_url, leader
    assert leader.points == 11, leader

    leader = await client.get(f"/api/v1/game/leaders/{user.username}")
    assert leader.status_code == 200, leader.text
    player_data = leader.json()
    player = Leader.model_validate(player_data)
    assert player.rank == 1, player
    assert player.name == user.name, player
    assert player.avatar_url == user.avatar_url, player
    assert player.points == 11, player

    boards = await client.get("/api/v1/game/boards")

    assert boards.status_code == 200, boards.text
    board_data = boards.json()
    assert len(board_data["data"]) == 1, board_data
    board = LeaderBoard.model_validate(board_data["data"][0])
    assert board.rank == 1, board
    assert board.name == "test-repo", board
    assert board.commit_count == 1, board
    assert board.points == 11, board
    assert board.url == "https://github.com/user/test-repo", board
    assert board.slug == "gh-user-test-repo", board

    # ban the project and user and verify they no longer show up
    game_service = GameService(db_session)
    await game_service.deactivate_project(board.project_id)
    await game_service.deactivate_user(user.id)
    await db_session.commit()

    leaders = await client.get("/api/v1/game/leaders")

    assert leaders.status_code == 200, leaders.text
    leader_data = leaders.json()
    assert len(leader_data["data"]) == 0, leader_data

    boards = await client.get("/api/v1/game/boards")

    assert boards.status_code == 200, boards.text
    board_data = boards.json()
    assert len(board_data["data"]) == 0, board_data
