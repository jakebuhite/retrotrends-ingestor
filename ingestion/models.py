"""
models.py

Shared data classes for the ingestion service.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional


@dataclass
class EbayItem:
    """Normalised representation of a single eBay listing."""
    ebay_listing_id: str
    raw_title: str
    condition: Optional[str]
    listing_type: str               # "AUCTION" | "FIXED_PRICE"
    listed_price: Optional[float]
    currency: str
    status: str                     # "ACTIVE" | "ENDED" | ...
    listed_at: Optional[datetime]
    seller_feedback_score: Optional[int]
    seller_positive_feedback_pct: Optional[float]
    shipping_cost: Optional[float]
    item_location: Optional[str]
    image_url: Optional[str]
    listing_url: Optional[str]
    raw_data: dict = field(repr=False)


@dataclass
class MatchResult:
    game_id: int
    game_title: str
    variant: str
    score: float            # 0–100 fuzzy match score
    matched: bool
