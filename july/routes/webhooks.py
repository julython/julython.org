import json
from typing import Annotated

import structlog
from fastapi import (
    APIRouter,
    Depends,
    Header,
    HTTPException,
    Request,
    Response,
    responses,
)
from sqlalchemy.ext.asyncio import AsyncSession

from july.dependencies import get_session
from july.services import webhook_service

logger = structlog.stdlib.get_logger(__name__)

router = APIRouter(tags=["Webhooks"])


async def process_and_respond(
    payload: webhook_service.WebhookPayload, db_session: AsyncSession
) -> responses.JSONResponse:
    service = webhook_service.WebhookService(db_session)
    commits = await service.process_webhook(payload)

    return responses.JSONResponse(
        content={
            "provider": payload.provider,
            "project": payload.repo.slug,
            "commits": [c.hash for c in commits],
        },
        status_code=201 if commits else 200,
    )


@router.post("/api/v1/github")
async def github_webhook(
    request: Request,
    db_session: Annotated[AsyncSession, Depends(get_session)],
    x_github_event: Annotated[str | None, Header()] = None,
    content_type: Annotated[str, Header()] = "application/json",
):
    if x_github_event == "ping":
        return Response(content="pong", media_type="text/plain")

    if "application/json" in content_type:
        data = await request.json()
    elif "form" in content_type:
        form = await request.form()
        if "payload" in form:
            data = json.loads(str(form.get("payload")))
        else:
            data = dict(form)
    else:
        raise HTTPException(400, f"Unsupported content type: {content_type}")

    payload = webhook_service.parse_github(data)
    logger.info(f"webhook for {payload.repo.slug} with {len(payload.commits)} commits")
    return await process_and_respond(payload, db_session)


@router.post("/api/v1/gitlab")
async def gitlab_webhook(
    request: Request,
    db_session: Annotated[AsyncSession, Depends(get_session)],
):
    data = await request.json()
    payload = webhook_service.parse_gitlab(data)
    return await process_and_respond(payload, db_session)


@router.post("/api/v1/bitbucket")
async def bitbucket_webhook(
    request: Request,
    db_session: Annotated[AsyncSession, Depends(get_session)],
    content_type: Annotated[str, Header()] = "application/json",
):
    if "application/json" in content_type:
        data = await request.json()
    elif "form" in content_type:
        form = await request.form()
        if "payload" in form:
            data = json.loads(str(form.get("payload")))
        else:
            data = dict(form)
    else:
        raise HTTPException(400, f"Unsupported content type: {content_type}")

    payload = webhook_service.parse_bitbucket(data)
    return await process_and_respond(payload, db_session)
