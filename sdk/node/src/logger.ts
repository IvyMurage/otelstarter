import { trace } from '@opentelemetry/api';
import type { Logger as PinoLogger } from 'pino';
import { environmentName, serviceName } from './config';
import { REDACTED, SENSITIVE_KEYS } from './redaction';

/**
 * Structured JSON logs, one line per entry, with the fields every log needs
 * to be useful (REQUIREMENTS FR-005):
 *
 *   logger.info('order placed', { orderId: 'ord_1' })
 *   {"severity":"INFO","timestamp":"2026-10-07T10:30:52.478Z","service.name":"shop",
 *    "environment":"local","trace_id":"…","span_id":"…","orderId":"ord_1","message":"order placed"}
 *
 * pino's own order works too, so pino users need not change habits:
 *
 *   logger.info({ orderId: 'ord_1' }, 'order placed')
 *   logger.error(err, 'payment failed')   // the error goes under "err"
 *
 * Under otelstarter/register each line is also sent to OpenTelemetry, so it
 * appears in Grafana next to its trace.
 */
export interface LogMethod {
  (message: string, fields?: Record<string, unknown>): void;
  (fields: Record<string, unknown> | Error, message?: string): void;
}

export interface Logger {
  debug: LogMethod;
  info: LogMethod;
  warn: LogMethod;
  error: LogMethod;
}

let instance: PinoLogger | undefined;

// pino is loaded on first use, so otelstarter/register has already set up
// OpenTelemetry's pino instrumentation when the logger is created.
function pinoLogger(): PinoLogger {
  if (instance) return instance;
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const pino = require('pino') as typeof import('pino');
  instance = pino({
    level: process.env.OTELSTARTER_LOG_LEVEL ?? 'info',
    base: undefined, // no pid/hostname: the resource attributes carry the host
    messageKey: 'message',
    timestamp: () => `,"timestamp":"${new Date().toISOString()}"`,
    formatters: { level: (label) => ({ severity: label.toUpperCase() }) },
    mixin: contextFields,
    redact: { paths: redactPaths(), censor: REDACTED },
  });
  return instance;
}

/** service, environment and, inside a request, trace and span IDs. */
function contextFields(): Record<string, unknown> {
  const fields: Record<string, unknown> = { 'service.name': serviceName() };
  const environment = environmentName();
  if (environment) fields.environment = environment;
  const span = trace.getActiveSpan()?.spanContext();
  if (span && trace.isSpanContextValid(span)) {
    fields.trace_id = span.traceId;
    fields.span_id = span.spanId;
  }
  return fields;
}

/** Sensitive keys at the top level and one level down, in pino's path syntax. */
function redactPaths(): string[] {
  return SENSITIVE_KEYS.flatMap((key) =>
    /^[A-Za-z_]\w*$/.test(key) ? [key, `*.${key}`] : [`["${key}"]`, `*["${key}"]`],
  );
}

function log(level: keyof Logger): LogMethod {
  return (first: unknown, second?: unknown): void => {
    // otelstarter's order: (message, fields).
    if (typeof first === 'string') {
      pinoLogger()[level](isRecord(second) ? second : {}, first);
      return;
    }
    // pino's order: (fields, message) or (error, message).
    const fields = first instanceof Error ? { err: first } : isRecord(first) ? first : {};
    pinoLogger()[level](fields, typeof second === 'string' ? second : undefined);
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export const logger: Logger = {
  debug: log('debug'),
  info: log('info'),
  warn: log('warn'),
  error: log('error'),
};

/** For tests: forget the cached pino instance. */
export function resetLoggerForTests(): void {
  instance = undefined;
}
