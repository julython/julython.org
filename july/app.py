from contextlib import asynccontextmanager
from typing import AsyncGenerator

import uvicorn
from fastapi import FastAPI
from starlette.middleware.cors import CORSMiddleware

from july.routes import github
from july.globals import context, settings
from july.utils.logger import setup_logging

setup_logging("july", debug=True, disable_access_log=False)


def init_middlewares(app: FastAPI) -> FastAPI:
    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
    return app


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator:
    await context.initialize(settings)
    yield
    await context.shutdown()


def create_app() -> FastAPI:
    """Factory pattern to generate an app instance."""
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

    app = init_middlewares(app)

    # URL routes
    app.include_router(github.router)
    return app


# The main instance of the app, which is run by uvicorn
app = create_app()

# The string reference to the app object.
# We need to use a string if we want to use reload=True during development.
app_import_path_string = "july.app:app"


def start_server(dev: bool = False) -> None:  # pragma: no cover
    uvicorn.run(
        app_import_path_string,
        host=settings.host,
        port=settings.port,
        log_config=None,
        reload=dev,
        access_log=False,
    )


if __name__ == "__main__":
    start_server()
