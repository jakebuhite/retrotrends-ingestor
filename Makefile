.PHONY: deps build test lint cover run-backfill run-ingest run-revisit docker-build

deps:
	go mod tidy

build:
	go build -o ingestor ./cmd/ingestor

test:
	go test ./...

lint:
	go vet ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

run-backfill: build
	./ingestor backfill

run-ingest: build
	./ingestor ingest

run-revisit: build
	./ingestor revisit

docker-build:
	docker build -t retrotrends-ingestor .
