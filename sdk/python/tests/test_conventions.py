"""The shared cases in sdk/conventions.json: the Node.js SDK runs the same ones."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest

from otelstarter.redaction import is_sensitive_key, redact_url
from otelstarter.sanitize import query_summary, sanitize_query

CASES: dict[str, Any] = json.loads((Path(__file__).parents[2] / "conventions.json").read_text())


@pytest.mark.parametrize(("sql", "want"), CASES["sanitize_query"])
def test_sanitize_query(sql: str, want: str) -> None:
    assert sanitize_query(sql) == want


@pytest.mark.parametrize(("sql", "want"), CASES["query_summary"])
def test_query_summary(sql: str, want: str | None) -> None:
    assert query_summary(sql) == want


@pytest.mark.parametrize("key", CASES["sensitive_keys"]["yes"])
def test_sensitive_keys(key: str) -> None:
    assert is_sensitive_key(key)


@pytest.mark.parametrize("key", CASES["sensitive_keys"]["no"])
def test_not_sensitive_keys(key: str) -> None:
    assert not is_sensitive_key(key)


@pytest.mark.parametrize(("url", "want"), CASES["redact_url"])
def test_redact_url(url: str, want: str) -> None:
    assert redact_url(url) == want
