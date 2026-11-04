"""Redaction: secrets never leave the app in logs or traces.

    log.info("login", extra={"user": "ann", "password": "hunter2"})
    -> {"message": "login", "user": "ann", "password": "[REDACTED]", ...}

The names and rules are shared with the Node.js SDK; see
docs/TELEMETRY_CONVENTIONS.md.
"""

from __future__ import annotations

import re
from urllib.parse import unquote

__all__ = ["REDACTED", "SENSITIVE_KEYS", "SENSITIVE_PAIRS", "SENSITIVE_WORDS", "is_sensitive_key", "redact_url"]

REDACTED = "[REDACTED]"

# A name is sensitive when one of its words is in SENSITIVE_WORDS, or two
# neighbouring words form a pair in SENSITIVE_PAIRS. Words are split at "_",
# "-", "." and camelCase, so "client_secret", "passwordHash" and "x-api-key"
# all count, while "tokens_used" and "passwordless" do not: only whole words
# match.
SENSITIVE_WORDS = (
    "password", "passwd", "passphrase", "secret", "token", "authorization", "cookie",
    "credential", "credentials", "cvv", "ssn", "otp", "jwt", "pin", "apikey", "sessionid", "privatekey",
)  # fmt: skip
SENSITIVE_PAIRS = (
    "api key", "private key", "access key", "session id", "card number", "credit card", "connection string",
)  # fmt: skip

# Deprecated: kept for code that imported it; is_sensitive_key uses the word rules.
SENSITIVE_KEYS = (
    "password", "passwd", "secret", "token",
    "access_token", "accessToken", "refresh_token", "refreshToken",
    "api_key", "apiKey", "x-api-key", "x-auth-token",
    "authorization", "proxy-authorization", "cookie", "set-cookie",
    "connection_string", "connectionString", "private_key", "privateKey",
    "credit_card", "creditCard", "card_number", "cardNumber", "cvv", "ssn",
)  # fmt: skip

_WORDS = frozenset(SENSITIVE_WORDS)
_PAIRS = frozenset(SENSITIVE_PAIRS)
_CAMEL = re.compile(r"([a-z0-9])([A-Z])")
_SEPARATORS = re.compile(r"[^a-z0-9]+")


def _split_words(name: str) -> list[str]:
    """ "passwordHash" -> ["password", "hash"]; "x-api-key" -> ["x", "api", "key"]."""
    return [w for w in _SEPARATORS.split(_CAMEL.sub(r"\1_\2", name).lower()) if w]


def is_sensitive_key(key: str) -> bool:
    """True when a name, or the last part of a dotted attribute name, is sensitive.

    "http.request.header.authorization" counts as "authorization".
    """
    words = _split_words(key.rsplit(".", 1)[-1])
    return any(w in _WORDS or (i > 0 and f"{words[i - 1]} {w}" in _PAIRS) for i, w in enumerate(words))


_QUERY_PARAM = re.compile(r"([?&])([^=&#]+)=([^&#]*)")
_USER_INFO = re.compile(r"^([a-z][a-z0-9+.-]*://)[^/?#@]*@", re.IGNORECASE)


def redact_url(url: str) -> str:
    """Hide credentials in a URL: a user and password before the host, and sensitive query parameters.

    "https://ann:hunter2@db.example.com/x" -> "https://[REDACTED]@db.example.com/x"
    "/x?token=abc&id=1" -> "/x?token=[REDACTED]&id=1"
    """

    def replace(match: re.Match[str]) -> str:
        sep, name = match.group(1), match.group(2)
        return f"{sep}{name}={REDACTED}" if is_sensitive_key(unquote(name)) else match.group(0)

    return _QUERY_PARAM.sub(replace, _USER_INFO.sub(rf"\1{REDACTED}@", url))
