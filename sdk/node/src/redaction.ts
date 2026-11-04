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

/**
 * A name is sensitive when one of its words is in SENSITIVE_WORDS, or two
 * neighbouring words form a pair in SENSITIVE_PAIRS. Words are split at
 * "_", "-", "." and camelCase, so "client_secret", "passwordHash" and
 * "x-api-key" all count, while "tokens_used" and "passwordless" do not:
 * only whole words match.
 */
export const SENSITIVE_WORDS = [
  'password', 'passwd', 'passphrase', 'secret', 'token', 'authorization', 'cookie',
  'credential', 'credentials', 'cvv', 'ssn', 'otp', 'jwt', 'pin', 'apikey', 'sessionid', 'privatekey',
];
export const SENSITIVE_PAIRS = [
  'api key', 'private key', 'access key', 'session id', 'card number', 'credit card', 'connection string',
];

/** @deprecated Kept for code that imported it; isSensitiveKey uses the word rules above. */
export const SENSITIVE_KEYS = [
  'password', 'passwd', 'secret', 'token',
  'access_token', 'accessToken', 'refresh_token', 'refreshToken',
  'api_key', 'apiKey', 'x-api-key', 'x-auth-token',
  'authorization', 'proxy-authorization', 'cookie', 'set-cookie',
  'connection_string', 'connectionString', 'private_key', 'privateKey',
  'credit_card', 'creditCard', 'card_number', 'cardNumber', 'cvv', 'ssn',
];

const words = new Set(SENSITIVE_WORDS);
const pairs = new Set(SENSITIVE_PAIRS);

/** "passwordHash" -> ["password", "hash"]; "x-api-key" -> ["x", "api", "key"]. */
function splitWords(name: string): string[] {
  return name
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter(Boolean);
}

/**
 * True when a field name, or the last part of a dotted attribute name
 * ("http.request.header.authorization" -> "authorization"), is sensitive.
 */
export function isSensitiveKey(key: string): boolean {
  const last = key.split('.').pop() ?? key;
  const parts = splitWords(last);
  return parts.some((word, i) => words.has(word) || (i > 0 && pairs.has(`${parts[i - 1]} ${word}`)));
}

const QUERY_PARAM = /([?&])([^=&#]+)=([^&#]*)/g;

const USER_INFO = /^([a-z][a-z0-9+.-]*:\/\/)[^/?#@]*@/i;

/**
 * Redact credentials in a URL: a user and password before the host, and
 * sensitive query parameters.
 *
 *   https://ann:hunter2@db.example.com/x  ->  https://[REDACTED]@db.example.com/x
 *   /x?token=abc&id=1                     ->  /x?token=[REDACTED]&id=1
 */
export function redactUrl(url: string): string {
  return url
    .replace(USER_INFO, `$1${REDACTED}@`)
    .replace(QUERY_PARAM, (match, sep: string, name: string) =>
      isSensitiveKey(safeDecode(name)) ? `${sep}${name}=${REDACTED}` : match,
    );
}

function safeDecode(name: string): string {
  try {
    return decodeURIComponent(name);
  } catch {
    return name; // a malformed %-escape: judge the name as written
  }
}

/**
 * Returns value with sensitive keys redacted in nested objects and arrays,
 * or value itself when nothing needed changing.
 *
 *   { user: { name: 'ann', apiKey: 'k' } }  ->  { user: { name: 'ann', apiKey: '[REDACTED]' } }
 */
export function redactValues(value: unknown, depth = 10): unknown {
  // The depth limit stops self-referencing objects (ORM models, requests)
  // from recursing forever; deeper values are left as they are.
  if (depth <= 0) return value;
  if (Array.isArray(value)) {
    const items = value.map((item) => redactValues(item, depth - 1));
    return items.some((item, i) => item !== value[i]) ? items : value;
  }
  if (value === null || typeof value !== 'object' || value instanceof Uint8Array || value instanceof Error) return value;
  let changed = false;
  const copy: Record<string, unknown> = {};
  for (const [key, item] of Object.entries(value)) {
    copy[key] = isSensitiveKey(key) ? REDACTED : redactValues(item, depth - 1);
    changed ||= copy[key] !== item;
  }
  return changed ? copy : value;
}
