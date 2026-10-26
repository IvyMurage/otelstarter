# ADR 0009: ES modules through the same loader, only when the app needs it

**Status:** Accepted

## Context

`otelstarter/register` is loaded with `--require`, which only sees `require()`. In an ES-module app (`"type": "module"`, `.mjs`), modules loaded with `import` were not patched. Measured on an Express + pg app with SDK 0.1.1:

- **What worked:** pg spans appeared, because pg loads its internals with `require`.
- **What was missing:** Express produced no spans, and requests were named only `GET`, with no route.

OpenTelemetry instruments `import` through `import-in-the-middle`. Its loader hook is registered with Node's `module.register()`.

Two findings shaped the design:

- **Node runs `--require` files a second time, in its internal module-loader thread,** once a hook is registered. Without a guard, a second SDK would start there and send duplicate runtime metrics. `worker_threads.isInternalThread`, available from Node.js 22.14 and 23.7, identifies that thread.
- **The hook is not free.** On a small Express + pg app it added about 220 ms of startup and 24 MB of memory, even when limited to the instrumented modules. CommonJS apps would pay this for nothing.

## Decision

- **Same flag for both kinds of app.** `otelstarter/register` registers the `import-in-the-middle` hook itself. Users keep `--require otelstarter/register`; `run` and `init` do not change.
- **The hook only intercepts the modules the loaded instrumentations patch.** These are read from each instrumentation's module definitions, such as `express` and `pg`.
- **The hook is only switched on when the script Node runs is an ES module.** That means `.mjs`, or `.js` under a `package.json` with `"type": "module"`, following Node's own rule. Under `npm start`, npm's own process stays without it. `OTELSTARTER_ESM=true|false` overrides the check.
- **Nothing runs in the internal loader thread.** `register` does nothing there.
- **On Node.js older than 22.14 the hook stays off,** because the loader thread cannot be recognised there. `validate` warns for ES-module apps on those versions.

## Alternatives

- **A second entry point for `--import`:** standard in OpenTelemetry's docs, but `run`, `init` and users would have to choose the right flag per app. Node also runs `--import` files in the loader thread, so the guard is needed either way.
- **Always register the hook:** simpler, but every CommonJS app, including every NestJS app compiled to CommonJS, would start about 200 ms slower.
- **Hook every module, without the include list:** catches libraries we do not know about, but wraps all of `node_modules`, which costs more and can break modules that `import-in-the-middle` cannot parse.

## Consequences

- An ES-module Express + pg app gets Express route spans, pg queries with SQL values removed, and `GET /orders/:email` as the request name. SIGTERM still exits.
- CommonJS apps measured the same startup time and memory as before.
- An ES-module app whose entry is not `.js`/`.mjs`, such as TypeScript run through a custom loader, needs `OTELSTARTER_ESM=true`.
- `import-in-the-middle` is now a direct dependency of the SDK, with the same major version as `@opentelemetry/instrumentation` uses, so npm installs one copy.
