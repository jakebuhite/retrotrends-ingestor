package jobs

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"

	"github.com/jakebuhite/retrotrends-ingestor/internal/config"
	"github.com/jakebuhite/retrotrends-ingestor/internal/igdb"
)

// igdbRateLimit represents the max number of requests per second to the IGDB API.
const igdbRateLimit = 4

// Backfill fetches all GameCube games from IGDB and upserts them into the games table.
// It is safe to re-run; existing rows are updated in place via ON CONFLICT (igdb_id).
func Backfill(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool) error {
	client := igdb.NewClient(cfg.IGDBClientID, cfg.IGDBClientSecret)
	limiter := rate.NewLimiter(rate.Limit(igdbRateLimit), 1)

	const pageSize = 500
	offset := 0
	total := 0

	for {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}

		games, err := client.FetchGameCubeGames(ctx, offset, pageSize)
		if err != nil {
			return err
		}
		if len(games) == 0 {
			break
		}

		for _, g := range games {
			if err := upsertGame(ctx, pool, g); err != nil {
				return err
			}
		}

		total += len(games)
		log.Printf("backfill: upserted %d games (total: %d)", len(games), total)

		if len(games) < pageSize {
			break
		}
		offset += pageSize
	}

	log.Printf("backfill complete: %d games total", total)
	return nil
}

func upsertGame(ctx context.Context, pool *pgxpool.Pool, g igdb.Game) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO games (igdb_id, title, slug, platform, release_year, cover_url, updated_at)
		VALUES ($1, $2, $3, 'Nintendo GameCube', $4, $5, NOW())
		ON CONFLICT (igdb_id) DO UPDATE SET
			title = EXCLUDED.title,
			slug = EXCLUDED.slug,
			release_year = EXCLUDED.release_year,
			cover_url = EXCLUDED.cover_url,
			updated_at = NOW()
	`, g.ID, g.Name, g.Slug, g.ReleaseYear, g.CoverURL)
	return err
}
