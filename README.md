# OTelStarter

OTelStarter is a command-line tool that adds production-quality observability to an existing application: structured logs, metrics, traces, correlation between them, and business-aware domain events.

It is a thin layer around [OpenTelemetry](https://opentelemetry.io/), not an observability backend. Your application emits standard OpenTelemetry data and you choose where it goes.

Target experience:

```bash
otelstarter init
otelstarter run
```

## Status

Early development. Working commands:

- `otelstarter analyze`: detect a project's language, framework and dependencies, and recommend OpenTelemetry instrumentation ([docs](docs/analyze.md)). Full detection for Python; Node.js and Go are recognised but not analysed yet.
- `otelstarter version` and `otelstarter help`.

The other commands print "not implemented yet".

## Build from source

Requires Go 1.24 or newer.

```bash
go build -o otelstarter ./cmd/otelstarter
./otelstarter help
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md): system design and boundaries
- [Requirements](docs/REQUIREMENTS.md): what the product must do
- [Engineering guidelines](docs/ENGINEERING_GUIDELINES.md): rules for contributors and AI agents
- [Domain events](docs/DOMAIN_EVENTS.md): the business event model
- [Architecture decision records](docs/adr/)
- [Contributing](CONTRIBUTING.md)

## License

[Apache License 2.0](LICENSE)
