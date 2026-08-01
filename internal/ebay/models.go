package ebay

import (
	"fmt"
	"strconv"
	"time"
)

// searchResponse mirrors the envelope returned by the Browse API item_summary/search endpoint.
type searchResponse struct {
	Total         int              `json:"total"`
	ItemSummaries []itemSummaryDTO `json:"itemSummaries"`
}

type itemSummaryDTO struct {
	ItemID string `json:"itemId"`
	Title  string `json:"title"`
	Price  struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"price"`
	ItemWebURL       string    `json:"itemWebUrl"`
	Condition        string    `json:"condition"`
	BuyingOptions    []string  `json:"buyingOptions"`
	ItemCreationDate time.Time `json:"itemCreationDate"`
}

// toSearchResult converts a raw API item summary into the SearchResult shape used
// by the rest of the ingestor.
func (dto itemSummaryDTO) toSearchResult() (SearchResult, error) {
	price, err := strconv.ParseFloat(dto.Price.Value, 64)
	if err != nil {
		return SearchResult{}, fmt.Errorf("parsing price %q for item %s: %w", dto.Price.Value, dto.ItemID, err)
	}

	var listingType string
	if len(dto.BuyingOptions) > 0 {
		listingType = dto.BuyingOptions[0]
	}

	return SearchResult{
		ItemID:      dto.ItemID,
		Title:       dto.Title,
		Price:       price,
		Currency:    dto.Price.Currency,
		ListingURL:  dto.ItemWebURL,
		ListedAt:    dto.ItemCreationDate,
		Condition:   dto.Condition,
		ListingType: listingType,
	}, nil
}
