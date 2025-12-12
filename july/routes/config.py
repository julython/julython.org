from fastapi import APIRouter, responses

from july.globals import settings

router = APIRouter(tags=["Configuration"])


@router.get("/api/config")
def health() -> responses.JSONResponse:
    return responses.JSONResponse(
        {
            "status": "ok",
            "is_dev": settings.is_dev,
            "is_local": settings.is_local,
            "api_title": settings.api_title,
            "version": settings.pyproject.get("version", "unknown"),
        }
    )
