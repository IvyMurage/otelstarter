from __future__ import annotations

import io
import json
import logging
import re
from collections.abc import Iterator

import pytest
from opentelemetry import trace

from otelstarter.instrumentor import OtelStarterInstrumentor
from otelstarter.structured_logs import JsonFormatter, install_json_logging


@pytest.fixture
def capture(monkeypatch: pytest.MonkeyPatch) -> Iterator[tuple[logging.Logger, io.StringIO]]:
    monkeypatch.setenv("OTEL_SERVICE_NAME", "shop")
    monkeypatch.setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment.name=test")
    stream = io.StringIO()
    handler = logging.StreamHandler(stream)
    handler.setFormatter(JsonFormatter())
    log = logging.getLogger("shop.test")
    log.addHandler(handler)
    log.setLevel(logging.DEBUG)
    log.propagate = False
    yield log, stream
    log.removeHandler(handler)


def last_entry(stream: io.StringIO) -> dict[str, object]:
    line = stream.getvalue().strip().splitlines()[-1]
    entry: dict[str, object] = json.loads(line)
    return entry


def test_fields_match_the_shared_conventions(capture: tuple[logging.Logger, io.StringIO]) -> None:
    log, stream = capture
    with trace.get_tracer("t").start_as_current_span("request") as span:
        log.info("order placed", extra={"orderId": "ord_1", "password": "hunter2", "user": {"apiKey": "k"}})

    entry = last_entry(stream)
    context = span.get_span_context()
    assert list(entry)[:6] == ["severity", "timestamp", "service.name", "environment", "trace_id", "span_id"]
    assert entry["severity"] == "INFO"
    assert re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z", str(entry["timestamp"]))
    assert entry["service.name"] == "shop"
    assert entry["environment"] == "test"
    assert entry["trace_id"] == format(context.trace_id, "032x")
    assert entry["span_id"] == format(context.span_id, "016x")
    assert entry["orderId"] == "ord_1"
    assert entry["password"] == "[REDACTED]"
    assert entry["user"] == {"apiKey": "[REDACTED]"}
    assert entry["message"] == "order placed"


@pytest.mark.parametrize(("level", "severity"), [(logging.WARNING, "WARN"), (logging.CRITICAL, "FATAL")])
def test_severity_names(capture: tuple[logging.Logger, io.StringIO], level: int, severity: str) -> None:
    log, stream = capture
    log.log(level, "x")
    assert last_entry(stream)["severity"] == severity


def test_outside_a_request_there_are_no_trace_ids(capture: tuple[logging.Logger, io.StringIO]) -> None:
    log, stream = capture
    log.info("starting")
    assert "trace_id" not in last_entry(stream)


def test_install_json_logging_on_a_bare_root_logger(monkeypatch: pytest.MonkeyPatch) -> None:
    root = logging.getLogger()
    monkeypatch.setattr(root, "handlers", [])
    monkeypatch.setattr(root, "level", logging.WARNING)

    install_json_logging()

    (handler,) = root.handlers
    assert isinstance(handler.formatter, JsonFormatter)
    assert root.level == logging.INFO


def test_install_json_logging_keeps_other_handlers(monkeypatch: pytest.MonkeyPatch) -> None:
    root = logging.getLogger()
    console, other = logging.StreamHandler(io.StringIO()), logging.NullHandler()
    monkeypatch.setattr(root, "handlers", [console, other])

    install_json_logging()

    assert isinstance(console.formatter, JsonFormatter)
    assert other.formatter is None
    assert root.handlers == [console, other]  # nothing added


def test_instrumentor_respects_opt_out(monkeypatch: pytest.MonkeyPatch) -> None:
    calls: list[bool] = []
    monkeypatch.setattr("otelstarter.instrumentor.install_json_logging", lambda: calls.append(True))
    monkeypatch.setenv("OTELSTARTER_JSON_LOGS", "false")

    OtelStarterInstrumentor()._instrument()

    assert calls == []
