import { trace } from '@opentelemetry/api';
import { logs, SeverityNumber } from '@opentelemetry/api-logs';
import { logger } from './logger';

/**
 * Domain events: business meaning, not business payloads (DOMAIN_EVENTS.md).
 * Same rules and field names as the Python SDK.
 *
 *   event('order.created', { domain: 'orders', outcome: 'success', entityType: 'order', entityId: 'ord_1' })
 *
 * One OpenTelemetry log-based event per call, linked to the current trace.
 * Never throws: problems are logged once as warnings.
 */
export interface EventFields {
  domain?: string;
  operation?: string;
  outcome?: string;
  entityType?: string;
  entityId?: string | number;
}

const NAME_PATTERN = /^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$/;
const MAX_VALUE_LENGTH = 256;
const ATTRIBUTE_NAMES: Record<keyof EventFields, string> = {
  domain: 'event.domain',
  outcome: 'event.outcome',
  operation: 'business.operation',
  entityType: 'business.entity.type',
  entityId: 'business.entity.id',
};

const warned = new Set<string>();

export function event(name: string, fields: EventFields = {}): void {
  try {
    emit(name, fields);
  } catch (err) {
    warnOnce('emit-failed', `otelstarter could not record event ${JSON.stringify(name)}: ${String(err)}`);
  }
}

function emit(name: unknown, fields: EventFields): void {
  if (typeof name !== 'string' || name === '') {
    warnOnce('name-type', `Event name must be a non-empty string, got ${JSON.stringify(name)}. The event was not recorded.`);
    return;
  }
  if (!NAME_PATTERN.test(name)) {
    warnOnce(`name:${name}`, `Event name ${JSON.stringify(name)} does not follow <domain>.<action> in lowercase, ` +
      "for example 'payment.completed'. The event was still recorded.");
  }

  const attributes: Record<string, string> = {};
  for (const [field, attribute] of Object.entries(ATTRIBUTE_NAMES) as [keyof EventFields, string][]) {
    const value = clean(name, attribute, fields[field]);
    if (value !== undefined) attributes[attribute] = value;
  }

  // Looked up on each call, so events work even when this module was imported
  // before otelstarter/register set up the logger provider.
  logs.getLogger('otelstarter').emit({
    eventName: name,
    body: name, // so log views that show only the body still show the event
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    attributes,
  });

  if (process.env.OTELSTARTER_CONSOLE_EVENTS?.toLowerCase() === 'true') {
    printSummary(name, attributes);
  }
}

function clean(name: string, attribute: string, value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined;
  if (typeof value !== 'string' && typeof value !== 'number') {
    warnOnce(`type:${attribute}`, `${attribute} on event ${JSON.stringify(name)} must be a string or number, got ` +
      `${typeof value}. It was left out: events carry business meaning, not payloads.`);
    return undefined;
  }
  let text = String(value);
  if (text.length > MAX_VALUE_LENGTH) {
    warnOnce(`long:${attribute}`, `${attribute} on event ${JSON.stringify(name)} is ${text.length} characters long, ` +
      `which looks like a payload. It was cut to ${MAX_VALUE_LENGTH}.`);
    text = text.slice(0, MAX_VALUE_LENGTH);
  }
  return text;
}

const SHORT_NAMES: Record<string, string> = {
  'event.domain': 'domain', 'event.outcome': 'outcome', 'business.operation': 'operation',
  'business.entity.type': 'entity_type', 'business.entity.id': 'entity_id',
};

function printSummary(name: string, attributes: Record<string, string>): void {
  const parts = [`[otelstarter] event ${name}`];
  for (const [key, value] of Object.entries(attributes)) parts.push(`${SHORT_NAMES[key]}=${value}`);
  const span = trace.getActiveSpan()?.spanContext();
  if (span && trace.isSpanContextValid(span)) parts.push(`trace_id=${span.traceId}`);
  process.stderr.write(parts.join(' ') + '\n');
}

function warnOnce(key: string, message: string): void {
  if (warned.has(key)) return;
  warned.add(key);
  logger.warn(message, { 'otelstarter.problem': key.split(':')[0] });
}

/** For tests: allow every warning again. */
export function resetWarningsForTests(): void {
  warned.clear();
}
