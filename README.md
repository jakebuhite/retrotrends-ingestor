# retrotrends-ingestor

eBay ingestion service for **RetroTrends**, a historical pricing service for retro video games.

This service is responsible for two jobs:

- **Ingestion** — sweeps eBay categories by platform, upserts active listings into PostgreSQL, and fuzzy-matches each listing title to a canonical game in the catalog.
- **Status Check** — periodically checks active listings to detect sales and record the final sold price.

---

## Quick start (local)

**Prerequisites:** Docker, Python 3.11+

```bash
# 1. Clone and enter the repo
git clone https://github.com/your-org/retrotrends-ingestor.git
cd retrotrends-ingestor

# 2. Set up environment variables
cp .env.example .env
# Edit .env — add your EBAY_CLIENT_ID and EBAY_CLIENT_SECRET

# 3. Start local Postgres (applies migrations automatically)
docker compose up -d db

# 4. Install Python dependencies
make dev

# 5. Seed the games catalog (required before matching works)
# See: docs/seeding.md — import from IGDB or a CSV export

# 6. Run one ingestion sweep
make run-ingest

# 7. Run the status checker
make run-check
```

---

## Running without Docker

```bash
pip install -e ".[dev]"
cp .env.example .env
# fill in .env values

# Apply migrations manually
make migrate

python -m ingestion.main ingest        # sweep next due platform
python -m ingestion.main check-status  # check active listings
python -m ingestion.main both          # run both in sequence
```

---

## Configuration

All configuration is via environment variables (or a `.env` file for local dev):

| Variable             | Required | Description                                      |
|----------------------|----------|--------------------------------------------------|
| `EBAY_CLIENT_ID`     | Yes      | eBay App ID from the developer portal            |
| `EBAY_CLIENT_SECRET` | Yes      | eBay Cert ID                                     |
| `DATABASE_URL`       | Yes      | PostgreSQL DSN                                   |
| `LOG_LEVEL`          | No       | `DEBUG` / `INFO` / `WARNING` (default: `INFO`)   |

eBay developer credentials: https://developer.ebay.com/my/keys

---

## Development

```bash
make dev        # install all deps including dev tools
make test       # run the test suite
make lint       # ruff linter
make fmt        # auto-format with ruff
make typecheck  # mypy
make check      # lint + typecheck + test in one go
```

CI runs on every push and pull request via GitHub Actions (`.github/workflows/ci.yml`).

---

## Database migrations

Migrations live in `migrations/` and are numbered sequentially. Apply them in order:

```bash
make migrate    # applies all migrations/*.sql to $DATABASE_URL
```

When running locally via docker compose, migrations are applied automatically on first start.

---

## Deployment (AWS)

The service is designed to run as an **ECS Fargate task** triggered by **EventBridge Scheduler**.

High-level steps:

1. Push the Docker image to ECR.
2. Create an ECS task definition pointing to the image; pass env vars via Secrets Manager / Parameter Store.
3. Create two EventBridge Scheduler rules — one for ingestion (e.g. every 6h), one for the status checker (every 12h).
4. Ensure the ECS task's IAM role has no outbound restrictions beyond what's needed (eBay API + RDS).
