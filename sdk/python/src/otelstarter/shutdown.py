"""Shutdown that sends the last telemetry, without holding the app up.

Two gaps in a plain OpenTelemetry setup, both found by compat/run.py:

- **SIGTERM drops the last batch.** Python's default SIGTERM stops the process
  at once, without running exit handlers, so spans and logs still waiting in
  the batch processors are lost: up to 5 seconds of telemetry on every
  deploy. If the app has not set its own SIGTERM handler, otelstarter sends
  them first, then raises SIGTERM again, so the process ends exactly as it
  would without otelstarter.
- **A Collector that is down holds shutdown up.** Traces, logs and metrics are
  flushed one after another, each retrying for up to the exporter's 10 s
  timeout: 20-30 s for a gunicorn or uvicorn worker, longer than Docker's
  10 s before it kills the container. otelstarter flushes the three at the
  same time and waits at most FLUSH_DEADLINE seconds in total.
"""

from __future__ import annotations

import atexit
import os
import signal
import threading
import time
from types import FrameType
from typing import Any

from opentelemetry import _logs, metrics, trace

FLUSH_DEADLINE = 3.0  # seconds


def flush(deadline: float = FLUSH_DEADLINE) -> None:
    """Shuts down the tracer, logger and meter providers at the same time,
    waiting at most deadline seconds. A provider that is still sending after
    that is left to finish, or not, while the process exits."""
    providers: list[Any] = [trace.get_tracer_provider(), _logs.get_logger_provider(), metrics.get_meter_provider()]
    threads = [
        threading.Thread(target=_shutdown, args=(provider,), name="otelstarter-flush", daemon=True)
        for provider in providers
        if callable(getattr(provider, "shutdown", None))
    ]
    for thread in threads:
        thread.start()
    end = time.monotonic() + deadline
    for thread in threads:
        thread.join(max(0.0, end - time.monotonic()))


_flushed = threading.Event()


def _shutdown(provider: Any) -> None:
    try:
        provider.shutdown()
    except Exception:  # noqa: BLE001 - shutting down must never raise into the app
        pass


def _flush_once() -> None:
    if not _flushed.is_set():
        _flushed.set()
        flush()


def _on_sigterm(signum: int, frame: FrameType | None) -> None:
    _flush_once()
    # Back to Python's default, then stop the way the app would have stopped.
    signal.signal(signal.SIGTERM, signal.SIG_DFL)
    os.kill(os.getpid(), signal.SIGTERM)


def install() -> None:
    """Flushes at exit, and on SIGTERM if the app keeps Python's default.

    Registered after OpenTelemetry's own exit handlers, so it runs before
    them; theirs then find the providers already shut down."""
    atexit.register(_flush_once)
    if threading.current_thread() is threading.main_thread() and signal.getsignal(signal.SIGTERM) is signal.SIG_DFL:
        signal.signal(signal.SIGTERM, _on_sigterm)
