package ebay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL              = "https://api.sandbox.ebay.com" // Prod will use "https://api.ebay.com"
	authURL              = "https://api.sandbox.ebay.com/identity/v1/oauth2/token"
	videoGamesCategoryID = "1249"
	pageLimit            = 200
)

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

type authResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// SearchGameCubeListings returns up to one page of active eBay listings matching
// the game title. page is 1-indexed; each page contains up to 50 results.
//
// Calls: GET /buy/browse/v1/item_summary/search
// Params: q="{title} gamecube", category_ids=1249, limit=200, offset=(page-1)*limit
func (c *Client) SearchGameCubeListings(ctx context.Context, gameTitle string, page int) ([]SearchResult, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	// Build URL with query parameters
	u, err := url.Parse(baseURL + "/buy/browse/v1/item_summary/search")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", fmt.Sprintf("%s gamecube", gameTitle))
	q.Set("category_ids", videoGamesCategoryID) // Video Games
	q.Set("limit", fmt.Sprint(pageLimit))
	q.Set("offset", fmt.Sprint((page-1)*pageLimit))
	u.RawQuery = q.Encode()

	// Make HTTP request
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+c.accessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &url.Error{
			Op:  "GET",
			URL: u.String(),
			Err: fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body),
		}
	}

	var raw searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(raw.ItemSummaries))
	for _, item := range raw.ItemSummaries {
		result, err := item.toSearchResult()
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// GetItemStatus fetches the current status of a single listing by its eBay item ID.
// Used by the revisit job to determine if a pending listing has sold.
//
// Calls: GET /buy/browse/v1/item/{itemId}
func (c *Client) GetItemStatus(ctx context.Context, itemID string) (*ItemStatus, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	u, err := url.Parse(baseURL + "/buy/browse/v1/item/" + url.PathEscape(itemID))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+c.accessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		now := time.Now()
		return &ItemStatus{ItemID: itemID, Sold: true, SoldAt: &now}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &url.Error{
			Op:  "GET",
			URL: u.String(),
			Err: fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body),
		}
	}

	return &ItemStatus{ItemID: itemID, Sold: false}, nil
}

// ensureToken fetches a new OAuth2 application token if the current one is missing
// or within 60 seconds of expiry.
//
// Calls: POST /identity/v1/oauth2/tokene
func (c *Client) ensureToken() error {
	if c.accessToken != "" && time.Now().Add(60*time.Second).Before(c.tokenExpiry) {
		return nil
	}

	formData := url.Values{}
	formData.Set("grant_type", "client_credentials")
	formData.Set("scope", "https://api.ebay.com/oauth/api_scope")
	encodedData := formData.Encode()

	resp, err := http.NewRequest(http.MethodPost, authURL, strings.NewReader(encodedData))
	if err != nil {
		return err
	}
	resp.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp.SetBasicAuth(c.clientID, c.clientSecret)

	response, err := http.DefaultClient.Do(resp)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return &url.Error{
			Op:  "POST",
			URL: authURL,
			Err: fmt.Errorf("unexpected status %d: %s", response.StatusCode, body),
		}
	}

	// Set c.accessToken and c.tokenExpiry
	var authResp authResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}
	c.accessToken = authResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	return nil
}
