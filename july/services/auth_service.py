from abc import ABC, abstractmethod

import hashlib
import base64
import secrets
from typing import override

import httpx
import structlog

from july.schema import EmailAddress, OAuthProvider, OAuthTokens, OAuthUser

logger = structlog.stdlib.get_logger(__name__)


class OAuthProviderBase(ABC):
    authorize_url: str
    token_url: str
    user_url: str
    supports_pkce: bool = False
    tokens_expire: bool = False
    scopes: str

    def __init__(self, client_id: str, client_secret: str, redirect_uri: str):
        self.client_id = client_id
        self.client_secret = client_secret
        self.redirect_uri = redirect_uri

    def get_authorization_url(
        self, state: str, pkce_challenge: str | None = None
    ) -> str:

        params = {
            "client_id": self.client_id,
            "redirect_uri": self.redirect_uri,
            "response_type": "code",
            "state": state,
            "scope": self.scopes,
        }
        if pkce_challenge and self.supports_pkce:
            params["code_challenge"] = pkce_challenge
            params["code_challenge_method"] = "S256"
        return f"{self.authorize_url}?{httpx.QueryParams(params)}"

    async def exchange_code(
        self, code: str, pkce_verifier: str | None = None
    ) -> OAuthTokens:

        data = {
            "client_id": self.client_id,
            "client_secret": self.client_secret,
            "code": code,
            "grant_type": "authorization_code",
            "redirect_uri": self.redirect_uri,
        }
        if pkce_verifier:
            data["code_verifier"] = pkce_verifier

        async with httpx.AsyncClient() as client:
            resp = await client.post(
                self.token_url, data=data, headers={"Accept": "application/json"}
            )
            resp.raise_for_status()
            data = resp.json()
            if "error" in data:
                raise ValueError(data.get("error_description", data["error"]))

            return OAuthTokens(
                access_token=data["access_token"],
                refresh_token=data.get("refresh_token"),
                expires_in=data.get("expires_in"),
            )

    async def refresh_access_token(self, refresh_token: str) -> OAuthTokens:

        async with httpx.AsyncClient() as client:
            resp = await client.post(
                self.token_url,
                data={
                    "client_id": self.client_id,
                    "client_secret": self.client_secret,
                    "refresh_token": refresh_token,
                    "grant_type": "refresh_token",
                    "redirect_uri": self.redirect_uri,
                },
            )
            resp.raise_for_status()
            data = resp.json()
            return OAuthTokens(
                access_token=data["access_token"],
                refresh_token=data.get("refresh_token"),
                expires_in=data.get("expires_in"),
            )

    @abstractmethod
    async def get_user(self, tokens: OAuthTokens) -> OAuthUser: ...

    @abstractmethod
    async def revoke_token(self, tokens: OAuthTokens) -> bool: ...


class GitHubOAuth(OAuthProviderBase):
    provider = OAuthProvider.GITHUB
    authorize_url = "https://github.com/login/oauth/authorize"
    token_url = "https://github.com/login/oauth/access_token"
    user_url = "https://api.github.com/user"
    supports_pkce = False
    tokens_expire = False
    scopes = "read:user user:email"

    async def refresh_access_token(self, refresh_token: str) -> OAuthTokens:
        # GitHub OAuth tokens don't expire, no refresh needed
        raise NotImplementedError("GitHub OAuth tokens don't expire")

    @override
    async def get_user(self, tokens: OAuthTokens) -> OAuthUser:
        headers = {
            "Authorization": f"Bearer {tokens.access_token}",
            "Accept": "application/vnd.github+json",
        }

        async with httpx.AsyncClient(headers=headers) as client:
            resp = await client.get(
                self.user_url,
            )
            resp.raise_for_status()
            data = resp.json()
            logger.info(f"User {data['login']} logged in via github")

            user_emails = await client.get(f"{self.user_url}/emails")
            user_emails.raise_for_status()
            email_data = user_emails.json()
            emails = [EmailAddress(**e) for e in email_data]

            return OAuthUser(
                id=str(data["id"]),
                provider=self.provider,
                username=data["login"],
                emails=emails,
                name=data.get("name"),
                avatar_url=data.get("avatar_url"),
                data=tokens.model_dump(mode="json"),
            )

    @override
    async def revoke_token(self, tokens: OAuthTokens) -> bool:

        async with httpx.AsyncClient() as client:
            resp = await client.request(
                "DELETE",
                f"https://api.github.com/applications/{self.client_id}/token",
                auth=(self.client_id, self.client_secret),
                headers={
                    "Accept": "application/vnd.github+json",
                    "X-GitHub-Api-Version": "2022-11-28",
                },
                json={"access_token": tokens.access_token},
            )
            return resp.status_code in (204, 404)


class GitLabOAuth(OAuthProviderBase):
    provider = OAuthProvider.GITLAB
    authorize_url = "https://gitlab.com/oauth/authorize"
    token_url = "https://gitlab.com/oauth/token"
    user_url = "https://gitlab.com/api/v4/user"
    revoke_url = "https://gitlab.com/oauth/revoke"
    supports_pkce = True
    tokens_expire = True
    scopes = "openid email"

    def __init__(
        self,
        client_id: str,
        client_secret: str,
        redirect_uri: str,
        base_url: str = "https://gitlab.com",
    ):
        super().__init__(client_id, client_secret, redirect_uri)
        self.base_url = base_url.rstrip("/")

    @override
    async def get_user(self, tokens: OAuthTokens) -> OAuthUser:

        async with httpx.AsyncClient() as client:
            resp = await client.get(
                self.user_url,
                headers={"Authorization": f"Bearer {tokens.access_token}"},
            )
            resp.raise_for_status()
            data = resp.json()
            return OAuthUser(
                id=str(data["id"]),
                provider=self.provider,
                username=data["username"],
                emails=data.get("emails", []),
                name=data.get("name"),
                avatar_url=data.get("avatar_url"),
                data=tokens.model_dump(mode="json"),
            )

    @override
    async def revoke_token(self, tokens: OAuthTokens) -> bool:

        async with httpx.AsyncClient() as client:
            resp = await client.post(
                self.revoke_url,
                data={
                    "client_id": self.client_id,
                    "client_secret": self.client_secret,
                    "token": tokens.access_token,
                },
            )
            return resp.status_code == 200


# PKCE helpers
def generate_pkce_pair() -> tuple[str, str]:
    """Returns (verifier, challenge)"""
    verifier = secrets.token_urlsafe(32)
    challenge = (
        base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest())
        .rstrip(b"=")
        .decode()
    )
    return verifier, challenge
