"""
Structured logging
==================

This module allows us to use json logs in production and pretty dev console locally.
Also it allows us to 'build' a logger that has context of the request or operation.
It can be used with the plain stdlib logging so it works with all existing apps/deps.

Setup
-----

First you'll need to replace the existing logging setup with the `setup_logging` function.
This needs to be called before the app is created or when a cli command is executed.
In our applications the two common places are `app.py` and `cli.py`.

In production you'll want to add a setting for `json_logging` then set that in the setup call::

    setup_logging(
        service_id=settings.service_id,
        debug=settings.debug,
        json_logging=settings.json_logging,
    )

Then remove the logger in the uvicorn setup::

    def start_server(dev: bool = False) -> None:  # pragma: no cover
        uvicorn.run(
            app_import_path_string,
            host=settings.host,
            port=settings.port,
            # disable logging so the process uses the logger we setup
            log_config=None,
            reload=dev,
            # also disable access_log as well ans use the logger to define it
            access_log=False,
        )


Extra Loggers
-------------

Sometimes you may need to modify the logging output of a noisy module or increase the output
for development. Since this builds off of the standard `dictConfig` it is an easy interface
to change. Example::

    extra_loggers = {
        # Quiet down a logger
        "some.noisy.logger": {"level": "ERROR"},
        # Disable third party logger
        "do.not.want.this": {"handlers": []},
    }
    setup_logging(
        service_id=settings.service_id,
        extra_loggers=extra_loggers,
    )
"""

import logging.config
from typing import Any, Dict, List, Optional

import structlog
from structlog.types import Processor


def setup_logging(
    service_id: str,
    debug: bool = False,
    json_logging: bool = False,
    disable_access_log: bool = True,
    extra_loggers: Optional[Dict[str, Any]] = None,
) -> None:
    """Configure structlog for service.

    Structlog enables our services to use json formatting in production
    along with a simple interface to add extra context to logs. It works
    with standard logging so there is no need to change loggers until you
    do want to use extra features.

    Example (app.py and cli.py)::

        # Override loggers for noisy things
        extra_loggers = {"some.noisy.logger": {"level": "ERROR"}}
        setup_logging(
            service_id=settings.service_id,
            debug=settings.debug,
            json_logging=settings.json_logging,
            extra_loggers=extra_loggers,
        )

    Args:
        service_id: The service_id that is logging aka 'bigbend'
        debug: Enable 'DEBUG' for the service_id else use 'INFO'
        json_logging: Format the output in json should be True in PROD.
        disable_access_log: Disable the very noisy and redundant uvicorn access log.
        extra_loggers: dictConfig for overriding the default levels for other loggers.

    """
    log_level = "DEBUG" if debug else "INFO"

    shared_processors: List[Processor] = [
        # Add structlog context variables to log lines
        structlog.contextvars.merge_contextvars,
        # Adds a timestamp for every log line
        structlog.processors.TimeStamper(fmt="iso"),
        # Add the name of the logger to the record
        structlog.stdlib.add_logger_name,
        # Adds the log level as a parameter of the log line
        structlog.stdlib.add_log_level,
        # Perform old school %-style formatting. on the log msg/event
        structlog.stdlib.PositionalArgumentsFormatter(),
        # If the log record contains a string in byte format, this will automatically convert it into a utf-8 string
        structlog.processors.UnicodeDecoder(),
        # Rename the events to `message` to work with datadog.
        structlog.processors.EventRenamer(to="message"),
    ]

    json_processors: List[Processor] = [
        *shared_processors,
        # Include stack traces in exceptions
        structlog.processors.format_exc_info,
    ]

    structlog_processors: List[Processor] = [
        *shared_processors,
        structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
    ]

    structlog.configure(
        processors=structlog_processors,
        # Defines how the logs will be printed out.
        logger_factory=structlog.stdlib.LoggerFactory(),
        cache_logger_on_first_use=True,
    )

    # Override the formatter with the correct output.
    formatter = "json_formatter" if json_logging else "dev_console"

    # Optionally disable access logs, in production these are basically just noise.
    # The istio proxy logs the request with more information like the external IP
    # and does that globally for all requests.
    _disable_access_log = (
        {"uvicorn.access": {"handlers": [], "propagate": False}}
        if disable_access_log
        else {}
    )
    _extra_loggers = extra_loggers or {}
    # defaults that can be overridden or extended with `extra_loggers`
    _default_loggers = {
        "": {
            "handlers": ["default"],
            "level": "INFO",
        },
        service_id: {"level": log_level},
        "uvicorn": {"level": "INFO"},
        "uvicorn.error": {"level": "INFO"},
        "aiokafka": {"level": "INFO"},
        "aiokafka.producer": {"level": "INFO"},
        "aiokafka.conn": {"level": "INFO"},
        "aiokafka.consumer.fetcher": {"level": "INFO"},
        "aiokafka.consumer.group_coordinator": {"level": "INFO"},
        "ddtrace": {"level": "INFO"},
        "ddtrace.monkey": {"level": "INFO"},
        "ddtrace.internal": {"level": "ERROR"},
    }

    all_loggers = {
        **_default_loggers,
        **_disable_access_log,
        **_extra_loggers,
    }

    logging.config.dictConfig(
        {
            "version": 1,
            "disable_existing_loggers": False,
            "formatters": {
                "json_formatter": {
                    "()": structlog.stdlib.ProcessorFormatter,
                    "processor": structlog.processors.JSONRenderer(),
                    "foreign_pre_chain": json_processors,
                },
                # Format log records into messages intended for the console
                "dev_console": {
                    "()": structlog.stdlib.ProcessorFormatter,
                    "processor": structlog.dev.ConsoleRenderer(
                        sort_keys=True,
                        colors=True,
                        event_key="message",
                    ),
                    "foreign_pre_chain": shared_processors,
                },
            },
            "handlers": {
                "default": {
                    "class": "logging.StreamHandler",
                    "formatter": formatter,
                    "stream": "ext://sys.stderr",
                },
            },
            "loggers": all_loggers,
        }
    )
