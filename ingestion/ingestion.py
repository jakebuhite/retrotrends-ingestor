"""
ingestion.py

Fetches all active eBay listings for a platform,
upserts them into the database, then runs fuzzy matching to link each
listing to a canonical game.
"""

from __future__ import annotations

import logging

from .ebay_client import EbayClient, EbayItem
from .db import (
    PgConnection,
    claim_next_queue_entry,
    complete_queue_entry,
    fail_queue_entry,
    get_games_for_platform,
    get_unmatched_listings,
    update_listing_game_id,
    update_queue_progress,
    upsert_listings,
)
from .matcher import Matcher

logger = logging.getLogger(__name__)

# How many items to accumulate before flushing to DB.
FLUSH_EVERY = 200

# How many hours before re-sweeping the same platform category.
RESWEEP_INTERVAL_HOURS = 24

# Fuzzy match confidence threshold (0–100). Tune this after reviewing
# false positives in your listings table.
MATCH_THRESHOLD = 85.0


def run_ingestion(conn: PgConnection, client: EbayClient) -> bool:
    """
    Claim the next platform due for ingestion, sweep its eBay category,
    and run the fuzzy matcher on newly ingested listings.

    Returns True if a platform was processed, False if nothing was due.
    """
    entry = claim_next_queue_entry(conn)
    if not entry:
        logger.info("No platforms due for ingestion.")
        return False

    queue_id    = entry["id"]
    platform_id = entry["platform_id"]
    category_id = entry["ebay_category_id"]
    start_page  = entry["last_page_fetched"]

    logger.info(
        "Starting ingestion: platform_id=%d category=%s (resuming from page %d).",
        platform_id, category_id, start_page,
    )

    try:
        _sweep_category(conn, client, queue_id, platform_id, category_id, start_page)
        _run_matcher(conn, platform_id)
        complete_queue_entry(conn, queue_id, interval_hours=RESWEEP_INTERVAL_HOURS)
        logger.info("Ingestion complete for platform_id=%d.", platform_id)

    except Exception as exc:
        logger.exception("Ingestion failed for platform_id=%d: %s", platform_id, exc)
        fail_queue_entry(conn, queue_id, str(exc))
        return False

    return True


def _sweep_category(conn: PgConnection, client: EbayClient, queue_id: int, platform_id: int, category_id: str, start_page: int) -> None:
    """
    Page through all eBay listings in a category and upsert them to the DB.
    Checkpoints progress after every flush so a restart can resume.
    """
    buffer: list[EbayItem] = []
    page = start_page
    total_upserted = 0

    for item in client.search_category(category_id, start_page=start_page):
        buffer.append(item)

        if len(buffer) >= FLUSH_EVERY:
            upserted = upsert_listings(conn, buffer)
            total_upserted += upserted
            page += len(buffer) // FLUSH_EVERY
            update_queue_progress(conn, queue_id, page, total_upserted)
            buffer.clear()
            logger.info("Flushed %d listings (total: %d).", upserted, total_upserted)

    # Final partial flush
    if buffer:
        upserted = upsert_listings(conn, buffer)
        total_upserted += upserted
        logger.info("Final flush: %d listings (total: %d).", upserted, total_upserted)

    logger.info("Category %s sweep complete. %d listings upserted.", category_id, total_upserted)


def _run_matcher(conn: PgConnection, platform_id: int) -> None:
    """
    Load the canonical game catalog for a platform, then fuzzy-match
    all unmatched listings to assign game_id and variant.
    """
    catalog = get_games_for_platform(conn, platform_id)
    if not catalog:
        logger.warning(
            "No games in catalog for platform_id=%d. "
            "Seed the games table before running the matcher.",
            platform_id,
        )
        return

    matcher = Matcher(catalog, threshold=MATCH_THRESHOLD)

    unmatched = get_unmatched_listings(conn, platform_id, limit=2000)
    if not unmatched:
        logger.info("No unmatched listings for platform_id=%d.", platform_id)
        return

    logger.info("Matching %d unmatched listings (platform_id=%d).", len(unmatched), platform_id)

    matched_count = 0
    results = matcher.match_batch(unmatched)

    for listing_id, result in results:
        if result.matched:
            update_listing_game_id(conn, listing_id, result.game_id, result.variant)
            matched_count += 1
        else:
            # Leave game_id NULL — a human review queue or re-run with a
            # lower threshold can pick these up later.
            pass

    logger.info(
        "Matching done: %d/%d linked (%.0f%%).",
        matched_count, len(unmatched),
        100 * matched_count / len(unmatched) if unmatched else 0,
    )
