/**
 * Query sanitising: database spans keep the shape of a query, never its values.
 *
 *   SELECT * FROM users WHERE email = 'ann@shop.rw' AND age > 30
 *   becomes
 *   SELECT * FROM users WHERE email = ? AND age > ?
 *
 * Values can be customer data (emails, IDs, card numbers), and leaving them in
 * also turns every query into a unique string, which backends cannot group.
 */

const STRING_LITERAL = /'(?:[^']|'')*'/g;
const NUMBER_LITERAL = /(?<![\w$])-?\d+(?:\.\d+)?(?![\w$])/g;
const VALUE_LIST = /\bIN\s*\(\s*\?(?:\s*,\s*\?)*\s*\)/gi;

/** Replace string and number literals with ?, and collapse IN (?, ?, …) to IN (?). */
export function sanitizeQuery(sql: string): string {
  return sql
    .replace(STRING_LITERAL, '?')
    .replace(NUMBER_LITERAL, '?')
    .replace(VALUE_LIST, 'IN (?)');
}

const OPERATION = /^\s*(SELECT|INSERT|UPDATE|DELETE|MERGE|UPSERT|CREATE|ALTER|DROP|TRUNCATE|CALL|EXEC|EXECUTE)\b/i;
const TABLE_AFTER = {
  SELECT: /\bFROM\s+([\w."`[\]]+)/i,
  DELETE: /\bFROM\s+([\w."`[\]]+)/i,
  INSERT: /\bINTO\s+([\w."`[\]]+)/i,
  UPDATE: /\bUPDATE\s+([\w."`[\]]+)/i,
} as const;

/**
 * A short, low-cardinality span name such as "SELECT users", or undefined when
 * the query is not recognised. Only the first table is used.
 */
export function querySummary(sql: string): string | undefined {
  const op = OPERATION.exec(sql)?.[1]?.toUpperCase();
  if (!op) return undefined;
  const tablePattern = TABLE_AFTER[op as keyof typeof TABLE_AFTER];
  const table = tablePattern?.exec(sql)?.[1]?.replace(/["`[\]]/g, '');
  return table ? `${op} ${table}` : op;
}
