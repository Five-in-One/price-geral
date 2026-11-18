package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceuser/internal/models"
)

var ErrNoEconomySummary = errors.New("no economy summary configured")

type EconomyRepo struct {
	pool *pgxpool.Pool
}

func NewEconomyRepo(pool *pgxpool.Pool) *EconomyRepo {
	return &EconomyRepo{pool: pool}
}

// Summary returns the latest economy summary row together with the monthly
// chart series, ordered by their configured sort_order.
func (r *EconomyRepo) Summary(ctx context.Context) (models.EconomySummary, error) {
	var summary models.EconomySummary

	err := r.pool.QueryRow(ctx, `
		SELECT total_saved, month_label, month_spend, month_savings, month_savings_percent
		FROM economy_summary
		ORDER BY updated_at DESC
		LIMIT 1
	`).Scan(&summary.TotalSaved, &summary.MonthLabel, &summary.MonthSpend, &summary.MonthSavings, &summary.MonthSavingsPercent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.EconomySummary{}, ErrNoEconomySummary
		}
		return models.EconomySummary{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT label, value, is_active
		FROM monthly_savings
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return models.EconomySummary{}, err
	}
	defer rows.Close()

	chart := make([]models.MonthlyPoint, 0)
	for rows.Next() {
		var p models.MonthlyPoint
		if err := rows.Scan(&p.Label, &p.Value, &p.IsActive); err != nil {
			return models.EconomySummary{}, err
		}
		chart = append(chart, p)
	}
	if err := rows.Err(); err != nil {
		return models.EconomySummary{}, err
	}

	summary.MonthlyChart = chart
	return summary, nil
}
