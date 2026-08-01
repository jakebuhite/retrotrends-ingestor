package models

import "time"

type Condition string

const (
	ConditionSealed  Condition = "sealed"
	ConditionCIB     Condition = "cib"
	ConditionLoose   Condition = "loose"
	ConditionUnknown Condition = "unknown"
)

type ListingType string

const (
	ListingTypeAuction    ListingType = "auction"
	ListingTypeBuyItNow  ListingType = "buy_it_now"
	ListingTypeBestOffer ListingType = "best_offer"
)

type ListingStatus string

const (
	ListingStatusPending      ListingStatus = "pending"
	ListingStatusSold         ListingStatus = "sold"
	ListingStatusEndedUnsold  ListingStatus = "ended_unsold"
)

type Listing struct {
	ID             int64
	GameID         int64
	EbayListingID  string
	RawTitle       string
	Condition      Condition
	ListingType    ListingType
	AskingPrice    *float64
	Currency       string
	ListingURL     *string
	ListedAt       *time.Time
	Status         ListingStatus
	SoldPrice      *float64
	SoldAt         *time.Time
	LastCheckedAt  *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
