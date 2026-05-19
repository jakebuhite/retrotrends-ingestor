"""
Tests for ingestion.ebay_client — item parsing and OAuth token handling.

All tests use unittest.mock to avoid real HTTP calls.
Run with:  pytest tests/test_ebay_client.py -v
"""

from datetime import datetime, timezone
from unittest.mock import MagicMock, patch

import pytest

from ingestion.ebay_client import EbayClient, EbayItem


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

RAW_ITEM = {
    "itemId":           "v1|123456789|0",
    "title":            "Super Mario Bros NES Cartridge Loose Tested",
    "condition":        "Used",
    "buyingOptions":    ["FIXED_PRICE"],
    "price":            {"value": "14.99", "currency": "USD"},
    "itemCreationDate": "2024-03-01T10:00:00Z",
    "seller": {
        "feedbackScore":      1500,
        "feedbackPercentage": "99.2",
    },
    "shippingOptions": [
        {"shippingCost": {"value": "4.99", "currency": "USD"}}
    ],
    "itemLocation":  {"country": "US"},
    "image":         {"imageUrl": "https://i.ebayimg.com/images/example.jpg"},
    "itemWebUrl":    "https://www.ebay.com/itm/123456789",
}


@pytest.fixture
def client():
    """EbayClient with a fake token already set (skips OAuth call)."""
    c = EbayClient("fake-client-id", "fake-client-secret")
    # Inject a pre-set token so _access_token() doesn't call eBay
    from ingestion.ebay_client import _Token
    c._token = _Token(
        access_token="fake-token",
        expires_at=datetime(2099, 1, 1, tzinfo=timezone.utc),
    )
    return c


# ---------------------------------------------------------------------------
# _parse_item
# ---------------------------------------------------------------------------

class TestParseItem:
    def test_basic_fields(self):
        item = EbayClient._parse_item(RAW_ITEM)
        assert item.ebay_listing_id == "v1|123456789|0"
        assert item.raw_title      == "Super Mario Bros NES Cartridge Loose Tested"
        assert item.condition      == "Used"
        assert item.listing_type   == "FIXED_PRICE"

    def test_price_parsed(self):
        item = EbayClient._parse_item(RAW_ITEM)
        assert item.listed_price == pytest.approx(14.99)
        assert item.currency     == "USD"

    def test_shipping_cost_parsed(self):
        item = EbayClient._parse_item(RAW_ITEM)
        assert item.shipping_cost == pytest.approx(4.99)

    def test_seller_fields_parsed(self):
        item = EbayClient._parse_item(RAW_ITEM)
        assert item.seller_feedback_score        == 1500
        assert item.seller_positive_feedback_pct == pytest.approx(99.2)

    def test_urls_parsed(self):
        item = EbayClient._parse_item(RAW_ITEM)
        assert item.image_url   == "https://i.ebayimg.com/images/example.jpg"
        assert item.listing_url == "https://www.ebay.com/itm/123456789"

    def test_listed_at_is_datetime(self):
        item = EbayClient._parse_item(RAW_ITEM)
        assert isinstance(item.listed_at, datetime)
        assert item.listed_at.year == 2024

    def test_missing_shipping_returns_none(self):
        raw = {**RAW_ITEM, "shippingOptions": []}
        item = EbayClient._parse_item(raw)
        assert item.shipping_cost is None

    def test_missing_creation_date_returns_none(self):
        raw = {**RAW_ITEM, "itemCreationDate": ""}
        item = EbayClient._parse_item(raw)
        assert item.listed_at is None

    def test_raw_data_stored(self):
        item = EbayClient._parse_item(RAW_ITEM)
        assert item.raw_data == RAW_ITEM

    def test_auction_listing_type(self):
        raw = {**RAW_ITEM, "buyingOptions": ["AUCTION"]}
        item = EbayClient._parse_item(raw)
        assert item.listing_type == "AUCTION"


# ---------------------------------------------------------------------------
# search_category — pagination behaviour
# ---------------------------------------------------------------------------

class TestSearchCategory:
    def _mock_response(self, items: list[dict], total: int) -> MagicMock:
        resp = MagicMock()
        resp.status_code = 200
        resp.json.return_value = {"itemSummaries": items, "total": total}
        return resp

    def test_yields_items_from_single_page(self, client):
        with patch.object(client._session, "get") as mock_get:
            mock_get.return_value = self._mock_response([RAW_ITEM], total=1)
            results = list(client.search_category("131761", max_pages=1))

        assert len(results) == 1
        assert results[0].ebay_listing_id == "v1|123456789|0"

    def test_stops_when_no_items_returned(self, client):
        with patch.object(client._session, "get") as mock_get:
            mock_get.return_value = self._mock_response([], total=0)
            results = list(client.search_category("131761"))

        assert results == []

    def test_respects_max_pages(self, client):
        """With max_pages=1 and PAGE_SIZE=200, only 1 API call is made."""
        with patch.object(client._session, "get") as mock_get:
            mock_get.return_value = self._mock_response([RAW_ITEM] * 200, total=10_000)
            results = list(client.search_category("131761", max_pages=1))

        assert mock_get.call_count == 1
        assert len(results) == 200


# ---------------------------------------------------------------------------
# get_items_by_id
# ---------------------------------------------------------------------------

class TestGetItemsById:
    def test_returns_empty_for_empty_input(self, client):
        results = client.get_items_by_id([])
        assert results == []

    def test_truncates_to_20_ids(self, client):
        """eBay allows max 20 IDs per call."""
        ids = [f"id{i}" for i in range(25)]
        with patch.object(client._session, "get") as mock_get:
            mock_get.return_value = MagicMock(
                status_code=200,
                json=lambda: {"items": []},
            )
            client.get_items_by_id(ids)

        call_params = mock_get.call_args[1]["params"]
        sent_ids = call_params["item_ids"].split(",")
        assert len(sent_ids) == 20
