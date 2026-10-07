import assert from 'node:assert/strict';
import { beforeEach, test } from 'node:test';
import { logs } from '@opentelemetry/api-logs';
import { InMemoryLogRecordExporter, LoggerProvider, SimpleLogRecordProcessor } from '@opentelemetry/sdk-logs';
import { event, resetWarningsForTests } from '../src/events';

// Set up after importing events: the logger must still be found (looked up per call).
const exporter = new InMemoryLogRecordExporter();
logs.setGlobalLoggerProvider(new LoggerProvider({ processors: [new SimpleLogRecordProcessor({ exporter })] }));

beforeEach(() => {
  exporter.reset();
  resetWarningsForTests();
});

test('an event is one named log record with business attributes', () => {
  event('order.created', { domain: 'orders', outcome: 'success', entityType: 'order', entityId: 42 });

  const records = exporter.getFinishedLogRecords();
  assert.equal(records.length, 1);
  assert.equal(records[0].eventName, 'order.created');
  assert.equal(records[0].body, 'order.created');
  assert.deepEqual(records[0].attributes, {
    'event.domain': 'orders',
    'event.outcome': 'success',
    'business.entity.type': 'order',
    'business.entity.id': '42',
  });
});

test('payload values are left out and long values are cut', () => {
  event('order.created', { entityId: { card: '4111' } as unknown as string, operation: 'x'.repeat(1000) });

  const attributes = exporter.getFinishedLogRecords()[0].attributes;
  assert.equal('business.entity.id' in attributes, false);
  assert.equal((attributes['business.operation'] as string).length, 256);
});

test('a badly named event is still recorded', () => {
  event('orderCreated');
  assert.equal(exporter.getFinishedLogRecords()[0].eventName, 'orderCreated');
});

test('an unusable name records nothing and never throws', () => {
  for (const name of ['', undefined, 42]) event(name as unknown as string);
  assert.equal(exporter.getFinishedLogRecords().length, 0);
});
