# OTelStarter

OTelStarter is a command-line tool that adds production-quality observability to an existing application: structured logs, metrics, traces, correlation between them, and business-aware domain events.

It is a thin layer around [OpenTelemetry](https://opentelemetry.io/), not an observability backend. Your application emits standard OpenTelemetry data and you choose where it goes.

```bash
otelstarter init                      # add OpenTelemetry to your app
otelstarter run -- uvicorn main:app   # start a local Collector + Grafana, then your app
```

**New here? Follow the [quickstart](docs/quickstart.md):** a Python app with traces, logs, metrics and business events in about 10 minutes.

## Install

Requires Go 1.25 or newer.

```bash
go install github.com/IvyMurage/otelstarter/cmd/otelstarter@latest
```

The program lands in `~/go/bin`. If `otelstarter` is then "command not found", add that folder to your `PATH`:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc   # or ~/.bashrc
```

`otelstarter run` also needs Docker, and Python apps need Python 3.10 or newer.

## Status

Early development. Working commands:

- `otelstarter analyze`: detect a project's language, framework and dependencies, and recommend OpenTelemetry instrumentation ([docs](docs/analyze.md)). Python and Node.js are analysed in full; Go is recognised but not analysed yet. `init`, `run` and `validate` support Python so far.
- `otelstarter init`: add OpenTelemetry to a Python project ([docs](docs/init.md)).
- `otelstarter run`: start a local Collector and Grafana, and optionally your app ([docs](docs/run.md)).
- `otelstarter validate` and `otelstarter doctor`: check the setup and explain how to fix problems ([docs](docs/validate.md)).
- `otelstarter version` and `otelstarter help`.

Domain events for Python apps: `observability.event("payment.completed", domain="payments", outcome="success")`. See the [Python SDK](sdk/python/README.md).

## Build from source

From a clone of this repository:

```bash
go install ./cmd/otelstarter    # or: go build -o otelstarter ./cmd/otelstarter
```

## Documentation

- [Quickstart](docs/quickstart.md): from an app to traces, logs, metrics and events
- [Performance](docs/performance.md): what otelstarter and instrumentation cost
- [Architecture](docs/ARCHITECTURE.md): system design and boundaries
- [Requirements](docs/REQUIREMENTS.md): what the product must do
- [Engineering guidelines](docs/ENGINEERING_GUIDELINES.md): rules for contributors and AI agents
- [Domain events](docs/DOMAIN_EVENTS.md): the business event model
- [Architecture decision records](docs/adr/)
- [Contributing](CONTRIBUTING.md)

## License

[Apache License 2.0](LICENSE)
