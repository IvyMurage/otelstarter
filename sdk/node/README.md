# otelstarter (Node.js)

Lightweight OpenTelemetry for Node.js, from the [otelstarter](https://github.com/IvyMurage/otelstarter) project:

- **Zero-code traces and metrics**, using only the instrumentations your app needs.
- **Structured JSON logs**, linked to their traces.
- **Redaction** of secrets.
- **Query sanitising.**
- **Business events.**

```js
const { logger, event } = require('otelstarter');

logger.info('order placed', { orderId: 'ord_1', password: 'hunter2' });
// {"severity":"INFO","timestamp":"…","service.name":"shop","trace_id":"…","span_id":"…","orderId":"ord_1","password":"[REDACTED]","message":"order placed"}

// pino's order works too:
logger.info({ orderId: 'ord_1' }, 'order placed');
logger.error(err, 'payment failed'); // the error is logged under "err", with its stack

event('order.created', { domain: 'orders', outcome: 'success', entityType: 'order', entityId: 'ord_1' });
```

Start the app with the loader. `otelstarter run` does this for you:

```bash
NODE_OPTIONS="--require otelstarter/register" node app.js
```

## What the loader does

- **Traces, metrics and logs** are sent over OTLP (HTTP/protobuf), configured by the standard `OTEL_*` environment variables, for example in `otel.env` from `otelstarter init`.
- **Only the instrumentations listed in `OTEL_NODE_ENABLED_INSTRUMENTATIONS` are loaded,** plus `http`, `undici` (fetch), `runtime-node` and `pino`. Install each one as `@opentelemetry/instrumentation-<name>`, for example `@opentelemetry/instrumentation-express`. A missing package is reported, never fatal.
- **No cloud metadata probing at startup.**
- **The app still exits on SIGTERM.** OpenTelemetry's own register script keeps it running ([upstream issue](https://github.com/open-telemetry/opentelemetry-js-contrib/issues/3808)). If your app has its own SIGTERM handler, that handler stays in charge.

## Size

The package is 12.5 kB. With its dependencies and two instrumentations, an Express + pg app's `node_modules` was 45 MB in testing, against more than 83 MB with `@opentelemetry/auto-instrumentations-node`.

## Privacy

Secrets are redacted in spans and in every log sent to OpenTelemetry, including logs from the app's own pino, winston or bunyan logger. SQL values are replaced with `?`. The exact rules, shared with the Python SDK, are in [docs/TELEMETRY_CONVENTIONS.md](../../docs/TELEMETRY_CONVENTIONS.md).

## Shutdown

On SIGTERM the SDK sends the telemetry still waiting, spending at most 3 seconds on it, then raises SIGTERM again. The app therefore stops exactly as it would without otelstarter, and a Collector that is down cannot hold it up. If the app has its own SIGTERM handler, that handler stays in charge.

## ES modules

Apps that use `import` (`"type": "module"` in `package.json`, or `.mjs` files) work with the same `--require otelstarter/register`. The SDK then also turns on OpenTelemetry's import hook, limited to the libraries it instruments, such as express and pg.

The hook costs about 200 ms of startup and 25 MB of memory, so it is only switched on when the script Node runs is an ES module. Override the check with `OTELSTARTER_ESM=true` or `OTELSTARTER_ESM=false`.

## Limitations

- **ES-module apps need Node.js 22.14 or newer** to have the libraries they `import` instrumented. On older versions, incoming HTTP requests are still traced.
- **Spans are redacted only after instrumentation has created them.** Capturing request bodies or headers is up to the instrumentation's own settings.

## Development

```bash
cd sdk/node
npm ci
npm test        # builds with tsc, then runs node --test
```
