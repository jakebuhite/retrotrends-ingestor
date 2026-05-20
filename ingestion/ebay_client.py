"""
ebay_client.py

Thin wrapper around the eBay Browse API.
https://developer.ebay.com/api-docs/buy/browse/resources/item_summary/methods/search
"""

from __future__ import annotations

import base64
import logging
import time
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import Generator, Optional

import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

from .models import EbayItem

logger = logging.getLogger(__name__)


class RateLimitError(Exception):
    """Raised when eBay returns 429 and all retry attempts are exhausted."""


EBAY_AUTH_URL   = "https://api.ebay.com/identity/v1/oauth2/token"
EBAY_BROWSE_URL = "https://api.ebay.com/buy/browse/v1"
EBAY_SCOPE      = "https://api.ebay.com/oauth/api_scope"

PAGE_SIZE     = 200   # eBay Browse API maximum
MAX_RETRIES   = 5
BACKOFF_BASE  = 2     # seconds; doubles each retry


@dataclass
class _Token:
    access_token: str
    expires_at: datetime


class EbayClient:
    """
    Usage::

        client = EbayClient(
            client_id="YOUR_APP_ID",
            client_secret="YOUR_CERT_ID",
        )
        for item in client.search_category("131761", max_pages=10):
            print(item.raw_title)
    """

    def __init__(self, client_id: str, client_secret: str):
        self._client_id     = client_id
        self._client_secret = client_secret
        self._token: Optional[_Token] = None
        self._session = self._build_session()

    def search_category(
        self,
        category_id: str,
        max_pages: int = 999,
        start_page: int = 0,
    ) -> Generator[EbayItem, None, None]:
        """
        Yield every active listing in the given eBay category.

        Args:
            category_id: eBay Browse API category_ids value (e.g. "131761")
            max_pages:   Safety cap on pages consumed per run (default: no cap)
            start_page:  Zero-based page index to resume a partial sweep.

        Yields:
            EbayItem for each listing returned by the API.
        """
        offset = start_page * PAGE_SIZE
        page   = start_page

        while page < max_pages:
            params = {
                "category_ids": category_id,
                "limit":        PAGE_SIZE,
                "offset":       offset,
                "sort":         "newlyListed",
                # Only including US listings for now; remove for global coverage
                "filter":       "itemLocationCountry:US",
            }
            data = self._get(f"{EBAY_BROWSE_URL}/item_summary/search", params=params)
            if not data:
                break

            items = data.get("itemSummaries", [])
            if not items:
                logger.info("Category %s: no more items at page %d.", category_id, page)
                break

            for raw in items:
                yield self._parse_item(raw)

            total = data.get("total", 0)
            offset += PAGE_SIZE
            page   += 1
            logger.info(
                "Category %s: fetched page %d/%d (%d items so far of %d total).",
                category_id, page, -(-total // PAGE_SIZE), offset, total,
            )

            if offset >= total:
                break

    def get_items_by_id(self, ebay_ids: list[str]) -> list[EbayItem]:
        """
        Fetch up to 20 listings by their eBay item IDs.
        Used by the status checker to bulk-verify sold/active state.

        Note: eBay enforces a max of 20 IDs per call — callers should chunk
        their lists before passing them here.
        """
        if not ebay_ids:
            return []

        params = {"item_ids": ",".join(ebay_ids[:20])}
        data   = self._get(f"{EBAY_BROWSE_URL}/item", params=params)
        if not data:
            return []

        return [self._parse_item(raw) for raw in data.get("items", [])]


    def _get(self, url: str, params: dict | None = None) -> dict | None:
        """GET with auth header, retry logic, and structured logging."""
        headers = {"Authorization": f"Bearer {self._access_token()}"}

        for attempt in range(1, MAX_RETRIES + 1):
            try:
                resp = self._session.get(url, headers=headers, params=params, timeout=20)
            except requests.RequestException as exc:
                logger.warning("Request error (attempt %d/%d): %s", attempt, MAX_RETRIES, exc)
                self._backoff(attempt)
                continue

            if resp.status_code == 200:
                return resp.json()

            if resp.status_code == 429:
                if attempt == MAX_RETRIES:
                    raise RateLimitError(
                        f"eBay rate limit persisted after {MAX_RETRIES} retries for {url}"
                    )
                retry_after = int(resp.headers.get("Retry-After", BACKOFF_BASE ** attempt))
                logger.warning("Rate limited. Sleeping %ds (attempt %d).", retry_after, attempt)
                time.sleep(retry_after)
                continue

            if resp.status_code in (500, 502, 503, 504):
                logger.warning("Server error %d (attempt %d).", resp.status_code, attempt)
                self._backoff(attempt)
                continue

            # 4xx that we shouldn't retry
            logger.error("Unrecoverable HTTP %d: %s", resp.status_code, resp.text[:200])
            return None

        logger.error("All %d attempts failed for %s", MAX_RETRIES, url)
        return None

    def _access_token(self) -> str:
        """Return a valid OAuth token, fetching a new one if needed."""
        now = datetime.now(tz=timezone.utc)
        if self._token and self._token.expires_at > now + timedelta(seconds=60):
            return self._token.access_token

        credentials = base64.b64encode(
            f"{self._client_id}:{self._client_secret}".encode()
        ).decode()

        resp = self._session.post(
            EBAY_AUTH_URL,
            headers={
                "Authorization":  f"Basic {credentials}",
                "Content-Type":   "application/x-www-form-urlencoded",
            },
            data={
                "grant_type": "client_credentials",
                "scope":      EBAY_SCOPE,
            },
            timeout=15,
        )
        resp.raise_for_status()
        payload = resp.json()

        self._token = _Token(
            access_token=payload["access_token"],
            expires_at=now + timedelta(seconds=payload["expires_in"]),
        )
        logger.info("eBay OAuth token refreshed (expires in %ds).", payload["expires_in"])
        return self._token.access_token

    @staticmethod
    def _parse_item(raw: dict) -> EbayItem:
        """Map raw eBay Browse API JSON to an EbayItem."""
        price_info  = raw.get("price", {})
        buy_box     = raw.get("currentBidPrice", price_info)
        shipping    = (raw.get("shippingOptions") or [{}])[0]

        try:
            listed_at = datetime.fromisoformat(
                raw.get("itemCreationDate", "").replace("Z", "+00:00")
            )
        except (ValueError, AttributeError):
            listed_at = None

        return EbayItem(
            ebay_listing_id             = raw.get("itemId", ""),
            raw_title                   = raw.get("title", ""),
            condition                   = raw.get("condition"),
            listing_type                = raw.get("buyingOptions", ["FIXED_PRICE"])[0],
            listed_price                = float(buy_box.get("value", 0)) or None,
            currency                    = buy_box.get("currency", "USD"),
            status                      = raw.get("itemAffiliateWebUrl", ""),
            listed_at                   = listed_at,
            seller_feedback_score       = raw.get("seller", {}).get("feedbackScore"),
            seller_positive_feedback_pct= raw.get("seller", {}).get("feedbackPercentage"),
            shipping_cost               = float(
                shipping.get("shippingCost", {}).get("value", 0)
            ) or None,
            item_location               = raw.get("itemLocation", {}).get("country"),
            image_url                   = raw.get("image", {}).get("imageUrl"),
            listing_url                 = raw.get("itemWebUrl"),
            raw_data                    = raw,
        )

    @staticmethod
    def _backoff(attempt: int) -> None:
        delay = BACKOFF_BASE ** attempt
        logger.debug("Backing off %ds.", delay)
        time.sleep(delay)

    @staticmethod
    def _build_session() -> requests.Session:
        session = requests.Session()
        retry = Retry(total=0, raise_on_status=False)
        adapter = HTTPAdapter(max_retries=retry)
        session.mount("https://", adapter)
        return session
