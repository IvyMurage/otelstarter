# otelstarter (Python)

Business-aware domain events for OpenTelemetry. Part of [otelstarter](https://github.com/IvyMurage/otelstarter).

```python
from otelstarter import observability

observability.event(
    "payment.completed",
    domain="payments",
    operation="payment",
    outcome="success",
    entity_type="payment",
    entity_id=payment.id,
)
```

Each call emits **one OpenTelemetry log-based event**: a log record whose event name is `payment.completed`, linked to the current trace. OpenTelemetry adds the trace and span IDs, service name and environment itself.

## Install

```bash
pip install otelstarter
```

`otelstarter init` adds it for you, pinned to the version it was tested with.

It depends only on `opentelemetry-api`. Run your app with OpenTelemetry configured (for example via `otelstarter run -- <your command>`), or `event()` quietly does nothing.

## Rules it enforces

- **Business meaning, not payloads.** Only the named fields above exist, with values that are short strings or ints. Anything else is left out with a warning.
- **`<domain>.<action>` names**, in lowercase, such as `order.created` or `payment.authorization.failed`. A name that breaks the rule is still recorded, and a warning is logged once.
- **Never breaks your app.** `event()` never raises. Internal errors are logged once as a warning.
- **One signal per event.** No duplicate span event or log line, and no automatic metric, so `entity_id` never becomes a metric label.

## Structured logs, redaction and query sanitising (0.3.0)

With the settings `otelstarter init` writes to `otel.env`, an app started by `otelstarter run` (or `opentelemetry-instrument`) gets three things, with no code changes:

- **Console logs become one JSON object per line,** using the fields in [docs/TELEMETRY_CONVENTIONS.md](https://github.com/IvyMurage/otelstarter/blob/main/docs/TELEMETRY_CONVENTIONS.md), shared with the Node.js SDK:
  ```python
  log.info("order placed", extra={"orderId": "ord_1", "password": "hunter2"})
  # {"severity": "INFO", "timestamp": "…", "service.name": "shop", "environment": "local",
  #  "trace_id": "…", "span_id": "…", "orderId": "ord_1", "password": "[REDACTED]", "message": "order placed"}
  ```
  Set `OTELSTARTER_JSON_LOGS=false` to keep your own format.
- **Secrets are redacted** in logs and in spans: passwords, tokens, API keys, `Authorization` and cookies.
- **SQL values become `?`:** `WHERE email = 'ann@shop.rw'` → `WHERE email = ?`.

This works because `otel.env` selects the SDK's exporters:

```bash
OTEL_TRACES_EXPORTER=otelstarter
OTEL_LOGS_EXPORTER=otelstarter
```

They make a cleaned copy of each span and log record, then send it with the standard OTLP HTTP exporter. Python freezes a finished span, so copying is the only safe way to change what gets exported.

**FastAPI note.** Recent FastAPI versions (0.142 in testing) log *"FastAPI automatic telemetry configuration failed"* with these settings. It is harmless: it means FastAPI did not add a second exporter. With `otlp`, it would, and every span would be sent twice. To silence the message: `FastAPI(telemetry={"auto_configure": False})`.

## Seeing events while you develop

Set `OTELSTARTER_CONSOLE_EVENTS=true` and each event also prints one line to stderr. `otelstarter run -- <command>` sets this for you.

```text
[otelstarter] event order.created domain=orders outcome=success entity_id=ord_1 trace_id=e566aa670b7683784763c4860c4f6e71
```

This line is only printed, never exported, so the event is still sent once.

Problems such as a badly named event are logged as warnings on the `otelstarter` logger. They appear in your terminal next to your own logs, pointing at the line that called `event()`, and in your backend:

```text
WARNING [otelstarter] [main.py:14] [trace_id=...] - Event name 'ordercreated' does not follow <domain>.<action> ...
```

To turn them off: `logging.getLogger("otelstarter").setLevel(logging.ERROR)`.

## Finding events in Grafana

With `otelstarter run`, open Grafana → Explore → Loki:

```text
{service_name="shop"} |= "payment."                       # all payment events
{service_name="shop"} | event_domain="payments"           # by domain
{service_name="shop"} | event_outcome="failure"           # failures only
{service_name="shop"} | scope_name="otelstarter" | detected_level="warn"   # event problems
```

Each event carries `trace_id`, so you can jump from it to the request's trace in Tempo. Loki does not store OpenTelemetry's event-name field, which is why the event name is also the log body.

See [docs/DOMAIN_EVENTS.md](https://github.com/IvyMurage/otelstarter/blob/main/docs/DOMAIN_EVENTS.md) for the event model.

## Development

```bash
cd sdk/python
python -m venv .venv && . .venv/bin/activate
pip install -e '.[dev]'
pytest
ruff check . && ruff format --check .
mypy
```
