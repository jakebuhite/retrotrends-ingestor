# API Reference

The RetroTrends API is a read-only REST API that exposes game catalog data and historical sold price information. It is served by the Java/Spring Boot service behind an Application Load Balancer.

**Base URL:** `https://api.retrotrends.example.com/v1`

**Format:** All responses are JSON. All timestamps are ISO 8601 in UTC. All prices are in USD unless otherwise noted.

**Pagination:** List endpoints accept `page` (0-indexed, default `0`) and `size` (default `20`, max `100`) query parameters. Paginated responses include a `pagination` envelope.

---

## Games

### List Games

Returns all tracked games, across all backfilled platforms.

```
GET /v1/games
```

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | `0` | Page number (0-indexed) |
| `size` | integer | `20` | Results per page (max 100) |
| `sort` | string | `title` | Sort field: `title`, `release_year` |

**Response:**

```json
{
  "data": [
    {
      "id": 42,
      "slug": "super-mario-sunshine",
      "title": "Super Mario Sunshine",
      "platform": "Nintendo GameCube",
      "release_year": 2002,
      "cover_url": "https://images.igdb.com/igdb/image/upload/t_cover_big/co1234.jpg"
    }
  ],
  "pagination": {
    "page": 0,
    "size": 20,
    "total_elements": 187,
    "total_pages": 10
  }
}
```

---

### Get Game

Returns a single game by ID or slug.

```
GET /v1/games/{id}
GET /v1/games/{slug}
```

**Response:**

```json
{
  "id": 42,
  "slug": "super-mario-sunshine",
  "title": "Super Mario Sunshine",
  "platform": "Nintendo GameCube",
  "release_year": 2002,
  "cover_url": "https://images.igdb.com/igdb/image/upload/t_cover_big/co1234.jpg",
  "price_summary": {
    "last_sold_at": "2024-11-15T14:22:00Z",
    "last_sold_price": 24.99,
    "avg_price_30d": 26.50,
    "sale_count_30d": 12
  }
}
```

The `price_summary` reflects all conditions combined. Returns `null` if no sold listings exist yet.

---

### Search Games

Full-text search over game titles using trigram similarity.

```
GET /v1/games/search
```

**Query parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | Yes | Search query (min 2 characters) |
| `page` | integer | No | Default `0` |
| `size` | integer | No | Default `20` |

**Response:** Same envelope as List Games.

**Example:** `GET /v1/games/search?q=zelda`

---

## Prices

### Price History

Returns the full list of confirmed sold listings for a game, ordered newest-first.

```
GET /v1/games/{id}/prices
```

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from` | date (YYYY-MM-DD) | — | Filter: sold on or after this date |
| `to` | date (YYYY-MM-DD) | — | Filter: sold on or before this date |
| `condition` | string | — | Filter: `sealed`, `cib`, `loose`, `unknown` |
| `page` | integer | `0` | Page number |
| `size` | integer | `50` | Results per page (max 200) |

**Response:**

```json
{
  "game_id": 42,
  "data": [
    {
      "sold_price": 24.99,
      "sold_at": "2024-11-15T14:22:00Z",
      "condition": "loose",
      "listing_type": "auction",
      "listing_url": "https://www.ebay.com/itm/395432918765"
    },
    {
      "sold_price": 45.00,
      "sold_at": "2024-11-10T09:11:00Z",
      "condition": "cib",
      "listing_type": "buy_it_now",
      "listing_url": "https://www.ebay.com/itm/394812033417"
    }
  ],
  "pagination": {
    "page": 0,
    "size": 50,
    "total_elements": 134,
    "total_pages": 3
  }
}
```

---

### Latest Price

Returns the most recent confirmed sold price, optionally filtered by condition.

```
GET /v1/games/{id}/prices/latest
```

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `condition` | string | Optional filter: `sealed`, `cib`, `loose` |

**Response:**

```json
{
  "game_id": 42,
  "condition": "loose",
  "sold_price": 24.99,
  "sold_at": "2024-11-15T14:22:00Z",
  "listing_url": "https://www.ebay.com/itm/395432918765"
}
```

Returns `404` if no sold listings exist for the given condition (or overall if no condition filter is provided).

---

### Price Summary

Returns aggregate statistics over a rolling time window.

```
GET /v1/games/{id}/prices/summary
```

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `days` | integer | `30` | Rolling window in days (max 365) |
| `condition` | string | — | Optional filter: `sealed`, `cib`, `loose`, `unknown` |

**Response:**

```json
{
  "game_id": 42,
  "condition": null,
  "period_days": 30,
  "avg_price": 26.50,
  "min_price": 18.00,
  "max_price": 55.00,
  "sale_count": 12,
  "from": "2024-10-16",
  "to": "2024-11-15"
}
```

Returns `null` for all price fields (not `404`) if no sales exist in the window.

---

### Price by Condition

Returns a price summary broken down by condition for a single game.

```
GET /v1/games/{id}/prices/by-condition
```

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `days` | integer | `30` | Rolling window in days (max 365) |

**Response:**

```json
{
  "game_id": 42,
  "period_days": 30,
  "conditions": {
    "sealed": {
      "avg_price": 120.00,
      "min_price": 99.00,
      "max_price": 145.00,
      "sale_count": 3
    },
    "cib": {
      "avg_price": 42.00,
      "min_price": 35.00,
      "max_price": 55.00,
      "sale_count": 7
    },
    "loose": {
      "avg_price": 22.50,
      "min_price": 18.00,
      "max_price": 27.00,
      "sale_count": 14
    },
    "unknown": {
      "avg_price": 20.00,
      "min_price": 20.00,
      "max_price": 20.00,
      "sale_count": 1
    }
  }
}
```

Conditions with no sales in the window are omitted from the response.

---

## Error Responses

All errors follow a consistent envelope:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Game with id 999 not found."
  }
}
```

| HTTP Status | Code | When |
|-------------|------|------|
| `400` | `BAD_REQUEST` | Invalid query parameters (e.g. `days` out of range) |
| `404` | `NOT_FOUND` | Resource does not exist |
| `500` | `INTERNAL_ERROR` | Unexpected server error |

---

## Design Notes

**Why is the API read-only?**
All writes are owned exclusively by the ingestor. Separating read and write paths prevents the API from accidentally corrupting ingestor state and makes the API trivially cacheable in future.

**Why no authentication?**
Not required for an MVP public price-history service. Authentication can be layered in later (e.g. API key via API Gateway) without changing the Spring Boot service.

**Why are slugs supported alongside numeric IDs on `GET /v1/games/{id}`?**
Slugs are human-readable and stable (IGDB slugs don't change for existing games), making them better for external links. Numeric IDs are more efficient for internal use.
