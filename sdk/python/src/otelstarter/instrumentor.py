"""Runs when the app starts under opentelemetry-instrument.

It switches on structured JSON logs, and a shutdown that sends the last
telemetry without holding the app up (see shutdown.py).

opentelemetry-instrument loads every installed instrumentor, so installing the
otelstarter package is enough; no code changes. Set OTELSTARTER_JSON_LOGS=false
to keep the app's own log format.
"""

from __future__ import annotations

import os
from collections.abc import Collection
from typing import Any

from opentelemetry.instrumentation.instrumentor import BaseInstrumentor  # type: ignore[attr-defined]

from otelstarter import shutdown
from otelstarter.structured_logs import install_json_logging


class OtelStarterInstrumentor(BaseInstrumentor):  # type: ignore[misc]
    def instrumentation_dependencies(self) -> Collection[str]:
        return []

    def _instrument(self, **kwargs: Any) -> None:
        if os.environ.get("OTELSTARTER_JSON_LOGS", "true").lower() != "false":
            install_json_logging()
        shutdown.install()

    def _uninstrument(self, **kwargs: Any) -> None:
        pass
