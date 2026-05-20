"""
Tests for ingestion.matcher — variant detection and fuzzy title matching.
"""

from ingestion.matcher import Matcher
from ingestion.title_utils import detect_variant, normalise


class TestDetectVariant:
    def test_sealed_keyword(self):
        assert detect_variant("Super Mario Bros NES Factory Sealed") == "sealed"

    def test_sealed_nib(self):
        assert detect_variant("Zelda II NIB Nintendo") == "sealed"

    def test_cib_acronym(self):
        assert detect_variant("Contra NES CIB Complete In Box") == "CIB"

    def test_cib_with_box(self):
        assert detect_variant("Mega Man 2 NES w/ Box and Manual") == "CIB"

    def test_cib_complete_alone(self):
        assert detect_variant("Castlevania NES Complete") == "CIB"

    def test_loose_cart_only(self):
        assert detect_variant("Tetris Game Boy Cart Only") == "loose"

    def test_loose_cartridge(self):
        assert detect_variant("Donkey Kong Country SNES Cartridge Loose") == "loose"

    def test_sealed_wins_over_cib(self):
        # "Factory Sealed Complete in Box" — sealed is more specific
        assert detect_variant("Final Fantasy VII PS1 Factory Sealed Complete In Box") == "sealed"

    def test_unknown_when_no_signal(self):
        assert detect_variant("Sonic the Hedgehog Genesis") == "unknown"

    def test_case_insensitive(self):
        assert detect_variant("MARIO KART SNES SEALED") == "sealed"
        assert detect_variant("zelda nes LOOSE") == "loose"


class TestNormalise:
    def test_strips_platform_noise(self):
        result = normalise("Super Mario Bros Nintendo NES")
        assert "nintendo" not in result
        assert "nes" not in result

    def test_strips_seller_noise(self):
        result = normalise("Mega Man 2 Authentic Tested Working Fast Ship")
        assert "authentic" not in result
        assert "working" not in result
        assert "ship" not in result

    def test_preserves_game_title_tokens(self):
        result = normalise("Super Mario Bros NES Cartridge Only Loose Tested")
        assert "super" in result
        assert "mario" in result
        assert "bros" in result

    def test_lowercases_output(self):
        result = normalise("CONTRA NES")
        assert result == result.lower()

    def test_collapses_whitespace(self):
        result = normalise("  Zelda   NES  ")
        assert "  " not in result


CATALOG = [
    {"id": 1, "title": "Super Mario Bros"},
    {"id": 2, "title": "Mega Man 2"},
    {"id": 3, "title": "The Legend of Zelda"},
    {"id": 4, "title": "Castlevania"},
    {"id": 5, "title": "Tetris"},
]


class TestMatcher:
    def _matcher(self, threshold=80.0) -> Matcher:
        return Matcher(CATALOG, threshold=threshold)

    def test_matches_exact_title(self):
        result = self._matcher().match("Super Mario Bros NES Cartridge Tested Working")
        assert result.matched
        assert result.game_id == 1

    def test_matches_with_noise(self):
        result = self._matcher().match(
            "Mega Man 2 Nintendo NES Authentic Tested Fast Ship Loose Cart"
        )
        assert result.matched
        assert result.game_id == 2

    def test_matches_with_word_order_variation(self):
        # token_set_ratio should handle this
        result = self._matcher().match("Bros Super Mario NES")
        assert result.matched
        assert result.game_id == 1

    def test_no_match_for_garbage_title(self):
        result = self._matcher().match("Lot of 10 Mixed NES Games Untested")
        assert not result.matched

    def test_no_match_below_threshold(self):
        # Typo ("Mairo") survives normalisation so the score stays below 99.9
        m = Matcher(CATALOG, threshold=99.9)
        result = m.match("Super Mairo Bros NES Loose")
        assert not result.matched

    def test_returns_correct_variant(self):
        result = self._matcher().match("Tetris Game Boy Sealed Factory New")
        assert result.variant == "sealed"

    def test_returns_unknown_variant_when_no_signal(self):
        result = self._matcher().match("Castlevania NES")
        assert result.variant == "unknown"

    def test_match_score_is_between_0_and_100(self):
        result = self._matcher().match("Super Mario Bros NES")
        assert 0 <= result.score <= 100

    def test_match_batch_returns_all_rows(self):
        m = self._matcher()
        rows = [
            {"id": 101, "raw_title": "Super Mario Bros NES Loose"},
            {"id": 102, "raw_title": "Mega Man 2 NES CIB"},
            {"id": 103, "raw_title": "Garbage lot of unknown games"},
        ]
        results = m.match_batch(rows)
        assert len(results) == 3
        listing_ids = [r[0] for r in results]
        assert listing_ids == [101, 102, 103]

    def test_empty_catalog_never_matches(self):
        m = Matcher([], threshold=50.0)
        result = m.match("Super Mario Bros NES")
        assert not result.matched

    def test_empty_title_never_matches(self):
        result = self._matcher().match("")
        assert not result.matched
