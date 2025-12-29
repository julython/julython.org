import httpx

from tests.fixtures import INAUGURAL_GAME

from july.services.user_service import UserService
from july.schema import IdentifierType


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

    leaders = await client.get(
        f"/api/v1/game/leaders?date={INAUGURAL_GAME.isoformat()}"
    )

    assert leaders.status_code == 200, leaders.text
    leader_data = leaders.json()
    assert len(leader_data["data"]) == 1, leader_data
    leader = leader_data["data"][0]
    assert leader["rank"] == 1
    assert leader["name"] == user.name
    assert leader["avatar_url"] == user.avatar_url
