package jobs

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jakebuhite/retrotrends-ingestor/internal/config"
	"github.com/jakebuhite/retrotrends-ingestor/internal/ebay"
	"github.com/jakebuhite/retrotrends-ingestor/internal/models"
)

// Ingest searches eBay for GameCube listings for each game in the catalog and stores
// new listings as pending. It stops early if the daily API call budget is exhausted.
func Ingest(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool) error {
	client := ebay.NewClient(cfg.EbayClientID, cfg.EbayClientSecret)

	games, err := fetchAllGames(ctx, pool)
	if err != nil {
		return err
	}

	callsUsed := 0

	for _, game := range games {
		for page := 1; page <= cfg.IngestMaxPages; page++ {
			if callsUsed >= cfg.EbayDailyCallLimit {
				log.Printf("ingest: daily call limit reached (%d), stopping early", cfg.EbayDailyCallLimit)
				return nil
			}

			results, err := client.SearchGameCubeListings(ctx, game.Title, page)
			callsUsed++
			if err != nil {
				log.Printf("ingest: search error for %q page %d: %v", game.Title, page, err)
				break
			}
			if len(results) == 0 {
				break // no more pages for this game
			}

			inserted := 0
			for _, r := range results {
				if err := insertListingIfNew(ctx, pool, game.ID, r); err != nil {
					log.Printf("ingest: failed to insert listing %s: %v", r.ItemID, err)
					continue
				}
				inserted++
			}

			log.Printf("ingest: %q page %d — %d found, %d inserted (calls used: %d)",
				game.Title, page, len(results), inserted, callsUsed)
		}
	}

	log.Printf("ingest complete: %d API calls used", callsUsed)
	return nil
}

func fetchAllGames(ctx context.Context, pool *pgxpool.Pool) ([]models.Game, error) {
	// TODO: implement
	//
	// SELECT id, title FROM games ORDER BY title
	_ = pool
	return nil, nil
}

func insertListingIfNew(ctx context.Context, pool *pgxpool.Pool, gameID int64, r ebay.SearchResult) error {
	// TODO: implement
	//
	// condition := ebay.ParseCondition(r.Condition, r.Title)
	// listingType := parseListingType(r.ListingType)
	//
	// INSERT INTO listings
	//   (game_id, ebay_listing_id, raw_title, condition, listing_type,
	//    asking_price, currency, listing_url, listed_at, status)
	// VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending')
	// ON CONFLICT (ebay_listing_id) DO NOTHING
	_ = pool
	_ = gameID
	_ = r
	return nil
}

func parseListingType(ebayBuyingOption string) models.ListingType {
	switch ebayBuyingOption {
	case "AUCTION":
		return models.ListingTypeAuction
	case "BEST_OFFER":
		return models.ListingTypeBestOffer
	default:
		return models.ListingTypeBuyItNow
	}
}
