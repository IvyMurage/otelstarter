/**
 * otelstarter for Node.js.
 *
 *   const { logger, event } = require('otelstarter');
 *   logger.info('order placed', { orderId: 'ord_1' });
 *   event('order.created', { domain: 'orders', outcome: 'success' });
 *
 * Start the app with NODE_OPTIONS="--require otelstarter/register" (otelstarter
 * run does this) to collect traces, metrics and these logs.
 */
export { event, type EventFields } from './events';
export { logger, type Logger } from './logger';
export { isSensitiveKey, redactUrl, REDACTED, SENSITIVE_KEYS } from './redaction';
export { querySummary, sanitizeQuery } from './sanitize';
