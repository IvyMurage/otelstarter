"""Shutdown, checked from outside the process: how it ends, and how fast."""

from __future__ import annotations

import signal
import subprocess
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

# An app with OpenTelemetry set up the way opentelemetry-instrument does,
# exporting to a Collector that is down, and a span waiting to be sent.
APP = """
import sys, time
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

provider = TracerProvider()
provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint="{endpoint}/v1/traces")))
trace.set_tracer_provider(provider)
trace.get_tracer("t").start_span("unsent").end()
{setup}
print("ready", flush=True)
time.sleep(60)
"""


DOWN = "http://127.0.0.1:9"  # nothing listens there


def run_and_stop(setup: str, endpoint: str = DOWN) -> tuple[int, float]:
    command = [sys.executable, "-c", APP.format(setup=setup, endpoint=endpoint)]
    with subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True) as child:
        assert child.stdout is not None
        assert child.stdout.readline().strip() == "ready"
        started = time.monotonic()
        child.send_signal(signal.SIGTERM)
        code = child.wait(timeout=30)
        return code, time.monotonic() - started


def test_sigterm_ends_the_process_like_without_otelstarter() -> None:
    code, took = run_and_stop("from otelstarter import shutdown; shutdown.install()")
    assert code == -signal.SIGTERM, "stopped by SIGTERM, as Python does by default"
    assert took < 5, f"took {took:.1f} s with the Collector down"


def test_an_apps_own_sigterm_handler_stays_in_charge() -> None:
    setup = (
        "import signal\n"
        "signal.signal(signal.SIGTERM, lambda *a: sys.exit(7))\n"
        "from otelstarter import shutdown; shutdown.install()"
    )
    code, took = run_and_stop(setup)
    assert code == 7
    assert took < 5, f"exit handlers took {took:.1f} s with the Collector down"


def test_sigterm_sends_the_waiting_span_first() -> None:
    received: list[str] = []

    class Collector(BaseHTTPRequestHandler):
        def do_POST(self) -> None:  # noqa: N802
            self.rfile.read(int(self.headers.get("Content-Length", 0)))
            received.append(self.path)
            self.send_response(200)
            self.send_header("Content-Length", "0")
            self.end_headers()

        def log_message(self, *args: object) -> None:
            pass

    server = ThreadingHTTPServer(("127.0.0.1", 0), Collector)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    try:
        endpoint = f"http://127.0.0.1:{server.server_address[1]}"
        code, _ = run_and_stop("from otelstarter import shutdown; shutdown.install()", endpoint)
    finally:
        server.shutdown()
        server.server_close()
    assert code == -signal.SIGTERM
    assert received == ["/v1/traces"], "the span was lost: Python's default SIGTERM skips exit handlers"
