.PHONY: install dev test lint fmt typecheck run-ingest run-check run-both \
        docker-build docker-up docker-down docker-logs migrate clean

# ── Local development ──────────────────────────────────────────────────────

install:          ## Install production dependencies
	pip install .

dev:              ## Install all dependencies including dev tools
	pip install -e ".[dev]"

# ── Quality ────────────────────────────────────────────────────────────────

test:             ## Run the test suite
	pytest tests/ -v

lint:             ## Check code style with ruff
	ruff check ingestion/ tests/

fmt:              ## Auto-format code with ruff
	ruff format ingestion/ tests/

typecheck:        ## Run mypy type checks
	mypy ingestion/

check: lint typecheck test  ## Run all quality checks

# ── Run locally (requires .env) ────────────────────────────────────────────

run-ingest:       ## Run one ingestion sweep against the next due platform
	python -m ingestion.main ingest

run-check:        ## Run the status check job
	python -m ingestion.main check-status

run-both:         ## Run ingestion then status check
	python -m ingestion.main both

# ── Docker ─────────────────────────────────────────────────────────────────

docker-build:     ## Build the Docker image
	docker build -t retrotrends-ingestor:local .

docker-up:        ## Start local Postgres + ingestor via docker compose
	docker compose up -d

docker-down:      ## Stop and remove containers
	docker compose down

docker-logs:      ## Tail ingestor logs
	docker compose logs -f ingestor

# ── Database ───────────────────────────────────────────────────────────────

migrate:          ## Apply all migrations to $DATABASE_URL
	@if [ -z "$$DATABASE_URL" ]; then \
		echo "ERROR: DATABASE_URL is not set."; exit 1; \
	fi
	@for f in migrations/*.sql; do \
		echo "Applying $$f ..."; \
		psql "$$DATABASE_URL" -f "$$f"; \
	done

# ── Housekeeping ───────────────────────────────────────────────────────────

clean:            ## Remove build artefacts and caches
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	rm -rf .pytest_cache .mypy_cache .ruff_cache dist build *.egg-info

help:             ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
