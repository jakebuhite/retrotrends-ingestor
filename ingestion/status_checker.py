"""
status_checker.py

Checks whether active listings have sold or expired by calling the
eBay Browse API's getItems endpoint.

Notes on detecting sold vs ended:
  - A "sold" listing will return an item with buyingOptions containing
    "SOLD" or the item will no longer be found (404 / empty response).
  - The Browse API does NOT return completed/sold items directly.
    Use the absence of the listing (404) combined with a recency check
    as a heuristic, OR use the eBay Trading API's GetItems call which
    does expose sold status. See TODO below.
"""

from __future__ import annotations

import logging
from collections.abc import Iterator
from itertools import islice

from .db import (
    PgConnection,
    get_listings_due_for_check,
    mark_listing_ended,
    mark_listing_sold,
    touch_listing,
)
from .ebay_client import EbayClient

logger = logging.getLogger(__name__)

BATCH_SIZE  = 20    # eBay Browse API max IDs per getItems call
CHECK_HOURS = 12    # check listings not seen in the last 12 hours
BATCH_LIMIT = 1000  # max listings to check per job run


def run_status_check(conn: PgConnection, client: EbayClient) -> None:
    """
    Main entry point for the status check job.

    1. Pull active listings from DB that haven't been checked recently.
    2. Batch them into groups of 20.
    3. Call eBay getItems for each batch.
    4. Update DB based on the response.
    """
    due = get_listings_due_for_check(conn, limit=BATCH_LIMIT, stale_hours=CHECK_HOURS)
    if not due:
        logger.info("No listings due for status check.")
        return

    logger.info("Checking status of %d listings.", len(due))

    # Build a dict of ebay_id → db_row_id for quick lookup
    id_map = {row["ebay_listing_id"]: row["id"] for row in due}
    ebay_ids = list(id_map.keys())

    sold   = 0
    ended  = 0
    active = 0

    for batch in _chunked(ebay_ids, BATCH_SIZE):
        results = client.get_items_by_id(batch)
        returned_ids = {item.ebay_listing_id for item in results}

        # Items the API returned — still exist, check their status
        for item in results:
            item_status = _resolve_status(item.raw_data)

            if item_status == "sold":
                sold_price = _extract_sold_price(item.raw_data)
                mark_listing_sold(conn, item.ebay_listing_id, sold_price)
                logger.debug("Marked SOLD: %s ($%.2f)", item.ebay_listing_id, sold_price)
                sold += 1

            elif item_status == "ended":
                mark_listing_ended(conn, item.ebay_listing_id)
                ended += 1

            else:
                touch_listing(conn, item.ebay_listing_id)
                active += 1

        # Items the API did NOT return → listing no longer exists
        # Treat as ended (no sale detected). If you want sold detection,
        # implement the Trading API path described in the TODO below.
        missing = set(batch) - returned_ids
        for ebay_id in missing:
            mark_listing_ended(conn, ebay_id)
            logger.debug("Listing not found, marking ENDED: %s", ebay_id)
            ended += 1

    logger.info(
        "Status check complete: %d sold, %d ended, %d still active.",
        sold, ended, active,
    )

def _resolve_status(raw_data: dict) -> str:
    """
    Determine listing status from raw eBay item data.

    The Browse API's item endpoint returns a field called "itemEndDate"
    and buyingOptions. A missing item or "SOLD" buyingOption indicates a sale.

    TODO: The Browse API is limited for sold-price detection. For accurate
    sold prices, supplement with the eBay Trading API's GetSellerTransactions
    or the Analytics API. The Browse Feed API (for sellers) also exposes
    sold status in bulk. For now we fall back to "ended" for missing items.
    """
    buying_options = raw_data.get("buyingOptions", [])

    if "SOLD" in buying_options:
        return "sold"

    # If eBay returned the item and it's not marked SOLD, it's still active
    # (FIXED_PRICE items don't always update immediately after sale)
    return "active"


def _extract_sold_price(raw_data: dict) -> float:
    """
    Extract the sale price from raw item data.
    Falls back to listed_price if sold price is not directly available.
    """
    # The Browse API does not cleanly expose sold price on the item endpoint.
    # If you switch to the Trading API, use Transaction.TransactionPrice instead.
    price = raw_data.get("price", {})
    return float(price.get("value", 0))


def _chunked(iterable, size: int) -> Iterator[list]:
    """Split an iterable into chunks of `size`."""
    it = iter(iterable)
    while chunk := list(islice(it, size)):
        yield chunk
