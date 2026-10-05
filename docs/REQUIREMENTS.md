# OTelStarter Requirements

**Status:** Draft  
**Version:** 1.0

## 1. Product Vision

OTelStarter makes production observability accessible by automatically configuring useful logs, metrics, traces, and business-aware domain events.

Target experience:

```bash
otelstarter init
otelstarter run
```

Within minutes:

```text
Logs         ✓
Metrics      ✓
Traces       ✓
Correlation  ✓
Domain Events ✓
Dashboard    ✓
```

## 2. Functional Requirements

### FR-001 Project Detection
The system MUST detect the primary language.

Initial support:
- Python
- Node.js
- Go

The system SHOULD detect frameworks and major dependencies.

### FR-002 Dependency Detection
The system MUST detect observability-relevant dependencies:
- PostgreSQL
- MySQL
- Redis
- Kafka
- RabbitMQ
- HTTP clients
- ORMs

### FR-003 Instrumentation Recommendation
The system MUST recommend appropriate OpenTelemetry instrumentation based on detected technologies.

### FR-004 Automatic Instrumentation
The system SHOULD use automatic instrumentation whenever possible.

### FR-005 Logs
The system MUST support structured logs.

Logs SHOULD contain:
- timestamp
- severity
- service.name
- environment
- trace_id
- span_id

### FR-006 Metrics
The MVP MUST support:
- Request rate
- Error rate
- Request latency
- Runtime metrics

Metrics MUST avoid uncontrolled high-cardinality dimensions.

### FR-007 Traces
The MVP MUST support:
- Incoming HTTP
- Outgoing HTTP
- Database operations

where instrumentation is available.

### FR-008 Correlation
Logs and traces MUST be correlated.

## 3. Domain Event Requirements

### FR-009 Domain Event API
The SDK MUST provide a lightweight API.

Example:

```python
observability.event(
    "payment.completed",
    domain="payments",
    outcome="success"
)
```

### FR-010 Automatic Technical Context
Domain events MUST inherit available:
- trace_id
- span_id
- service.name
- environment
- request_id

### FR-011 Business Context
Domain events SHOULD support:
- domain
- operation
- entity.type
- entity.id
- outcome

All are optional except event name.

### FR-012 Privacy
Domain events MUST NOT automatically capture sensitive business payloads.

### FR-013 Domain Event Naming
Use:

```text
<domain>.<action>
```

Examples:
```text
payment.completed
payment.failed
loan.submitted
order.created
order.cancelled
user.authenticated
```

Avoid:
```text
paymentDone
paymentStuff
handlePayment
paymentEvent1
```

## 4. CLI Requirements

MVP:
```text
otelstarter init
otelstarter analyze
otelstarter run
otelstarter validate
otelstarter doctor
otelstarter version
```

Future:
```text
otelstarter deploy
otelstarter recommend
otelstarter upgrade
otelstarter uninstall
```

## 5. Safety Requirements

The system MUST warn against:
- Secrets
- Passwords
- Tokens
- Authorization headers
- Private keys
- Sensitive request bodies
- Sensitive identifiers

The system SHOULD provide redaction/filtering mechanisms.

## 6. Configuration Requirements

Generated configuration MUST:
- Be deterministic.
- Be human-readable.
- Avoid unnecessary complexity.
- Avoid overwriting user configuration without confirmation.
- Be validated before being applied.
- Support rollback where practical.

## 7. Idempotency

Running `otelstarter init` multiple times MUST be safe.

The tool MUST recognize existing configuration and avoid duplication.

## 8. Validation

`otelstarter validate` MUST validate:
- Dependencies
- Configuration
- Instrumentation
- Collector
- Environment
- Telemetry generation

## 9. Diagnostics

`otelstarter doctor` MUST provide actionable errors.

Bad:
```text
Configuration failed.
```

Good:
```text
FastAPI instrumentation is installed but no OTLP endpoint is configured.

Try:
Set OTEL_EXPORTER_OTLP_ENDPOINT.
```

## 10. Language Requirements

### Python
- FastAPI
- Flask
- Django

### Node.js
- Express
- NestJS
- Fastify

### Go
- net/http
- Gin
- Fiber

## 11. Non-Functional Requirements

### Performance
Observability MUST introduce minimal overhead.

The project MUST benchmark:
- CPU
- Memory
- Request latency
- Telemetry throughput

before making performance claims.

### Reliability
Telemetry failures MUST NOT bring down the application.

### Security
The project MUST follow secure defaults.

### Privacy
Telemetry should favor metadata over payloads.

### Portability
Generated configurations SHOULD support:
- Local
- Docker
- Kubernetes
- Cloud

## 12. Open Source Requirements

The project MUST include:
```text
README.md
CONTRIBUTING.md
CODE_OF_CONDUCT.md
SECURITY.md
LICENSE
CHANGELOG.md
```

Documentation MUST explain:
- Architecture
- Development setup
- Adding languages
- Adding integrations
- Testing
- Contribution process

## 13. MVP Scope

Included:
- CLI
- Python
- Node.js
- Go
- Project detection
- Dependency detection
- Logs
- Metrics
- Traces
- Domain events
- Business context
- Technical context
- Correlation
- OTEL Collector
- Local environment
- Basic dashboard
- Safety checks
- Validation
- Doctor

Not MVP:
- AI recommendations
- Kubernetes automation
- Hosted SaaS
- Advanced anomaly detection
- Large-scale analytics

## 14. Success Criteria

A developer should be able to take a supported application and reach working observability in under 10 minutes.

Success flow:

```text
Clone application
      ↓
otelstarter init
      ↓
otelstarter run
      ↓
Generate traffic
      ↓
View metrics
      ↓
Open trace
      ↓
Find correlated logs
      ↓
Understand business event
```

Judge the product primarily on developer experience and operational usefulness, not feature count.
