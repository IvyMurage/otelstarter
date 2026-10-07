"""Structured JSON logs, one line per entry (docs/TELEMETRY_CONVENTIONS.md).

    log.info("order placed", extra={"orderId": "ord_1", "password": "hunter2"})
    {"severity": "INFO", "timestamp": "2026-10-07T10:36:20.852Z", "service.name": "shop",
     "environment": "local", "trace_id": "...", "span_id": "...", "orderId": "ord_1",
     "password": "[REDACTED]", "message": "order placed"}

Under opentelemetry-instrument, otelstarter's instrumentor turns this on for
the root logger automatically. Set OTELSTARTER_JSON_LOGS=false to keep your
own log format.
"""

from __future__ import annotations

import json
import logging
import os
import sys
from datetime import datetime, timezone
from typing import Any

from opentelemetry import trace

from otelstarter.redaction import REDACTED, is_sensitive_key

__all__ = ["JsonFormatter", "environment_name", "install_json_logging", "service_name"]

_SEVERITY = {"DEBUG": "DEBUG", "INFO": "INFO", "WARNING": "WARN", "ERROR": "ERROR", "CRITICAL": "FATAL"}

# Attributes every LogRecord has; anything else came from extra={...}.
_STANDARD = frozenset(vars(logging.LogRecord("", 0, "", 0, "", None, None))) | {"message", "asctime", "taskName"}


def service_name() -> str:
    return os.environ.get("OTEL_SERVICE_NAME") or "unknown_service"


def environment_name() -> str:
    """deployment.environment.name from OTEL_RESOURCE_ATTRIBUTES, or ""."""
    for pair in os.environ.get("OTEL_RESOURCE_ATTRIBUTES", "").split(","):
        key, _, value = pair.partition("=")
        if key.strip() in ("deployment.environment.name", "deployment.environment"):
            return value.strip()
    return ""


def _redact(value: Any, depth: int = 1) -> Any:
    """Redact sensitive keys in a dict, and in dicts one level down."""
    if not isinstance(value, dict):
        return value
    return {
        key: REDACTED if is_sensitive_key(str(key)) else (_redact(item, depth - 1) if depth > 0 else item)
        for key, item in value.items()
    }


class JsonFormatter(logging.Formatter):
    """Formats each record as one JSON object with the shared fields."""

    def format(self, record: logging.LogRecord) -> str:
        entry: dict[str, Any] = {
            "severity": _SEVERITY.get(record.levelname, record.levelname),
            "timestamp": datetime.fromtimestamp(record.created, tz=timezone.utc)
            .isoformat(timespec="milliseconds")
            .replace("+00:00", "Z"),
            "service.name": service_name(),
        }
        environment = environment_name()
        if environment:
            entry["environment"] = environment
        # Formatting runs synchronously where the log call happened, so the
        # current span is the one the log belongs to.
        span = trace.get_current_span().get_span_context()
        if span.is_valid:
            entry["trace_id"] = format(span.trace_id, "032x")
            entry["span_id"] = format(span.span_id, "016x")

        extras = {
            key: value for key, value in vars(record).items() if key not in _STANDARD and not key.startswith("otel")
        }
        entry.update(_redact(extras))
        entry["message"] = record.getMessage()
        if record.exc_info:
            entry["exception"] = self.formatException(record.exc_info)
        return json.dumps(entry, default=str)


def install_json_logging() -> None:
    """Use JsonFormatter on the root logger's console output.

    Plain console handlers get the JSON format. Other handlers, such as
    OpenTelemetry's own handler that exports logs, are left alone. With no
    console handler yet, one writing to stdout is added at INFO level.
    """
    root = logging.getLogger()
    consoles = [h for h in root.handlers if type(h) is logging.StreamHandler]
    if not consoles:
        handler = logging.StreamHandler(sys.stdout)
        root.addHandler(handler)
        consoles = [handler]
        if root.level == logging.NOTSET or root.level > logging.INFO:
            root.setLevel(logging.INFO)
    for handler in consoles:
        handler.setFormatter(JsonFormatter())
