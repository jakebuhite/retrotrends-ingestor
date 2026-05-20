"""
main.py

Entry point for the ingestion service.
"""

from __future__ import annotations

import logging
import os
import sys

from dotenv import load_dotenv

from .db import get_connection, release_connection
from .ebay_client import EbayClient
from .ingestion import run_ingestion
from .status_checker import run_status_check

load_dotenv()

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO").upper(),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S",
)
logger = logging.getLogger(__name__)


def _require_env(key: str) -> str:
    value = os.getenv(key)
    if not value:
        logger.critical("Missing required environment variable: %s", key)
        sys.exit(1)
    return value


def main() -> None:
    command = sys.argv[1] if len(sys.argv) > 1 else "both"

    if command not in ("ingest", "check-status", "both"):
        print("Usage: python main.py [ingest|check-status|both]")
        sys.exit(1)

    ebay_client_id     = _require_env("EBAY_CLIENT_ID")
    ebay_client_secret = _require_env("EBAY_CLIENT_SECRET")
    database_url       = _require_env("DATABASE_URL")

    client = EbayClient(ebay_client_id, ebay_client_secret)
    conn   = get_connection(database_url)

    try:
        if command in ("ingest", "both"):
            logger.info("=== Starting ingestion run ===")
            run_ingestion(conn, client)

        if command in ("check-status", "both"):
            logger.info("=== Starting status check run ===")
            run_status_check(conn, client)

    finally:
        release_connection(conn)

    logger.info("=== Done ===")


if __name__ == "__main__":
    main()
