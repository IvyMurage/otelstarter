import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import path from 'node:path';
import { test } from 'node:test';

// These tests run real child processes: the logger's JSON output and the
// SIGTERM behaviour can only be checked from outside the process.

const register = path.join(__dirname, '..', 'src', 'register.js');
const index = path.join(__dirname, '..', 'src', 'index.js');
const quiet = {
  ...process.env,
  OTEL_TRACES_EXPORTER: 'none', OTEL_METRICS_EXPORTER: 'none', OTEL_LOGS_EXPORTER: 'none',
  OTEL_SERVICE_NAME: 'shop', OTEL_RESOURCE_ATTRIBUTES: 'deployment.environment.name=test',
};

function run(args: string[], script: string) {
  const child = spawn(process.execPath, [...args, '-e', script], { env: quiet });
  let out = '';
  child.stdout.on('data', (d) => (out += d));
  child.stderr.on('data', (d) => (out += d));
  return { child, output: () => out };
}

async function waitFor(output: () => string, text: string): Promise<void> {
  for (let i = 0; i < 100 && !output().includes(text); i++) await new Promise((r) => setTimeout(r, 50));
  assert.ok(output().includes(text), `child never printed ${text}; got: ${output()}`);
}

test('logger prints structured JSON with context fields and redaction', async () => {
  const { child, output } = run(['--require', register], `
    const { trace } = require('@opentelemetry/api');
    const { logger } = require(${JSON.stringify(index)});
    trace.getTracer('t').startActiveSpan('request', (span) => {
      logger.info('order placed', { orderId: 'ord_1', password: 'hunter2', user: { apiKey: 'k' } });
      span.end();
    });`);
  await once(child, 'exit');

  const line = JSON.parse(output().trim().split('\n').find((l) => l.startsWith('{'))!);
  assert.equal(line.severity, 'INFO');
  assert.equal(line.message, 'order placed');
  assert.equal(line['service.name'], 'shop');
  assert.equal(line.environment, 'test');
  assert.match(line.timestamp, /^\d{4}-\d\d-\d\dT/);
  assert.match(line.trace_id, /^[0-9a-f]{32}$/);
  assert.match(line.span_id, /^[0-9a-f]{16}$/);
  assert.equal(line.orderId, 'ord_1');
  assert.equal(line.password, '[REDACTED]');
  assert.equal(line.user.apiKey, '[REDACTED]');
});

test("pino's argument order keeps the fields", async () => {
  const { child, output } = run([], `
    const { logger } = require(${JSON.stringify(index)});
    logger.info({ orderId: 'ord_1', password: 'hunter2' }, 'order placed');
    logger.error(new Error('card declined'), 'payment failed');
    logger.warn({ retry: 2 });`);
  await once(child, 'exit');

  const lines = output().trim().split('\n').filter((l) => l.startsWith('{')).map((l) => JSON.parse(l));
  assert.equal(lines.length, 3, output());
  assert.equal(lines[0].message, 'order placed');
  assert.equal(lines[0].orderId, 'ord_1');
  assert.equal(lines[0].password, '[REDACTED]');
  assert.equal(lines[1].severity, 'ERROR');
  assert.equal(lines[1].message, 'payment failed');
  assert.equal(lines[1].err.message, 'card declined');
  assert.match(lines[1].err.stack, /Error: card declined/);
  assert.equal(lines[2].retry, 2);
});

test('with otelstarter/register the app still exits on SIGTERM', async () => {
  const { child, output } = run(['--require', register], `
    require('http').createServer((q, r) => r.end()).listen(0, () => console.log('listening'));`);
  await waitFor(output, 'listening');

  child.kill('SIGTERM');
  const [code, signal] = await Promise.race([
    once(child, 'exit'),
    new Promise<never>((_, reject) => setTimeout(() => reject(new Error('still running 5 s after SIGTERM')), 5000)),
  ]);
  assert.equal(code, 143, `exit code ${code}, signal ${signal}`);
});

test("an app's own SIGTERM handler stays in charge", async () => {
  const { child, output } = run(['--require', register], `
    const server = require('http').createServer((q, r) => r.end()).listen(0, () => console.log('listening'));
    process.on('SIGTERM', () => { console.log('app closing'); setTimeout(() => process.exit(7), 200); });`);
  await waitFor(output, 'listening');

  child.kill('SIGTERM');
  const [code] = await once(child, 'exit');
  assert.equal(code, 7, 'the app chose its own exit code');
  assert.ok(output().includes('app closing'));
});
