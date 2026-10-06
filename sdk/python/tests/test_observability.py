from __future__ import annotations

import subprocess
import sys
import warnings

import pytest
from opentelemetry import trace
from opentelemetry._logs import SeverityNumber
from opentelemetry.sdk._logs.export import InMemoryLogRecordExporter
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter

from otelstarter import observability
from otelstarter.observability import OtelStarterWarning


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


def test_badly_named_event_warns_once_and_is_still_recorded(logs: InMemoryLogRecordExporter) -> None:
    with pytest.warns(OtelStarterWarning, match="<domain>.<action>"):
        observability.event("paymentDone")

    with warnings.catch_warnings():
        warnings.simplefilter("error")  # a second warning would fail here
        observability.event("paymentDone")

    assert [r.log_record.event_name for r in logs.get_finished_logs()] == ["paymentDone", "paymentDone"]


@pytest.mark.parametrize("name", ["", None, 123])
def test_unusable_name_is_skipped_with_a_warning(logs: InMemoryLogRecordExporter, name: object) -> None:
    with pytest.warns(OtelStarterWarning, match="non-empty string"):
        observability.event(name)  # type: ignore[arg-type]

    assert logs.get_finished_logs() == ()


def test_payload_values_are_kept_out(logs: InMemoryLogRecordExporter) -> None:
    with pytest.warns(OtelStarterWarning, match="must be a string or int"):
        observability.event("order.created", entity_id={"card": "4111 1111 1111 1111"})  # type: ignore[arg-type]

    assert "business.entity.id" not in (only_record(logs).log_record.attributes or {})


def test_long_values_are_cut(logs: InMemoryLogRecordExporter) -> None:
    with pytest.warns(OtelStarterWarning, match="looks like a payload"):
        observability.event("order.created", operation="x" * 1000)

    assert len((only_record(logs).log_record.attributes or {})["business.operation"]) == 256


def test_no_catch_all_argument_exists() -> None:
    # Events carry business meaning, not payloads: arbitrary fields are a
    # programming error that type checkers and Python itself reject.
    with pytest.raises(TypeError):
        observability.event("order.created", body={"items": []})  # type: ignore[call-arg]


def test_internal_failure_never_reaches_the_app(
    logs: InMemoryLogRecordExporter, monkeypatch: pytest.MonkeyPatch
) -> None:
    def broken_emit(*args: object, **kwargs: object) -> None:
        raise RuntimeError("exporter exploded")

    monkeypatch.setattr(observability._logger, "emit", broken_emit)

    with pytest.warns(OtelStarterWarning, match="could not record event"):
        observability.event("payment.completed")  # must not raise


def test_works_without_opentelemetry_configured() -> None:
    # A fresh interpreter has no SDK configured: event() must be a quiet no-op.
    result = subprocess.run(
        [sys.executable, "-W", "error", "-c", "from otelstarter import observability; observability.event('a.b')"],
        capture_output=True,
        text=True,
        check=False,
    )
    assert result.returncode == 0, result.stderr
