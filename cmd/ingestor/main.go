package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/jakebuhite/retrotrends-ingestor/internal/config"
	"github.com/jakebuhite/retrotrends-ingestor/internal/db"
	"github.com/jakebuhite/retrotrends-ingestor/internal/jobs"
)

func main() {
	// Load .env for local development; no-op in production where vars are injected.
	_ = godotenv.Load()

	cfg := config.Load()

	root := &cobra.Command{
		Use:   "ingestor",
		Short: "RetroTrends data ingestor",
	}

	root.AddCommand(
		newBackfillCmd(cfg),
		newIngestCmd(cfg),
		newRevisitCmd(cfg),
	)

	if err := root.ExecuteContext(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func newBackfillCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "backfill",
		Short: "Seed the games table from IGDB (safe to re-run)",
		RunE: func(cmd *cobra.Command, args []string) error {
			pool := db.Connect(cfg.DatabaseURL)
			defer pool.Close()
			return jobs.Backfill(cmd.Context(), cfg, pool)
		},
	}
}

func newIngestCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "ingest",
		Short: "Search eBay for new GameCube listings",
		RunE: func(cmd *cobra.Command, args []string) error {
			pool := db.Connect(cfg.DatabaseURL)
			defer pool.Close()
			return jobs.Ingest(cmd.Context(), cfg, pool)
		},
	}
}

func newRevisitCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "revisit",
		Short: "Check pending listings for sold status",
		RunE: func(cmd *cobra.Command, args []string) error {
			pool := db.Connect(cfg.DatabaseURL)
			defer pool.Close()
			return jobs.Revisit(cmd.Context(), cfg, pool)
		},
	}
}
