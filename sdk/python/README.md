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
- **`<domain>.<action>` names**, in lowercase, such as `order.created` or `payment.authorization.failed`. A name that breaks the rule is still recorded, with a one-time `OtelStarterWarning`.
- **Never breaks your app.** `event()` never raises. Internal errors become a one-time `OtelStarterWarning`.
- **One signal per event.** No duplicate span event or log line, and no automatic metric, so `entity_id` never becomes a metric label.

## Finding events in Grafana

With `otelstarter run`, open Grafana → Explore → Loki:

```text
{service_name="shop"} |= "payment."                       # all payment events
{service_name="shop"} | event_domain="payments"           # by domain
{service_name="shop"} | event_outcome="failure"           # failures only
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
