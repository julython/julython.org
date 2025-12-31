from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlmodel import col
from structlog.stdlib import get_logger

from july.db.models import UserIdentifier
from july.dependencies import get_session, current_user
from july.globals import settings
from july.schema import SessionData
from july.services.github_service import GitHubRepo, GitHubService, GitHubWebhook

logger = get_logger(__name__)
router = APIRouter(prefix="/api/github", tags=["GitHub"])


class CreateWebhookRequest(BaseModel):
    owner: str
    repo: str


class RepoListResponse(BaseModel):
    repos: list[GitHubRepo]
    webhook_url: str  # So the UI knows what to look for


async def get_github_token(
    user: Annotated[SessionData, Depends(current_user)],
    session: Annotated[AsyncSession, Depends(get_session)],
) -> str:
    """Extract GitHub access token from user's stored identity."""
    identity_key = user.identity_key
    if not identity_key:
        raise HTTPException(401, "Not authenticated")

    if not identity_key.startswith("github:"):
        raise HTTPException(400, "Not authenticated with GitHub")

    stmt = select(UserIdentifier).where(col(UserIdentifier.value) == identity_key)
    result = await session.execute(stmt)
    identifier = result.scalar_one_or_none()

    if identifier is None:
        raise HTTPException(401, "Identity not found")

    if identifier.data is None:
        raise HTTPException(403, "Identifier missing token data")

    access_token = identifier.data.get("access_token")
    if not access_token:
        raise HTTPException(403, "No GitHub access token")

    return access_token


@router.get("/repos", response_model=RepoListResponse)
async def list_repos(
    token: Annotated[str, Depends(get_github_token)],
    include_webhooks: bool = False,
):
    """
    List GitHub repositories the user can manage webhooks for.

    Query params:
        include_webhooks: If true, fetch existing webhooks for each repo (slower)
    """
    github = GitHubService(token)

    try:
        repos = await github.list_repos(include_webhooks=include_webhooks)
    except Exception as e:
        logger.exception("Failed to list GitHub repos")
        raise HTTPException(502, f"GitHub API error: {e}")

    return RepoListResponse(
        repos=repos,
        webhook_url=settings.github_webhook_url,
    )


@router.get("/repos/{owner}/{repo}/webhooks", response_model=list[GitHubWebhook])
async def get_repo_webhooks(
    owner: str,
    repo: str,
    token: Annotated[str, Depends(get_github_token)],
):
    """Get webhooks for a specific repository."""
    github = GitHubService(token)

    try:
        return await github.get_repo_webhooks(owner, repo)
    except Exception as e:
        logger.exception("Failed to get webhooks", owner=owner, repo=repo)
        raise HTTPException(502, f"GitHub API error: {e}")


@router.post("/repos/{owner}/{repo}/webhooks")
async def create_webhook(
    owner: str,
    repo: str,
    token: Annotated[str, Depends(get_github_token)],
    session: Annotated[AsyncSession, Depends(get_session)],
) -> GitHubWebhook:
    """
    Create a webhook for commits on a repository.

    Also creates a Project record to track the repo.
    """
    github = GitHubService(token)
    full_name = f"{owner}/{repo}"

    # Check if we already have a webhook
    try:
        existing_hooks = await github.get_repo_webhooks(owner, repo)
        for hook in existing_hooks:
            if hook.config.get("url") == settings.github_webhook_url:
                raise HTTPException(409, "Webhook already exists for this repository")
    except HTTPException:
        raise
    except Exception as e:
        logger.exception("Failed to check existing webhooks")
        raise HTTPException(502, f"GitHub API error: {e}")

    # Create the webhook
    try:
        webhook = await github.create_webhook(
            owner=owner,
            repo=repo,
            webhook_url=settings.github_webhook_url,
            events=["push"],
        )
    except Exception as e:
        logger.exception(
            f"Failed to create webhook {full_name}", owner=owner, repo=repo
        )
        raise HTTPException(502, f"GitHub API error: {e}")

    await session.commit()

    logger.info(f"Created github webhook {full_name}", repo=full_name)

    return webhook


@router.delete("/repos/{owner}/{repo}/webhooks/{hook_id}")
async def delete_webhook(
    owner: str,
    repo: str,
    hook_id: int,
    token: Annotated[str, Depends(get_github_token)],
):
    """Delete a webhook from a repository."""
    github = GitHubService(token)

    try:
        deleted = await github.delete_webhook(owner, repo, hook_id)
    except Exception as e:
        logger.exception("Failed to delete webhook")
        raise HTTPException(502, f"GitHub API error: {e}")

    if not deleted:
        raise HTTPException(404, "Webhook not found")

    return {"ok": True}


@router.post("/repos/{owner}/{repo}/webhooks/{hook_id}/ping")
async def ping_webhook(
    owner: str,
    repo: str,
    hook_id: int,
    token: Annotated[str, Depends(get_github_token)],
):
    """Trigger a test ping for a webhook."""
    github = GitHubService(token)

    try:
        success = await github.ping_webhook(owner, repo, hook_id)
    except Exception as e:
        logger.exception("Failed to ping webhook")
        raise HTTPException(502, f"GitHub API error: {e}")

    if not success:
        raise HTTPException(500, "Ping failed")

    return {"ok": True}
