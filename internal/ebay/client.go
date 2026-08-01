package ebay

import (
	"context"
	"net/http"
	"time"
)

const baseURL = "https://api.ebay.com"

// Client is a minimal eBay Browse API client with OAuth2 client credentials auth.
type Client struct {
	http         *http.Client
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
}

// SearchResult holds the fields extracted from a single eBay listing summary.
type SearchResult struct {
	ItemID      string
	Title       string
	Price       float64
	Currency    string
	ListingURL  string
	ListedAt    time.Time
	Condition   string // raw eBay condition label, e.g. "Used", "New"
	ListingType string // "AUCTION", "FIXED_PRICE", "BEST_OFFER"
}

// ItemStatus holds the resolved status of a previously-stored listing.
type ItemStatus struct {
	ItemID    string
	Sold      bool
	SoldPrice *float64
	SoldAt    *time.Time
	EndedAt   *time.Time
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		http:         &http.Client{Timeout: 30 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// SearchGameCubeListings returns up to one page of active eBay listings matching
// the game title. page is 1-indexed; each page contains up to 50 results.
//
// Calls: GET /buy/browse/v1/item_summary/search
// Params: q="{title} gamecube", category_ids=1249, limit=50, offset=(page-1)*50
func (c *Client) SearchGameCubeListings(ctx context.Context, gameTitle string, page int) ([]SearchResult, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	// TODO: implement HTTP call and response parsing
	return nil, nil
}

// GetItemStatus fetches the current status of a single listing by its eBay item ID.
// Used by the revisit job to determine if a pending listing has sold or ended unsold.
//
// Calls: GET /buy/browse/v1/item/v1|{itemID}|0
func (c *Client) GetItemStatus(ctx context.Context, itemID string) (*ItemStatus, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	// TODO: implement HTTP call and response parsing
	return nil, nil
}

// ensureToken fetches a new OAuth2 application token if the current one is missing
// or within 60 seconds of expiry.
//
// Calls: POST /identity/v1/oauth2/token
// Body:  grant_type=client_credentials&scope=https://api.ebay.com/oauth/api_scope
func (c *Client) ensureToken(ctx context.Context) error {
	if c.accessToken != "" && time.Now().Add(60*time.Second).Before(c.tokenExpiry) {
		return nil
	}
	// TODO: implement token fetch and set c.accessToken / c.tokenExpiry
	return nil
}
