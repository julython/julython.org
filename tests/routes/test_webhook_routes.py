import json

import httpx


class TestGithubWebhook:

    async def test_ping(self, client: httpx.AsyncClient):
        response = await client.post(
            "/api/v1/github", headers={"X-GitHub-Event": "ping"}
        )
        assert response.status_code == 200, response.text
        assert response.text == "pong"

    async def test_json_payload(self, github_payload: dict, client: httpx.AsyncClient):
        response = await client.post("/api/v1/github", json=github_payload)
        assert response.status_code == 201, response.text

        repeated = await client.post("/api/v1/github", json=github_payload)
        assert repeated.status_code == 200, repeated.text

    async def test_formdata_payload(
        self, github_payload: dict, client: httpx.AsyncClient
    ):
        response = await client.post(
            "/api/v1/github",
            data={"payload": json.dumps(github_payload)},
            headers={"Content-type": "application/x-www-form-urlencoded"},
        )
        assert response.status_code == 201, response.text

        repeated = await client.post(
            "/api/v1/github",
            data={"payload": json.dumps(github_payload)},
            headers={"Content-type": "application/x-www-form-urlencoded"},
        )
        assert repeated.status_code == 200, repeated.text


class TestGitlabWebhook:
    async def test_json_payload(self, gitlab_payload: dict, client: httpx.AsyncClient):
        response = await client.post("/api/v1/gitlab", json=gitlab_payload)
        assert response.status_code == 201, response.text

        repeated = await client.post("/api/v1/gitlab", json=gitlab_payload)
        assert repeated.status_code == 200, repeated.text


class TestBitbucketWebhook:
    async def test_json_payload(
        self, bitbucket_payload: dict, client: httpx.AsyncClient
    ):
        response = await client.post("/api/v1/bitbucket", json=bitbucket_payload)
        assert response.status_code == 201, response.text

        repeated = await client.post("/api/v1/bitbucket", json=bitbucket_payload)
        assert repeated.status_code == 200, repeated.text
