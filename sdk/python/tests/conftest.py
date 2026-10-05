"""Configure an in-memory OpenTelemetry SDK once for the whole test run.

OpenTelemetry allows the global providers to be set only once per process,
so each test clears the exporters instead of creating new providers.
"""

from __future__ import annotations

from collections.abc import Iterator

import pytest
from opentelemetry import trace
from opentelemetry._logs import set_logger_provider
from opentelemetry.sdk._logs import LoggerProvider
from opentelemetry.sdk._logs.export import InMemoryLogRecordExporter, SimpleLogRecordProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter

from otelstarter import observability

RESOURCE = Resource.create({"service.name": "shop", "deployment.environment.name": "test"})

_logs = InMemoryLogRecordExporter()
_log_provider = LoggerProvider(resource=RESOURCE)
_log_provider.add_log_record_processor(SimpleLogRecordProcessor(_logs))
set_logger_provider(_log_provider)

_spans = InMemorySpanExporter()
_tracer_provider = TracerProvider(resource=RESOURCE)
_tracer_provider.add_span_processor(SimpleSpanProcessor(_spans))
trace.set_tracer_provider(_tracer_provider)


@pytest.fixture
def logs() -> Iterator[InMemoryLogRecordExporter]:
    _logs.clear()
    observability._warned.clear()  # each test sees warnings afresh
    yield _logs


@pytest.fixture
def spans() -> Iterator[InMemorySpanExporter]:
    _spans.clear()
    yield _spans
