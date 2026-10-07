import assert from 'node:assert/strict';
import { test } from 'node:test';
import { trace } from '@opentelemetry/api';
import { InMemorySpanExporter, SimpleSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { NodeTracerProvider } from '@opentelemetry/sdk-trace-node';
import { PrivacySpanProcessor } from '../src/processor';
import { isSensitiveKey, redactUrl } from '../src/redaction';
import { querySummary, sanitizeQuery } from '../src/sanitize';

test('sanitizeQuery replaces values, keeps the shape', () => {
  const cases: [string, string][] = [
    ["SELECT * FROM users WHERE email = 'ann@shop.rw' AND age > 30", 'SELECT * FROM users WHERE email = ? AND age > ?'],
    ["SELECT * FROM t WHERE name = 'O''Brien'", 'SELECT * FROM t WHERE name = ?'],
    ['SELECT * FROM orders WHERE id IN (1, 2, 3)', 'SELECT * FROM orders WHERE id IN (?)'],
    ['SELECT * FROM t2 WHERE price > -1.5', 'SELECT * FROM t2 WHERE price > ?'], // "t2" is a name, not a value
    ['SELECT * FROM users WHERE id = $1', 'SELECT * FROM users WHERE id = $1'], // placeholders stay
  ];
  for (const [input, want] of cases) assert.equal(sanitizeQuery(input), want);
});

test('querySummary gives a short span name', () => {
  assert.equal(querySummary("SELECT id FROM users WHERE email = 'x'"), 'SELECT users');
  assert.equal(querySummary('insert into "orders" (id) values (1)'), 'INSERT orders');
  assert.equal(querySummary('UPDATE stock SET n = n - 1'), 'UPDATE stock');
  assert.equal(querySummary('BEGIN'), undefined);
});

test('isSensitiveKey matches names, any case, and dotted attribute names', () => {
  for (const key of ['password', 'Authorization', 'accessToken', 'access_token', 'http.request.header.authorization', 'x-api-key']) {
    assert.equal(isSensitiveKey(key), true, key);
  }
  for (const key of ['orderId', 'tokens_used', 'http.method']) assert.equal(isSensitiveKey(key), false, key);
});

test('redactUrl hides sensitive query parameters only', () => {
  assert.equal(redactUrl('/reset?token=abc123&lang=en'), '/reset?token=[REDACTED]&lang=en');
  assert.equal(redactUrl('https://x.rw/a?id=7'), 'https://x.rw/a?id=7');
});

test('PrivacySpanProcessor cleans spans before they are exported', async () => {
  const exporter = new InMemorySpanExporter();
  const provider = new NodeTracerProvider({
    spanProcessors: [new PrivacySpanProcessor(new SimpleSpanProcessor(exporter))],
  });
  const tracer = provider.getTracer('test');

  const span0 = tracer.startSpan('pg.query', {
    attributes: { 'http.request.header.authorization': 'Bearer secret', 'db.system': 'postgresql' },
  });
  // Like the real pg instrumentation: attributes arrive after the span started.
  span0.setAttribute('db.query.text', "SELECT * FROM users WHERE email = 'ann@shop.rw'");
  span0.setAttribute('url.full', 'https://api.rw/x?api_key=k1&page=2');
  span0.end();

  const [span] = exporter.getFinishedSpans();
  assert.deepEqual(span.attributes, {
    'db.query.text': 'SELECT * FROM users WHERE email = ?',
    'http.request.header.authorization': '[REDACTED]',
    'url.full': 'https://api.rw/x?api_key=[REDACTED]&page=2',
    'db.system': 'postgresql',
  });
  await provider.shutdown();
  trace.disable();
});
