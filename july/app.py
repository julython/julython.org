from contextlib import asynccontextmanager
from typing import AsyncGenerator

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles
from starlette.middleware.cors import CORSMiddleware

from july.routes import config, ui, webhooks
from july.globals import context, settings, Settings
from july.middleware import CacheHeadersMiddleware
from july.utils.logger import setup_logging


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator:
    await context.initialize(settings)
    yield
    await context.shutdown(settings)


def create_app(settings: Settings) -> FastAPI:
    """Factory pattern to generate an app instance."""
    setup_logging(
        "july",
        debug=settings.debug,
        json_logging=settings.json_logging,
        disable_access_log=settings.disable_access_log,
    )

    app = FastAPI(
        title=settings.api_title,
        description=settings.api_description,
        servers=settings.openapi_servers,
        openapi_url=settings.openapi_url,
        docs_url=settings.docs_url,
        redoc_url=settings.redoc_url,
        root_path=settings.root_path,
        lifespan=lifespan,
    )

    # CORS Settings
    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # Cache Settings
    app.add_middleware(
        CacheHeadersMiddleware,
        asset_prefix=settings.asset_prefix,
        api_prefix=settings.api_prefix,
    )

    # Add API routes
    app.include_router(config.router)
    app.include_router(webhooks.router)

    # Add UI routes last as they are greedy
    app.mount(
        "/assets",
        StaticFiles(directory=settings.static_dir / "assets"),
        name="assets",
    )
    app.include_router(ui.router)

    return app
