.PHONY: deps build test lint run-backfill run-ingest run-revisit docker-build

deps:
	go mod tidy

build:
	go build -o ingestor ./cmd/ingestor

test:
	go test ./...

lint:
	go vet ./...

run-backfill: build
	./ingestor backfill

run-ingest: build
	./ingestor ingest

run-revisit: build
	./ingestor revisit

docker-build:
	docker build -t retrotrends-ingestor .
