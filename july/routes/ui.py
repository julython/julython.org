from fastapi import APIRouter, HTTPException, Request, responses
from structlog.stdlib import get_logger
from july.globals import settings

router = APIRouter(include_in_schema=False)

log = get_logger(__name__)


@router.get("/{path:path}")
@router.head("/{path:path}")
async def spa_fallback(request: Request, path: str):
    file_path = settings.static_dir / path
    is_asset = path.startswith(settings.asset_prefix[1:])
    is_file = file_path.is_file()

    if is_asset and not is_file:
        raise HTTPException(status_code=404, detail=f"{path} not found")

    elif not is_file:
        file_path = settings.static_dir / "index.html"

    response = responses.FileResponse(file_path)
    etag = response.headers.get("etag")

    if etag and request.headers.get("if-none-match") == etag:
        return responses.Response(status_code=304)

    return response
