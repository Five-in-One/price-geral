package models

import "time"

// ListCompareItem is one line of the shopping list. Either a catalog item or
// any listing of it (product_id) identifies what the user wants.
type ListCompareItem struct {
	CatalogID string `json:"catalog_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// ListCompareRequest is the body of POST /api/comparisons/list.
type ListCompareRequest struct {
	Items []ListCompareItem `json:"items"`
	// AllowEquivalents lets a market fill a gap with another brand of the same
	// product (same kind and size) — but only for products where switching
	// brand is normal (rice, beans, sugar...), never for beer or wine. The
	// requested brand always wins when the market has it.
	AllowEquivalents bool `json:"allow_equivalents"`
	// Lat/Lng (both or neither) add each market's distance; with RadiusKm,
	// markets known to be farther away are left out.
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	RadiusKm *float64 `json:"radius_km"`
}

// ListLine is one list item as one market would sell it.
type ListLine struct {
	CatalogID     string    `json:"catalog_id"`
	Name          string    `json:"name"`
	Quantity      int       `json:"quantity"`
	ProductID     string    `json:"product_id"`
	ProductName   string    `json:"product_name"`
	Brand         string    `json:"brand,omitempty"`
	SizeLabel     string    `json:"size_label,omitempty"`
	Price         float64   `json:"price"`
	LineTotal     float64   `json:"line_total"`
	UnitPrice     *float64  `json:"unit_price"`
	UnitPriceUnit string    `json:"unit_price_unit,omitempty"`
	CollectedAt   time.Time `json:"collected_at"`
	// IsSubstitute: another brand of the requested product.
	IsSubstitute bool `json:"is_substitute"`
	// IsCheapest: no market sells this item for less.
	IsCheapest bool `json:"is_cheapest"`
}

// MissingItem is a list item a market doesn't sell.
type MissingItem struct {
	CatalogID string `json:"catalog_id"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
}

// MarketComparison is what the whole list costs at one market.
type MarketComparison struct {
	Market           Market   `json:"market"`
	DistanceKm       *float64 `json:"distance_km"`
	TotalCost        float64  `json:"total_cost"`
	IsComplete       bool     `json:"is_complete"`
	FoundItems       int      `json:"found_items"`
	TotalItems       int      `json:"total_items"`
	SubstitutesCount int      `json:"substitutes_count"`
	// Difference from the cheapest *complete* market; 0 for incomplete ones,
	// whose total covers fewer items and so can't be compared with it.
	Difference float64       `json:"difference"`
	Lines      []ListLine    `json:"lines"`
	Missing    []MissingItem `json:"missing"`
}

// ListItemSummary describes a requested item across all markets.
type ListItemSummary struct {
	CatalogID        string   `json:"catalog_id"`
	Name             string   `json:"name"`
	Brand            string   `json:"brand,omitempty"`
	SizeLabel        string   `json:"size_label,omitempty"`
	Quantity         int      `json:"quantity"`
	Available        bool     `json:"available"`
	CheapestMarketID string   `json:"cheapest_market_id,omitempty"`
	CheapestPrice    *float64 `json:"cheapest_price"`
}

// Stop is one market visited in the best combination.
type Stop struct {
	Market     Market     `json:"market"`
	DistanceKm *float64   `json:"distance_km"`
	Subtotal   float64    `json:"subtotal"`
	Lines      []ListLine `json:"lines"`
}

// BestCombination buys every item where it is cheapest, even if that means
// several markets.
type BestCombination struct {
	TotalCost float64 `json:"total_cost"`
	// SavingsVsBestSingle is how much less it costs than the cheapest single
	// market that has the whole list; nil when no single market does.
	SavingsVsBestSingle *float64 `json:"savings_vs_best_single"`
	Stops               []Stop   `json:"stops"`
}

// ListComparison is the answer to POST /api/comparisons/list.
type ListComparison struct {
	AllowEquivalents bool              `json:"allow_equivalents"`
	Items            []ListItemSummary `json:"items"`
	// Markets is ranked: markets with the whole list first, cheapest first;
	// then incomplete ones, most items first.
	Markets            []MarketComparison `json:"markets"`
	BestSingleMarketID string             `json:"best_single_market_id,omitempty"`
	BestCombination    *BestCombination   `json:"best_combination"`
	// MaxSavings: cheapest vs most expensive market among those with the
	// whole list; 0 with fewer than two of them.
	MaxSavings        float64 `json:"max_savings"`
	MaxSavingsPercent float64 `json:"max_savings_percent"`
}
