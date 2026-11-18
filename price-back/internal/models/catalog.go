package models

import "time"

// CatalogOffer is one market's listing of a catalog item.
type CatalogOffer struct {
	ProductID     string    `json:"product_id"`
	ProductName   string    `json:"product_name"`
	MarketID      string    `json:"market_id"`
	MarketName    string    `json:"market_name"`
	Price         float64   `json:"price"`
	UnitPrice     *float64  `json:"unit_price"`
	UnitPriceUnit string    `json:"unit_price_unit,omitempty"`
	CollectedAt   time.Time `json:"collected_at"`
}

// CatalogItem is a real-world product — brand, kind and package size — with
// every market's offer of it, cheapest first.
type CatalogItem struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Brand        string   `json:"brand,omitempty"`
	GenericKey   string   `json:"generic_key"`
	SizeKind     string   `json:"size_kind,omitempty"`
	SizeAmount   *float64 `json:"size_amount"`
	SizeLabel    string   `json:"size_label,omitempty"`
	PackCount    int      `json:"pack_count"`
	SoldByWeight bool     `json:"sold_by_weight"`

	Offers       []CatalogOffer `json:"offers"`
	MarketsCount int            `json:"markets_count"`
	MinPrice     *float64       `json:"min_price"`
	MaxPrice     *float64       `json:"max_price"`
}

// CatalogSuggestion is a doubtful match waiting for a person: "is this
// listing the same product as that catalog item?".
type CatalogSuggestion struct {
	ID        int64        `json:"id"`
	Status    string       `json:"status"`
	Score     float64      `json:"score"`
	Reason    string       `json:"reason"` // "brand-missing" | "similar-name"
	CreatedAt time.Time    `json:"created_at"`
	Product   CatalogOffer `json:"product"`
	Item      CatalogItem  `json:"item"`
}

// MatchSummary is what a catalog matching run did.
type MatchSummary struct {
	Processed    int `json:"processed"`
	Linked       int `json:"linked"`
	ItemsCreated int `json:"items_created"`
	Suggestions  int `json:"suggestions"`
	Skipped      int `json:"skipped"`
}
