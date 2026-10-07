import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { test } from 'node:test';
import { trace } from '@opentelemetry/api';
import { InMemorySpanExporter, SimpleSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { NodeTracerProvider } from '@opentelemetry/sdk-trace-node';
import { PrivacySpanProcessor } from '../src/processor';
import { isSensitiveKey, redactUrl } from '../src/redaction';
import { querySummary, sanitizeQuery } from '../src/sanitize';

// The shared cases in sdk/conventions.json: the Python SDK runs the same ones.
const cases = JSON.parse(readFileSync(path.join(__dirname, '..', '..', '..', 'conventions.json'), 'utf8'));

test('sanitizeQuery (shared cases)', () => {
  for (const [sql, want] of cases.sanitize_query) assert.equal(sanitizeQuery(sql), want, sql);
});

test('querySummary (shared cases)', () => {
  for (const [sql, want] of cases.query_summary) assert.equal(querySummary(sql), want ?? undefined, sql);
});

test('isSensitiveKey (shared cases)', () => {
  for (const key of cases.sensitive_keys.yes) assert.equal(isSensitiveKey(key), true, key);
  for (const key of cases.sensitive_keys.no) assert.equal(isSensitiveKey(key), false, key);
});

test('redactUrl (shared cases)', () => {
  for (const [url, want] of cases.redact_url) assert.equal(redactUrl(url), want, url);
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
