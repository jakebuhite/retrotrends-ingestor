# ---- Build stage ----
FROM python:3.12-slim AS builder

WORKDIR /build

# Install build deps
RUN pip install --no-cache-dir setuptools wheel

COPY pyproject.toml .
COPY ingestion/ ./ingestion/

# Install only production deps into a prefix we can copy
RUN pip install --no-cache-dir --prefix=/install .


# ---- Runtime stage ----
FROM python:3.12-slim

# Non-root user for security
RUN useradd --create-home --shell /bin/bash appuser

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /install /usr/local

# Copy source
COPY ingestion/ ./ingestion/

USER appuser

# Default: run both ingestion and status check.
# Override CMD in your ECS task definition or docker-compose.yml
# for a service that does only one job.
#
# Valid commands: ingest | check-status | both
CMD ["python", "-m", "ingestion.main", "both"]
