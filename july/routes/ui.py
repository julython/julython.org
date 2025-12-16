from fastapi import APIRouter, Request, responses
from structlog.stdlib import get_logger
from july.globals import settings

router = APIRouter(include_in_schema=False)

log = get_logger(__name__)


@router.get("/{path:path}")
@router.head("/{path:path}")
async def spa_fallback(request: Request, path: str) -> responses.Response:
    file_path = settings.static_dir / path

    if not file_path.is_file():
        file_path = settings.static_dir / "index.html"

    response = responses.FileResponse(file_path)
    etag = response.headers.get("etag")

    if etag and request.headers.get("if-none-match") == etag:
        return responses.Response(status_code=304)

    return response
