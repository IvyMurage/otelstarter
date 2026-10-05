# OTelStarter Architecture

**Status:** Draft  
**Version:** 1.0  
**Audience:** Maintainers, contributors, AI coding agents, reviewers

## 1. Purpose

OTelStarter is a lightweight, developer-first observability bootstrapper that makes it easy to add useful production observability to applications.

The system provides:
- Logs
- Metrics
- Traces
- Domain events
- Business context
- Technical context
- Correlation between telemetry signals
- Automatic instrumentation
- Safe defaults
- Lightweight configuration
- Multi-language support

OTelStarter is **not an observability backend**. It builds on OpenTelemetry and existing observability technologies.

The goal is:

> **Make production observability easy without requiring developers to become observability experts.**

## 2. Core Philosophy

### 2.1 Automatic by default
Developers should not manually configure telemetry that can be safely inferred.

### 2.2 Explicit when necessary
Some information cannot be inferred automatically. Business meaning is one example.

Example:

```python
observability.event(
    "payment.completed",
    domain="payments",
    outcome="success"
)
```

### 2.3 Business context + technical context
Technical telemetry answers: **What happened technically?**

Business context answers: **What did this mean to the business?**

Both should be available without unnecessarily collecting business data.

### 2.4 Lightweight
Prefer:
- Small SDK
- Small CLI
- OpenTelemetry
- Existing backends
- Minimal configuration

Avoid:
- Large runtime
- Proprietary telemetry protocol
- Custom telemetry backend
- Unnecessary dependencies
- Complex abstractions

### 2.5 KISS
> Keep It Simple, Stupid.

When two solutions are functionally equivalent, choose the simpler one. Complexity must be justified.

## 3. High-Level Architecture

```text
                         ┌───────────────────────┐
                         │      Developer        │
                         └───────────┬───────────┘
                                     │
                                     ▼
                         ┌───────────────────────┐
                         │     OTelStarter CLI   │
                         └───────────┬───────────┘
                                     │
                                     ▼
                         ┌───────────────────────┐
                         │   Project Analyzer    │
                         │ Language              │
                         │ Framework             │
                         │ Dependencies          │
                         │ Environment           │
                         └───────────┬───────────┘
                                     │
                                     ▼
                         ┌───────────────────────┐
                         │ Recommendation Engine │
                         │ Instrumentation       │
                         │ Configuration         │
                         │ Safety                │
                         │ Sampling              │
                         └───────────┬───────────┘
                                     │
                                     ▼
              ┌─────────────────────────────────────────┐
              │          Application Observability      │
              │  Lightweight OTel instrumentation       │
              │                                         │
              │  ┌────────┐ ┌─────────┐ ┌──────────┐   │
              │  │ Logs   │ │ Metrics │ │ Traces   │   │
              │  └────┬───┘ └────┬────┘ └────┬─────┘   │
              │       │           │           │         │
              │       └───────────┼───────────┘         │
              │                   │                     │
              │            Domain Events                │
              │                   │                     │
              │          Business Context              │
              │          Technical Context              │
              └───────────────────┬─────────────────────┘
                                  │
                                  ▼
                         ┌───────────────────────┐
                         │ OpenTelemetry         │
                         │ Collector             │
                         └───────────┬───────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
                    ▼                ▼                ▼
                  Logs            Metrics          Traces
                    │                │                │
                    └────────────────┼────────────────┘
                                     │
                                     ▼
                         Observability Backend
```

## 4. Architectural Layers

### Layer 1 — CLI

Commands:
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

The CLI should primarily orchestrate other components. Core business logic should not live in CLI handlers.

### Layer 2 — Project Analyzer

Identify:
- Language
- Framework
- Runtime
- Package manager
- Databases
- Caches
- Message brokers
- HTTP clients
- ORMs
- Deployment environment
- Existing OpenTelemetry configuration

### Layer 3 — Recommendation Engine

Determine:
- Applicable instrumentation
- Safe configuration
- Sampling defaults
- Sensitive-data risks
- Useful telemetry

The engine should favor **minimum useful telemetry**, not maximum telemetry.

### Layer 4 — Application Observability

Provides:
- Logs
- Metrics
- Traces
- Domain events
- Business context
- Technical context
- Correlation

### Layer 5 — OpenTelemetry Collector

Handles:
- Batching
- Processing
- Filtering
- Sampling
- Export

### Layer 6 — Backend

OTelStarter remains backend agnostic.

## 5. Language Adapter Architecture

```text
core/
├── detector/
├── configuration/
├── recommendation/
├── validation/
└── safety/

adapters/
├── python/
├── node/
└── go/
```

Each adapter should expose a small common interface:

```text
detect()
dependencies()
instrumentation()
configuration()
validate()
```

Adding a language should not require changes throughout the core.

## 6. Initial Language Support

### Python
- FastAPI
- Flask
- Django
- PostgreSQL
- SQLAlchemy
- Redis
- Requests/httpx
- Celery

### Node.js
- Express
- NestJS
- Fastify
- PostgreSQL
- Redis
- Axios
- Kafka

### Go
- net/http
- Gin
- Fiber
- PostgreSQL
- Redis

## 7. Telemetry Architecture

OTelStarter uses OpenTelemetry as the telemetry foundation.

Primary signals:
- Logs
- Metrics
- Traces

All signals should share common resource attributes where applicable:
- `service.name`
- `service.version`
- `deployment.environment`

## 8. Logs

Logs should be structured.

Example:

```json
{
  "severity": "ERROR",
  "message": "Payment failed",
  "service.name": "payment-service",
  "trace_id": "abc123",
  "span_id": "def456"
}
```

Technical context should be attached automatically where available.

## 9. Metrics

Initial metrics:
- Request count
- Request duration
- Error count
- Active requests
- CPU
- Memory
- Runtime metrics

HTTP services should support RED-style metrics:
- Rate
- Errors
- Duration

Avoid uncontrolled high-cardinality labels such as:
- `user_id`
- `request_id`
- `transaction_id`

## 10. Traces

Initial instrumentation:
- Incoming HTTP requests
- Outgoing HTTP requests
- Database calls
- Messaging where supported

Tracing should support context propagation.

## 11. Domain Events

Domain events provide business meaning.

Example:

```python
observability.event(
    "payment.completed",
    domain="payments",
    outcome="success"
)
```

The domain-event API must remain small. Technical context is attached automatically.

## 12. Domain Event Model

Potential fields:

```text
event.name
event.domain
event.outcome
business.operation
business.entity.type
business.entity.id
service.name
environment
trace_id
span_id
request_id
timestamp
```

Fields are optional except the event name.

## 13. Business Context

Business context describes what the application was trying to accomplish.

Examples:
- payment
- loan_application
- order
- transfer
- authentication
- subscription
- disbursement

Prefer business meaning over payloads.

## 14. Technical Context

Technical context includes:
- service
- endpoint
- HTTP status
- database
- queue
- latency
- runtime
- deployment
- trace
- span
- error
- version

## 15. Business + Technical Correlation

Example:

```text
Business Event:
payment.authorization.failed

Technical Context:
service = payment-service
endpoint = POST /payments
HTTP status = 504
trace_id = abc123
```

This enables:

```text
Metric
  ↓
Problematic endpoint
  ↓
Trace
  ↓
Database/external service
  ↓
Logs
  ↓
Domain event
  ↓
Business impact
```

## 16. Business Context Does Not Mean Business Payloads

Do not automatically collect:
- Passwords
- Tokens
- Credit card numbers
- National IDs
- Authentication headers
- Personal information
- Full request bodies

The principle is:

> **Business meaning, not business payloads.**

## 17. Three Instrumentation Levels

### Level 1 — Automatic
HTTP, databases, frameworks, runtimes, HTTP clients.

### Level 2 — Contextual
Developers add business or operational context when automatic detection is insufficient.

### Level 3 — Domain Events
Developers explicitly record meaningful business events.

This gives:

> **Automatic by default, explicit when necessary.**

## 18. Backend Agnosticism

Support backend profiles such as:
```text
local
custom-otlp
grafana
openobserve
```

The application remains OpenTelemetry-native.

## 19. Local Development

Example:

```bash
otelstarter run
```

Potential architecture:

```text
Application
    ↓
OTEL Collector
    ↓
 ┌──┼──────────┐
 ↓  ↓          ↓
Logs Metrics Traces
 └────┼────────┘
      ↓
   Grafana
```

## 20. Kubernetes

Kubernetes support is a later phase.

Possible model:

```text
Services
   ↓
OTEL Collector
   ↓
Logs / Metrics / Traces
```

OTelStarter should generate configuration rather than permanently control the cluster.

## 21. Failure Isolation

Telemetry failure must not cause application failure.

If the Collector is unavailable, the application should continue operating.

Telemetry loss is an observability degradation, not an application failure.

## 22. Lightweight Runtime

The CLI should primarily perform setup. It should not remain in the application's runtime unnecessarily.

Desired architecture:

```text
CLI
 ↓
Configure application
 ↓
Application + OTel
 ↓
Collector
```

## 23. Future Intelligence Layer

The architecture should allow future AI-assisted recommendations.

```text
Application
    ↓
Project analysis
    ↓
Dependencies
    ↓
Operational context
    ↓
Recommendation engine
    ↓
Telemetry recommendation
```

The system should optimize for:
- Diagnostic value
- Low overhead
- Low cost
- Privacy
- Security

## 24. Core Architectural Rule

> **OTelStarter is a thin intelligent layer around OpenTelemetry, not another observability platform.**
