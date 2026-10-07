"""Exporters that clean telemetry before sending it over OTLP.

Python finishes a span by freezing its attributes, so nothing can change them
afterwards. Instead, these exporters pass on a cleaned copy of each span and
log record: SQL values become ?, secrets become [REDACTED]. They are selected
in otel.env and found by OpenTelemetry through entry points:

    OTEL_TRACES_EXPORTER=otelstarter
    OTEL_LOGS_EXPORTER=otelstarter

They wrap the OTLP HTTP/protobuf exporters, which otelstarter init installs.
"""

from __future__ import annotations

import copy
import dataclasses
from collections.abc import Mapping, Sequence
from typing import Any

from opentelemetry.sdk._logs import ReadableLogRecord
from opentelemetry.sdk._logs.export import LogRecordExporter, LogRecordExportResult
from opentelemetry.sdk.trace import ReadableSpan
from opentelemetry.sdk.trace.export import SpanExporter, SpanExportResult

from otelstarter.redaction import REDACTED, is_sensitive_key, redact_url
from otelstarter.sanitize import sanitize_query

__all__ = ["PrivacyLogExporter", "PrivacySpanExporter", "clean_attributes"]

_QUERY_ATTRIBUTES = frozenset({"db.query.text", "db.statement"})
_URL_ATTRIBUTES = frozenset({"url.full", "url.query", "http.url", "http.target"})


def clean_attributes(attributes: Mapping[str, Any] | None) -> dict[str, Any]:
    """Return a cleaned copy of span or log attributes."""
    cleaned: dict[str, Any] = {}
    for key, value in (attributes or {}).items():
        if is_sensitive_key(key):
            cleaned[key] = REDACTED
        elif isinstance(value, str) and key in _QUERY_ATTRIBUTES:
            cleaned[key] = sanitize_query(value)
        elif isinstance(value, str) and key in _URL_ATTRIBUTES:
            cleaned[key] = redact_url(value)
        else:
            cleaned[key] = value
    return cleaned


def _clean_span(span: ReadableSpan) -> ReadableSpan:
    return ReadableSpan(
        name=span.name,
        context=span.context,
        parent=span.parent,
        resource=span.resource,
        attributes=clean_attributes(span.attributes),
        events=span.events,
        links=span.links,
        kind=span.kind,
        status=span.status,
        start_time=span.start_time,
        end_time=span.end_time,
        instrumentation_scope=span.instrumentation_scope,
    )


def _clean_log(record: ReadableLogRecord) -> ReadableLogRecord:
    # A shallow copy keeps trace IDs and timestamps; only the attributes change.
    log_record = copy.copy(record.log_record)
    log_record.attributes = clean_attributes(record.log_record.attributes)
    return dataclasses.replace(record, log_record=log_record)


class PrivacySpanExporter(SpanExporter):
    """OTLP span exporter that cleans each span first."""

    def __init__(self, inner: SpanExporter | None = None) -> None:
        if inner is None:
            from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter

            inner = OTLPSpanExporter()
        self._inner = inner

    def export(self, spans: Sequence[ReadableSpan]) -> SpanExportResult:
        return self._inner.export([_clean_span(span) for span in spans])

    def shutdown(self) -> None:
        self._inner.shutdown()

    def force_flush(self, timeout_millis: int = 30000) -> bool:
        return self._inner.force_flush(timeout_millis)


class PrivacyLogExporter(LogRecordExporter):
    """OTLP log exporter that cleans each record's attributes first."""

    def __init__(self, inner: LogRecordExporter | None = None) -> None:
        if inner is None:
            from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter

            inner = OTLPLogExporter()
        self._inner = inner

    def export(self, batch: Sequence[ReadableLogRecord]) -> LogRecordExportResult:
        return self._inner.export([_clean_log(record) for record in batch])

    def shutdown(self) -> None:
        self._inner.shutdown()  # type: ignore[no-untyped-call]  # untyped upstream

    def force_flush(self, timeout_millis: int = 30000) -> bool:
        return self._inner.force_flush(timeout_millis)
