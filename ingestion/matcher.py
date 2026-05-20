"""
matcher.py

Fuzzy-matches raw eBay listing titles against the canonical games catalog,
and extracts the variant (loose / CIB / sealed) from the title text.
"""

from __future__ import annotations

import logging

from rapidfuzz import fuzz, process

from .models import MatchResult
from .title_utils import detect_variant, normalise

logger = logging.getLogger(__name__)


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
