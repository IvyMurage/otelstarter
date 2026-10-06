from __future__ import annotations

import logging
import subprocess
import sys

import pytest
from opentelemetry import trace
from opentelemetry._logs import SeverityNumber
from opentelemetry.sdk._logs.export import InMemoryLogRecordExporter
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter

from otelstarter import observability


def warnings_logged(caplog: pytest.LogCaptureFixture) -> list[logging.LogRecord]:
    records = [r for r in caplog.records if r.name == "otelstarter" and r.levelno == logging.WARNING]
    for r in records:
        # Every warning must point at the caller, never inside otelstarter.
        assert r.filename == "test_observability.py", f"warning points at {r.filename}:{r.lineno}"
    return records


def only_record(logs: InMemoryLogRecordExporter):  # type: ignore[no-untyped-def]
    finished = logs.get_finished_logs()
    assert len(finished) == 1, f"want exactly one log record, got {len(finished)}"
    return finished[0]


def test_event_is_one_named_log_record(logs: InMemoryLogRecordExporter) -> None:
    observability.event(
        "payment.completed",
        domain="payments",
        operation="payment",
        outcome="success",
        entity_type="payment",
        entity_id="pay_123",
    )

    readable = only_record(logs)
    record = readable.log_record
    assert record.event_name == "payment.completed"
    assert record.body == "payment.completed"
    assert record.severity_number == SeverityNumber.INFO
    assert dict(record.attributes or {}) == {
        "event.domain": "payments",
        "event.outcome": "success",
        "business.operation": "payment",
        "business.entity.type": "payment",
        "business.entity.id": "pay_123",
    }
    # Technical context comes from OpenTelemetry's resource, not the caller.
    assert readable.resource.attributes["service.name"] == "shop"
    assert readable.resource.attributes["deployment.environment.name"] == "test"


def test_only_the_name_is_required(logs: InMemoryLogRecordExporter) -> None:
    observability.event("order.created")

    record = only_record(logs).log_record
    assert record.event_name == "order.created"
    assert dict(record.attributes or {}) == {}


def test_event_is_linked_to_the_active_span(logs: InMemoryLogRecordExporter, spans: InMemorySpanExporter) -> None:
    tracer = trace.get_tracer("test")
    with tracer.start_as_current_span("POST /payments") as span:
        observability.event("payment.completed", outcome="success")

    record = only_record(logs).log_record
    assert record.trace_id == span.get_span_context().trace_id
    assert record.span_id == span.get_span_context().span_id
    # One signal only: no duplicate span event (DOMAIN_EVENTS section 13).
    (finished,) = spans.get_finished_spans()
    assert finished.events == ()


def test_integer_entity_id_is_recorded_as_text(logs: InMemoryLogRecordExporter) -> None:
    observability.event("loan.submitted", entity_id=42)

    assert only_record(logs).log_record.attributes == {"business.entity.id": "42"}


def test_badly_named_event_warns_once_and_is_still_recorded(
    logs: InMemoryLogRecordExporter, caplog: pytest.LogCaptureFixture
) -> None:
    observability.event("paymentDone")
    observability.event("paymentDone")

    (warning,) = warnings_logged(caplog)  # exactly one, despite two calls
    assert "<domain>.<action>" in warning.getMessage()
    # The record points at the caller, so formats showing file:line help.
    assert warning.filename == "test_observability.py"
    assert [r.log_record.event_name for r in logs.get_finished_logs()] == ["paymentDone", "paymentDone"]


@pytest.mark.parametrize("name", ["", None, 123])
def test_unusable_name_is_skipped_with_a_warning(
    logs: InMemoryLogRecordExporter, caplog: pytest.LogCaptureFixture, name: object
) -> None:
    observability.event(name)  # type: ignore[arg-type]

    assert "non-empty string" in warnings_logged(caplog)[0].getMessage()
    assert logs.get_finished_logs() == ()


def test_payload_values_are_kept_out(logs: InMemoryLogRecordExporter, caplog: pytest.LogCaptureFixture) -> None:
    observability.event("order.created", entity_id={"card": "4111 1111 1111 1111"})  # type: ignore[arg-type]

    (warning,) = warnings_logged(caplog)
    assert "must be a string or int" in warning.getMessage()
    assert "4111" not in warning.getMessage()  # the payload itself is never logged
    assert "business.entity.id" not in (only_record(logs).log_record.attributes or {})


def test_long_values_are_cut(logs: InMemoryLogRecordExporter, caplog: pytest.LogCaptureFixture) -> None:
    observability.event("order.created", operation="x" * 1000)

    assert "looks like a payload" in warnings_logged(caplog)[0].getMessage()
    assert len((only_record(logs).log_record.attributes or {})["business.operation"]) == 256


def test_no_catch_all_argument_exists() -> None:
    # Events carry business meaning, not payloads: arbitrary fields are a
    # programming error that type checkers and Python itself reject.
    with pytest.raises(TypeError):
        observability.event("order.created", body={"items": []})  # type: ignore[call-arg]


def test_internal_failure_never_reaches_the_app(
    logs: InMemoryLogRecordExporter, monkeypatch: pytest.MonkeyPatch, caplog: pytest.LogCaptureFixture
) -> None:
    def broken_emit(*args: object, **kwargs: object) -> None:
        raise RuntimeError("exporter exploded")

    monkeypatch.setattr(observability._logger, "emit", broken_emit)

    observability.event("payment.completed")  # must not raise

    assert "could not record event" in warnings_logged(caplog)[0].getMessage()


def test_works_without_opentelemetry_configured() -> None:
    # A fresh interpreter has no SDK configured: event() must be a quiet no-op.
    result = subprocess.run(
        [sys.executable, "-W", "error", "-c", "from otelstarter import observability; observability.event('a.b')"],
        capture_output=True,
        text=True,
        check=False,
    )
    assert result.returncode == 0, result.stderr


def test_console_summary_is_off_by_default(
    logs: InMemoryLogRecordExporter, capsys: pytest.CaptureFixture[str], monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.delenv("OTELSTARTER_CONSOLE_EVENTS", raising=False)

    observability.event("order.created", domain="orders")

    assert capsys.readouterr().err == ""


def test_console_summary_for_local_development(
    logs: InMemoryLogRecordExporter, capsys: pytest.CaptureFixture[str], monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setenv("OTELSTARTER_CONSOLE_EVENTS", "true")
    tracer = trace.get_tracer("test")

    with tracer.start_as_current_span("POST /orders") as span:
        observability.event("order.created", domain="orders", outcome="success", entity_id="ord_1")

    trace_id = f"{span.get_span_context().trace_id:032x}"
    assert capsys.readouterr().err == (
        f"[otelstarter] event order.created domain=orders outcome=success entity_id=ord_1 trace_id={trace_id}\n"
    )
    # Printing is not exporting: still exactly one record.
    only_record(logs)
