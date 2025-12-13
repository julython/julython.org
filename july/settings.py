from __future__ import annotations

import tomllib
from functools import cached_property
from pathlib import Path
from typing import Optional
from structlog.stdlib import get_logger
from pydantic_settings import BaseSettings, SettingsConfigDict

BASE_PATH = Path(__file__).resolve().parents[1]
ENV_VAR_PREFIX = "JULY_"


log = get_logger(__name__)


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        env_prefix=ENV_VAR_PREFIX,
        extra="ignore",
    )

    # The base path of the service
    base_path: Path = BASE_PATH

    # App settings, used to tell uvicorn how to run
    debug: bool = False
    host: str = "0.0.0.0"
    port: int = 8000

    # Logging settings
    service_id: str = "july"
    json_logging: bool = False
    disable_access_log: bool = False

    # TODO: Consider integrating this somehow with HOST or DEBUG settings
    is_dev: bool = True
    is_local: bool = True

    database_scheme: str = "postgresql+asyncpg://"
    database_hostname: str = "localhost"
    database_username: str = "postgres"
    database_password: str = "postgres"
    database_name: str = "july"
    database_available: bool = False

    cors_origins: list[str] = []

    # The parameters used by FastAPI when generating API docs
    api_prefix: str = ""
    api_title: str = "Julython API"
    api_description: str = ""

    auth_providers: set[str] = {"JWTAuthProvider", "DirectPassportProvider"}
    auth_api_url: str = "https://auth.stage.anaconda.com/api/auth"
    auth_oauth_client_id: str = "local-dev-host"
    auth_oauth_client_secret: str = ""

    openapi_url: str = "/api/openapi.json"
    openapi_servers: list[dict[str, str]] = []
    docs_url: str = "/api/docs"
    redoc_url: Optional[str] = "/api"
    root_path: str = ""

    @cached_property
    def database_uri(self) -> str:
        """Return the database URI construct from individual components."""
        return f"{self.database_scheme}{self.database_username}:{self.database_password}@{self.database_hostname}/{self.database_name}"

    @cached_property
    def pyproject(self) -> dict:
        log.info("looking for pyproject")
        pyproject_path = self.base_path / "pyproject.toml"
        with open(pyproject_path, "rb") as f:
            return tomllib.load(f).get("project", {})
