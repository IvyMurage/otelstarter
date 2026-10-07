# Telemetry conventions shared by the SDKs

The Node.js and Python SDKs must produce the same structured logs, apply the same redaction and sanitise queries the same way, so the backend looks identical whichever language a service uses. This page is the contract. Each SDK has tests that check its output against it.

## Structured logs

One JSON object per line on stdout. The same entry is also sent to OpenTelemetry, linked to its trace, when the app runs under otelstarter.

| Field | Always | Example |
|-------|--------|---------|
| `timestamp` | yes | `"2026-10-07T10:36:20.852Z"`, ISO 8601, UTC |
| `severity` | yes | `"DEBUG"`, `"INFO"`, `"WARN"` or `"ERROR"` |
| `message` | yes | `"order placed"` |
| `service.name` | yes | from `OTEL_SERVICE_NAME` |
| `environment` | when set | from `deployment.environment.name` in `OTEL_RESOURCE_ATTRIBUTES` |
| `trace_id`, `span_id` | inside a traced request | 32 and 16 hex characters |
| *your fields* | when given | `"orderId": "ord_1"` |

```json
{"severity":"INFO","timestamp":"2026-10-07T10:36:20.852Z","service.name":"shop","environment":"local","trace_id":"6dc47d148bc575361bf3d9fbd9abdc93","span_id":"8930bdd53b53f51f","orderId":"ord_1","password":"[REDACTED]","message":"order placed"}
```

## Redaction

The value of any field or attribute with one of these names becomes `[REDACTED]`. Names are compared without case; for dotted attribute names such as `http.request.header.authorization` the last part counts.

`password`, `passwd`, `secret`, `token`, `access_token`/`accessToken`, `refresh_token`/`refreshToken`, `api_key`/`apiKey`, `x-api-key`, `x-auth-token`, `authorization`, `proxy-authorization`, `cookie`, `set-cookie`, `connection_string`/`connectionString`, `private_key`/`privateKey`, `credit_card`/`creditCard`, `card_number`/`cardNumber`, `cvv`, `ssn`

- **Logs:** top-level fields and fields one level down (`user.apiKey`).
- **Spans:** any attribute. In URLs (`url.full`, `url.query`, `http.url`, `http.target`), sensitive query parameters are hidden: `/reset?token=[REDACTED]&lang=en`.

## Query sanitising

In `db.query.text` and `db.statement`:

- **String and number literals become `?`:** `WHERE email = 'ann@shop.rw' AND age > 30` → `WHERE email = ? AND age > ?`
- **Lists collapse:** `IN (1, 2, 3)` → `IN (?)`
- **Unchanged:** placeholders (`$1`, `?`) and names that contain digits (`t2`).

Spans are cleaned when they end, before any exporter sees them, because instrumentations may set the query after the span starts.

## Domain events

See [DOMAIN_EVENTS.md](DOMAIN_EVENTS.md). Both SDKs emit one log-based event with the same attribute names (`event.domain`, `event.outcome`, `business.operation`, `business.entity.type`, `business.entity.id`).
