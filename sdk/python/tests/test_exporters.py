from __future__ import annotations

from opentelemetry._logs import SeverityNumber
from opentelemetry.sdk._logs import LoggerProvider
from opentelemetry.sdk._logs.export import InMemoryLogRecordExporter, SimpleLogRecordProcessor
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter

from otelstarter.exporters import PrivacyLogExporter, PrivacySpanExporter


def test_span_exporter_cleans_attributes_set_after_start() -> None:
    inner = InMemorySpanExporter()
    provider = TracerProvider()
    provider.add_span_processor(SimpleSpanProcessor(PrivacySpanExporter(inner)))
    tracer = provider.get_tracer("test")

    with tracer.start_as_current_span("SELECT", attributes={"db.system": "postgresql"}) as span:
        # Like real database instrumentation: the query arrives after the span started.
        span.set_attribute("db.statement", "SELECT * FROM users WHERE email = 'ann@shop.rw'")
        span.set_attribute("http.request.header.authorization", "Bearer secret")
        span.set_attribute("url.full", "https://api.rw/x?api_key=k1&page=2")

    (exported,) = inner.get_finished_spans()
    assert dict(exported.attributes or {}) == {
        "db.system": "postgresql",
        "db.statement": "SELECT * FROM users WHERE email = ?",
        "http.request.header.authorization": "[REDACTED]",
        "url.full": "https://api.rw/x?api_key=[REDACTED]&page=2",
    }
    assert exported.context.trace_id == span.get_span_context().trace_id  # still the same span


def test_log_exporter_cleans_attributes_and_keeps_the_rest() -> None:
    inner = InMemoryLogRecordExporter()
    provider = LoggerProvider()
    provider.add_log_record_processor(SimpleLogRecordProcessor(PrivacyLogExporter(inner)))

    provider.get_logger("test").emit(
        body="login",
        severity_number=SeverityNumber.INFO,
        attributes={"user": "ann", "password": "hunter2"},
    )

    (record,) = inner.get_finished_logs()
    assert record.log_record.body == "login"
    assert record.log_record.severity_number == SeverityNumber.INFO
    assert dict(record.log_record.attributes or {}) == {"user": "ann", "password": "[REDACTED]"}
