import type { Context } from '@opentelemetry/api';
import type { ReadableSpan, Span, SpanProcessor } from '@opentelemetry/sdk-trace-base';
import { isSensitiveKey, redactUrl, REDACTED } from './redaction';
import { sanitizeQuery } from './sanitize';

const QUERY_ATTRIBUTES = new Set(['db.query.text', 'db.statement']);
const URL_ATTRIBUTES = new Set(['url.full', 'url.query', 'http.url', 'http.target']);

/**
 * Cleans spans before the exporter sees them: SQL values become ?, sensitive
 * attributes become [REDACTED], and sensitive URL query parameters are hidden.
 *
 * It wraps the processor that exports, and cleans each span when it ends:
 * instrumentations may set attributes at any point (pg sets the query after
 * the span starts), and only at the end are they all present.
 *
 *   new PrivacySpanProcessor(new BatchSpanProcessor(exporter))
 */
export class PrivacySpanProcessor implements SpanProcessor {
  constructor(private readonly next: SpanProcessor) {}

  onStart(span: Span, parentContext: Context): void {
    this.next.onStart(span, parentContext);
  }

  onEnd(span: ReadableSpan): void {
    cleanAttributes(span.attributes as Record<string, unknown>);
    this.next.onEnd(span);
  }

  forceFlush(): Promise<void> {
    return this.next.forceFlush();
  }

  shutdown(): Promise<void> {
    return this.next.shutdown();
  }
}

/**
 * Cleans attributes in place. A finished span's attributes are no longer
 * changed through the span API, so the object itself is updated: it is the
 * one the exporter reads.
 */
export function cleanAttributes(attributes: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(attributes)) {
    if (isSensitiveKey(key)) {
      attributes[key] = REDACTED;
    } else if (typeof value === 'string' && QUERY_ATTRIBUTES.has(key)) {
      attributes[key] = sanitizeQuery(value);
    } else if (typeof value === 'string' && URL_ATTRIBUTES.has(key)) {
      attributes[key] = redactUrl(value);
    }
  }
}
