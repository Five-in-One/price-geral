package models

// CompareItem is one line of a comparison request: a product and how many
// units of it are in the user's shopping list.
type CompareItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// CompareRequest is the body of POST /api/comparisons.
type CompareRequest struct {
	Items []CompareItem `json:"items"`
}

// ItemPrice is the price of a single product within one market's result.
type ItemPrice struct {
	ProductID string  `json:"product_id"`
	Price     float64 `json:"price"`
}

// ComparisonResult is one market's total cost for the requested shopping
// list, ranked against the other markets.
type ComparisonResult struct {
	Market            Market      `json:"market"`
	TotalCost         float64     `json:"total_cost"`
	MatchedItemsCount int         `json:"matched_items_count"`
	Difference        float64     `json:"difference"`
	Items             []ItemPrice `json:"items"`
}
