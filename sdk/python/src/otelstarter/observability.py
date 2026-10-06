"""Record business events alongside your app's OpenTelemetry telemetry.

    from otelstarter import observability

    observability.event("payment.completed", domain="payments", outcome="success")

Each call emits one OpenTelemetry log-based event: a log record whose event
name is the event, linked to the current trace. OpenTelemetry adds the
technical context (trace and span IDs, service name, environment) itself.

Recording an event never raises, so observability can never break the app.
Problems are logged once each as warnings on the "otelstarter" logger, so they
appear with your other logs, in the terminal and in your backend.

For local development, set OTELSTARTER_CONSOLE_EVENTS=true (otelstarter run
does this) to also print a one-line summary of each event to stderr. The
summary is not exported, so each event is still sent exactly once.
"""

from __future__ import annotations

import logging
import os
import re
import sys

from opentelemetry import trace
from opentelemetry._logs import SeverityNumber, get_logger

__all__ = ["event"]

# <domain>.<action>, lowercase: "payment.completed", "payment.authorization.failed".
_NAME_PATTERN = re.compile(r"^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$")

# Longer values look like payloads, not business meaning.
_MAX_VALUE_LENGTH = 256

# A proxy logger: it starts sending once the app configures OpenTelemetry
# (for example via opentelemetry-instrument) and does nothing before that.
_logger = get_logger("otelstarter")

# Problems with an event are logged here. Through OpenTelemetry's logging
# instrumentation they reach the backend like any other warning log.
_log = logging.getLogger("otelstarter")

_CONSOLE_ENV = "OTELSTARTER_CONSOLE_EVENTS"

_warned: set[str] = set()


def event(
    name: str,
    *,
    domain: str | None = None,
    operation: str | None = None,
    outcome: str | None = None,
    entity_type: str | None = None,
    entity_id: str | int | None = None,
) -> None:
    """Record that something meaningful happened in the business.

    Args:
        name: What happened, as ``<domain>.<action>``, for example
            ``"payment.completed"``. The only required argument.
        domain: The business capability, for example ``"payments"``.
        operation: The business activity, for example ``"payment"``.
        outcome: Preferably ``"success"``, ``"failure"``, ``"cancelled"``
            or ``"rejected"``.
        entity_type: The business object involved, for example ``"order"``.
        entity_id: Its identifier. Recorded on the event only; otelstarter
            never turns it into a metric label.

    There is deliberately no way to attach other data: events carry business
    meaning, not business payloads.
    """
    try:
        _emit(name, domain, operation, outcome, entity_type, entity_id)
    except Exception as exc:  # noqa: BLE001 - recording must never break the app
        _warn_once("emit-failed", f"otelstarter could not record event {name!r}: {exc!r}", depth=1)


def _emit(
    name: object,
    domain: object,
    operation: object,
    outcome: object,
    entity_type: object,
    entity_id: object,
) -> None:
    if not isinstance(name, str) or not name:
        _warn_once(
            "name-type", f"Event name must be a non-empty string, got {name!r}. The event was not recorded.", depth=2
        )
        return
    if not _NAME_PATTERN.match(name):
        _warn_once(
            "name:" + name,
            f"Event name {name!r} does not follow <domain>.<action> in lowercase, "
            "for example 'payment.completed'. The event was still recorded.",
            depth=2,
        )

    fields = {
        "event.domain": domain,
        "event.outcome": outcome,
        "business.operation": operation,
        "business.entity.type": entity_type,
        "business.entity.id": entity_id,
    }
    attributes: dict[str, str] = {}
    for key, value in fields.items():
        cleaned = _clean(name, key, value)
        if cleaned is not None:
            attributes[key] = cleaned

    _logger.emit(
        event_name=name,
        body=name,  # so log views that show only the body still show the event
        severity_number=SeverityNumber.INFO,
        severity_text="INFO",
        attributes=attributes,
    )

    if os.environ.get(_CONSOLE_ENV, "").lower() == "true":
        _print_summary(name, attributes)


def _print_summary(name: str, attributes: dict[str, str]) -> None:
    """Print one line about the event to stderr, for local development."""
    parts = [f"[otelstarter] event {name}"]
    short = {
        "event.domain": "domain",
        "event.outcome": "outcome",
        "business.operation": "operation",
        "business.entity.type": "entity_type",
        "business.entity.id": "entity_id",
    }
    parts += [f"{short[key]}={value}" for key, value in attributes.items()]
    span_context = trace.get_current_span().get_span_context()
    if span_context.is_valid:
        parts.append(f"trace_id={span_context.trace_id:032x}")
    print(" ".join(parts), file=sys.stderr, flush=True)


def _clean(name: str, key: str, value: object) -> str | None:
    """Return the value as a short string, or None to leave the field out."""
    if value is None:
        return None
    # bool is a subclass of int, so rule it out explicitly.
    if isinstance(value, bool) or not isinstance(value, (str, int)):
        _warn_once(
            f"type:{key}",
            f"{key} on event {name!r} must be a string or int, got {type(value).__name__}. "
            "It was left out: events carry business meaning, not payloads.",
            depth=3,
        )
        return None
    text = str(value)
    if len(text) > _MAX_VALUE_LENGTH:
        _warn_once(
            f"long:{key}",
            f"{key} on event {name!r} is {len(text)} characters long, which looks like a payload. "
            f"It was cut to {_MAX_VALUE_LENGTH}.",
            depth=3,
        )
        text = text[:_MAX_VALUE_LENGTH]
    return text


def _warn_once(key: str, message: str, depth: int) -> None:
    """Log a warning once per key.

    depth is how many otelstarter functions sit between the user's event()
    call and this one, so the record points at the user's file and line.
    """
    if key in _warned:
        return
    _warned.add(key)
    _log.warning(message, stacklevel=depth + 2)
