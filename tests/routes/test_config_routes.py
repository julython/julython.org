import httpx

from july.globals import settings


async def test_config_endpoint(client: httpx.AsyncClient):
    response = await client.get("/api/config")
    assert response.status_code == 200, response.text
    assert response.json() == {
        "status": "ok",
        "is_dev": settings.is_dev,
        "is_local": settings.is_local,
        "api_title": settings.api_title,
        "version": settings.pyproject.get("version", "unknown"),
    }
