/**
 * Loaded with --require, before the app:
 *
 *   NODE_OPTIONS="--require otelstarter/register" node app.js
 *
 * Starts OpenTelemetry (see sdk.ts) and, unlike OpenTelemetry's own register
 * script, still lets the app exit on SIGTERM. Adding a SIGTERM listener turns
 * off Node's default "exit on SIGTERM"; so after flushing telemetry we exit,
 * unless the app has a SIGTERM handler of its own, which then stays in charge.
 * See https://github.com/open-telemetry/opentelemetry-js-contrib/issues/3808
 */
import { isInternalThread } from 'node:worker_threads';
import { start } from './sdk';

// Node also runs --require files in its internal module-loader thread, which
// exists once the import hook is on (see esm.ts). The app never runs there,
// so a second SDK there would only send duplicate runtime metrics.
if (!isInternalThread) main();

function main(): void {
  const started = start();

  if (started.missing.length > 0 && process.env.OTEL_NODE_ENABLED_INSTRUMENTATIONS) {
    process.stderr.write(
      `[otelstarter] Not instrumented, package not installed: ${started.missing.join(', ')}. ` +
        `Install @opentelemetry/instrumentation-<name> for each, or run otelstarter init again.\n`,
    );
  }

  function onSigterm(): void {
    process.removeListener('SIGTERM', onSigterm);
    const appHandlesSigterm = process.listenerCount('SIGTERM') > 0;
    void started.shutdown().finally(() => {
      if (!appHandlesSigterm) process.exit(128 + 15); // what Node itself would exit with
    });
  }
  process.on('SIGTERM', onSigterm);
  process.once('beforeExit', () => void started.shutdown());
}
