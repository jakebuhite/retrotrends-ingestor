package models

import "time"

type Game struct {
	ID          int64
	IGDBID      int
	Title       string
	Slug        string
	Platform    string
	ReleaseYear *int16
	CoverURL    *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
