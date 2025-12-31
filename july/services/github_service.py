from typing import Any

import httpx
from pydantic import BaseModel
from structlog.stdlib import get_logger

logger = get_logger(__name__)


class GitHubRepo(BaseModel):
    id: int
    name: str
    full_name: str
    owner: str
    private: bool
    html_url: str
    description: str | None = None
    default_branch: str = "main"
    hooks_url: str
    webhooks: list["GitHubWebhook"] | None = None  # None = not fetched / no permission


class GitHubWebhook(BaseModel):
    id: int
    name: str
    active: bool
    events: list[str]
    config: dict[str, Any]


class GitHubService:
    """Client for GitHub API operations."""

    BASE_URL = "https://api.github.com"

    def __init__(self, access_token: str):
        self.access_token = access_token
        self.headers = {
            "Authorization": f"Bearer {access_token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        }

    async def list_repos(
        self,
        include_webhooks: bool = False,
        per_page: int = 50,
    ) -> list[GitHubRepo]:
        """
        List repositories the user has push access to.

        Args:
            include_webhooks: If True, fetch webhooks for each repo (slower, may fail on some)
            per_page: Number of repos per page (max 100)
        """
        repos: list[GitHubRepo] = []
        page = 1

        async with httpx.AsyncClient(headers=self.headers, timeout=30) as client:
            while True:
                resp = await client.get(
                    f"{self.BASE_URL}/user/repos",
                    params={
                        "per_page": per_page,
                        "page": page,
                        "sort": "updated",
                        "direction": "desc",
                        # Only repos where user can manage webhooks
                        "affiliation": "owner,organization_member",
                    },
                )
                resp.raise_for_status()
                data = resp.json()

                if not data:
                    break

                for repo_data in data:
                    # Skip if user doesn't have admin access (can't manage hooks)
                    permissions = repo_data.get("permissions", {})
                    if not permissions.get("admin", False):
                        continue

                    repo = GitHubRepo(
                        id=repo_data["id"],
                        name=repo_data["name"],
                        full_name=repo_data["full_name"],
                        owner=repo_data["owner"]["login"],
                        private=repo_data["private"],
                        html_url=repo_data["html_url"],
                        description=repo_data["description"],
                        default_branch=repo_data.get("default_branch", "main"),
                        hooks_url=repo_data["hooks_url"],
                    )
                    repos.append(repo)

                page += 1

            if include_webhooks:
                for repo in repos:
                    repo.webhooks = await self._get_webhooks(
                        client, repo.owner, repo.name
                    )

        return repos

    async def _get_webhooks(
        self,
        client: httpx.AsyncClient,
        owner: str,
        repo: str,
    ) -> list[GitHubWebhook] | None:
        """Fetch webhooks for a repo. Returns None if no permission."""
        try:
            resp = await client.get(f"{self.BASE_URL}/repos/{owner}/{repo}/hooks")
            if resp.status_code == 404:
                return None  # No permission or repo not found
            resp.raise_for_status()

            return [
                GitHubWebhook(
                    id=hook["id"],
                    name=hook["name"],
                    active=hook["active"],
                    events=hook["events"],
                    config=hook["config"],
                )
                for hook in resp.json()
            ]
        except httpx.HTTPStatusError as e:
            logger.warning(
                "Failed to fetch webhooks", owner=owner, repo=repo, error=str(e)
            )
            return None

    async def get_repo_webhooks(self, owner: str, repo: str) -> list[GitHubWebhook]:
        """Get webhooks for a specific repo."""
        async with httpx.AsyncClient(headers=self.headers, timeout=30) as client:
            resp = await client.get(f"{self.BASE_URL}/repos/{owner}/{repo}/hooks")
            resp.raise_for_status()

            return [
                GitHubWebhook(
                    id=hook["id"],
                    name=hook["name"],
                    active=hook["active"],
                    events=hook["events"],
                    config=hook["config"],
                )
                for hook in resp.json()
            ]

    async def create_webhook(
        self,
        owner: str,
        repo: str,
        webhook_url: str,
        events: list[str] | None = None,
    ) -> GitHubWebhook:
        """
        Create a webhook on a repository.

        Args:
            owner: Repository owner
            repo: Repository name
            webhook_url: URL to receive webhook events
            secret: Secret for webhook signature verification
            events: List of events to subscribe to (default: ["push"])
        """
        if events is None:
            events = ["push"]

        async with httpx.AsyncClient(headers=self.headers, timeout=30) as client:
            resp = await client.post(
                f"{self.BASE_URL}/repos/{owner}/{repo}/hooks",
                json={
                    "name": "web",
                    "active": True,
                    "events": events,
                    "config": {
                        "url": webhook_url,
                        "content_type": "json",
                        "insecure_ssl": "0",
                    },
                },
            )
            resp.raise_for_status()
            hook = resp.json()

            return GitHubWebhook(
                id=hook["id"],
                name=hook["name"],
                active=hook["active"],
                events=hook["events"],
                config=hook["config"],
            )

    async def delete_webhook(self, owner: str, repo: str, hook_id: int) -> bool:
        """Delete a webhook. Returns True if deleted, False if not found."""
        async with httpx.AsyncClient(headers=self.headers, timeout=30) as client:
            resp = await client.delete(
                f"{self.BASE_URL}/repos/{owner}/{repo}/hooks/{hook_id}"
            )
            return resp.status_code in (204, 404)

    async def ping_webhook(self, owner: str, repo: str, hook_id: int) -> bool:
        """Trigger a ping event for a webhook."""
        async with httpx.AsyncClient(headers=self.headers, timeout=30) as client:
            resp = await client.post(
                f"{self.BASE_URL}/repos/{owner}/{repo}/hooks/{hook_id}/pings"
            )
            return resp.status_code == 204
