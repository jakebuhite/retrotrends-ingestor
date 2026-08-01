# RetroTrends Ingestor

Go service that populates the RetroTrends database with GameCube game data from IGDB and historical sold prices from eBay.

## Prerequisites

- Go 1.23+
- PostgreSQL 16 (local or via Docker)
- eBay developer account — [API call limits](https://developer.ebay.com/develop/get-started/api-call-limits)
- IGDB developer account

## Local Setup

```bash
# 1. Install dependencies
make deps

# 2. Copy and fill in credentials
cp .env.sample .env

# 3. Start a local Postgres instance (adjust connection string in .env as needed)
docker run -d \
  --name retrotrends-pg \
  -e POSTGRES_USER=retrotrends \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=retrotrends \
  -p 5432:5432 \
  postgres:16-alpine

# 4. Apply the schema (migration lives in the retrotrends-api repo)
psql postgres://retrotrends:password@localhost:5432/retrotrends -f path/to/api/migrations/001_initial_schema.sql
```

## Running the Jobs

```bash
# Seed the games table from IGDB (run once before ingesting)
make run-backfill

# Search eBay for new GameCube listings
make run-ingest

# Check pending listings for sold status
make run-revisit
```

Or build the binary and run subcommands directly:

```bash
make build
./ingestor backfill
./ingestor ingest
./ingestor revisit
./ingestor --help
```

## Project Structure

```
cmd/ingestor/       entry point and CLI subcommand wiring
internal/
  config/           environment variable loading
  db/               PostgreSQL connection pool
  models/           shared Go types (Game, Listing, enums)
  ebay/             eBay Browse API client + condition parsing
  igdb/             IGDB API client
  jobs/             job implementations (backfill, ingest, revisit)
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `EBAY_CLIENT_ID` | Yes | — | eBay OAuth client ID |
| `EBAY_CLIENT_SECRET` | Yes | — | eBay OAuth client secret |
| `IGDB_CLIENT_ID` | Yes | — | IGDB (Twitch) client ID |
| `IGDB_CLIENT_SECRET` | Yes | — | IGDB (Twitch) client secret |
| `EBAY_DAILY_CALL_LIMIT` | No | `5000` | Maximum eBay API calls per day |
| `INGEST_MAX_PAGES` | No | `5` | Maximum eBay search pages per game |

## Docker

```bash
make docker-build
docker run --env-file .env retrotrends-ingestor ingest
```
