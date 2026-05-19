"""
matcher.py
----------
Fuzzy-matches raw eBay listing titles against the canonical games catalog,
and extracts the variant (loose / CIB / sealed) from the title text.

Depends on:
    pip install rapidfuzz

Design notes:
  - Build one Matcher per platform per ingestion run (load catalog once).
  - Threshold of 85 is a reasonable starting point; tune by reviewing
    false matches in your listings table.
  - For titles with strong keyword signals (e.g. exact platform + game name),
    you can raise the threshold to 90+ to reduce false positives.
"""

from __future__ import annotations

import logging
import re
from dataclasses import dataclass
from typing import Optional

from rapidfuzz import fuzz, process

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Variant detection
# ---------------------------------------------------------------------------

# Ordered most-specific → least-specific so that "sealed" wins over "CIB"
# when a listing says "factory sealed complete in box" (rare but it happens).
_VARIANT_PATTERNS: list[tuple[str, re.Pattern]] = [
    ("sealed", re.compile(
        r"\b(sealed|factory.?sealed|new.?in.?box|nib|shrink.?wrapped?)\b",
        re.IGNORECASE,
    )),
    ("CIB", re.compile(
        r"\b(cib|complete.?in.?box|complete|w[/\\].?box|with.?box|"
        r"w[/\\].?manual|with.?manual|box.?and.?manual)\b",
        re.IGNORECASE,
    )),
    ("loose", re.compile(
        r"\b(loose|cart(ridge)?.?only|game.?only|no.?box|cartridge)\b",
        re.IGNORECASE,
    )),
]


def detect_variant(title: str) -> str:
    """
    Return "sealed", "CIB", "loose", or "unknown" based on title keywords.
    """
    for variant, pattern in _VARIANT_PATTERNS:
        if pattern.search(title):
            return variant
    return "unknown"


# ---------------------------------------------------------------------------
# Title normalisation
# ---------------------------------------------------------------------------

# Tokens that add noise without helping matching
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


def normalise(title: str) -> str:
    """Strip noise tokens and normalise whitespace for fuzzy comparison."""
    cleaned = _NOISE.sub(" ", title)
    return _WHITESPACE.sub(" ", cleaned).strip().lower()


# ---------------------------------------------------------------------------
# Matcher
# ---------------------------------------------------------------------------

@dataclass
class MatchResult:
    game_id: int
    game_title: str
    variant: str
    score: float            # 0–100 fuzzy match score
    matched: bool


class Matcher:
    """
    Fuzzy title matcher for a single platform's game catalog.

    Args:
        catalog: List of dicts with keys "id" and "title" (from db.get_games_for_platform).
        threshold: Minimum rapidfuzz score (0–100) to accept a match.
    """

    def __init__(self, catalog: list[dict], threshold: float = 85.0):
        self._threshold = threshold

        # Build a lookup dict: normalised_title → (game_id, original_title)
        self._lookup: dict[str, tuple[int, str]] = {}
        for row in catalog:
            key = normalise(row["title"])
            if key:
                self._lookup[key] = (row["id"], row["title"])

        self._choices = list(self._lookup.keys())
        logger.info("Matcher loaded %d games into catalog.", len(self._choices))

    def match(self, raw_title: str) -> MatchResult:
        """
        Attempt to match a raw eBay title to a canonical game.

        Returns a MatchResult with matched=True if confidence ≥ threshold,
        otherwise matched=False (game_id will be -1).
        """
        variant       = detect_variant(raw_title)
        normalised    = normalise(raw_title)

        if not normalised or not self._choices:
            return MatchResult(-1, "", variant, 0.0, False)

        # token_set_ratio handles word-order differences better than
        # simple ratio for game titles (e.g. "Mario Super Land" vs "Super Mario Land")
        result = process.extractOne(
            normalised,
            self._choices,
            scorer=fuzz.token_set_ratio,
            score_cutoff=self._threshold,
        )

        if result is None:
            logger.debug("No match for: %r (normalised: %r)", raw_title, normalised)
            return MatchResult(-1, "", variant, 0.0, False)

        best_key, score, _ = result
        game_id, game_title = self._lookup[best_key]

        logger.debug(
            "Matched %r → %r (score=%.1f, variant=%s)",
            raw_title, game_title, score, variant,
        )
        return MatchResult(game_id, game_title, variant, score, True)

    def match_batch(self, rows: list[dict]) -> list[tuple[int, MatchResult]]:
        """
        Match a batch of listing rows (each with "id" and "raw_title" keys).
        Returns a list of (listing_id, MatchResult) pairs.
        """
        return [(row["id"], self.match(row["raw_title"])) for row in rows]
