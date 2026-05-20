"""
models.py

Shared data classes for the ingestion service.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime


@dataclass
class EbayItem:
    """Normalised representation of a single eBay listing."""

    ebay_listing_id: str
    raw_title: str
    condition: str | None
    listing_type: str  # "AUCTION" | "FIXED_PRICE"
    listed_price: float | None
    currency: str
    status: str  # "ACTIVE" | "ENDED" | ...
    listed_at: datetime | None
    seller_feedback_score: int | None
    seller_positive_feedback_pct: float | None
    shipping_cost: float | None
    item_location: str | None
    image_url: str | None
    listing_url: str | None
    raw_data: dict = field(repr=False)


@dataclass
class MatchResult:
    game_id: int
    game_title: str
    variant: str
    score: float  # 0–100 fuzzy match score
    matched: bool
