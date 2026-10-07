"""Redaction: secrets never leave the app in logs or traces.

    log.info("login", extra={"user": "ann", "password": "hunter2"})
    -> {"message": "login", "user": "ann", "password": "[REDACTED]", ...}

The names and rules are shared with the Node.js SDK; see
docs/TELEMETRY_CONVENTIONS.md.
"""

from __future__ import annotations

import re
from urllib.parse import unquote

__all__ = ["REDACTED", "SENSITIVE_KEYS", "is_sensitive_key", "redact_url"]

REDACTED = "[REDACTED]"

SENSITIVE_KEYS = (
    "password", "passwd", "secret", "token",
    "access_token", "accessToken", "refresh_token", "refreshToken",
    "api_key", "apiKey", "x-api-key", "x-auth-token",
    "authorization", "proxy-authorization", "cookie", "set-cookie",
    "connection_string", "connectionString", "private_key", "privateKey",
    "credit_card", "creditCard", "card_number", "cardNumber", "cvv", "ssn",
)  # fmt: skip

_SENSITIVE = frozenset(key.lower() for key in SENSITIVE_KEYS)


def is_sensitive_key(key: str) -> bool:
    """True when a name, or the last part of a dotted attribute name, is sensitive.

    "http.request.header.authorization" counts as "authorization".
    """
    lower = key.lower()
    return lower in _SENSITIVE or lower.rsplit(".", 1)[-1] in _SENSITIVE


_QUERY_PARAM = re.compile(r"([?&])([^=&#]+)=([^&#]*)")


def redact_url(url: str) -> str:
    """Hide sensitive query parameters: "/x?token=abc&id=1" -> "/x?token=[REDACTED]&id=1"."""

    def replace(match: re.Match[str]) -> str:
        sep, name = match.group(1), match.group(2)
        return f"{sep}{name}={REDACTED}" if is_sensitive_key(unquote(name)) else match.group(0)

    return _QUERY_PARAM.sub(replace, url)
