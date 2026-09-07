package models

// Product is an item that can be searched for and priced across markets.
type Product struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Unit              string  `json:"unit"`
	Category          string  `json:"category"`
	DefaultMarketName string  `json:"default_market_name"`
	DefaultPrice      float64 `json:"default_price"`
	// Source is "seed" (demo data), "company" (self-registered via the
	// partner page) or "bot" (created by the admin's on-demand collection).
	Source string `json:"source"`
}
