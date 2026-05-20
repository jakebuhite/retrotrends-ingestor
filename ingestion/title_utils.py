"""
title_utils.py

Title normalisation and variant detection for eBay listing titles.
"""

from __future__ import annotations

import re

# Ordered most-specific → least-specific so that "sealed" wins over "CIB"
# when a listing says "factory sealed complete in box" (rare but it happens).
_VARIANT_PATTERNS: list[tuple[str, re.Pattern]] = [
    (
        "sealed",
        re.compile(
            r"\b(sealed|factory.?sealed|new.?in.?box|nib|shrink.?wrapped?)\b",
            re.IGNORECASE,
        ),
    ),
    (
        "CIB",
        re.compile(
            r"\b(cib|complete.?in.?box|complete|w[/\\].?box|with.?box|"
            r"w[/\\].?manual|with.?manual|box.?and.?manual)\b",
            re.IGNORECASE,
        ),
    ),
    (
        "loose",
        re.compile(
            r"\b(loose|cart(ridge)?.?only|game.?only|no.?box|cartridge)\b",
            re.IGNORECASE,
        ),
    ),
]

_NOISE = re.compile(
    r"\b("
    r"nintendo|nes|snes|sega|genesis|n64|ps1|playstation|atari|"
    r"game\s?boy|gba|gbc|turbografx|"
    r"authentic|original|tested|working|fast|ship|free|shipping|"
    r"lot|bundle|"
    r"cib|loose|sealed|complete|cartridge|cart|only|no.?box|"
    r"rare|htf|vhtf|lqqk|look|"
    r"great|good|nice|excellent|mint|condition|"
    r"vintage|retro|classic|"
    r"video\s?game|game\s?cart|"
    r"[\(\)\[\]!\"\'#\*]"
    r")\b",
    re.IGNORECASE,
)

_WHITESPACE = re.compile(r"\s+")


def detect_variant(title: str) -> str:
    """Return "sealed", "CIB", "loose", or "unknown" based on title keywords."""
    for variant, pattern in _VARIANT_PATTERNS:
        if pattern.search(title):
            return variant
    return "unknown"


def normalise(title: str) -> str:
    """Strip noise tokens and normalise whitespace for fuzzy comparison."""
    cleaned = _NOISE.sub(" ", title)
    return _WHITESPACE.sub(" ", cleaned).strip().lower()
