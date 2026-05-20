"""
db.py

All database I/O for the ingestion service.
"""
from __future__ import annotations

import logging
from contextlib import contextmanager
from typing import Generator, Optional

import psycopg2
import psycopg2.extras
import psycopg2.pool
from psycopg2.extensions import connection as PgConnection

from .models import EbayItem

logger = logging.getLogger(__name__)

_pool: Optional[psycopg2.pool.ThreadedConnectionPool] = None


def get_connection(dsn: str, minconn: int = 2, maxconn: int = 10) -> PgConnection:
    """Return a connection from the module-level ThreadedConnectionPool."""
    global _pool
    if _pool is None:
        _pool = psycopg2.pool.ThreadedConnectionPool(minconn, maxconn, dsn)
    conn = _pool.getconn()
    conn.autocommit = False
    return conn


def release_connection(conn: PgConnection) -> None:
    """Return a connection to the pool."""
    if _pool is not None:
        _pool.putconn(conn)


@contextmanager
def transaction(conn: PgConnection) -> Generator[None, None, None]:
    """Context manager: commit on exit, rollback on exception."""
    try:
        yield
        conn.commit()
    except Exception:
        conn.rollback()
        raise


def claim_next_queue_entry(conn: PgConnection) -> Optional[dict]:
    """
    Atomically claim the highest-priority platform due for ingestion.
    Returns a dict with platform info, or None if nothing is due.

    Uses SELECT ... FOR UPDATE SKIP LOCKED so multiple workers don't
    double-claim the same entry.
    """
    sql = """
        UPDATE ingestion_queue
        SET    status    = 'in_progress',
               updated_at = NOW()
        WHERE  id = (
            SELECT id
            FROM   ingestion_queue
            WHERE  status        IN ('pending', 'failed')
              AND  next_fetch_at <= NOW()
            ORDER  BY priority ASC, next_fetch_at ASC
            LIMIT  1
            FOR UPDATE SKIP LOCKED
        )
        RETURNING id, platform_id, ebay_category_id, last_page_fetched
    """
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute(sql)
        conn.commit()
        return cur.fetchone()


def update_queue_progress(conn: PgConnection, queue_id: int, last_page_fetched: int, total_results: int) -> None:
    """Record sweep progress so a partial run can be resumed."""
    sql = """
        UPDATE ingestion_queue
        SET    last_page_fetched = %s,
               total_results    = %s,
               updated_at       = NOW()
        WHERE  id = %s
    """
    with conn.cursor() as cur:
        cur.execute(sql, (last_page_fetched, total_results, queue_id))
    conn.commit()


def complete_queue_entry(conn: PgConnection, queue_id: int, interval_hours: int = 24) -> None:
    """Mark a sweep as complete and schedule the next one."""
    sql = """
        UPDATE ingestion_queue
        SET    status            = 'pending',
               last_fetched_at  = NOW(),
               last_page_fetched = 0,
               next_fetch_at    = NOW() + (%s * INTERVAL '1 hour'),
               error_message    = NULL,
               updated_at       = NOW()
        WHERE  id = %s
    """
    with conn.cursor() as cur:
        cur.execute(sql, (interval_hours, queue_id))
    conn.commit()


def fail_queue_entry(conn: PgConnection, queue_id: int, error: str) -> None:
    """Mark a queue entry as failed with an error message."""
    sql = """
        UPDATE ingestion_queue
        SET    status        = 'failed',
               error_message = %s,
               updated_at    = NOW()
        WHERE  id = %s
    """
    with conn.cursor() as cur:
        cur.execute(sql, (error[:1000], queue_id))
    conn.commit()


def upsert_listings(conn: PgConnection, items: list[EbayItem]) -> int:
    """
    Bulk-upsert a list of EbayItems into the listings table.

    On conflict (same ebay_listing_id): updates price and last_checked_at
    but does NOT overwrite sold_price / sold_at if already set.

    Returns the number of rows inserted or updated.
    """
    if not items:
        return 0

    rows = [
        (
            item.ebay_listing_id,
            item.raw_title,
            item.condition,
            item.listing_type,
            item.listed_price,
            item.currency,
            item.listed_at,
            item.seller_feedback_score,
            item.seller_positive_feedback_pct,
            item.shipping_cost,
            item.item_location,
            item.image_url,
            item.listing_url,
            psycopg2.extras.Json(item.raw_data),
        )
        for item in items
    ]

    sql = """
        INSERT INTO listings (
            ebay_listing_id,
            raw_title,
            condition,
            listing_type,
            listed_price,
            currency,
            listed_at,
            seller_feedback_score,
            seller_positive_feedback_pct,
            shipping_cost,
            item_location,
            image_url,
            listing_url,
            raw_data
        )
        VALUES %s
        ON CONFLICT (ebay_listing_id) DO UPDATE
            SET listed_price   = EXCLUDED.listed_price,
                raw_title      = EXCLUDED.raw_title,
                last_checked_at = NOW(),
                raw_data       = EXCLUDED.raw_data,
                updated_at     = NOW()
            -- Don't touch sold_price / sold_at / game_id once set
            WHERE listings.status = 'active'
    """
    with conn.cursor() as cur:
        psycopg2.extras.execute_values(cur, sql, rows, page_size=200)
        count = cur.rowcount
    conn.commit()

    logger.info("Upserted %d listings.", count)
    return count


def get_listings_due_for_check(conn: PgConnection, limit: int = 1000, stale_hours: int = 12) -> list[dict]:
    """
    Return active listings that haven't been checked recently.
    The status checker will call the eBay API to see if they've sold.
    """
    sql = """
        SELECT id, ebay_listing_id
        FROM   listings
        WHERE  status         = 'active'
          AND  last_checked_at < NOW() - (%s * INTERVAL '1 hour')
        ORDER  BY last_checked_at ASC
        LIMIT  %s
    """
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute(sql, (stale_hours, limit))
        return cur.fetchall()


def mark_listing_sold(conn: PgConnection, ebay_listing_id: str, sold_price: float) -> None:
    sql = """
        UPDATE listings
        SET    status     = 'sold',
               sold_price = %s,
               sold_at    = NOW(),
               last_checked_at = NOW(),
               updated_at = NOW()
        WHERE  ebay_listing_id = %s
    """
    with conn.cursor() as cur:
        cur.execute(sql, (sold_price, ebay_listing_id))
    conn.commit()


def mark_listing_ended(conn: PgConnection, ebay_listing_id: str) -> None:
    """Listing closed without a sale (expired / cancelled)."""
    sql = """
        UPDATE listings
        SET    status         = 'ended',
               last_checked_at = NOW(),
               updated_at    = NOW()
        WHERE  ebay_listing_id = %s
    """
    with conn.cursor() as cur:
        cur.execute(sql, (ebay_listing_id,))
    conn.commit()


def touch_listing(conn: PgConnection, ebay_listing_id: str) -> None:
    """Update last_checked_at for a listing that is still active."""
    sql = """
        UPDATE listings
        SET    last_checked_at = NOW(),
               updated_at     = NOW()
        WHERE  ebay_listing_id = %s
    """
    with conn.cursor() as cur:
        cur.execute(sql, (ebay_listing_id,))
    conn.commit()


def get_games_for_platform(conn: PgConnection, platform_id: int) -> list[dict]:
    """
    Return the canonical game list for a platform.
    Used by the matcher to build its in-memory lookup table.
    """
    sql = """
        SELECT id, title
        FROM   games
        WHERE  platform_id = %s
        ORDER  BY title
    """
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute(sql, (platform_id,))
        return cur.fetchall()


def update_listing_game_id(conn: PgConnection, listing_id: int, game_id: int, variant: Optional[str]) -> None:
    sql = """
        UPDATE listings
        SET    game_id    = %s,
               variant   = %s,
               updated_at = NOW()
        WHERE  id = %s
          AND  game_id IS NULL
    """
    with conn.cursor() as cur:
        cur.execute(sql, (game_id, variant, listing_id))
    conn.commit()


def get_unmatched_listings(conn: PgConnection, platform_id: int, limit: int = 500) -> list[dict]:
    """
    Return unmatched listings for a given platform (joined via platform →
    ingestion_queue → category, then back to listings via raw_data or
    a separate platform_id column you may want to add to listings).

    For now we use a heuristic: listings without a game_id created recently.
    You may want to add a platform_id column to listings for cleaner querying.
    """
    sql = """
        SELECT l.id, l.raw_title
        FROM   listings l
        WHERE  l.game_id IS NULL
          AND  l.status  = 'active'
        ORDER  BY l.created_at ASC
        LIMIT  %s
    """
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute(sql, (limit,))
        return cur.fetchall()
