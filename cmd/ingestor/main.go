package main

import (
	"context"
	"fmt"
	"log"

	"github.com/alecthomas/kong"
	"github.com/joho/godotenv"

	"github.com/jakebuhite/retrotrends-ingestor/internal/config"
	"github.com/jakebuhite/retrotrends-ingestor/internal/db"
	"github.com/jakebuhite/retrotrends-ingestor/internal/jobs"
	"github.com/jakebuhite/retrotrends-ingestor/internal/platform"
)

type BackfillCmd struct {
	Platform string `help:"Retro platform to backfill from IGDB. Run 'ingestor platforms' for the full list." default:"${platformDefault}"`
}

func (c *BackfillCmd) Run(cfg *config.Config) error {
	plat, err := platform.BySlug(c.Platform)
	if err != nil {
		return err
	}

	pool := db.Connect(cfg.DatabaseURL)
	defer pool.Close()
	return jobs.Backfill(context.Background(), cfg, pool, plat)
}

type PlatformsCmd struct{}

func (c *PlatformsCmd) Run() error {
	for _, p := range platform.All {
		fmt.Printf("%-24s %s\n", p.Slug, p.Name)
	}
	return nil
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
	Backfill  BackfillCmd  `cmd:"" help:"Seed the games table from IGDB for a platform (safe to re-run)."`
	Ingest    IngestCmd    `cmd:"" help:"Search eBay for new listings for tracked games."`
	Revisit   RevisitCmd   `cmd:"" help:"Check pending listings for sold status."`
	Platforms PlatformsCmd `cmd:"" help:"List supported retro platforms and their --platform slugs."`
}

func main() {
	// Load .env for local development; no-op in production where vars are injected.
	_ = godotenv.Load()

	cfg := config.Load()

	ctx := kong.Parse(&cli,
		kong.Name("ingestor"),
		kong.Description("RetroTrends data ingestor."),
		kong.Bind(cfg),
		kong.Vars{"platformDefault": platform.DefaultSlug},
	)
	if err := ctx.Run(); err != nil {
		log.Fatal(err)
	}
}
