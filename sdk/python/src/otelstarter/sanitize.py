"""Query sanitising: database spans keep the shape of a query, never its values.

    SELECT * FROM users WHERE email = 'ann@shop.rw' AND age > 30
    -> SELECT * FROM users WHERE email = ? AND age > ?

Rules shared with the Node.js SDK; see docs/TELEMETRY_CONVENTIONS.md.
"""

from __future__ import annotations

import re

__all__ = ["query_summary", "sanitize_query"]

# PostgreSQL strings the plain rule would cut short: E'...' allows backslash
# escapes (E'it\\'s'), and $$...$$ or $tag$...$tag$ quote anything.
_ESCAPE_STRING = re.compile(r"\b[Ee]'(?:[^'\\]|\\.|'')*'")
_DOLLAR_STRING = re.compile(r"\$((?:[A-Za-z_]\w*)?)\$[\s\S]*?\$\1\$")
_STRING_LITERAL = re.compile(r"'(?:[^']|'')*'")
# MySQL also accepts "..." for strings. Elsewhere "..." quotes a name, which
# ORMs use everywhere ("users"."email"), so only a "..." compared with
# something is treated as a value.
_QUOTED_VALUE = re.compile(r'(=|<>|!=|<=|>=|<|>|\bLIKE|\bILIKE)(\s*)"(?:[^"]|"")*"', re.IGNORECASE)
_HEX_LITERAL = re.compile(r"\b0x[0-9a-f]+\b", re.IGNORECASE)
_NUMBER_LITERAL = re.compile(r"(?<![\w$])-?\d+(?:\.\d+)?(?![\w$])")
_VALUE_LIST = re.compile(r"\bIN\s*\(\s*\?(?:\s*,\s*\?)*\s*\)", re.IGNORECASE)


def sanitize_query(sql: str) -> str:
    """Replace string and number literals with ?, and collapse IN (?, ?, ...) to IN (?)."""
    sql = _ESCAPE_STRING.sub("?", sql)
    sql = _DOLLAR_STRING.sub("?", sql)
    sql = _STRING_LITERAL.sub("?", sql)
    sql = _QUOTED_VALUE.sub(r"\1\2?", sql)
    sql = _HEX_LITERAL.sub("?", sql)
    sql = _NUMBER_LITERAL.sub("?", sql)
    return _VALUE_LIST.sub("IN (?)", sql)


_OPERATION = re.compile(
    r"^\s*(SELECT|INSERT|UPDATE|DELETE|MERGE|UPSERT|CREATE|ALTER|DROP|TRUNCATE|CALL|EXEC|EXECUTE)\b", re.IGNORECASE
)
_TABLE_AFTER = {
    "SELECT": re.compile(r"\bFROM\s+([\w.\"`\[\]]+)", re.IGNORECASE),
    "DELETE": re.compile(r"\bFROM\s+([\w.\"`\[\]]+)", re.IGNORECASE),
    "INSERT": re.compile(r"\bINTO\s+([\w.\"`\[\]]+)", re.IGNORECASE),
    "UPDATE": re.compile(r"\bUPDATE\s+([\w.\"`\[\]]+)", re.IGNORECASE),
}


_QUOTES = re.compile(r"[\"`\[\]]")


def query_summary(sql: str) -> str | None:
    """A short, low-cardinality name such as "SELECT users", or None when not recognised."""
    match = _OPERATION.match(sql)
    if not match:
        return None
    operation = match.group(1).upper()
    pattern = _TABLE_AFTER.get(operation)
    table = pattern.search(sql) if pattern else None
    if not table:
        return operation
    name = _QUOTES.sub("", table.group(1))
    return f"{operation} {name}"
