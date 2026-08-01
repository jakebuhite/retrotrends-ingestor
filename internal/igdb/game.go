package igdb

import "time"

// Game holds the fields extracted from an IGDB game record.
type Game struct {
	ID          int
	Name        string
	Slug        string
	ReleaseYear *int16
	CoverURL    *string
}

// gameJSON mirrors the raw shape of an IGDB /games response entry.
type gameJSON struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	FirstReleaseDate *int64 `json:"first_release_date"`
	Cover            *struct {
		URL string `json:"url"`
	} `json:"cover"`
}

func (r gameJSON) toGame() Game {
	g := Game{ID: r.ID, Name: r.Name, Slug: r.Slug}

	if r.FirstReleaseDate != nil {
		year := int16(time.Unix(*r.FirstReleaseDate, 0).UTC().Year())
		g.ReleaseYear = &year
	}
	if r.Cover != nil && r.Cover.URL != "" {
		g.CoverURL = &r.Cover.URL
	}

	return g
}
