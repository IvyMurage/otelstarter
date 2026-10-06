# OTelStarter Domain Events

**Status:** Draft  
**Version:** 1.0

## 1. Purpose

Domain events give telemetry business meaning.

Technical telemetry can tell us:

```text
POST /payments → 504 → 4.2s
```

A domain event can tell us:

```text
payment.authorization.failed
```

Together they allow engineers to understand both:

> What happened technically?

and:

> What business operation was affected?

## 2. Design Principle

> **Business meaning, not business payloads.**

Domain events should identify what happened without automatically capturing sensitive business data.

## 3. Lightweight API

Preferred API:

```python
observability.event(
    "payment.completed",
    domain="payments",
    outcome="success"
)
```

The SDK should automatically attach available technical context.

Developers should not manually pass:
- trace_id
- span_id
- service.name
- environment
- request_id

unless there is a specific reason.

## 4. Event Schema

A domain event may contain:

```text
event.name
event.domain
event.outcome

business.operation
business.entity.type
business.entity.id

service.name
service.version
deployment.environment

trace_id
span_id
request_id

timestamp
```

Only `event.name` is required.

## 5. Event Naming

Use:

```text
<domain>.<action>
```

Examples:

```text
payment.completed
payment.failed
payment.authorization.started
payment.authorization.failed

loan.submitted
loan.approved
loan.rejected

order.created
order.fulfilled
order.cancelled

user.authenticated
user.authentication.failed

subscription.created
subscription.cancelled
```

Avoid:

```text
paymentDone
PaymentCompleted
doPayment
handlePayment
paymentEvent
paymentEvent1
```

An event name describes **what happened**, not which function executed.

## 6. Domain Names

Domain names should represent business capabilities.

Good:
```text
payments
lending
orders
identity
subscriptions
inventory
```

Avoid technical domains:
```text
controllers
database
http
api
utils
```

unless those are genuinely the business domain.

## 7. Outcomes

Use a small consistent vocabulary where possible:

```text
success
failure
cancelled
rejected
```

Do not create dozens of synonymous outcome values.

## 8. Business Operation

The operation describes the business activity.

Examples:

```text
payment
loan_application
order
authentication
transfer
subscription
```

## 9. Entity Type

Entity types describe the business object involved.

Examples:

```text
payment
loan_application
order
user
subscription
invoice
```

## 10. Entity ID

Entity IDs may be useful for logs/traces when permitted, but they should NOT automatically become metric labels.

The implementation must consider:
- Privacy
- Security
- Cardinality
- Data retention
- Regulatory requirements

## 11. Technical Context

A domain event should inherit technical context from the active execution context.

Example:

```json
{
  "event.name": "payment.completed",
  "event.domain": "payments",
  "event.outcome": "success",
  "service.name": "payment-service",
  "deployment.environment": "production",
  "trace_id": "abc123",
  "span_id": "def456"
}
```

## 12. Correlation

Domain events should be associated with the current trace/span whenever possible.

Desired flow:

```text
HTTP Request
    ↓
Trace
    ↓
Payment Processing
    ↓
payment.authorization.started
    ↓
External Provider
    ↓
payment.completed
```

## 13. Logging Domain Events

Domain events may be represented in logs, but the system should avoid creating redundant telemetry.

Do not automatically emit:
1. A domain event.
2. A duplicate application log.
3. Another custom trace event.

if the same information can be represented cleanly.

The implementation should prefer semantic reuse.

## 14. Domain Events and Metrics

Do not turn every domain event into a metric automatically.

For example:

```text
payment.completed
```

could be represented as a metric when aggregate counts are operationally useful:

```text
payments_completed_total
```

But high-cardinality event attributes must not automatically become metric labels.

## 15. Domain Events and Traces

Domain events belong to the active span. Conceptually:

```text
Trace
└── payment.process
    ├── event: payment.authorization.started
    ├── PostgreSQL span
    ├── external-provider span
    └── event: payment.completed
```

They are linked to the span through its `trace_id` and `span_id`, not added to it as span events. Each event is an OpenTelemetry **log-based event**: a log record with an event name, emitted through the Logs API. OpenTelemetry is [deprecating the Span Events API](https://opentelemetry.io/blog/2026/deprecating-span-events/) in favour of log-based events ([OTEP 4430](https://github.com/open-telemetry/opentelemetry-specification/blob/main/oteps/4430-span-event-api-deprecation-plan.md)). Using one signal also keeps section 13's rule against redundant telemetry.

Backends show the event next to the span's logs and link from it to the trace. See [ADR 0006](adr/0006-domain-events-are-log-based-events.md).

## 16. Privacy

Never automatically include:
- Passwords
- Tokens
- API keys
- Authorization headers
- Full request bodies
- Credit card numbers
- National IDs
- Sensitive personal information

Prefer metadata.

## 17. Cardinality

Never automatically use domain entity IDs as metric labels.

Bad:

```text
payment_completed{payment_id="123456"}
```

Better:

```text
payment_completed_total
```

Use IDs in traces/logs only when justified and controlled.

## 18. Domain Event Examples

### Successful payment

```python
observability.event(
    "payment.completed",
    domain="payments",
    operation="payment",
    outcome="success"
)
```

### Failed payment

```python
observability.event(
    "payment.failed",
    domain="payments",
    operation="payment",
    outcome="failure"
)
```

### Loan application

```python
observability.event(
    "loan.submitted",
    domain="lending",
    operation="loan_application",
    outcome="success"
)
```

## 19. Automatic vs Explicit Context

Use automatic instrumentation for technical context.

Use explicit domain events for business meaning.

This is the core model:

```text
Automatic:
HTTP
Database
Runtime
Messaging
Trace context

Explicit:
Business events
Business operations
Business outcomes
```

## 20. Future Direction

A future intelligent recommendation engine may analyze an application and suggest useful domain events.

For example:

```text
Detected:
POST /payments
PaymentService
PaymentRepository
PaymentProvider

Suggested domain events:

payment.authorization.started
payment.authorization.failed
payment.completed
```

This should remain a recommendation until the developer approves it.

The AI must never silently introduce business events based on uncertain assumptions.

## 21. Core Principle

> **Capture enough business context to explain operational impact, but never turn observability into uncontrolled business-data collection.**
