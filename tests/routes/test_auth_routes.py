async def test_auth_login(client):
    response = await client.get("/auth/login/github")
    assert response.status_code == 307, response.text
    response = await client.get("/auth/login/gitlab")
    assert response.status_code == 307, response.text
    response = await client.get("/auth/login/bad")
    assert response.status_code == 422, response.text


async def test_auth_callback(client):
    response = await client.get("/auth/callback")
    assert response.status_code == 422, response.text

    response = await client.get("/auth/callback?scope=234")
    assert response.status_code == 422, response.text
