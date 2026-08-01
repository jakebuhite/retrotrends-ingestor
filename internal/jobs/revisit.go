package jobs

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jakebuhite/retrotrends-ingestor/internal/config"
	"github.com/jakebuhite/retrotrends-ingestor/internal/ebay"
	"github.com/jakebuhite/retrotrends-ingestor/internal/models"
)

// Revisit checks each pending listing against the eBay API and updates its status
// to sold or ended_unsold. Listings are processed oldest-last-checked-first so no
// listing is starved indefinitely. Stops when the API call budget is exhausted.
func Revisit(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool) error {
	client := ebay.NewClient(cfg.EbayClientID, cfg.EbayClientSecret)

	// ebay API limits are split evenly with the ingest job.
	revisitBudget := cfg.EbayDailyCallLimit / 2

	pending, err := fetchPendingListings(ctx, pool, revisitBudget)
	if err != nil {
		return err
	}

	log.Printf("revisit: checking %d pending listings (budget: %d API calls)", len(pending), revisitBudget)

	resolved, failed := 0, 0
	for _, listing := range pending {
		status, err := client.GetItemStatus(ctx, listing.EbayListingID)
		if err != nil {
			log.Printf("revisit: failed to check listing %s: %v", listing.EbayListingID, err)
			failed++
			continue
		}

		if err := applyListingStatus(ctx, pool, listing.ID, status); err != nil {
			log.Printf("revisit: failed to update listing %s: %v", listing.EbayListingID, err)
			failed++
			continue
		}
		resolved++
	}

	log.Printf("revisit complete: %d resolved, %d failed", resolved, failed)
	return nil
}

func fetchPendingListings(ctx context.Context, pool *pgxpool.Pool, limit int) ([]models.Listing, error) {
	rows, err := pool.Query(ctx, `SELECT id, ebay_listing_id FROM listings
		WHERE status = 'pending'
		ORDER BY last_checked_at ASC NULLS FIRST
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Listing])
}

func applyListingStatus(ctx context.Context, pool *pgxpool.Pool, listingID int64, status *ebay.ItemStatus) error {
	var err error
	if status.Sold {
		_, err = pool.Exec(ctx, `UPDATE listings
			SET status = 'sold', sold_at = $1, last_checked_at = NOW(), updated_at = NOW()
			WHERE id = $2`, status.SoldAt, listingID)
	} else {
		// Still active
		_, err = pool.Exec(ctx, `UPDATE listings
			SET last_checked_at = NOW(), updated_at = NOW()
			WHERE id = $1`, listingID)
	}
	return err
}
