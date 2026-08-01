package main

import (
	"context"
	"log"

	"github.com/alecthomas/kong"
	"github.com/joho/godotenv"

	"github.com/jakebuhite/retrotrends-ingestor/internal/config"
	"github.com/jakebuhite/retrotrends-ingestor/internal/db"
	"github.com/jakebuhite/retrotrends-ingestor/internal/jobs"
)

type BackfillCmd struct{}

func (c *BackfillCmd) Run(cfg *config.Config) error {
	pool := db.Connect(cfg.DatabaseURL)
	defer pool.Close()
	return jobs.Backfill(context.Background(), cfg, pool)
}

type IngestCmd struct{}

func (c *IngestCmd) Run(cfg *config.Config) error {
	pool := db.Connect(cfg.DatabaseURL)
	defer pool.Close()
	return jobs.Ingest(context.Background(), cfg, pool)
}

type RevisitCmd struct{}

func (c *RevisitCmd) Run(cfg *config.Config) error {
	pool := db.Connect(cfg.DatabaseURL)
	defer pool.Close()
	return jobs.Revisit(context.Background(), cfg, pool)
}

var cli struct {
	Backfill BackfillCmd `cmd:"" help:"Seed the games table from IGDB (safe to re-run)."`
	Ingest   IngestCmd   `cmd:"" help:"Search eBay for new GameCube listings."`
	Revisit  RevisitCmd  `cmd:"" help:"Check pending listings for sold status."`
}

func main() {
	// Load .env for local development; no-op in production where vars are injected.
	_ = godotenv.Load()

	cfg := config.Load()

	ctx := kong.Parse(&cli,
		kong.Name("ingestor"),
		kong.Description("RetroTrends data ingestor."),
		kong.Bind(cfg),
	)
	if err := ctx.Run(); err != nil {
		log.Fatal(err)
	}
}
