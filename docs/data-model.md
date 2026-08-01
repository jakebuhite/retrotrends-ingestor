# Data Model

RetroTrends uses a single PostgreSQL database with two core tables: `games` (the catalog) and `listings` (eBay price records).

---

## Tables

### `games`

Populated by the IGDB backfill job. Represents the canonical list of GameCube titles the system tracks.

```sql
CREATE TABLE games (
    id           BIGSERIAL    PRIMARY KEY,
    igdb_id      INTEGER      NOT NULL UNIQUE,
    title        TEXT         NOT NULL,
    slug         TEXT         NOT NULL UNIQUE,  -- URL-safe title, e.g. "super-mario-sunshine"
    platform     TEXT         NOT NULL DEFAULT 'Nintendo GameCube',
    release_year SMALLINT,
    cover_url    TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

**Notes:**
- `igdb_id` is the canonical identifier from IGDB. It is the key used during upserts in the backfill job.
- `slug` is derived from the IGDB slug field and used to construct clean API URLs (e.g. `/v1/games/super-mario-sunshine`).
- `cover_url` stores the IGDB cover image URL for display in the website.

---

### `listings`

Populated by the ingestor. Each row represents a single eBay listing for a tracked game.

```sql
CREATE TABLE listings (
    id               BIGSERIAL     PRIMARY KEY,
    game_id          BIGINT        NOT NULL REFERENCES games(id),
    ebay_listing_id  TEXT          NOT NULL UNIQUE,
    raw_title        TEXT          NOT NULL,
    condition        TEXT          NOT NULL DEFAULT 'unknown'
                                   CHECK (condition IN ('sealed', 'cib', 'loose', 'unknown')),
    listing_type     TEXT          NOT NULL
                                   CHECK (listing_type IN ('auction', 'buy_it_now', 'best_offer')),
    asking_price     NUMERIC(10,2),
    currency         CHAR(3)       NOT NULL DEFAULT 'USD',
    listing_url      TEXT,
    listed_at        TIMESTAMPTZ,
    status           TEXT          NOT NULL DEFAULT 'pending'
                                   CHECK (status IN ('pending', 'sold', 'ended_unsold')),
    sold_price       NUMERIC(10,2),
    sold_at          TIMESTAMPTZ,
    last_checked_at  TIMESTAMPTZ,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
```

**Column notes:**

| Column | Description |
|--------|-------------|
| `ebay_listing_id` | eBay's own item ID (e.g. `"395432918765"`). Unique constraint prevents duplicate inserts. |
| `raw_title` | The full listing title as returned by eBay. Used for condition parsing and debugging. |
| `condition` | Parsed from the eBay condition field and listing title. See [Condition Parsing](architecture.md#condition-parsing). |
| `listing_type` | `auction`, `buy_it_now`, or `best_offer`. Auctions and BIN have different price dynamics. |
| `asking_price` | The listed price at the time of ingestion (starting bid for auctions, BIN price otherwise). |
| `status` | Lifecycle state: `pending` → `sold` or `ended_unsold`. |
| `sold_price` | The final transaction price. Only set when `status = 'sold'`. For auctions this is the winning bid; for BIN this equals `asking_price`. |
| `sold_at` | Timestamp when the item was confirmed sold. Sourced from eBay's `itemEndDate`. |
| `last_checked_at` | Timestamp of the most recent revisit check. Used by the revisit job to prioritize stale records. |

---

## Indexes

```sql
-- Revisit job: find pending listings sorted by stalest-first
CREATE INDEX idx_listings_pending_revisit
    ON listings (last_checked_at ASC NULLS FIRST)
    WHERE status = 'pending';

-- Price history queries: sold listings for a game ordered by time
CREATE INDEX idx_listings_price_history
    ON listings (game_id, sold_at DESC)
    WHERE status = 'sold';

-- Condition-filtered price queries
CREATE INDEX idx_listings_condition_price
    ON listings (game_id, condition, sold_at DESC)
    WHERE status = 'sold';

-- Game search by title
CREATE INDEX idx_games_title_trgm
    ON games USING gin (title gin_trgm_ops);
```

The last index requires the `pg_trgm` extension for trigram-based fuzzy text search:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

---

## Entity Relationships

```
games (1) ─────── (many) listings
  id ◄────────────── game_id
```

Each listing belongs to exactly one game. A game may have zero or many listings.

---

## Lifecycle State Machine

```
              [ingest job]
                   │
                   ▼
              ┌─────────┐
              │ pending │
              └────┬────┘
                   │ [revisit job checks eBay status]
          ┌────────┴──────────┐
          ▼                   ▼
      ┌────────┐       ┌──────────────┐
      │  sold  │       │ ended_unsold │
      └────────┘       └──────────────┘
```

- A listing enters as `pending` when first discovered by the ingest job.
- The revisit job transitions it to `sold` (with `sold_price` and `sold_at` populated) or `ended_unsold` (no sale).
- Transitions are terminal — once `sold` or `ended_unsold`, a listing is never re-checked.

---

## Full Schema (Migration)

```sql
-- migrations/001_initial_schema.sql

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE games (
    id           BIGSERIAL    PRIMARY KEY,
    igdb_id      INTEGER      NOT NULL UNIQUE,
    title        TEXT         NOT NULL,
    slug         TEXT         NOT NULL UNIQUE,
    platform     TEXT         NOT NULL DEFAULT 'Nintendo GameCube',
    release_year SMALLINT,
    cover_url    TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE listings (
    id               BIGSERIAL     PRIMARY KEY,
    game_id          BIGINT        NOT NULL REFERENCES games(id),
    ebay_listing_id  TEXT          NOT NULL UNIQUE,
    raw_title        TEXT          NOT NULL,
    condition        TEXT          NOT NULL DEFAULT 'unknown'
                                   CHECK (condition IN ('sealed', 'cib', 'loose', 'unknown')),
    listing_type     TEXT          NOT NULL
                                   CHECK (listing_type IN ('auction', 'buy_it_now', 'best_offer')),
    asking_price     NUMERIC(10,2),
    currency         CHAR(3)       NOT NULL DEFAULT 'USD',
    listing_url      TEXT,
    listed_at        TIMESTAMPTZ,
    status           TEXT          NOT NULL DEFAULT 'pending'
                                   CHECK (status IN ('pending', 'sold', 'ended_unsold')),
    sold_price       NUMERIC(10,2),
    sold_at          TIMESTAMPTZ,
    last_checked_at  TIMESTAMPTZ,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_listings_pending_revisit
    ON listings (last_checked_at ASC NULLS FIRST)
    WHERE status = 'pending';

CREATE INDEX idx_listings_price_history
    ON listings (game_id, sold_at DESC)
    WHERE status = 'sold';

CREATE INDEX idx_listings_condition_price
    ON listings (game_id, condition, sold_at DESC)
    WHERE status = 'sold';

CREATE INDEX idx_games_title_trgm
    ON games USING gin (title gin_trgm_ops);
```

---

## Design Notes

**Why no separate `price_history` table?**
The `listings` table *is* the price history. Each sold listing is one data point in the time series. Materializing a separate history table would duplicate data and add complexity without benefit at this scale.

**Why store `asking_price` separately from `sold_price`?**
For auctions, `asking_price` is the starting bid — often far below final sale price. Keeping both allows analysis of how much prices move during bidding, and avoids confusion when a BIN listing is cross-referenced.

**Why `NUMERIC(10,2)` for prices?**
`FLOAT` introduces rounding errors unsuitable for currency. `NUMERIC(10,2)` stores up to $99,999,999.99 exactly, which is more than sufficient for any retro game.
