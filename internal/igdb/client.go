package igdb

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
	authURL = "https://id.twitch.tv/oauth2/token"
	apiURL  = "https://api.igdb.com/v4"

	// gameCubePlatformID is the IGDB platform ID for Nintendo GameCube.
	gameCubePlatformID = 21
	gameTypes          = "0,3,8,11" // main, bundle, remake, port
)

// Client is a minimal IGDB API client with Twitch OAuth2 client credentials auth.
type Client struct {
	http         *http.Client
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
}

type authResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
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
func (c *Client) FetchGameCubeGames(ctx context.Context, offset, limit int) ([]Game, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	// Filtering criteria is IGDB query language in raw body.
	respBody := `fields id,name,slug,first_release_date,cover.url;
	where platforms = (` + fmt.Sprint(gameCubePlatformID) + `) & game_type = (` + gameTypes + `);
	sort id asc; limit ` + fmt.Sprint(limit) + `; offset ` + fmt.Sprint(offset) + `;`

	u := apiURL + "/games"

	resp, err := http.NewRequest(http.MethodPost, u, strings.NewReader(respBody))
	if err != nil {
		return nil, err
	}
	resp.Header.Add("Client-ID", c.clientID)
	resp.Header.Add("Authorization", "Bearer "+c.accessToken)

	response, err := http.DefaultClient.Do(resp)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, &url.Error{
			Op:  "POST",
			URL: u,
			Err: fmt.Errorf("unexpected status %d: %s", response.StatusCode, body),
		}
	}

	var raw []gameJSON
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		return nil, err
	}

	games := make([]Game, len(raw))
	for i, r := range raw {
		games[i] = r.toGame()
	}

	return games, nil
}

// ensureToken fetches a new Twitch OAuth2 token if the current one is missing
// or within 60 seconds of expiry.
//
// Calls: POST https://id.twitch.tv/oauth2/token
// Params: client_id, client_secret, grant_type=client_credentials
func (c *Client) ensureToken() error {
	if c.accessToken != "" && time.Now().Add(60*time.Second).Before(c.tokenExpiry) {
		return nil
	}

	// We need to refetch the token
	u, err := url.Parse(authURL)
	if err != nil {
		return err
	}
	params := u.Query()
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("grant_type", "client_credentials")

	u.RawQuery = params.Encode()

	resp, err := c.http.Post(u.String(), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &url.Error{
			Op:  "POST",
			URL: u.String(),
			Err: fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body),
		}
	}

	// Set c.accessToken / c.tokenExpiry
	var authResp authResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}
	c.accessToken = authResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	return nil
}
