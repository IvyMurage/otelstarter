/**
 * Redaction: secrets never leave the app in logs or traces.
 *
 *   logger.info('login', { user: 'ann', password: 'hunter2' })
 *   prints  {"message":"login","user":"ann","password":"[REDACTED]", …}
 *
 * The same list protects span attributes, for example a captured
 * "authorization" header or "?token=…" in a URL.
 */

export const REDACTED = '[REDACTED]';

/** Field and header names whose values are always replaced, compared without case. */
export const SENSITIVE_KEYS = [
  'password', 'passwd', 'secret', 'token',
  'access_token', 'accessToken', 'refresh_token', 'refreshToken',
  'api_key', 'apiKey', 'x-api-key', 'x-auth-token',
  'authorization', 'proxy-authorization', 'cookie', 'set-cookie',
  'connection_string', 'connectionString', 'private_key', 'privateKey',
  'credit_card', 'creditCard', 'card_number', 'cardNumber', 'cvv', 'ssn',
];

const sensitive = new Set(SENSITIVE_KEYS.map((k) => k.toLowerCase()));

/** True when a field name, or the last part of a dotted attribute name, is sensitive. */
export function isSensitiveKey(key: string): boolean {
  const lower = key.toLowerCase();
  if (sensitive.has(lower)) return true;
  const last = lower.split('.').pop() ?? lower; // "http.request.header.authorization" -> "authorization"
  return sensitive.has(last);
}

const QUERY_PARAM = /([?&])([^=&#]+)=([^&#]*)/g;

/** Redact sensitive query parameters in a URL: "/x?token=abc&id=1" -> "/x?token=[REDACTED]&id=1". */
export function redactUrl(url: string): string {
  return url.replace(QUERY_PARAM, (match, sep: string, name: string) =>
    isSensitiveKey(decodeURIComponent(name)) ? `${sep}${name}=${REDACTED}` : match,
  );
}
