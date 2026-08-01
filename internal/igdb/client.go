package igdb

import (
	"context"
	"net/http"
	"time"
)

const (
	authURL = "https://id.twitch.tv/oauth2/token"
	apiURL  = "https://api.igdb.com/v4"

	// gameCubePlatformID is the IGDB platform ID for Nintendo GameCube.
	gameCubePlatformID = 21
)

// Client is a minimal IGDB API client with Twitch OAuth2 client credentials auth.
type Client struct {
	http         *http.Client
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
}

// Game holds the fields extracted from an IGDB game record.
type Game struct {
	ID          int
	Name        string
	Slug        string
	ReleaseYear *int16
	CoverURL    *string
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		http:         &http.Client{Timeout: 30 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// FetchGameCubeGames returns a single page of GameCube games from IGDB.
// offset is the number of records to skip; limit is the page size (max 500).
// Returns an empty slice when there are no more results.
//
// Calls: POST /games
// Body:  fields id,name,slug,first_release_date,cover.url;
//
//	where platforms = (21);
//	limit {limit}; offset {offset};
func (c *Client) FetchGameCubeGames(ctx context.Context, offset, limit int) ([]Game, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	// TODO: implement HTTP call and response parsing
	return nil, nil
}

// ensureToken fetches a new Twitch OAuth2 token if the current one is missing
// or within 60 seconds of expiry.
//
// Calls: POST https://id.twitch.tv/oauth2/token
// Params: client_id, client_secret, grant_type=client_credentials
func (c *Client) ensureToken(ctx context.Context) error {
	if c.accessToken != "" && time.Now().Add(60*time.Second).Before(c.tokenExpiry) {
		return nil
	}
	// TODO: implement token fetch and set c.accessToken / c.tokenExpiry
	return nil
}
