import httpx


class TestSpaFallback:

    async def test_404(self, client: httpx.AsyncClient):
        response = await client.get("/some-non-existent-file")
        assert response.status_code == 200, response.text
        assert "etag" in response.headers
        assert "last-modified" in response.headers
        assert response.headers["cache-control"] == "no-cache"
        assert "Welcome Julython Testers!" in response.text

    async def test_200(self, client: httpx.AsyncClient):
        response = await client.get("/index.html")
        assert response.status_code == 200, response.text
        assert "etag" in response.headers
        assert "last-modified" in response.headers
        assert response.headers["cache-control"] == "no-cache"
        assert "Welcome Julython Testers!" in response.text

    async def test_head(self, client: httpx.AsyncClient):
        response = await client.head("/index.html")
        assert response.status_code == 200, response.text
        assert "etag" in response.headers
        assert "last-modified" in response.headers
        assert response.headers["cache-control"] == "no-cache"
        assert response.text == ""


class TestStaticAssets:

    async def test_404(self, client: httpx.AsyncClient):
        response = await client.get("/assets/missing.js")
        assert response.status_code == 404, response.text
        assert "etag" not in response.headers
        assert "last-modified" not in response.headers
        assert "cache-control" not in response.headers

    async def test_200(self, client: httpx.AsyncClient):
        response = await client.get("/assets/fake.js")
        assert response.status_code == 200, response.text
        assert "some fake ass Julython Javascript" in response.text
        assert "etag" in response.headers
        assert "last-modified" in response.headers
        assert (
            response.headers["cache-control"] == "public, max-age=31536000, immutable"
        )

    async def test_head(self, client: httpx.AsyncClient):
        response = await client.head("/assets/fake.js")
        assert response.status_code == 200, response.text
        assert "etag" in response.headers
        assert "last-modified" in response.headers
        assert (
            response.headers["cache-control"] == "public, max-age=31536000, immutable"
        )
        assert response.text == ""
