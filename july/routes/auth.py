import secrets
from typing import Annotated, Literal

from fastapi import APIRouter, Depends, HTTPException, Request, responses
from sqlalchemy.ext.asyncio import AsyncSession
from structlog.stdlib import get_logger

from july.dependencies import get_session
from july.globals import settings
from july.services import auth_service, user_service

logger = get_logger(__name__)
router = APIRouter(tags=["Auth"])

PROVIDERS: dict[str, auth_service.OAuthProviderBase] = {
    "github": auth_service.GitHubOAuth(
        client_id=settings.github_client_id,
        client_secret=settings.github_client_secret,
        redirect_uri=settings.auth_callback,
    ),
    "gitlab": auth_service.GitLabOAuth(
        client_id=settings.gitlab_client_id,
        client_secret=settings.gitlab_client_secret,
        redirect_uri=settings.auth_callback,
    ),
}


@router.get("/auth/login/{provider}")
async def login(request: Request, provider: Literal["github", "gitlab"]):
    state = secrets.token_urlsafe(16)
    verifier, challenge = auth_service.generate_pkce_pair()
    request.session["oauth_state"] = state
    request.session["oauth_provider"] = provider
    request.session["oauth_verifier"] = verifier

    auth_provider = PROVIDERS[provider]

    auth_url = auth_provider.get_authorization_url(state, challenge)
    return responses.RedirectResponse(auth_url)


@router.get("/auth/callback")
async def callback(
    request: Request,
    code: str,
    state: str,
    session: Annotated[AsyncSession, Depends(get_session)],
):
    verifier = request.session.pop("oauth_verifier", None)
    expected_state = request.session.pop("oauth_state", None)
    if not expected_state or state != expected_state:
        raise HTTPException(400, "Invalid state")

    provider = request.session.get("oauth_provider", "")
    auth_provider = PROVIDERS.get(provider)
    if not auth_provider:
        raise HTTPException(400, "Invalid provider")

    users = user_service.UserService(session)
    tokens = await auth_provider.exchange_code(code, verifier)
    user_info = await auth_provider.get_user(tokens)
    user, created = await users.oauth_login_or_register(user_info)

    if created:
        logger.info(f"{user.name} created")

    # Store in session
    request.session["user"] = user.model_dump(mode="json")
    request.session["identity_key"] = user_info.key

    return responses.RedirectResponse("/", status_code=302)


@router.get("/auth/session")
async def get_user_session(request: Request) -> dict:
    return request.session


@router.get("/auth/logout")
async def logout(request: Request):
    access_token = request.session.get("access_token")
    auth_provider = PROVIDERS.get(request.session.get("oauth_provider", ""))

    if auth_provider and access_token:
        await auth_provider.revoke_token(access_token)

    request.session.clear()
    return {"ok": True}
