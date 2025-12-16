from starlette.datastructures import MutableHeaders
from starlette.types import ASGIApp, Message, Receive, Scope, Send
from structlog.stdlib import get_logger

log = get_logger(__name__)


class CacheHeadersMiddleware:
    def __init__(self, app: ASGIApp, asset_prefix: str, api_prefix: str):
        self.app = app
        self.asset_prefix = asset_prefix
        log.info(
            "Setting up CacheHeaderMiddleware",
            asset_prefix=asset_prefix,
        )

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        path: str = scope["path"]
        is_asset = path.startswith(self.asset_prefix)

        async def send_wrapper(message: Message) -> None:
            if message["type"] == "http.response.start":
                headers = MutableHeaders(raw=list(message["headers"]))

                if "etag" in headers:
                    if is_asset:
                        headers["Cache-Control"] = "public, max-age=31536000, immutable"
                    else:
                        headers["Cache-Control"] = "no-cache"
                else:
                    headers.setdefault("Cache-Control", "no-store")

                message["headers"] = headers.raw
            await send(message)

        await self.app(scope, receive, send_wrapper)
