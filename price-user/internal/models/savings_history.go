package models

// SavingsHistoryItem is one past comparison shown in the Economy tab's
// "recent comparisons" list.
type SavingsHistoryItem struct {
	DateLabel  string  `json:"date_label"`
	MarketName string  `json:"market_name"`
	ItemsCount int     `json:"items_count"`
	TotalCost  float64 `json:"total_cost"`
	Savings    float64 `json:"savings"`
}
