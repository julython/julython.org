import asyncio
import json
from pathlib import Path

import click
import yaml
from alembic import command
from alembic.config import Config

from july.app import app, start_server
from july.globals import settings


@click.group()
def cli() -> None:
    pass


@cli.command()
def dev() -> None:
    """Start the server in development mode."""
    start_server(dev=True)


@cli.command()
def prod() -> None:
    """Start the server in production mode."""
    start_server()


@cli.command()
@click.argument("api_output_file", required=True)
def swagger(api_output_file: str) -> None:
    with Path(api_output_file).open("w") as f:
        # json hack to properly serialize pydantic objects into yaml
        data = json.loads(json.dumps(app.openapi()))
        yaml.dump(data, f)


@cli.group()
def db() -> None:
    pass


@db.command(name="create")
def create_if_not_exists() -> None:
    """Create a new database if it doesn't already exist. Should only be used in local development."""
    from july.db import ensure_database_exists

    if not settings.is_local:
        raise ValueError("Only use in local development")

    loop = asyncio.get_event_loop()
    loop.run_until_complete(ensure_database_exists(settings.database_uri))


def _load_alembic_config(uri: str) -> Config:
    alembic_config = Config(str(settings.base_path / "alembic.ini"))
    alembic_config.set_main_option(
        "script_location", str(settings.base_path / "migrations")
    )
    alembic_config.set_main_option("sqlalchemy.url", uri)
    return alembic_config


@db.command()
@click.option("--uri", "-u", default=settings.database_uri)
@click.option("--revision", "-r", default="head")
@click.option("--fake", "-f", is_flag=True, default=False)
def upgrade(uri: str, revision: str, fake: bool) -> None:
    """
    Upgrades db to revision and adds a license key from ENV

    Args:
        uri: database uri
        revision: revision to upgrade
        fake: to bump migration version without applying changes
    """
    alembic_config = _load_alembic_config(uri)
    if fake:
        command.stamp(alembic_config, revision)
    else:
        command.upgrade(alembic_config, revision)


@db.command()
@click.option("--uri", "-u", default=settings.database_uri)
@click.option("--revision", "-r", default="-1")
@click.option("--fake", "-f", is_flag=True, default=False)
def downgrade(uri: str, revision: str, fake: bool) -> None:
    """
    Downgrades db to revision.

    Args:
        uri: database uri
        revision: revision to downgrade
        fake: to bump migration version without applying changes
    """
    alembic_config = _load_alembic_config(uri)
    if fake:
        command.stamp(alembic_config, revision)
    else:
        command.downgrade(alembic_config, revision)


@db.command()
@click.option("--uri", "-u", default=settings.database_uri)
@click.option("--message", "-m", required=True)
def revision(uri: str, message: str) -> None:
    """
    Creates a new db revision.

    Args:
        uri: database uri
        message: revision message
    """
    alembic_config = _load_alembic_config(uri)
    command.revision(alembic_config, message, autogenerate=True)


@db.command()
@click.option("--uri", "-u", default=settings.database_uri)
def history(uri: str) -> None:
    """
    Shows db revision history.

    Args:
        uri: database uri
    """
    alembic_config = _load_alembic_config(uri)
    command.history(alembic_config, verbose=False)
