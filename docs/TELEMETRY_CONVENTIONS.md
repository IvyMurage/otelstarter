# Telemetry conventions shared by the SDKs

The Node.js and Python SDKs must produce the same structured logs, apply the same redaction and sanitise queries the same way, so the backend looks identical whichever language a service uses. This page is the contract. The test cases in [`sdk/conventions.json`](../sdk/conventions.json) are run by both SDKs, so they cannot drift apart.

## Structured logs

One JSON object per line on stdout. The same entry is also sent to OpenTelemetry, linked to its trace, when the app runs under otelstarter.

| Field | Always | Example |
|-------|--------|---------|
| `timestamp` | yes | `"2026-10-07T10:36:20.852Z"`, ISO 8601, UTC |
| `severity` | yes | `"DEBUG"`, `"INFO"`, `"WARN"`, `"ERROR"` or `"FATAL"` |
| `message` | yes | `"order placed"` |
| `service.name` | yes | from `OTEL_SERVICE_NAME` |
| `environment` | when set | from `deployment.environment.name` in `OTEL_RESOURCE_ATTRIBUTES` |
| `trace_id`, `span_id` | inside a traced request | 32 and 16 hex characters |
| *your fields* | when given | `"orderId": "ord_1"` |

```json
{"severity":"INFO","timestamp":"2026-10-07T10:36:20.852Z","service.name":"shop","environment":"local","trace_id":"6dc47d148bc575361bf3d9fbd9abdc93","span_id":"8930bdd53b53f51f","orderId":"ord_1","password":"[REDACTED]","message":"order placed"}
```

## Redaction

The value of a field or attribute becomes `[REDACTED]` when its **name** looks sensitive. The name is split into words at `_`, `-`, `.` and camelCase; for dotted attribute names such as `http.request.header.authorization`, only the last part counts. A name is sensitive when:

- **one of its words** is `password`, `passwd`, `passphrase`, `secret`, `token`, `authorization`, `cookie`, `credential(s)`, `cvv`, `ssn`, `otp`, `jwt`, `pin`, `apikey`, `sessionid` or `privatekey`; or
- **two neighbouring words** are `api key`, `private key`, `access key`, `session id`, `card number`, `credit card` or `connection string`.

So `client_secret`, `passwordHash`, `aws_secret_access_key`, `x-api-key`, `id_token` and `session_id` are redacted. Only whole words count, so `tokens_used`, `passwordless_login_enabled` and `secretary` are not.

- **Logs:** fields at any depth, up to 10 levels (`user.credentials.apiKey`).
- **Spans:** any attribute. In URLs (`url.full`, `url.query`, `http.url`, `http.target`):
  - a user and password before the host are hidden: `https://[REDACTED]@db.example.com/x`;
  - sensitive query parameters are hidden: `/reset?token=[REDACTED]&lang=en`.

## Query sanitising

In `db.query.text` and `db.statement`:

- **String and number literals become `?`:** `WHERE email = 'ann@shop.rw' AND age > 30` → `WHERE email = ? AND age > ?`. This covers:
  - PostgreSQL's `E'…'` and `$$…$$` strings;
  - hex literals (`0xDEADBEEF`);
  - MySQL's double-quoted strings when compared with something (`WHERE email = "ann@shop.rw"`).
- **Lists collapse:** `IN (1, 2, 3)` → `IN (?)`
- **Unchanged:**
  - placeholders (`$1`, `?`);
  - names that contain digits (`t2`);
  - double-quoted names (`"users"."email"`), which ORMs generate everywhere.

Spans are cleaned when they end, before any exporter sees them, because instrumentations may set the query after the span starts.

## Domain events

See [DOMAIN_EVENTS.md](DOMAIN_EVENTS.md). Both SDKs emit one log-based event with the same attribute names (`event.domain`, `event.outcome`, `business.operation`, `business.entity.type`, `business.entity.id`).
