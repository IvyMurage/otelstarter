# OTelStarter Engineering & AI Agent Guidelines

**Status:** Mandatory  
**Audience:** AI coding agents, maintainers, contributors, reviewers

## 1. Purpose

This document defines how OTelStarter should be designed, implemented, reviewed, and maintained.

The goal is to ensure:
- A beginner can understand the codebase.
- A senior engineer can extend it confidently.
- Open-source contributors can contribute without understanding the entire system.
- AI agents can modify the code safely.
- The system remains simple as it grows.

## 2. AI Agent Role

When working on this repository, the AI agent MUST behave like a senior software engineer with 10+ years of production engineering experience.

This means:
- Understand before modifying.
- Challenge unnecessary complexity.
- Prefer simple solutions.
- Consider security and reliability.
- Reuse existing code.
- Write tests.
- Protect backwards compatibility.
- Consider maintainability.
- Explain important architectural decisions.

The AI agent must NOT blindly implement instructions that introduce obvious technical debt.

## 3. Before Coding

Before changing code, the AI agent MUST:
1. Read README.md.
2. Read docs/ARCHITECTURE.md.
3. Read docs/REQUIREMENTS.md.
4. Read this document.
5. Inspect relevant source code.
6. Search for existing functionality.
7. Inspect related tests.
8. Understand existing patterns.
9. Identify the smallest appropriate change.

Do not immediately create new abstractions.

## 4. KISS

Always prefer the simplest correct solution.

Ask:

> Can this be simpler?

before introducing a new framework, dependency, abstraction, service, configuration layer, or extension point.

## 5. YAGNI

Do not build functionality merely because it might be useful someday.

Avoid unused configuration, future-proof abstractions, and unused extension points.

Build what is required now.

## 6. DRY — But Carefully

Avoid unnecessary duplication.

However, do not create complicated abstractions merely to remove a few duplicated lines.

Three simple implementations can be preferable to one difficult abstraction.

## 7. Single Responsibility

Functions and classes should have clear responsibilities.

Avoid giant functions that perform detection, installation, configuration, validation, Docker startup, and telemetry.

Prefer focused functions such as:
```text
detect_project()
detect_dependencies()
generate_config()
validate_config()
start_environment()
```

## 8. Naming

Names must communicate intent.

Prefer:
```text
detect_framework()
generate_collector_config()
validate_configuration()
redact_sensitive_fields()
emit_domain_event()
```

Avoid vague names:
```text
process()
handle()
run()
manager()
helper()
data()
thing()
utils()
```

unless genuinely clear in context.

## 9. Functions

Functions should:
- Do one thing.
- Have clear inputs.
- Have clear outputs.
- Avoid unnecessary side effects.
- Be easy to test.

## 10. Classes

Do not create a class when a function or simple data structure is sufficient.

Classes should represent:
- A meaningful domain concept.
- A stateful component.
- A clear abstraction boundary.

Avoid classes used only to group unrelated utilities.

## 11. Reusability

Before implementing something new:
1. Search existing code.
2. Search related modules.
3. Search tests.
4. Identify reusable functions.

Do not duplicate existing behavior.

But do not force unrelated components into one abstraction.

## 12. Abstraction Rule

Create an abstraction when:
- Multiple implementations genuinely share behavior.
- The abstraction improves clarity.
- The abstraction represents a real concept.
- The abstraction is likely to remain stable.

Do not abstract solely because it may be needed later.

## 13. Dependency Management

Every dependency has a cost.

Before adding one, ask:
- Can the standard library solve this?
- Can an existing dependency solve this?
- Is it actively maintained?
- Does it introduce security risk?
- Does it materially reduce complexity?

Prefer fewer dependencies.

## 14. Configuration

Configuration should be explicit and predictable.

Do not hardcode environment-specific values.

But do not make everything configurable.

## 15. Error Handling

Every user-facing error should answer:
1. What happened?
2. Why?
3. How can I fix it?

## 16. Logging

OTelStarter's own logs must be useful and non-sensitive.

Never log:
- Passwords
- Tokens
- Secrets
- API keys
- Private keys
- Full credentials

## 17. Security

Security is part of design.

Consider:
- Authentication
- Authorization
- Secrets
- Input validation
- Dependency vulnerabilities
- Path traversal
- Command injection
- Unsafe file modification
- Sensitive telemetry

CLI functionality that executes commands or modifies files requires explicit security consideration.

## 18. Privacy

Collect the minimum information required.

Prefer:
```text
payment.completed
```

over customer payloads.

Business context means business meaning, not business payloads.

## 19. Telemetry Cardinality

Avoid high-cardinality metric attributes.

Do not automatically use:
- user_id
- request_id
- transaction_id
- email
- phone

as metric labels.

## 20. Domain Event Naming

Use:
```text
<domain>.<action>
```

Examples:
```text
payment.completed
payment.failed
order.created
order.cancelled
loan.submitted
loan.approved
user.authenticated
```

Event names describe what happened, not what code function executed.

## 21. Business Context

Use business metadata sparingly.

Good:
```text
domain = payments
operation = payment
outcome = success
```

Avoid unnecessary payloads.

Goal:

> High diagnostic value with minimal data.

## 22. Technical Context

Automatically attach where possible:
- service.name
- service.version
- environment
- trace_id
- span_id
- HTTP route
- HTTP status
- runtime
- deployment

Developers should not manually add these everywhere.

## 23. OpenTelemetry First

Do not create proprietary telemetry mechanisms when OpenTelemetry provides the capability.

Use OpenTelemetry for:
- Logs
- Metrics
- Traces
- Context propagation
- Resource attributes
- Export

OTelStarter is an opinionated developer experience around OpenTelemetry, not a replacement.

## 24. Backend Agnostic

Do not couple the SDK to:
- Grafana
- Loki
- Tempo
- Prometheus
- OpenObserve

These are backend choices.

## 25. Tests

Every meaningful feature requires tests.

Minimum:
- Unit tests
- Integration tests
- CLI tests
- Configuration tests
- Safety tests
- Language adapter tests

Test behavior rather than internal implementation.

## 26. Test Pyramid

Prefer:
```text
        E2E
       /   \
  Integration
     /     \
    Unit tests
```

Most tests should be fast unit tests.

## 27. Backwards Compatibility

Treat generated configuration as a public interface.

Before changing generated output, consider existing projects, configurations, CLI commands, dashboards, and users.

Breaking changes require explicit documentation.

## 28. Documentation

Every public feature should explain:
- What
- Why
- How
- Example
- Limitations

If something cannot be explained simply, reconsider the design.

## 29. Code Comments

Comments should explain **why**, not what.

Avoid comments that merely repeat the code.

## 30. File and Module Organization

Prefer:
```text
src/
├── cli/
├── core/
├── adapters/
├── telemetry/
├── safety/
└── configuration/
```

Avoid giant files and dumping unrelated functionality into `utils.py`, `helpers.py`, or `common.py`.

## 31. AI Agent Change Discipline

The AI agent should make focused changes.

Do not combine a new feature with unrelated refactoring, formatting, or dependency upgrades unless required.

A pull request should have one clear purpose.

## 32. AI Agent Must Review Its Own Diff

Before completing work, review:
- Changed files
- Added files
- Deleted files
- Dependencies
- Tests
- Documentation
- Security implications
- Telemetry implications

Remove unnecessary changes.

## 33. AI Agent Must Run Verification

Before declaring a task complete:
- Run tests.
- Run lint.
- Run type checking where applicable.
- Validate generated configuration.
- Run relevant integration tests.

If something cannot be run, state why.

Never claim tests passed if they were not run.

## 34. AI Agent Must Challenge Bad Designs

If a requested implementation introduces unnecessary complexity, explain the concern and recommend a simpler approach.

The goal is good engineering, not blind compliance.

## 35. Open Source Standards

A contributor should be able to understand:
- How to run the project.
- How the architecture works.
- Where language adapters live.
- Where tests live.
- How to add instrumentation.
- How to submit a PR.

## 36. Adding a Language

A language adapter should be isolated:

```text
adapters/
└── python/
    ├── detector
    ├── instrumentation
    ├── configuration
    ├── validator
    └── tests
```

A contributor should not need to modify unrelated parts of the system.

Document the process in:
```text
docs/adding-a-language.md
```

## 37. Pull Requests

Every PR should answer:
- What problem does this solve?
- Why is this implementation appropriate?
- What alternatives were considered?
- What tests were added?
- Does it introduce dependencies?
- Does it affect performance?
- Does it affect security?
- Does it affect telemetry volume?
- Does it introduce a breaking change?

## 38. Architecture Decision Records

Significant decisions should use ADRs.

Location:
```text
docs/adr/
```

Each ADR contains:
- Context
- Decision
- Alternatives
- Consequences

## 39. Definition of Done

A feature is not complete until:
- Implementation exists.
- Tests exist.
- Relevant documentation exists.
- Security reviewed.
- Privacy considered.
- Performance considered.
- Error handling exists.
- Existing behavior preserved.
- Code is understandable.
- Diff is focused.

## 40. Golden Rules

1. Understand before changing.
2. Prefer simple solutions.
3. Reuse before creating.
4. Do not abstract prematurely.
5. Keep functions focused.
6. Name things clearly.
7. Treat security as a requirement.
8. Treat privacy as a requirement.
9. Collect useful telemetry, not maximum telemetry.
10. Business context is not business payload.
11. OpenTelemetry is the foundation; do not reinvent it.
12. Do not couple the SDK to a backend.
13. Write tests for behavior.
14. Document public behavior.
15. Keep pull requests focused.
16. Challenge unnecessary complexity.
17. Never claim verification that was not performed.

## 41. Final Engineering Principle

> **Simple enough for a beginner to understand. Structured enough for a senior engineer to extend. Documented enough for an open-source contributor to participate. Explicit enough for an AI agent to modify safely.**

Product principle:

> **Automatically collect technical context, explicitly capture meaningful business context, correlate logs/metrics/traces/domain events, and collect only the telemetry that provides operational value.**
