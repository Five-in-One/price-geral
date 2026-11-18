package models

// MonthlyPoint is one bar of the "savings per month" chart on the Economy
// tab.
type MonthlyPoint struct {
	Label    string  `json:"label"`
	Value    float64 `json:"value"`
	IsActive bool    `json:"is_active"`
}

// EconomySummary aggregates the headline numbers shown on the Economy tab.
type EconomySummary struct {
	TotalSaved          float64        `json:"total_saved"`
	MonthLabel          string         `json:"month_label"`
	MonthSpend          float64        `json:"month_spend"`
	MonthSavings        float64        `json:"month_savings"`
	MonthSavingsPercent float64        `json:"month_savings_percent"`
	MonthlyChart        []MonthlyPoint `json:"monthly_chart"`
}
