package jobs

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"

	"github.com/jakebuhite/retrotrends-ingestor/internal/config"
	"github.com/jakebuhite/retrotrends-ingestor/internal/igdb"
	"github.com/jakebuhite/retrotrends-ingestor/internal/platform"
)

// igdbRateLimit represents the max number of requests per second to the IGDB API.
const (
	igdbRateLimit = 4
	igdbPageSize  = 500
)

// Backfill fetches all games for the given platform from IGDB and upserts
// them into the games table. It is safe to re-run; existing rows are updated
// in place via ON CONFLICT (igdb_id).
func Backfill(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, plat platform.Platform) error {
	client := igdb.NewClient(cfg.IGDBClientID, cfg.IGDBClientSecret)
	limiter := rate.NewLimiter(rate.Limit(igdbRateLimit), 1)
	offset := 0
	total := 0

	for {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}

		games, err := client.FetchGames(ctx, plat.ID, offset, igdbPageSize)
		if err != nil {
			return err
		}

		log.Printf("backfill: fetched %d games (offset: %d)", len(games), offset)
		if len(games) == 0 {
			break
		}

		for _, g := range games {
			if err := upsertGame(ctx, pool, g, plat.Name); err != nil {
				return err
			}
		}

		total += len(games)
		log.Printf("backfill: upserted %d games (total: %d)", len(games), total)

		if len(games) < igdbPageSize {
			break
		}
		offset += igdbPageSize
	}

	log.Printf("backfill complete: %d games total", total)
	return nil
}

func upsertGame(ctx context.Context, pool *pgxpool.Pool, g igdb.Game, platformName string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO games (igdb_id, title, slug, platform, release_year, cover_url, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (igdb_id) DO UPDATE SET
			title = EXCLUDED.title,
			slug = EXCLUDED.slug,
			release_year = EXCLUDED.release_year,
			cover_url = EXCLUDED.cover_url,
			updated_at = NOW()
	`, g.ID, g.Name, g.Slug, platformName, g.ReleaseYear, g.CoverURL)
	return err
}
