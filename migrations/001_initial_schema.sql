-- =============================================================================
-- RetroTrends Database Schema
-- PostgreSQL 15+
-- =============================================================================

-- ---------------------------------------------------------------------------
-- PLATFORMS
-- Reference table for gaming consoles/platforms.
-- Seed this once; rarely changes.
-- ---------------------------------------------------------------------------
CREATE TABLE platforms (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,               -- "Nintendo Entertainment System"
    short_name  VARCHAR(20)  NOT NULL UNIQUE,         -- "NES"
    ebay_category_id VARCHAR(20),                    -- eBay Browse API category_ids value
    active      BOOLEAN      NOT NULL DEFAULT TRUE,   -- set FALSE to pause ingestion
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- GAMES
-- Canonical game catalog. Seed from IGDB / VGChartz / manual CSV before
-- ingestion runs. Clean titles here are what listings get matched against.
-- ---------------------------------------------------------------------------
CREATE TABLE games (
    id           SERIAL PRIMARY KEY,
    title        VARCHAR(255) NOT NULL,
    platform_id  INT          NOT NULL REFERENCES platforms(id) ON DELETE RESTRICT,
    release_year SMALLINT,
    publisher    VARCHAR(100),
    developer    VARCHAR(100),
    genre        VARCHAR(50),
    upc          VARCHAR(20),
    igdb_id      INT UNIQUE,                          -- for future enrichment
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (title, platform_id)
);

CREATE INDEX idx_games_platform ON games(platform_id);
CREATE INDEX idx_games_title ON games USING gin(to_tsvector('english', title));

-- ---------------------------------------------------------------------------
-- LISTINGS
-- One row per eBay listing. Inserted on first discovery; updated when the
-- status checker detects a sale or expiry. game_id is NULL until the matcher
-- runs and links the listing to a canonical game.
-- ---------------------------------------------------------------------------
CREATE TABLE listings (
    id                          BIGSERIAL PRIMARY KEY,
    ebay_listing_id             VARCHAR(50)   NOT NULL UNIQUE,
    game_id                     INT           REFERENCES games(id) ON DELETE SET NULL,
    raw_title                   VARCHAR(500)  NOT NULL,

    -- Condition / variant parsed from the listing title or eBay condition field
    condition                   VARCHAR(50),   -- "New", "Used", "For parts or not working"
    variant                     VARCHAR(20),   -- "loose", "CIB", "sealed", "unknown"

    listing_type                VARCHAR(20),   -- "AUCTION", "FIXED_PRICE"
    listed_price                NUMERIC(10,2),
    sold_price                  NUMERIC(10,2),
    currency                    CHAR(3)        NOT NULL DEFAULT 'USD',

    -- Lifecycle
    status                      VARCHAR(20)    NOT NULL DEFAULT 'active',
    -- active | sold | ended | cancelled
    listed_at                   TIMESTAMPTZ,
    sold_at                     TIMESTAMPTZ,
    last_checked_at             TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    -- Seller signals (useful for outlier filtering)
    seller_feedback_score       INT,
    seller_positive_feedback_pct NUMERIC(5,2),

    -- Logistics
    shipping_cost               NUMERIC(10,2),
    item_location               VARCHAR(100),

    -- Links
    image_url                   TEXT,
    listing_url                 TEXT,

    -- Full API response — store raw so you can re-parse without re-fetching
    raw_data                    JSONB,

    created_at                  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- Ingestion: fast lookup by eBay ID (already covered by UNIQUE constraint)
-- Status checker: find active listings due for a check
CREATE INDEX idx_listings_status_checked ON listings(status, last_checked_at)
    WHERE status = 'active';

-- Analytics: join listings to games by sold date
CREATE INDEX idx_listings_game_sold ON listings(game_id, sold_at)
    WHERE status = 'sold';

-- Matching job: find unmatched listings
CREATE INDEX idx_listings_unmatched ON listings(created_at)
    WHERE game_id IS NULL;

-- ---------------------------------------------------------------------------
-- INGESTION QUEUE
-- Controls when each platform's eBay category is swept next.
-- One row per platform. The ingestion service reads this to decide what
-- to fetch, then updates next_fetch_at when done.
-- ---------------------------------------------------------------------------
CREATE TABLE ingestion_queue (
    id                  SERIAL PRIMARY KEY,
    platform_id         INT          NOT NULL REFERENCES platforms(id) ON DELETE CASCADE,
    ebay_category_id    VARCHAR(20)  NOT NULL,
    priority            SMALLINT     NOT NULL DEFAULT 5,  -- lower = fetched first
    next_fetch_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_fetched_at     TIMESTAMPTZ,
    last_page_fetched   INT          NOT NULL DEFAULT 0,  -- resume partial sweeps
    total_results       INT,                               -- populated after first page
    status              VARCHAR(20)  NOT NULL DEFAULT 'pending',
    -- pending | in_progress | completed | failed
    error_message       TEXT,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (platform_id)
);

-- Scheduler query: pick the highest-priority platform due for a fetch
CREATE INDEX idx_queue_next_fetch ON ingestion_queue(priority, next_fetch_at)
    WHERE status IN ('pending', 'failed');

-- ---------------------------------------------------------------------------
-- PRICE SUMMARIES
-- Pre-aggregated sold-price stats per game per variant per time period.
-- Recomputed nightly by a simple SQL job. Your API serves from this table
-- rather than aggregating millions of listings rows on the fly.
-- ---------------------------------------------------------------------------
CREATE TABLE price_summaries (
    id                  BIGSERIAL   PRIMARY KEY,
    game_id             INT         NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    variant             VARCHAR(20) NOT NULL,   -- "loose" | "CIB" | "sealed" | "any"
    period_type         VARCHAR(10) NOT NULL,   -- "week" | "month" | "quarter"
    period_start        DATE        NOT NULL,
    period_end          DATE        NOT NULL,
    avg_sold_price      NUMERIC(10,2),
    median_sold_price   NUMERIC(10,2),
    min_sold_price      NUMERIC(10,2),
    max_sold_price      NUMERIC(10,2),
    sample_size         INT         NOT NULL DEFAULT 0,
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (game_id, variant, period_type, period_start)
);

CREATE INDEX idx_summaries_game_variant ON price_summaries(game_id, variant, period_type);

-- ---------------------------------------------------------------------------
-- UTILITY: auto-update updated_at on row changes
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_games_updated_at
    BEFORE UPDATE ON games
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_listings_updated_at
    BEFORE UPDATE ON listings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_queue_updated_at
    BEFORE UPDATE ON ingestion_queue
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- SEED DATA: Platforms
-- ebay_category_id values are Browse API category_ids for Video Games subcats.
-- Verify / extend these against the eBay category tree before running.
-- ---------------------------------------------------------------------------
INSERT INTO platforms (name, short_name, ebay_category_id) VALUES
    ('Nintendo Entertainment System',   'NES',      '131761'),
    ('Super Nintendo',                  'SNES',     '131763'),
    ('Nintendo 64',                     'N64',      '131762'),
    ('Sega Genesis',                    'Genesis',  '131286'),
    ('Sega Master System',              'SMS',      '131285'),
    ('Sega Saturn',                     'Saturn',   '131288'),
    ('Sega Dreamcast',                  'DC',       '131289'),
    ('Sony PlayStation',                'PS1',      '131292'),
    ('Sony PlayStation 2',              'PS2',      '131293'),
    ('Game Boy',                        'GB',       '131290'),
    ('Game Boy Color',                  'GBC',      '131291'),
    ('Game Boy Advance',                'GBA',      '131283'),
    ('Atari 2600',                      'Atari2600','131280'),
    ('TurboGrafx-16',                   'TG16',     '131284')
ON CONFLICT (short_name) DO NOTHING;

-- Seed the ingestion queue from the platform list
INSERT INTO ingestion_queue (platform_id, ebay_category_id, priority)
SELECT id, ebay_category_id, ROW_NUMBER() OVER (ORDER BY id)
FROM platforms
WHERE active = TRUE AND ebay_category_id IS NOT NULL
ON CONFLICT (platform_id) DO NOTHING;
