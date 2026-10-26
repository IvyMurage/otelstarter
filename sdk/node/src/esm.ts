import { readFileSync } from 'node:fs';
import * as nodeModule from 'node:module';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { isInternalThread } from 'node:worker_threads';
import type { Instrumentation } from '@opentelemetry/instrumentation';

/**
 * ES-module support. `--require` only sees `require()`, so an app that uses
 * `import` (`"type": "module"`, `.mjs`) would get HTTP spans but no Express
 * routes or database queries. Node's module.register() lets the loader see
 * `import` too: it runs OpenTelemetry's import-in-the-middle hook, which lets
 * the instrumentations patch modules loaded with `import`.
 *
 * The hook only intercepts the modules the loaded instrumentations patch
 * (for example express and pg), so every other import is left alone. It
 * still costs about 200 ms of startup and 25 MB of memory, so it is only
 * switched on for ES-module apps (see isESMEntry), unless OTELSTARTER_ESM
 * says otherwise.
 */

/** The module names the instrumentations patch, such as "express" and "pg". */
export function moduleNames(instrumentations: Instrumentation[]): string[] {
  const names = new Set<string>();
  for (const instrumentation of instrumentations) {
    const definitions = (instrumentation as { getModuleDefinitions?: () => { name: string }[] }).getModuleDefinitions?.() ?? [];
    for (const definition of definitions) names.add(definition.name);
  }
  return [...names];
}

/**
 * Registers the import hook for names when the app is an ES module, or when
 * OTELSTARTER_ESM=true. Returns false, and changes nothing, on Node.js
 * older than 22.14, when OTELSTARTER_ESM=false, or when there is nothing to
 * hook.
 */
export function hookImports(names: string[], env: NodeJS.ProcessEnv = process.env): boolean {
  const register = (nodeModule as { register?: typeof nodeModule.register }).register;
  // The hook runs in an internal thread that also loads --require files.
  // register.ts skips that thread using worker_threads.isInternalThread
  // (Node 22.14+); without it a second SDK would start there, so older
  // Node versions keep CommonJS-only instrumentation.
  if (typeof register !== 'function' || isInternalThread === undefined) return false;
  if (names.length === 0) return false;
  const wanted = env.OTELSTARTER_ESM === 'true' || (env.OTELSTARTER_ESM !== 'false' && isESMEntry(process.argv[1]));
  if (!wanted) return false;
  register('import-in-the-middle/hook.mjs', pathToFileURL(__filename), { data: { include: names } });
  return true;
}

/**
 * Whether Node will run entry, the script it was started with, as an ES
 * module: ".mjs", or ".js" (or no extension) under a package.json with
 * "type": "module". Under npm start this is false for npm's own process and
 * true only for the app's.
 */
export function isESMEntry(entry: string | undefined): boolean {
  if (!entry) return false; // node -e, or the REPL
  const ext = path.extname(entry);
  if (ext === '.mjs') return true;
  if (ext !== '.js' && ext !== '') return false;
  for (let dir = path.dirname(path.resolve(entry)); ; dir = path.dirname(dir)) {
    try {
      const pkg = JSON.parse(readFileSync(path.join(dir, 'package.json'), 'utf8')) as { type?: unknown };
      return pkg.type === 'module';
    } catch {
      // no package.json here, or unreadable: keep looking upwards
    }
    if (path.dirname(dir) === dir) return false;
  }
}
