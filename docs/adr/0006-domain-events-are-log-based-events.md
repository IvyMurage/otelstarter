# ADR 0006: Domain events are OpenTelemetry log-based events

**Status:** Accepted
**Date:** 2026-10-05

## Context

`DOMAIN_EVENTS.md` asks for `observability.event("payment.completed", domain=..., outcome=...)`, with technical context attached automatically, no payloads, no redundant telemetry (section 13), and events "attached to the active span where supported" (section 15). The API runs inside the user's application, so it must be written in the application's language. Python is first.

OpenTelemetry has since decided that events are **log records with an event name**, emitted through the Logs API and correlated with traces through context. It is [deprecating the Span Events API](https://opentelemetry.io/blog/2026/deprecating-span-events/) (`Span.AddEvent`) in favour of them ([OTEP 4430](https://github.com/open-telemetry/opentelemetry-specification/blob/main/oteps/4430-span-event-api-deprecation-plan.md)). The Python API's `Logger.emit()` has an `event_name` parameter.

## Decision

- **Each `event()` call emits one log record** through the OpenTelemetry Logs API:
  - `event_name` and `body` are both the event name
  - severity is `INFO`
  - attributes are `event.domain`, `event.outcome`, `business.operation`, `business.entity.type` and `business.entity.id`, only those given

  Trace and span IDs come from the current context. Service name and environment come from the resource.
- **No span event, no extra log line and no automatic metric.** This departs from section 15's "attach to the active span", because OpenTelemetry is deprecating that API. The log-based event is still linked to the span through its trace and span IDs.
- **The body repeats the event name,** because some backends do not store the event-name field. Loki in `grafana/otel-lgtm:0.35.0` drops it; a debug Collector confirmed it is sent.
- **The SDK lives in this repo at `sdk/python/`,** is named `otelstarter`, and is imported as `from otelstarter import observability`. It depends only on `opentelemetry-api`: without a configured SDK, `event()` does nothing.
- **Payloads are prevented by design:**
  - the function has only the named fields, with no `**kwargs`
  - values must be short strings or ints; other types are dropped and values over 256 characters are cut, each with a warning
- **`event()` never raises.** Bad names (not `<domain>.<action>`) are still recorded, with a one-time `OtelStarterWarning`; internal errors become the same warning.
- **`request_id` is not attached.** OpenTelemetry has no standard source for it.

## Alternatives

- **Span events:** what section 15 describes, but the API is being deprecated.
- **A span event plus a log record:** duplicate telemetry, which section 13 forbids.
- **A separate repository for each language SDK:** cleaner release cycles, but two repositories to keep in sync for a solo maintainer.
- **Dropping or rejecting badly named events:** stricter naming, but production data would be lost, or the app could break.

## Consequences

- Events appear in Loki next to the request's logs, and link to the trace in Tempo through `trace_id`.
- In Loki, events are found by body text (`|= "payment.completed"`) or by attribute (`event_domain="payments"`), not by event name, until Loki stores that field.
- `DOMAIN_EVENTS.md` section 15 now describes this approach and points here.
- The package is not on PyPI yet. Until it is published, users install it from GitHub, and `otelstarter init` does not add it.
